package monitor

import (
	"context"
	"errors"
	"time"

	"elv-monitor/internal/models"
)

var (
	// Used when native ICMP needs admin rights, and the user doesn't have them.
	ErrPermission = errors.New("permission denied for ICMP")
)

type Status string

const (
	StatusUp   Status = "up"
	StatusDown Status = "down"
)

type CheckResult struct {
	DeviceID  string
	IP        string
	Status    Status
	RTT       time.Duration // 0 if not available
	CheckedAt time.Time
	Reason    string // Optional reason for status change
}

type Checker interface {
	Name() string
	Check(ctx context.Context, d models.Device) CheckResult
}
