package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
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
	store   *storage.SQLiteStore
}

type ImportResult struct {
	Total   int      `json:"total"`
	Added   int      `json:"added"`
	Updated int      `json:"updated"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors"`
}

func newUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// NewApp creates a new App application struct
func NewApp() *App {
	logger := &storage.CSVLogger{BaseDir: "logs"}
	mg := monitor.NewManager(logger)

	dbfilepath := filepath.Join("data", "elv-monitor.db")
	store, err := storage.NewSqliteStore(dbfilepath)
	if err != nil {
		panic(err) // for V1 dev; later return error nicely
	}
	return &App{
		devices: make(map[string]models.Device),
		manager: mg,
		store:   store,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	devs, err := a.store.ListDevices()
	if err != nil {
		return
	}
	a.mu.Lock()
	for _, d := range devs {
		a.devices[d.Id] = d
	}
	a.manager.SetDevices(a.deviceSliceUnsafe())
	a.mu.Unlock()
}

func (a *App) ListDevices() []models.Device {
	a.mu.RLock()
	defer a.mu.RUnlock()

	out := make([]models.Device, 0, len(a.devices))
	for _, d := range a.devices {
		out = append(out, d)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].IP < out[j].IP
		}
		return out[i].Name < out[j].Name
	})

	return out
}

func (a *App) AddDevice(d models.Device) error {
	if d.IP == "" || d.Name == "" {
		return errors.New("IP and Name are required")
	}

	if id, ok, err := a.store.GetIdByIP(d.IP); err != nil {
		return err
	} else if ok {
		d.Id = id // update existing record by IP
	} else {
		d.Id = newUID() // create new record
	}

	if d.Interval <= 0 {
		d.Interval = 10
	}
	d.Enabled = true

	if d.Mode == "" {
		d.Mode = models.CheckModePingCmd
	}
	a.mu.Lock()
	a.devices[d.Id] = d
	a.mu.Unlock()
	if err := a.store.UpsertDevice(d); err != nil {
		return err
	}

	a.mu.RLock()
	a.manager.SetDevices(a.deviceSliceUnsafe())
	a.mu.RUnlock()
	return nil

}

func (a *App) DeleteDevice(id string) error {
	if id == "" {
		return errors.New("ID is required")
	}
	// delete from db first
	if err := a.store.DeleteDevice(id); err != nil {
		return err
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

func (a *App) ImportDevices(devs []models.Device) (ImportResult, error) {
	res := ImportResult{Total: len(devs)}
	if len(devs) == 0 {
		return res, nil
	}
	existingByIP, err := a.store.MapIPToId()
	if err != nil {
		return res, err
	}
	a.mu.RLock()
	existing := make(map[string]struct{}, len(a.devices))
	for _, d := range a.devices {
		existing[d.Id] = struct{}{}
	}
	a.mu.RUnlock()
	clean := make([]models.Device, 0, len(devs))
	for i, d := range devs {
		if d.Name == "" || d.IP == "" {
			res.Failed++
			res.Errors = append(res.Errors, "row "+itoa(i+1)+": missing name or ip")
			continue
		}

		if d.Interval <= 0 {
			d.Interval = 10
		}
		if d.Mode == "" {
			d.Mode = models.CheckModePingCmd
		}
		if d.Mode == models.CheckModeTCP && d.TCPPort <= 0 {
			d.TCPPort = 80
		}
		if existingId, ok := existingByIP[d.IP]; ok {
			d.Id = existingId
			res.Updated++
		} else {
			d.Id = newUID()
			res.Added++
		}
		clean = append(clean, d)
	}

	if err := a.store.UpsertDevices(clean); err != nil {
		return res, err
	}
	devList, err := a.store.ListDevices()
	if err != nil {
		return res, nil
	}

	a.mu.Lock()
	a.devices = make(map[string]models.Device, len(devList))
	for _, d := range devList {
		a.devices[d.Id] = d
	}
	a.mu.Unlock()
	a.manager.SetDevices(a.deviceSliceUnsafe())
	return res, nil

}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
