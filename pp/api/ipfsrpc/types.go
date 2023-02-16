package ipfsrpc

const (
	FAILED  string = "-1"
	SUCCESS string = "0"

	UPLOAD_DATA   string = "1"
	DOWNLOAD_DATA string = "1"
)

type DownloadResult struct {
	Return  string `json:"return"`
	Message string `json:"message,omitempty"`
}

type UploadResult struct {
	Return  string `json:"return"`
	Message string `json:"message,omitempty"`
}