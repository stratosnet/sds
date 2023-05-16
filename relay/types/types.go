package types

import (
	"math/big"

	"github.com/stratosnet/sds/msg/protos"
)

type ActivatedPPReq struct {
	PPList []*protos.ReqActivatedPP `json:"pp_list"`
}

type UpdatedStakePPReq struct {
	PPList []*protos.ReqUpdatedStakePP `json:"pp_list"`
}

type UnbondingPPReq struct {
	PPList []*protos.ReqUnbondingPP `json:"pp_list"`
}

type DeactivatedPPReq struct {
	PPList []*protos.ReqDeactivatedPP `json:"pp_list"`
}

type UpdatedStakeSPReq struct {
	SPList []*protos.ReqUpdatedStakeSP `json:"sp_list"`
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

type UpdatedEffectiveStakePP struct {
	P2PAddress                string   `json:"p2p_address"`
	IsUnsuspendedDuringUpdate bool     `json:"is_unsuspended_during_update"`
	EffectiveStakeAfter       *big.Int `json:"effective_stake_after"`
}

type UpdatedEffectiveStakePPReq struct {
	PPList []UpdatedEffectiveStakePP `json:"pp_list"`
	TxHash string                    `json:"tx_hash"`
}

type WithdrawnStakeSPReq struct {
	SPList []*protos.ReqWithdrawnStakeSP `json:"sp_list"`
	TxHash string                        `json:"tx_hash"`
}
