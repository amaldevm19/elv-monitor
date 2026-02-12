package main

import (
	"context"
	"errors"
	"sync"

	"elv-monitor/internal/models"
	"elv-monitor/internal/monitor"
	"elv-monitor/internal/storage"
)

// App struct
type App struct {
	ctx context.Context

	mu      sync.RWMutex
	devices map[string]models.Device
	manager *monitor.Manager
}

// NewApp creates a new App application struct
func NewApp() *App {
	logger := &storage.CSVLogger{BaseDir: "logs"}
	mg := monitor.NewManager(logger)
	return &App{
		devices: make(map[string]models.Device),
		manager: mg,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) ListDevices() []models.Device {
	a.mu.RLock()
	defer a.mu.RUnlock()

	out := make([]models.Device, 0, len(a.devices))
	for _, d := range a.devices {
		out = append(out, d)
	}
	return out
}

func (a *App) AddDevice(d models.Device) error {
	if d.ID == "" || d.IP == "" {
		return errors.New("Device ID and IP are required")
	}
	if d.Interval <= 0 {
		d.Interval = 10
	}
	d.Enabled = true

	if d.Mode == "" {
		d.Mode = models.CheckModePingCmd
	}
	a.mu.Lock()
	a.devices[d.ID] = d

	a.manager.SetDevices(a.deviceSliceUnsafe())

	a.mu.Unlock()
	return nil

}

func (a *App) DeleteDevice(id string) error {
	if id == "" {
		return errors.New("ID is required")
	}
	a.mu.Lock()
	delete(a.devices, id)
	a.manager.SetDevices(a.deviceSliceUnsafe())
	a.mu.Unlock()
	return nil
}

// helper: caller must hold a.mu Lock or RLock if you change it
func (a *App) deviceSliceUnsafe() []models.Device {
	out := make([]models.Device, 0, len(a.devices))
	for _, d := range a.devices {
		out = append(out, d)
	}
	return out

}

func (a *App) StartMonitoring() {
	// ensure manager has latest device list
	a.mu.RLock()
	a.manager.SetDevices(a.deviceSliceUnsafe())
	a.mu.RUnlock()

	a.manager.Start()

}

func (a *App) StopMonitoring() {
	a.manager.Stop()
}

func (a *App) GetStatusSnapshot() map[string]monitor.DeviceState {
	return a.manager.Snapshot()
}
