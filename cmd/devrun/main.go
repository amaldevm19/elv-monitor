package main

import (
	"time"

	"elv-monitor/internal/models"
	"elv-monitor/internal/monitor"
	"elv-monitor/internal/storage"
)

func main() {
	logger := &storage.CSVLogger{BaseDir: "logs"}
	m := monitor.NewManager(logger)

	devs := []models.Device{
		{Id: "1", Name: "win-11", IP: "192.168.70.99", Enabled: true, Interval: 5, Mode: models.CheckModePingCmd},
		{Id: "2", Name: "Dev Server", IP: "192.168.70.166", Enabled: true, Interval: 5, Mode: models.CheckModePingCmd},
	}
	m.SetDevices(devs)
	m.Start()
	time.Sleep(120 * time.Second)
	m.Stop()
}
