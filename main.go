package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	appMenu := buildAppMenu(app)

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "elv-monitor",
		Width:  1024,
		Height: 768,
		Menu:   appMenu,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour:  &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:         app.startup,
		HideWindowOnClose: true,
		//OnBeforeClose:     app.OnBeforeClose,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

func buildAppMenu(app *App) *menu.Menu {
	m := menu.NewMenu()

	//------IP Devices----------
	ipDev := m.AddSubmenu("IP Devices")

	ipDev.AddText("Add IP Device...", nil, func(_ *menu.CallbackData) {
		app.MenuAddIpDevice()
	})

	ipDev.AddText("Import CSV...", nil, func(_ *menu.CallbackData) {
		app.MenuImportCSV()
	})

	ipDev.AddSeparator()

	ipDev.AddText("Refresh Device List...", nil, func(_ *menu.CallbackData) {
		app.MenuRefreshDevices()
	})

	// ---- Monitoring IP devices----

	app.menuStart = ipDev.AddText("Start Monitoring", nil, func(_ *menu.CallbackData) {
		app.MenuStartMonitoring()
	})

	app.menuStop = ipDev.AddText("Stop Monitoring", nil, func(_ *menu.CallbackData) {
		app.MenuStopMonitoring()
	})

	app.menuStop.Disable()

	//----------Logs-----------
	logs := m.AddSubmenu("Logs")
	logs.AddText("Open Logs folder", nil, func(_ *menu.CallbackData) {
		app.MenuOpenLogsFolder()
	})
	logs.AddText("System Event Log...", nil, func(_ *menu.CallbackData) {
		app.MenuShowAuditLog()
	})

	//-------------Help-----------

	help := m.AddSubmenu("Help")
	help.AddText("About", nil, func(_ *menu.CallbackData) {
		app.MenuAbout()
	})

	return m
}
