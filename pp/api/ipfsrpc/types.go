package ipfsrpc

const (
	FAILED  string = "-1"
	SUCCESS string = "0"
)

type DownloadResult struct {
	Return  string `json:"return"`
	Message string `json:"message,omitempty"`
}

type UploadResult struct {
	Return  string `json:"return"`
	Message string `json:"message,omitempty"`
}
