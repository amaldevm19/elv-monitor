package monitor

import (
	"context"
	"log"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"elv-monitor/internal/models"
)

type PingCmdChecker struct {
	Timeout time.Duration
}

func (p *PingCmdChecker) Name() string {
	return "pingcmd"
}

func (p *PingCmdChecker) Check(ctx context.Context, d models.Device) CheckResult {
	// Implementation of ping command checking would go here
	// For now, return a default result
	now := time.Now()
	log.Printf("[PingCmd] Pinging %s\n", d.IP)

	if runtime.GOOS != "windows" {
		return CheckResult{
			DeviceID:  d.Id,
			IP:        d.IP,
			Status:    StatusDown,
			RTT:       0,
			CheckedAt: now,
		}
	}

	timeoutMs := int(p.Timeout.Milliseconds())
	if timeoutMs <= 0 {
		timeoutMs = 1000 // Default to 1 second if invalid
	}

	// -n 1 : one echo request
	// -w ms: timeout in milliseconds

	cmd := exec.CommandContext(ctx, "ping", "-n", "1", "-w", strconv.Itoa(timeoutMs), d.IP)

	out, err := cmd.CombinedOutput()

	text := string(out)

	low := strings.ToLower(text)

	log.Printf("[PingCmd] Output for %s:\n%s\n", d.IP, text)

	if err == nil && strings.Contains(low, "reply from") && strings.Contains(low, "ttl=") {

		// Try parse time=XXms (best effort)
		rtt := parsePingRTT(text)

		return CheckResult{
			DeviceID:  d.Id,
			IP:        d.IP,
			Status:    StatusUp,
			RTT:       rtt,
			CheckedAt: now,
		}
	}

	// Distinguish common reasons (best effort)
	reason := "Timeout"

	if strings.Contains(low, "destination host unreachable") {
		reason = "Unreachable"
	} else if strings.Contains(low, "could not find host") {
		reason = "Host Not Found"
	} else if strings.Contains(low, "general failure") {
		reason = "General Failure"
	} else if strings.Contains(low, "request timed out") {
		reason = "Request Timed Out"
	}

	return CheckResult{
		DeviceID:  d.Id,
		IP:        d.IP,
		Status:    StatusDown,
		RTT:       0,
		CheckedAt: now,
		Reason:    reason,
	}

}

var rttRe = regexp.MustCompile(`time[=<]\s*(\d+)\s*ms`)

func parsePingRTT(output string) time.Duration {

	m := rttRe.FindStringSubmatch(output)

	if len(m) != 2 {
		return 0
	}

	ms, err := strconv.Atoi(m[1])

	if err != nil {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}
