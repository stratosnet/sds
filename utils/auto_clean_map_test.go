package utils

import (
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"
)

const (
	LAST_RECONNECT_KEY               = "last_reconnect"
	MIN_RECONNECT_INTERVAL_THRESHOLD = 3  // seconds
	MAX_RECONNECT_INTERVAL_THRESHOLD = 20 // seconds
	RECONNECT_INTERVAL_MULTIPLIER    = 3
)

var SPMaintenanceMap *AutoCleanMap

type LastReconnectRecord struct {
	SpP2PAddress                string
	Time                        time.Time
	NextAllowableReconnectInSec int64
}

func TestAutoClean(t *testing.T) {
	autoCleanMap := NewAutoCleanMap(5 * time.Second)

	autoCleanMap.Store("a", 1)
	autoCleanMap.Store("b", 2)

	time.Sleep(4 * time.Second)
	fmt.Println("Load key 1 and check if key 1 is still in the map before clean time")
	if _, ok := autoCleanMap.Load("a"); !ok {
		t.Fatal()
	}

	time.Sleep(2 * time.Second)
	fmt.Println("check if key 2 is cleared after clean time")
	if _, ok := autoCleanMap.Load("b"); ok {
		t.Fatal()
	}
	fmt.Println("Load key 1 and check if key 1 is still in the map before clean time")
	if _, ok := autoCleanMap.Load("a"); !ok {
		t.Fatal()
	}

	time.Sleep(6 * time.Second)
	fmt.Println("check if key 1 is cleared after clean time")
	if _, ok := autoCleanMap.Load("a"); ok {
		t.Fatal()
	}
}

func TestDoubleStore(t *testing.T) {
	autoCleanMap := NewAutoCleanMap(5 * time.Second)

	autoCleanMap.Store("a", 1)

	time.Sleep(4 * time.Second)
	autoCleanMap.Store("a", 2)

	time.Sleep(2 * time.Second)
	fmt.Println("check value after first insert expires")
	if value, ok := autoCleanMap.Load("a"); ok {
		v := value.(int)
		if v != 2 {
			t.Fatal()
		}
	} else {
		t.Fatal()
	}
}

func TestDeleteAndStore(t *testing.T) {
	autoCleanMap := NewAutoCleanMap(5 * time.Second)

	autoCleanMap.Store("a", 1)

	time.Sleep(3 * time.Second)
	autoCleanMap.Delete("a")

	autoCleanMap.Store("a", 2)

	time.Sleep(3 * time.Second)
	fmt.Println("check value after first insert expires")
	if value, ok := autoCleanMap.Load("a"); ok {
		v := value.(int)
		if v != 2 {
			t.Fatal()
		}
	} else {
		t.Fatal()
	}
}

func TestAutoCleanUnsafe(t *testing.T) {
	autoCleanUnsafeMap := NewAutoCleanUnsafeMap(5 * time.Second)

	autoCleanUnsafeMap.Store("a", 1)
	autoCleanUnsafeMap.Store("b", 2)

	time.Sleep(4 * time.Second)
	fmt.Println("Load key 1 and check if key 1 is still in the map before clean time")
	if _, ok := autoCleanUnsafeMap.Load("a"); !ok {
		t.Fatal()
	}

	time.Sleep(2 * time.Second)
	fmt.Println("check if key 2 is cleared after clean time")
	if _, ok := autoCleanUnsafeMap.Load("b"); ok {
		t.Fatal()
	}
	fmt.Println("Load key 1 and check if key 1 is still in the map before clean time")
	if _, ok := autoCleanUnsafeMap.Load("a"); !ok {
		t.Fatal()
	}

	time.Sleep(6 * time.Second)
	fmt.Println("check if key 1 is cleared after clean time")
	if _, ok := autoCleanUnsafeMap.Load("a"); ok {
		t.Fatal()
	}
}

func TestDoubleStoreUnsafe(t *testing.T) {
	autoCleanUnsafeMap := NewAutoCleanUnsafeMap(5 * time.Second)

	autoCleanUnsafeMap.Store("a", 1)

	time.Sleep(4 * time.Second)
	autoCleanUnsafeMap.Store("a", 2)

	time.Sleep(2 * time.Second)
	fmt.Println("check value after first insert expires")
	if value, ok := autoCleanUnsafeMap.Load("a"); ok {
		v := value.(int)
		if v != 2 {
			t.Fatal()
		}
	} else {
		t.Fatal()
	}
}

func TestDeleteAndStoreUnsafe(t *testing.T) {
	autoCleanUnsafeMap := NewAutoCleanUnsafeMap(5 * time.Second)

	autoCleanUnsafeMap.Store("a", 1)

	time.Sleep(3 * time.Second)
	autoCleanUnsafeMap.Delete("a")

	autoCleanUnsafeMap.Store("a", 2)

	time.Sleep(3 * time.Second)
	fmt.Println("check value after first insert expires")
	if value, ok := autoCleanUnsafeMap.Load("a"); ok {
		v := value.(int)
		if v != 2 {
			t.Fatal()
		}
	} else {
		t.Fatal()
	}
}

func TestStoreStructUnsafe(t *testing.T) {
	autoCleanUnsafeMap := NewAutoCleanUnsafeMap(5 * time.Second)

	type testStruct struct {
		fieldA string
		fieldB int64
	}
	autoCleanUnsafeMap.Store("a", testStruct{
		fieldA: "a",
		fieldB: 1,
	})

	time.Sleep(3 * time.Second)

	fmt.Println("check struct fields after first insert expires")
	if value, ok := autoCleanUnsafeMap.Load("a"); ok {
		v := value.(testStruct)
		if v.fieldB != 1 || v.fieldA != "a" {
			t.Fatal()
		}
	} else {
		t.Fatal()
	}
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
	SPMaintenanceMap = NewAutoCleanMap(time.Duration(nextReconnectInterval) * time.Second)
	SPMaintenanceMap.Store(LAST_RECONNECT_KEY, &LastReconnectRecord{
		SpP2PAddress:                spP2pAddress,
		Time:                        recordTime,
		NextAllowableReconnectInSec: nextReconnectInterval,
	})
}
