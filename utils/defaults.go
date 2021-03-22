package utils

const (
	MaxConnections      = 1000
	DefTaskPoolCount    = 200
	TaskSize            = 1024
	ReadTimeOut         = 180             // in seconds
	WriteTimeOut        = 180             // in seconds
	ClientSendHeartTime = 60              // in seconds
	MsgHeaderLen        = 16              // in bytes
	MessageBeatLen      = 4 * 1024 * 1024 // in bytes
)
