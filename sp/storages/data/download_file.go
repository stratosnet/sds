package data

type DownloadFile struct {
	FileName      string
	FileHash      string
	SliceNum      uint64
	WalletAddress string
	List          map[uint64]bool
}

// GetCacheKey
func (df *DownloadFile) GetCacheKey() string {
	return "download_file#" + df.FileHash + "-" + df.WalletAddress
}

// SetSliceFinish
func (df *DownloadFile) SetSliceFinish(num uint64) {
	if len(df.List) > 0 {
		df.List[num] = true
	}
}

// IsUploadFinished
func (df *DownloadFile) IsDownloadFinished() bool {

	if len(df.List) > 0 {
		c := df.SliceNum
		for _, ok := range df.List {
			if ok {
				c--
			}
		}
		return c == 0
	}
	return true
}
