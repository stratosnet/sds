package utils

const (
	DefTaskPoolCount             = 200
	TaskSize                     = 1024
	ReadTimeOut                  = 360             // in seconds
	WriteTimeOut                 = 360             // in seconds
	HandshakeTimeOut             = 5               // in seconds
	MessageBeatLen               = 4 * 1024 * 1024 // in bytes
	LatencyCheckSpListInterval   = 24 * 3600       // in seconds
	LatencyCheckSpListTimeout    = 3               // in seconds
	LatencyCheckTopSpsConsidered = 3               // number of SPs

	PpMinTier = 0
	PpMaxTier = 3
)
