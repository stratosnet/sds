package relay

import (
	"math/big"

	"github.com/stratosnet/sds/sds-msg/protos"
)

type ActivatedPPReq struct {
	PPList []*protos.ReqActivatedPP `json:"pp_list"`
}

type UpdatedDepositPPReq struct {
	PPList []*protos.ReqUpdatedDepositPP `json:"pp_list"`
}

type UnbondingPPReq struct {
	PPList []*protos.ReqUnbondingPP `json:"pp_list"`
}

type UnbondingSPReq struct {
	SPList []*protos.ReqUnbondingSP `json:"sp_list"`
}

type DeactivatedPPReq struct {
	PPList []*protos.ReqDeactivatedPP `json:"pp_list"`
}

type UpdatedDepositSPReq struct {
	SPList []*protos.ReqUpdatedDepositSP `json:"sp_list"`
}

type ActivatedSPReq struct {
	SPList []*protos.ReqActivatedSP `json:"sp_list"`
}

type PrepaidReq struct {
	WalletList []*protos.ReqPrepaid `json:"wallet_list"`
}

type FileUploadedReq struct {
	UploadList []*protos.Uploaded `json:"upload_list"`
}

type VolumeReportedReq struct {
	Epochs []string `json:"epochs"`
}

type SlashedPP struct {
	P2PAddress string   `json:"p2p_address"`
	QueryFirst bool     `json:"query_first"`
	Suspended  bool     `json:"suspended"`
	SlashedAmt *big.Int `json:"slashed_amt"`
}

type SlashedPPReq struct {
	PPList []SlashedPP `json:"pp_list"`
	TxHash string      `json:"tx_hash"`
}

type UpdatedEffectiveDepositPP struct {
	P2PAddress                string   `json:"p2p_address"`
	IsUnsuspendedDuringUpdate bool     `json:"is_unsuspended_during_update"`
	EffectiveDepositAfter     *big.Int `json:"effective_deposit_after"`
}

type UpdatedEffectiveDepositPPReq struct {
	PPList []UpdatedEffectiveDepositPP `json:"pp_list"`
	TxHash string                      `json:"tx_hash"`
}

//type WithdrawnDepositSPReq struct {
//	SPList []*protos.ReqWithdrawnDepositSP `json:"sp_list"`
//	TxHash string                          `json:"tx_hash"`
//}
