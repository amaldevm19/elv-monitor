package monitor

import (
	"context"
	"elv-monitor/internal/models"
	"elv-monitor/internal/storage"
	"log"
	"sync"
	"time"
)

type DeviceState struct {
	LastStatus Status
	LastSeen   time.Time
	LastRTT    time.Duration
	LastReason string
}

type Manager struct {
	mu      sync.RWMutex
	devs    map[string]models.Device
	state   map[string]DeviceState
	running bool
	workers int
	icmp    ICMPChecker
	tcp     TCPChecker
	ping    PingCmdChecker
	logger  *storage.CSVLogger
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func NewManager(logger *storage.CSVLogger) *Manager {
	return &Manager{
		devs:    make(map[string]models.Device),
		state:   make(map[string]DeviceState),
		logger:  logger,
		icmp:    ICMPChecker{Timeout: 2 * time.Second},
		ping:    PingCmdChecker{Timeout: 2 * time.Second},
		tcp:     TCPChecker{Timeout: 2 * time.Second},
		workers: 10,
	}
}

func (m *Manager) SetDevices(devices []models.Device) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.devs = make(map[string]models.Device, len(devices))
	for _, d := range devices {
		m.devs[d.Id] = d
	}
}

func (m *Manager) Snapshot() map[string]DeviceState {
	m.mu.Lock()
	defer m.mu.Unlock()
	snapshot := make(map[string]DeviceState, len(m.state))
	for k, v := range m.state {
		snapshot[k] = v
	}
	return snapshot
}

func (m *Manager) Start() {
	log.Println("[Manager] Started monitoring")
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	deviceCount := len(m.devs)
	minWorkers := 5
	maxWorkers := 50
	// Adjust workers based on device count
	if deviceCount < minWorkers {
		m.workers = minWorkers
	} else if deviceCount > maxWorkers {
		m.workers = maxWorkers
	} else {
		m.workers = deviceCount
	}

	if deviceCount == 0 {
		m.running = false
		m.mu.Unlock()
		return
	}

	m.mu.Unlock()

	jobs := make(chan models.Device, 2000)

	for i := 0; i < m.workers; i++ {
		m.wg.Add(1)
		go func(workerID int) {
			defer m.wg.Done()
			log.Printf("[Worker-%d] started\n", workerID)
			for d := range jobs {
				log.Printf("[Worker-%d] checking %s (%s)\n", workerID, d.Name, d.IP)
				if !d.Enabled {
					continue
				}
				result := m.CheckAuto(ctx, d)
				m.ApplyResult(d, result)
			}
		}(i)
	}
	// Scheduler
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		lastrun := make(map[string]time.Time)

		for {
			select {
			case <-ctx.Done():
				close(jobs)
				return
			case <-ticker.C:
				m.mu.RLock()
				for _, d := range m.devs {

					if !d.Enabled {
						continue
					}
					iv := d.Interval
					if iv <= 0 {
						iv = 10
					}
					prev := lastrun[d.Id]
					if prev.IsZero() || time.Since(prev) >= time.Duration(iv)*time.Second {
						lastrun[d.Id] = time.Now()
						select {
						case jobs <- d:
							log.Printf("[Scheduler] Enqueue device %s (%s)\n", d.Name, d.IP)
						default:
							log.Println("[Scheduler] Job queue full, skipping")
							// Drop the job if the channel is full to avoid blocking
						}
					}
				}
				m.mu.RUnlock()
			}
		}
	}()
}

func (m *Manager) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	if m.cancel != nil {
		m.cancel()
	}
	m.mu.Unlock()
	m.wg.Wait()

}

func (m *Manager) CheckAuto(ctx context.Context, d models.Device) CheckResult {
	switch d.Mode {
	case models.CheckModeICMP:
		return m.icmp.Check(ctx, d)
	case models.CheckModeTCP:
		return m.tcp.Check(ctx, d)
	case models.CheckModePingCmd:
		return m.ping.Check(ctx, d)
	default:
		res := m.icmp.Check(ctx, d)
		if res.Reason == ErrPermission.Error() || res.Reason == "permission denied" {
			return m.ping.Check(ctx, d)
		}
		return res
	}

}

func (m *Manager) ApplyResult(d models.Device, r CheckResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prev := m.state[d.Id]

	isFirst := prev.LastSeen.IsZero()

	changed := !isFirst && prev.LastStatus != r.Status

	m.state[d.Id] = DeviceState{
		LastStatus: r.Status,
		LastSeen:   r.CheckedAt,
		LastRTT:    r.RTT,
		LastReason: r.Reason,
	}

	if changed {
		log.Printf(
			"[StateChange] %s (%s): %s -> %s\n",
			d.Name, d.IP, prev.LastStatus, r.Status,
		)
	}

	if changed && m.logger != nil {
		rttms := int64(0)
		if r.RTT > 0 {
			rttms = r.RTT.Milliseconds()
		}
		_ = m.logger.LogStateChange(d.Name, d.IP, string(r.Status), r.Reason, r.CheckedAt, rttms)

	}
}
