package models

type CheckMode string

const (
	CheckModeAuto    CheckMode = "auto"
	CheckModeICMP    CheckMode = "icmp"
	CheckModeTCP     CheckMode = "tcp"
	CheckModePingCmd CheckMode = "pingcmd"
)

type Device struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	IP       string    `json:"ip"`
	Interval int       `json:"interval"`
	Mode     CheckMode `json:"mode"`
	TCPPort  int       `json:"tcpPort"`
	Enabled  bool      `json:"enabled"`
}
