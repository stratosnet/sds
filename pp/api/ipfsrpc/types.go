package ipfsrpc

const (
	FAILED  string = "0"
	SUCCESS string = "1"

	UPLOAD_DATA   string = "2"
	DOWNLOAD_DATA string = "2"
)

type DownloadResult struct {
	Return  string `json:"return"`
	Message string `json:"message,omitempty"`
}

type UploadResult struct {
	Return  string `json:"return"`
	Message string `json:"message,omitempty"`
}

type FileListResult struct {
	Return      string     `json:"return"`
	Message     string     `json:"message,omitempty"`
	FileInfo    []FileInfo `json:"fileinfo,omitempty"`
	TotalNumber uint64     `json:"totalnumber,omitempty"`
	PageId      uint64     `json:"page,omitempty"`
}

type FileInfo struct {
	FileHash    string `json:"filehash"`
	FileSize    uint64 `json:"filesize"`
	FileName    string `json:"filename"`
	CreateTime  uint64 `json:"createtime,omitempty"`
	LinkTime    uint64 `json:"linktime,omitempty"`
	LinkTimeExp uint64 `json:"linktimeexp,omitempty"`
	ShareId     string `json:"shareid,omitempty"`
	ShareLink   string `json:"sharelink,omitempty"`
}
