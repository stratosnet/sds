package event

import (
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/stratosnet/sds/utils"
)

const (
	LAST_RECONNECT_KEY               = "last_reconnect"
	MIN_RECONNECT_INTERVAL_THRESHOLD = 3  // seconds
	MAX_RECONNECT_INTERVAL_THRESHOLD = 20 // seconds
	RECONNECT_INTERVAL_MULTIPLIER    = 3
)

var SPMaintenanceMap *utils.AutoCleanMap

type LastReconnectRecord struct {
	SpP2PAddress                string
	Time                        time.Time
	NextAllowableReconnectInSec int64
}

func TestSpMaintenance(t *testing.T) {
	//var SPMaintenanceMap *AutoCleanMap
	//SPMaintenanceMap = NewAutoCleanMap(time.Duration(MIN_RECONNECT_INTERVAL_THRESHOLD) * time.Second)
	ITER := 100
	SLEEP := 1
	for i := 0; i < ITER; i++ {
		RecordSpMaintenance("aldskfjlsdfj", time.Now())
		time.Sleep(time.Duration(SLEEP) * time.Second)
		//SLEEP++
	}
}

// RecordSpMaintenance, return boolean flag of switching to new SP
func RecordSpMaintenance(spP2pAddress string, recordTime time.Time) bool {
	if SPMaintenanceMap == nil {
		resetSPMaintenanceMap(spP2pAddress, recordTime, MIN_RECONNECT_INTERVAL_THRESHOLD)
		fmt.Println("init&reset Map interval: " + strconv.Itoa(MIN_RECONNECT_INTERVAL_THRESHOLD) + " returning true")
		return true
	}
	if value, ok := SPMaintenanceMap.Load(LAST_RECONNECT_KEY); ok {
		lastRecord := value.(*LastReconnectRecord)
		if time.Now().Before(lastRecord.Time.Add(time.Duration(lastRecord.NextAllowableReconnectInSec) * time.Second)) {
			// if new maintenance rsp incoming in between the interval, extend the KV by storing it again (not changing value)
			SPMaintenanceMap.Store(LAST_RECONNECT_KEY, lastRecord)
			fmt.Println("found&extend Map interval: " + strconv.FormatInt(lastRecord.NextAllowableReconnectInSec, 10) + " returning False")
			return false
		}
		// if new maintenance rsp incoming beyond the interval, reset the map and modify the NextAllowableReconnectInSec
		nextReconnectInterval := int64(math.Min(MAX_RECONNECT_INTERVAL_THRESHOLD,
			float64(lastRecord.NextAllowableReconnectInSec*RECONNECT_INTERVAL_MULTIPLIER)))
		resetSPMaintenanceMap(spP2pAddress, recordTime, nextReconnectInterval)
		fmt.Println("found&reset Map interval: " + strconv.FormatInt(nextReconnectInterval, 10) + " returning true")
		return true
	}
	resetSPMaintenanceMap(spP2pAddress, recordTime, MIN_RECONNECT_INTERVAL_THRESHOLD)
	fmt.Println("not found&reset Map interval: " + strconv.Itoa(MIN_RECONNECT_INTERVAL_THRESHOLD) + " returning true")
	return true
}

func resetSPMaintenanceMap(spP2pAddress string, recordTime time.Time, nextReconnectInterval int64) {
	// reset the interval to 60s
	SPMaintenanceMap = nil
	SPMaintenanceMap = utils.NewAutoCleanMap(time.Duration(nextReconnectInterval) * time.Second)
	SPMaintenanceMap.Store(LAST_RECONNECT_KEY, &LastReconnectRecord{
		SpP2PAddress:                spP2pAddress,
		Time:                        recordTime,
		NextAllowableReconnectInSec: nextReconnectInterval,
	})
}
