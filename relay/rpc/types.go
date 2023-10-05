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

type ParamReqSync struct {
	TxHash string `json:"tx_hash"`
}

type SyncResult struct {
	Return string `json:"return"`
}

type Signature struct {
	Address   string `json:"address"`
	Pubkey    string `json:"pubkey"`
	Signature string `json:"signature"` // Hex-encoded
}
