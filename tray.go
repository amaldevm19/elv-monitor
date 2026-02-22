package main

import (
	"elv-monitor/internal/models"
	_ "embed"
	"fmt"

	//"log"
	"os/exec"
	"path/filepath"
	"time"

	systray "github.com/ra1phdd/systray-on-wails"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed internal/assets/appicon.ico
var trayIcon []byte

func (a *App) startTray() {
	systray.Run(func() {
		systray.SetIcon(trayIcon)
		systray.SetTooltip("ELV-Monitor")

		mShow := systray.AddMenuItem("Show", "Show the window")
		mStart := systray.AddMenuItem("Start Monitoring", "Start polling")
		mStop := systray.AddMenuItem("Stop Monitoring", "Stop polling")
		mStop.Disable()
		mHide := systray.AddMenuItem("Hide", "Hide the window")
		mOpenLog := systray.AddMenuItem("Open Logs", "Open Logs folder")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit ELV Monitor")

		go func() {
			for {
				select {
				case <-mShow.ClickedCh:
					runtime.WindowShow(a.ctx)
					runtime.WindowUnminimise(a.ctx)
					a.setWindowVisible(true)
				case <-mStart.ClickedCh:
					if err := a.StartMonitoring(); err != nil {
						// optionally show dialog or emit auth-required event already done inside requireLogin
						continue
					}
					mStart.Disable()
					mStop.Enable()
				case <-mStop.ClickedCh:
					if err := a.StopMonitoring(); err != nil {
						continue
					}
					mStart.Enable()
					mStop.Disable()
				case <-mHide.ClickedCh:
					runtime.WindowHide(a.ctx)
					a.setWindowVisible(false)
				case <-mOpenLog.ClickedCh:
					if err := a.requireLogin(models.AuditOpenLogsFolder, "logs"); err != nil {
						continue
					}
					a.audit(models.AuditOpenLogsFolder, "logs", "")
					openFolder("logs")
				case <-mQuit.ClickedCh:
					a.QuitApp()
				}
			}
		}()

		go func() {
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				total, down := a.manager.Summary()
				systray.SetTooltip(fmt.Sprintf("ELV Monitor\n%d down / %d total", down, total))
			}
		}()

		go func() {
			t := time.NewTicker(500 * time.Millisecond)
			defer t.Stop()

			last := false
			for range t.C {
				running := a.IsPolling()
				if running == last {
					continue
				}
				last = running
				if running {
					mStart.Disable()
					mStop.Enable()
				} else {
					mStart.Enable()
					mStop.Disable()
				}
			}
		}()

	}, func() { /*log.Println("ELV Monitor Closed")*/ })
}

func openFolder(path string) {
	abs, _ := filepath.Abs(path)
	_ = exec.Command("explorer", abs).Start()
}
