package rpc


const (
	
	SIGNATURE_FAILURE     string = "-3"
	WRONG_FILE_SIZE       string = "-4"
	TIME_OUT              string = "-5"
	FILE_REQ_FAILURE      string = "-6"
	WRONG_INPUT           string = "-7"
	WRONG_PP_ADDRESS      string = "-8"
	INTERNAL_DATA_FAILURE string = "-9"
	INTERNAL_COMM_FAILURE string = "-10"

	UPLOAD_DATA           string = "1"
	DOWNLOAD_OK           string = "2"
	DL_OK_ASK_INFO        string = "3"
	SUCCESS               string = "0"
)

// upload: request upload file
type ParamReqUploadFile struct {
	FileName      string    `json:"filename"`
	FileSize      int       `json:"filesize"`
	FileHash      string    `json:"filehash"`
	WalletAddr    string    `json:"walletaddr"`
	WalletPubkey  string    `json:"walletpubkey"`
	Signature     string    `json:"signature"`
}

// upload: upload file data
type ParamUploadData struct {
	FileHash      string   `json:"filehash"`
	Data          string   `json:"data"`
}

// download: request download file
type ParamReqDownloadFile struct {
	FileHash      string   `json:"filehash"`
	WalletAddr    string   `json:"walletaddr"`
}

// download: download file data
type ParamDownloadData struct {
	FileHash      string   `json:"filehash"`
	ReqId         string   `json:"reqid"`
}

// download: downloaded file info
type ParamDownloadFileInfo struct {
	FileHash      string   `json:"filehash"`
	FileSize      uint64   `json:"filesize"`
	ReqId         string   `json:"reqid"`
}

// list: request file list
type ParamReqFileList struct {
	WalletAddr    string   `json:"walletaddr"`
	PageId        uint64   `json:"page"`
}

type FileInfo struct {
	FileHash      string   `json:"filehash"`
	FileSize      uint64   `json:"filesize"`
	FileName      string   `json:"filename"`
	CreateTime    uint64   `json:"createtime"`
}

type Result struct {
	Return        string     `json:"return"`
	ReqId         string     `json:"reqid,omitempty"`
	OffsetStart   *uint64    `json:"offsetstart,omitempty"`
	OffsetEnd     *uint64    `json:"offsetend,omitempty"`
	FileData      string     `json:"filedata,omitempty"`
}

type FileListResult struct {
	Return        string     `json:"return"`
	FileInfo      []FileInfo `json:"fileinfo,omitempty"`
	TotalNumber   uint64     `json:"totalnumber,omitempty"`
	PageId        uint64     `json:"page,omitempty"`
}
