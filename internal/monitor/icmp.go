package monitor

import (
	"context"
	"time"

	"elv-monitor/internal/models"
)

type ICMPChecker struct {
	Timeout time.Duration
}

func (i ICMPChecker) Name() string { return "icmp" }

func (i ICMPChecker) Check(ctx context.Context, d models.Device) CheckResult {

	// We’ll implement native ICMP next. For now, return permission error so Auto mode falls back.
	return CheckResult{
		DeviceID:  d.ID,
		IP:        d.IP,
		CheckedAt: time.Now(),
		Status:    StatusDown,
		Reason:    ErrPermission.Error(),
	}
}
