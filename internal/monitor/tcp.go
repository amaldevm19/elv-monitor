package monitor

import (
	"context"
	"net"
	"strconv"
	"time"

	"elv-monitor/internal/models"
)

type TCPChecker struct {
	Timeout time.Duration
}

func (t TCPChecker) Name() string { return "tcp" }

func (t TCPChecker) Check(ctx context.Context, d models.Device) CheckResult {
	now := time.Now()

	port := d.TCPPort

	if port <= 0 {
		port = 80 // Default to port 80 if not specified
	}

	addr := net.JoinHostPort(d.IP, strconv.Itoa(port))

	timeout := t.Timeout

	if timeout <= 0 {
		timeout = 5 * time.Second // Default to 5 seconds if invalid
	}

	var dnet net.Dialer
	dnet.Timeout = timeout
	start := time.Now()
	conn, err := dnet.DialContext(ctx, "tcp", addr)

	if err != nil {
		return CheckResult{
			DeviceID:  d.ID,
			IP:        d.IP,
			Status:    StatusDown,
			CheckedAt: now,
			Reason:    err.Error(),
		}
	}

	_ = conn.Close()

	return CheckResult{
		DeviceID:  d.ID,
		IP:        d.IP,
		Status:    StatusUp,
		CheckedAt: now,
		RTT:       time.Since(start),
	}
}
