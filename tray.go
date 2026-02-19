package main

import (
	_ "embed"
	"fmt"
	"log"
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
					runtime.WindowMinimise(a.ctx)
					a.setWindowVisible(true)
				case <-mStart.ClickedCh:
					a.StartMonitoring()
					mStart.Disable()
					mStop.Enable()
				case <-mStop.ClickedCh:
					a.StopMonitoring()
					mStart.Enable()
					mStop.Disable()
				case <-mHide.ClickedCh:
					runtime.WindowHide(a.ctx)
					a.setWindowVisible(false)
				case <-mOpenLog.ClickedCh:
					openFolder("logs")
				case <-mQuit.ClickedCh:
					a.mu.RLock()
					running := a.polling
					a.mu.RUnlock()
					if running {
						btn, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
							Type:          runtime.QuestionDialog,
							Title:         "Close ELV Monitoring",
							Message:       "Polling is currently running.\nDo you really want to quit?",
							Buttons:       []string{"Yes", "No"},
							DefaultButton: "No",
							CancelButton:  "No",
						})
						if err != nil || btn != "Yes" {
							continue
						}
					}
					a.manager.Stop()
					systray.Quit()
					runtime.Quit(a.ctx)
					return

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

	}, func() { log.Println("ELV Monitor Closed") })
}

func openFolder(path string) {
	abs, _ := filepath.Abs(path)
	_ = exec.Command("explorer", abs).Start()
}
