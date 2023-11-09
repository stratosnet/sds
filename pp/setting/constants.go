package setting

import (
	"math"
	"time"

	"github.com/alecthomas/units"
)

const (
	Version       = "v0.11.6"
	AppVersion    = 11
	MinAppVersion = 11

	HDPath        = "m/44'/606'/0'/0/0"
	P2pServerType = "tcp4"

	NodeReportIntervalSec         = 5 * 60       // Interval of node stat report, in seconds
	PpLatencyCheckInterval        = 60 * 60 * 24 // interval for checking the latency peer PPs, in seconds
	DEFAULT_DATA_BUFFER_POOL_SIZE = 2000

	MaxData            = 1024 * 1024 * 3 // max size of a piece in a slice
	MaxSliceSize       = 1024 * 1024 * 32
	ImagePath          = "./images/"
	VideoPath          = "./videos"
	DownloadPathMinLen = 88

	StreamCacheMaxSlice = 2

	DefaultMaxConnections = 1000

	DefaultMinUnsuspendDeposit = "1stos" // 1 stos

	SpamThresholdSpSignLatency   = 60 // in second
	SpamThresholdSliceOperations = 6 * time.Hour

	SoftRamLimitTier0     = int64(3 * units.GiB)
	SoftRamLimitTier1     = int64(7 * units.GiB)
	SoftRamLimitTier2     = int64(15 * units.GiB)
	SoftRamLimitUnlimited = math.MaxInt64
	SoftRamLimitTier0Dev  = int64(300 * units.MiB)
	SoftRamLimitTier1Dev  = int64(500 * units.MiB)
	SoftRamLimitTier2Dev  = int64(700 * units.MiB)

	DefaultHlsSegmentBuffer = 4
	DefaultHlsSegmentLength = 10
	DefaultSliceBlockSize   = 33554432

	// http code
	FAILCode       = 500
	SUCCESSCode    = 0
	ShareErrorCode = 1002
)
