package rpc

const (
	GENERIC_ERR           string = "-1"
	SIGNATURE_FAILURE     string = "-3"
	WRONG_FILE_SIZE       string = "-4"
	TIME_OUT              string = "-5"
	FILE_REQ_FAILURE      string = "-6"
	WRONG_INPUT           string = "-7"
	WRONG_PP_ADDRESS      string = "-8"
	INTERNAL_DATA_FAILURE string = "-9"
	INTERNAL_COMM_FAILURE string = "-10"
	WRONG_FILE_INFO       string = "-11"
	WRONG_WALLET_ADDRESS  string = "-12"

	UPLOAD_DATA     string = "1"
	DOWNLOAD_OK     string = "2"
	DL_OK_ASK_INFO  string = "3"
	SHARED_DL_START string = "4"
	SUCCESS         string = "0"
)

// upload: request upload file
type ParamReqUploadFile struct {
	FileName        string `json:"filename"`
	FileSize        int    `json:"filesize"`
	FileHash        string `json:"filehash"`
	WalletAddr      string `json:"walletaddr"`
	WalletPubkey    string `json:"walletpubkey"`
	Signature       string `json:"signature"`
	DesiredTier     uint32 `json:"desired_tier"`
	AllowHigherTier bool   `json:"allow_higher_tier"`
}

// upload: upload file data
type ParamUploadData struct {
	FileHash string `json:"filehash"`
	Data     string `json:"data"`
}

// download: request download file
type ParamReqDownloadFile struct {
	FileHandle   string `json:"filehandle"`
	WalletAddr   string `json:"walletaddr"`
	WalletPubkey string `json:"walletpubkey"`
	Signature    string `json:"signature"`
}

// download: download file data
type ParamReqDownloadData struct {
	FileHash       string `json:"filehash"`
	ReqId          string `json:"reqid"`
	SliceHash      string `json:"slicehash"`
	SliceNumber    uint64 `json:"slicenumber"`
	NetworkAddress string `json:"networkaddress"`
	P2PAddress     string `json:"p2pAddress"`
}

// download: download file data
type ParamDownloadData struct {
	FileHash string `json:"filehash"`
	ReqId    string `json:"reqid"`
}

// download: downloaded file info
type ParamDownloadFileInfo struct {
	FileHash string `json:"filehash"`
	FileSize uint64 `json:"filesize"`
	ReqId    string `json:"reqid"`
}

// list: request file list
type ParamReqFileList struct {
	WalletAddr string `json:"walletaddr"`
	PageId     uint64 `json:"page"`
}

// share: request share a file
type ParamReqShareFile struct {
	FileHash    string `json:"filehash"`
	WalletAddr  string `json:"walletaddr"`
	Duration    int64  `json:"duration"`
	PrivateFlag bool   `json:"bool"`
}

// share: request list shared files
type ParamReqListShared struct {
	WalletAddr string `json:"walletaddr"`
	PageId     uint64 `json:"page"`
}

// share: request stop sharing a file
type ParamReqStopShare struct {
	WalletAddr string `json:"walletaddr"`
	ShareId    string `json:"shareid"`
}

// share: request download a shared file
type ParamReqGetShared struct {
	WalletAddr   string `json:"walletaddr"`
	WalletPubkey string `json:"walletpubkey"`
	ShareLink    string `json:"sharelink"`
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

// share: request list shared files
type ParamReqDownloadShared struct {
	FileHash     string `json:"filehash"`
	WalletAddr   string `json:"walletaddr"`
	WalletPubkey string `json:"walletpubkey"`
	Signature    string `json:"signature"`
	ReqId        string `json:"reqid"`
}

// ozone: get ozone
type ParamReqGetOzone struct {
	WalletAddr string `json:"walletaddr"`
}

// result for all upload and download messages
type Result struct {
	Return         string  `json:"return"`
	ReqId          string  `json:"reqid,omitempty"`
	OffsetStart    *uint64 `json:"offsetstart,omitempty"`
	OffsetEnd      *uint64 `json:"offsetend,omitempty"`
	FileHash       string  `json:"filehash,omitempty"`
	FileName       string  `json:"filename,omitempty"`
	FileData       string  `json:"filedata,omitempty"`
	SequenceNumber string  `json:"sequencenumber,omitempty"`
}

type FileListResult struct {
	Return      string     `json:"return"`
	FileInfo    []FileInfo `json:"fileinfo,omitempty"`
	TotalNumber uint64     `json:"totalnumber,omitempty"`
	PageId      uint64     `json:"page,omitempty"`
}

type FileShareResult struct {
	Return         string     `json:"return"`
	ShareId        string     `json:"shareid,omitempty"`
	ShareLink      string     `json:"sharelink,omitempty"`
	FileInfo       []FileInfo `json:"fileinfo,omitempty"`
	TotalNumber    uint64     `json:"totalnumber,omitempty"`
	PageId         uint64     `json:"page,omitempty"`
	SequenceNumber string     `json:"sequencynumber,omitempty"`
}

// result for getozone
type GetOzoneResult struct {
	Return         string `json:"return"`
	Ozone          string `json:"ozone,omitempty"`
	SequenceNumber string `json:"sequencynumber,omitempty"`
}

// rp: request RegisterNewPP
type ParamReqRP struct {
	//P2PAddr    string `json:"p2paddr"`
	WalletAddr string `json:"walletaddr"`
}

type RPResult struct {
	Return    string `json:"return"`
	AlreadyPp bool   `json:"alreadypp"`
}

// activate: request to activate pp node
type ParamReqActivate struct {
	WalletAddr string `json:"walletaddr"`
	Stake      string `json:"stake"`
	Fee        string `json:"fee"`
	Gas        uint64 `json:"gas"`
}

type ActivateResult struct {
	Return          string `json:"return"`
	ActivationState uint32 `json:"activationstate"`
}

// prepay: request to buy ozone using token
type ParamReqPrepay struct {
	WalletAddr   string `json:"walletaddr"`
	PrepayAmount string `json:"prepayamount"`
	Fee          string `json:"fee"`
	Gas          uint64 `json:"gas"`
}

type PrepayResult struct {
	Return string `json:"return"`
}

// startmining: request to startmining
type ParamReqStartMining struct {
	WalletAddr string `json:"walletaddr"`
	//P2PAddr    string `json:"p2paddr"`
}

type StartMiningResult struct {
	Return string `json:"return"`
}
