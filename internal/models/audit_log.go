package models

import "time"

type AuditAction string

const (
	AuditAdminCreated AuditAction = "ADMIN_CREATED"
	AuditLoginSuccess AuditAction = "LOGIN_SUCCESS"
	AuditLoginFail    AuditAction = "LOGIN_FAIL"
	AuditLogout       AuditAction = "LOGOUT"

	AuditMonitorStart AuditAction = "MONITOR_START"
	AuditMonitorStop  AuditAction = "MONITOR_STOP"

	AuditDeviceAdd    AuditAction = "DEVICE_ADD"
	AuditDeviceDelete AuditAction = "DEVICE_DELETE"
	AuditDeviceImport AuditAction = "DEVICE_IMPORT"

	AuditOpenLogsFolder AuditAction = "OPEN_LOGS_FOLDER"

	AuditQuitAttempt AuditAction = "QUIT_ATTEMPT"
	AuditQuitDenied  AuditAction = "QUIT_DENIED"
	AuditQuitCancel  AuditAction = "QUIT_CANCELLED"
	AuditQuitConfirm AuditAction = "QUIT_CONFIRMED"

	AuditViewLogs AuditAction = "VIEW_LOGS"

	// Generic denied action (optional pattern)
	AuditDenied AuditAction = "DENIED"
)

// AuditLog is the DB model for "who did what, when".
type AuditLog struct {
	Id          string      `json:"id" db:"id"`
	Timestamp   time.Time   `json:"ts" db:"ts"`
	UserId      string      `json:"userId,omitempty" db:"user_id"`
	Username    string      `json:"username,omitempty" db:"username"`
	Action      AuditAction `json:"action" db:"action"`
	Target      string      `json:"target,omitempty" db:"target"`            // e.g. deviceId, IP, filename
	DetailsJSON string      `json:"detailsJson,omitempty" db:"details_json"` // JSON string for extra info
}
