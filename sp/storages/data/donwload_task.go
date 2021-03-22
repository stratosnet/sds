package data

type DownloadTask struct {
	TaskId               string
	SliceSize            uint64
	SliceHash            string
	SliceNumber          uint64
	StorageWalletAddress string
	WalletAddressList    []string
	Time                 uint64
}

func (dt *DownloadTask) GetCacheKey() string {
	return "download_task#" + dt.TaskId
}
