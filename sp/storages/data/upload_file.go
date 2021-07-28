package data

// UploadFile
type UploadFile struct {
	Key           string
	TaskID        string
	FileName      string
	FileHash      string
	FileSize      uint64
	SliceNum      uint64
	FilePath      string
	WalletAddress string
	IsCover       bool
	List          map[uint64]bool
	IsVideoStream bool
}

// GetCacheKey
func (am *UploadFile) GetCacheKey() string {
	return "upload_file#" + am.Key
}

// SetSliceFinish
func (am *UploadFile) SetSliceFinish(num uint64) {
	if len(am.List) > 0 {
		am.List[num] = true
	}
}

// IsUploadFinished
func (am *UploadFile) IsUploadFinished() bool {

	if len(am.List) > 0 {
		c := am.SliceNum
		for _, ok := range am.List {
			if ok {
				c--
			}
		}
		return c == 0
	}
	return true
}
