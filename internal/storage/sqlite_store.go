package storage

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"elv-monitor/internal/models"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSqliteStore(dbpath string) (*SQLiteStore, error) {
	// check dbpath is empty or not
	if dbpath == "" {
		return nil, errors.New("db path is empty")
	}

	//Create folder if not exisit
	if err := os.MkdirAll(filepath.Dir(dbpath), 0755); err != nil {
		return nil, err
	}

	//open sql db in the filepath
	db, err := sql.Open("sqlite", dbpath)

	if err != nil {
		return nil, err
	}

	// Create new instance of SQLiteStore and Point SQLiteStore struct's db
	s := &SQLiteStore{db: db}

	// Migrate the db
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Implement db close method on SQLIteStore

func (s *SQLiteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Implement migrate method on SQLIteStore

func (s *SQLiteStore) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS devices(
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			ip TEXT NOT NULL,
			interval_sec INTEGER NOT NULL,
			mode TEXT NOT NULL,
			tcp_port INTEGER NOT NULL,
			enabled INTEGER NOT NULL
		);
	`)
	if err != nil {
		return err
	}
	// remove bad/duplicate IPs (one-time safety)
	_, _ = s.db.Exec(`
		DELETE FROM devices
		WHERE rowid NOT IN (
		SELECT MIN(rowid) FROM devices GROUP BY ip
		);
	`)

	// enforce IP uniqueness for upsert
	_, err = s.db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_devices_ip_unique ON devices(ip);
	`)

	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS users(
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			created_at TIMESTAMP NOT NULL,
			last_login_at TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS audit_logs(
			id TEXT PRIMARY KEY,
			ts TIMESTAMP NOT NULL,
			user_id TEXT,
			username TEXT,
			action TEXT NOT NULL,
			target TEXT,
			details_json TEXT
		);
	`)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit_logs(ts);`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id);`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action);`)
	if err != nil {
		return err
	}

	return nil
}

// Implement ListDevices method on SQLiteStore

func (s *SQLiteStore) ListDevices() ([]models.Device, error) {
	rows, err := s.db.Query(`
		SELECT id, name, ip, interval_sec, mode, tcp_port, enabled
		FROM devices
		ORDER BY name ASC, ip ASC;
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Device
	for rows.Next() {
		var d models.Device
		var mode string
		var enabled int
		if err := rows.Scan(&d.Id, &d.Name, &d.IP, &d.Interval, &mode, &d.TCPPort, &enabled); err != nil {
			return nil, err
		}

		d.Mode = models.CheckMode(mode)
		d.Enabled = enabled != 0
		out = append(out, d)
	}

	return out, rows.Err()

}

// Implement UpsertDevice method on SQLiteStore

func (s *SQLiteStore) UpsertDevice(d models.Device) error {
	enabled := 0
	if d.Enabled {
		enabled = 1
	}

	_, err := s.db.Exec(`
		INSERT INTO devices (id, name, ip, interval_sec, mode, tcp_port, enabled)
		VALUES (?,?,?,?,?,?,?)
		ON CONFLICT (ip) DO UPDATE SET
		name=excluded.name,
		ip=excluded.ip,
		interval_sec=excluded.interval_sec,
		mode=excluded.mode,
		tcp_port=excluded.tcp_port,
		enabled=excluded.enabled;
	`, d.Id, d.Name, d.IP, d.Interval, string(d.Mode), d.TCPPort, enabled)
	return err
}

// Implement DeleteDevice method on SQLiteStore

func (s *SQLiteStore) DeleteDevice(id string) error {
	_, err := s.db.Exec(`DELETE FROM devices WHERE id = ?;`, id)
	return err
}

func (s *SQLiteStore) MapIPToId() (map[string]string, error) {
	rows, err := s.db.Query(`SELECT ip, id FROM devices;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var ip, id string
		if err := rows.Scan(&ip, &id); err != nil {
			return nil, err
		}
		out[ip] = id
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetIdByIP(ip string) (string, bool, error) {
	var id string
	err := s.db.QueryRow(`SELECT id FROM devices WHERE ip = ? LIMIT 1;`, ip).Scan(&id)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return id, true, nil
}

// Implement User methods on SQLiteStore

func (s *SQLiteStore) HasAnyUsers() (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users;`).Scan(&count)
	return count > 0, err
}

// Create user with unique username and hashed password

func (s *SQLiteStore) CreateUser(u models.User) error {
	_, err := s.db.Exec(`
		INSERT INTO users (id, username, password_hash, role, created_at, last_login_at)
		VALUES (?,?,?,?,?,?);
	`, u.Id, u.Username, u.PasswordHash, string(u.Role), toTS(u.CreatedAt), toTS(u.LastLoginAt))
	return err
}

// Get user by username

func (s *SQLiteStore) GetUserByUsername(username string) (models.User, bool, error) {
	var u models.User
	var createdAt, lastLoginAt string
	var role string

	err := s.db.QueryRow(`
		SELECT id, username, password_hash, role, created_at, last_login_at
		FROM users WHERE username = ? LIMIT 1;
	`, username).Scan(&u.Id, &u.Username, &u.PasswordHash, &role, &createdAt, &lastLoginAt)

	if err == sql.ErrNoRows {
		return models.User{}, false, nil
	}
	if err != nil {
		return models.User{}, false, err
	}
	u.Role = models.UserRole(role)
	u.CreatedAt = fromTS(createdAt)
	u.LastLoginAt = fromTS(lastLoginAt)

	return u, true, nil
}

// Update user's last login time

func (s *SQLiteStore) UpdateUserLastLogin(userId string, lastLogin time.Time) error {
	_, err := s.db.Exec(`UPDATE users SET last_login_at = ? WHERE id = ?;`, toTS(lastLogin), userId)
	return err
}

// insert audit log entry

func (s *SQLiteStore) InsertAuditLog(log models.AuditLog) error {
	_, err := s.db.Exec(`
		INSERT INTO audit_logs (id, ts, user_id, username, action, target, details_json)
		VALUES (?,?,?,?,?,?,?);
	`, log.Id, toTS(log.Timestamp), nullIfEmpty(log.UserId), nullIfEmpty(log.Username), string(log.Action), nullIfEmpty(log.Target), nullIfEmpty(log.DetailsJSON))
	return err
}

// List audit logs with pagination (limit and offset)

func (s *SQLiteStore) ListAuditLogs(limit int) ([]models.AuditLog, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.Query(`
		SELECT id, ts, user_id, username, action, target, details_json
		FROM audit_logs
		ORDER BY ts DESC
		LIMIT ?;
	`, limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	out := make([]models.AuditLog, 0, limit)
	for rows.Next() {
		var log models.AuditLog
		var ts, action string
		// nullable columns come back as NULL -> Scan into sql.NullString is safer
		var nUserId, nUsername, nTarget, nDetails sql.NullString
		if err := rows.Scan(&log.Id, &ts, &nUserId, &nUsername, &log.Action, &nTarget, &nDetails); err != nil {
			return nil, err
		}
		log.Timestamp = fromTS(ts)
		log.UserId = nUserId.String
		log.Username = nUsername.String
		log.Action = models.AuditAction(action)
		log.Target = nTarget.String
		log.DetailsJSON = nDetails.String
		out = append(out, log)
	}
	return out, rows.Err()
}

// Helper functions to convert time.Time to string and vice versa for SQLite storage
func toTS(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func fromTS(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
