package storage

import (
	"elv-monitor/internal/models"
)

func (s *SQLiteStore) UpsertDevices(devs []models.Device) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
		INSERT INTO devices (id, name, ip, interval_sec, mode, tcp_port, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(ip) DO UPDATE SET
		name=excluded.name,
		ip=excluded.ip,
		interval_sec=excluded.interval_sec,
		mode=excluded.mode,
		tcp_port=excluded.tcp_port,
		enabled=excluded.enabled;
	`)

	if err != nil {
		return err
	}

	defer stmt.Close()
	for _, d := range devs {
		enabled := 0

		if d.Enabled {
			enabled = 1
		}

		if _, err := stmt.Exec(d.Id, d.Name, d.IP, d.Interval, string(d.Mode), d.TCPPort, enabled); err != nil {
			return err
		}

	}
	return tx.Commit()
}
