package data

type DownloadTask struct {
	TaskId            string
	SliceSize         uint64
	SliceHash         string
	SliceNumber       uint64
	StorageP2PAddress string
	P2PAddressList    []string
	Time              uint64
}

func (dt *DownloadTask) GetCacheKey() string {
	return "download_task#" + dt.TaskId
}
