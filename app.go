package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	//"log"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"elv-monitor/internal/models"
	"elv-monitor/internal/monitor"
	"elv-monitor/internal/storage"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/crypto/bcrypt"
)

var trayOnce sync.Once

// App struct
type App struct {
	ctx context.Context

	mu               sync.RWMutex
	devices          map[string]models.Device
	manager          *monitor.Manager
	store            *storage.SQLiteStore
	polling          bool
	windowVisible    bool
	menuStart        *menu.MenuItem
	menuStop         *menu.MenuItem
	menuAdd          *menu.MenuItem
	menuImport       *menu.MenuItem
	menuRefresh      *menu.MenuItem
	menuOpenLog      *menu.MenuItem
	menuShowAuditLog *menu.MenuItem
	currentUser      *models.UserDTO
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
	a.updateMenuState()
	trayOnce.Do(func() {
		go a.startTray()
	})
	a.setWindowVisible(true)
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

	a.manager.SetOnStateChange(func(d models.Device, prev, cur monitor.Status, r monitor.CheckResult) {
		//  emit a generic change event if you want:
		payload := map[string]any{
			"id":     d.Id,
			"ip":     d.IP,
			"prev":   string(prev),
			"cur":    string(cur),
			"reason": r.Reason,
		}

		if a.IsWindowVisible() {
			runtime.EventsEmit(a.ctx, "device:changed", payload)
		} else {
			runtime.WindowShow(a.ctx)
			runtime.WindowUnminimise(a.ctx)
			runtime.WindowExecJS(a.ctx, "window.focus()")
			// give window time to appear before toast
			go func() {
				time.Sleep(150 * time.Millisecond)
				runtime.EventsEmit(a.ctx, "device:changed", payload)
			}()
		}

	})

	has, _ := a.store.HasAnyUsers()
	if !has {
		runtime.EventsEmit(a.ctx, "auth:bootstrap")
	}

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
	if err := a.requireLogin(models.AuditDeviceAdd, d.IP); err != nil {
		return err
	}
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
	a.audit(models.AuditDeviceAdd, d.IP, fmt.Sprintf(`{"name":"%s"}`, d.Name))
	return nil

}

func (a *App) DeleteDevice(id string) error {
	if err := a.requireLogin(models.AuditDeviceDelete, "Device Delete"); err != nil {
		return err
	}
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

func (a *App) StartMonitoring() error {

	if err := a.requireLogin(models.AuditMonitorStart, ""); err != nil {
		return err
	}
	a.audit(models.AuditMonitorStart, "", "")
	// ensure manager has latest device list
	a.manager.SetDevices(a.deviceSliceUnsafe())
	a.SetPolling(true)

	a.manager.Start()
	return nil

}

func (a *App) StopMonitoring() error {
	if err := a.requireLogin(models.AuditMonitorStop, ""); err != nil {
		return err
	}
	a.audit(models.AuditMonitorStop, "", "")
	a.manager.Stop()
	a.SetPolling(false)
	return nil
}

func (a *App) QuitApp() {
	if err := a.requireLogin(models.AuditQuitAttempt, "app"); err != nil {
		a.audit(models.AuditQuitDenied, "app", "")
		return
	}
	if a.IsPolling() {
		btn, _ := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:          runtime.QuestionDialog,
			Title:         "Close ELV Monitoring",
			Message:       "Polling is currently running.\nDo you really want to quit?",
			Buttons:       []string{"Yes", "No"},
			DefaultButton: "No",
			CancelButton:  "No",
		})
		if btn != "Yes" {
			a.audit(models.AuditQuitCancel, "app", "")
			return
		}
	}
	a.audit(models.AuditQuitConfirm, "app", "")
	a.manager.Stop()
	runtime.Quit(a.ctx)
}

func (a *App) GetStatusSnapshot() map[string]monitor.DeviceState {
	return a.manager.Snapshot()
}

func (a *App) ImportDevices(devs []models.Device) (ImportResult, error) {
	if err := a.requireLogin(models.AuditDeviceImport, "csv"); err != nil {
		return ImportResult{}, err
	}
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

	a.audit(models.AuditDeviceImport, "csv", fmt.Sprintf(`{"total":%d,"added":%d,"updated":%d,"failed":%d}`,
		res.Total, res.Added, res.Updated, res.Failed))
	return res, nil

}

func (a *App) IsPolling() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.polling
}

func (a *App) SetPolling(v bool) {
	//log.Printf("[SetPolling] v=%v ctxNil=%v", v, a.ctx == nil)
	a.mu.Lock()
	a.polling = v
	a.mu.Unlock()
	a.updateMenuState()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "polling:changed", v)
	}
	if a.menuStart != nil && a.menuStop != nil && a.ctx != nil {
		if v {
			a.menuStart.Disable()
			a.menuStop.Enable()
		} else {
			a.menuStart.Enable()
			a.menuStop.Disable()
		}
		// Crucial step: Apply the changes to the UI
		runtime.MenuUpdateApplicationMenu(a.ctx)
	}

}

func (a *App) setWindowVisible(v bool) {
	a.mu.Lock()
	a.windowVisible = v
	a.mu.Unlock()
}

func (a *App) IsWindowVisible() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.windowVisible
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

func (a *App) MenuAddIpDevice() {
	runtime.EventsEmit(a.ctx, "ui:addDevice", nil)
}

func (a *App) MenuImportCSV() {
	runtime.EventsEmit(a.ctx, "ui:importCSV", nil)
}

func (a *App) MenuStartMonitoring() {
	a.StartMonitoring()
}

func (a *App) MenuStopMonitoring() {
	a.StopMonitoring()
}

func (a *App) MenuRefreshDevices() {
	runtime.EventsEmit(a.ctx, "ui:refreshDevice", nil)
}

func (a *App) MenuOpenLogsFolder() {
	if err := a.requireLogin(models.AuditOpenLogsFolder, "logs"); err != nil {
		return
	}
	a.audit(models.AuditOpenLogsFolder, "logs", "")
	openFolder("logs")
}

func (a *App) MenuShowAuditLog() {
	if err := a.requireLogin(models.AuditViewLogs, "audit_logs"); err != nil {
		return
	}
	a.audit(models.AuditViewLogs, "audit_logs", "")
	runtime.EventsEmit(a.ctx, "ui:showAudit", nil)
}

func (a *App) MenuAbout() {
	_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   "About ELV-Monitor",
		Message: "ELV Monitor\nV1 - IP monitoring tool \n\nCreated by @Amaldev Mahadevan",
	})
}

// User authentication methods

func (a *App) CreateFirstAdminUser(username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return errors.New("username and password required")
	}
	// Check if any users exist
	hasUsers, err := a.store.HasAnyUsers()
	if err != nil {
		return err
	}
	if hasUsers {
		return errors.New("admin already exists") // Already has users
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Create the first admin user
	user := models.User{
		Id:           newUID(),
		Username:     username,
		PasswordHash: string(hashedPassword),
		Role:         models.RoleAdmin,
		CreatedAt:    time.Now(),
	}
	err = a.store.CreateUser(user)
	if err != nil {
		return err
	}
	// Log the admin creation action
	auditLog := models.AuditLog{
		Id:        newUID(),
		Timestamp: time.Now(),
		UserId:    user.Id,
		Username:  user.Username,
		Action:    models.AuditAdminCreated,
	}
	err = a.store.InsertAuditLog(auditLog)
	if err != nil {
		//log.Printf("Failed to log admin creation: %v", err)
	}

	return nil
}

func (a *App) Login(username, password string) (*models.UserDTO, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, errors.New("username and password required")
	}
	user, ok, err := a.store.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}
	if !ok {
		a.auditLoginFail(username)
		return nil, errors.New("Invalid credentials")
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}
	a.currentUser = &models.UserDTO{
		Id:       user.Id,
		Username: user.Username,
		Role:     user.Role,
	}

	a.updateMenuState()
	_ = a.store.UpdateUserLastLogin(user.Id, time.Now().UTC())

	_ = a.store.InsertAuditLog(models.AuditLog{
		Id:        newUID(),
		Timestamp: time.Now().UTC(),
		UserId:    user.Id,
		Username:  user.Username,
		Action:    models.AuditLoginSuccess,
	})
	return a.currentUser, nil
}

func (a *App) auditLoginFail(username string) {
	_ = a.store.InsertAuditLog(models.AuditLog{
		Id:        newUID(),
		Timestamp: time.Now().UTC(),
		Username:  username,
		Action:    models.AuditLoginFail,
	})
}

func (a *App) CurrentUser() *models.UserDTO {
	return a.currentUser
}

func (a *App) Logout() {
	if a.currentUser == nil {
		return
	}
	_ = a.store.InsertAuditLog(models.AuditLog{
		Id:        newUID(),
		Timestamp: time.Now().UTC(),
		UserId:    a.currentUser.Id,
		Username:  a.currentUser.Username,
		Action:    models.AuditLogout,
	})

	a.currentUser = nil
	a.updateMenuState()
}

func (a *App) requireLogin(action models.AuditAction, target string) error {
	if a.currentUser != nil {
		return nil
	}

	_ = a.store.InsertAuditLog(models.AuditLog{
		Id:          newUID(),
		Timestamp:   time.Now().UTC(),
		Action:      models.AuditDenied,
		Target:      target,
		Username:    "",
		DetailsJSON: fmt.Sprintf(`{"required":"login","action":"%s"}`, action),
	})

	// Tell UI to open Login modal
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "auth:required", map[string]any{
			"action": string(action),
			"target": target,
		})
	}
	return errors.New("login required")
}

func (a *App) audit(action models.AuditAction, target string, detailsJSON string) {
	var uid, uname string
	if a.currentUser != nil {
		uid = a.currentUser.Id
		uname = a.currentUser.Username
	}

	_ = a.store.InsertAuditLog(models.AuditLog{
		Id:          newUID(),
		Timestamp:   time.Now().UTC(),
		UserId:      uid,
		Username:    uname,
		Action:      action,
		Target:      target,
		DetailsJSON: detailsJSON,
	})
}

func (a *App) HasAnyUser() (bool, error) { return a.store.HasAnyUsers() }

func (a *App) updateMenuState() {
	if a.ctx == nil {
		return
	}
	a.mu.RLock()
	loggedIn := a.currentUser != nil
	polling := a.IsPolling()
	a.mu.RUnlock()

	// helper
	set := func(it *menu.MenuItem, enabled bool) {
		if it == nil {
			return
		}
		if enabled {
			it.Enable()
		} else {
			it.Disable()
		}
	}
	// Restricted actions require login
	set(a.menuAdd, loggedIn)
	set(a.menuImport, loggedIn)
	set(a.menuOpenLog, loggedIn)

	// Start/Stop require login + depend on polling state
	set(a.menuStart, loggedIn && !polling)
	set(a.menuStop, loggedIn && polling)

	// Refresh: your choice
	// If you want refresh allowed even when logged out -> true
	// If you want refresh restricted -> loggedIn
	set(a.menuRefresh, true)

	// Apply changes
	runtime.MenuUpdateApplicationMenu(a.ctx)
}

func (a *App) GetAuditLogs(limit int) ([]models.AuditLog, error) {
	return a.store.ListAuditLogs(limit)
}
