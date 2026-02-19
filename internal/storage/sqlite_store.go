package storage

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"

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
	return err
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
