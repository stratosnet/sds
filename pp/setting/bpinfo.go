package setting


import (
	"encoding/csv"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/utils"
	"math/rand"
	"os"
)

// BPList
var BPList []string

// InitBPList
func InitBPList() bool {
	csvFile, err := os.OpenFile(Config.BPListDir, os.O_CREATE|os.O_RDWR, 0777)
	defer csvFile.Close()
	if utils.CheckError(err) {
		utils.Log("InitBPList err", err)
		return false
	}
	reader := csv.NewReader(csvFile)
	reader.FieldsPerRecord = -1
	record, err := reader.ReadAll()
	if utils.CheckError(err) {
		utils.Log("InitBPList err", err)
		return false
	}
	if len(record) > 0 {
		for _, item := range record {
			BPList = append(BPList, item[0])
		}
	} else {
		utils.Log("BPList == nil")
		return false
	}
	return true
}

// SaveBPListLocal
func SaveBPListLocal(target *protos.RspGetBPList) {
	BPList = make([]string, 0)
	for _, address := range target.BpList {
		BPList = append(BPList, address.NetworkAddress)
	}
	rwmutex.Lock()
	defer rwmutex.Unlock()
	// 保存本地时先清空原来的文件
	os.Truncate(Config.BPListDir, 0)
	csvFile, err := os.OpenFile(Config.BPListDir, os.O_CREATE|os.O_RDWR, 0777)
	if utils.CheckError(err) {
		utils.ErrorLog("saveBPListLocal err", err)
		return
	}
	defer csvFile.Close()
	writer := csv.NewWriter(csvFile)
	utils.DebugLog("BPList len", len(BPList))
	for _, post := range BPList {
		line := []string{post}
		err := writer.Write(line)
		if utils.CheckError(err) {
			utils.ErrorLog("csv line ", err)
		}
	}
	writer.Flush()
}

// GetRandomBP
func GetRandomBP() string {
	if len(BPList) > 0 {
		i := rand.Intn(len(BPList))
		return BPList[i]
	}
	return ""
}
