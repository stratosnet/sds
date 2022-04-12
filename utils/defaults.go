package utils

const (
	MaxConnections             = 100000
	DefTaskPoolCount           = 200
	TaskSize                   = 1024
	ReadTimeOut                = 360             // in seconds
	WriteTimeOut               = 360             // in seconds
	MsgHeaderLen               = 24              // in bytes
	MessageBeatLen             = 4 * 1024 * 1024 // in bytes
	LatencyCheckSpListInterval = 1800            // in seconds
	LatencyCheckSpListTimeout  = 3               // in seconds
	StatusReportPunishInterval = 10000
)
