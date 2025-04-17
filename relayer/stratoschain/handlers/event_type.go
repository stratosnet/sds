package handlers

import (
	"github.com/stratosnet/sds/relayer/stratoschain/types"
)

const (
	// register module ---------------------------------------------------------

	EventTypeCompleteUnbondingResourceNode = "stratos.register.v1.EventCompleteUnBondingResourceNode"
	EventTypeCompleteUnbondingMetaNode     = "stratos.register.v1.EventCompleteUnBondingMetaNode"
	EventTypeCreateResourceNode            = "stratos.register.v1.EventCreateResourceNode"
	EventTypeUnbondingResourceNode         = "stratos.register.v1.EventUnBondingResourceNode"
	EventTypeUpdateResourceNode            = "stratos.register.v1.EventUpdateResourceNode"
	EventTypeUpdateResourceNodeDeposit     = "stratos.register.v1.EventUpdateResourceNodeDeposit"
	EventTypeUpdateEffectiveDeposit        = "stratos.register.v1.EventUpdateEffectiveDeposit"
	EventTypeCreateMetaNode                = "stratos.register.v1.EventCreateMetaNode"
	EventTypeUnbondingMetaNode             = "stratos.register.v1.EventUnBondingMetaNode"
	EventTypeUpdateMetaNode                = "stratos.register.v1.EventUpdateMetaNode"
	EventTypeUpdateMetaNodeDeposit         = "stratos.register.v1.EventUpdateMetaNodeDeposit"
	EventTypeMetaNodeRegistrationVote      = "stratos.register.v1.EventMetaNodeRegistrationVote"
	EventTypeMerkleDataUpdated             = "stratos.register.v1.EventMerkleDataUpdated"
	EventTypeCommitmentAckowledged         = "stratos.register.v1.EventCommitmentAcknowledged"

	AttributeKeyResourceNode            = "resource_node"
	AttributeKeyMetaNode                = "meta_node"
	AttributeKeyNetworkAddress          = "network_address"
	AttributeKeyPubKey                  = "pubkey"
	AttributeKeyCandidateNetworkAddress = "candidate_network_address"
	AttributeKeyVoterNetworkAddress     = "voter_network_address"
	AttributeKeyCandidateStatus         = "candidate_status"
	AttributeKeyBeneficiaryAddress      = "beneficiary_address"
	AttributeKeyUnbondingMatureTime     = "unbonding_mature_time"
	AttributeKeyOZoneLimitChanges       = "ozone_limit_changes"
	AttributeKeyInitialDeposit          = "initial_deposit"
	AttributeKeyCurrentDeposit          = "current_deposit"
	AttributeKeyAvailableTokenBefore    = "available_token_before"
	AttributeKeyAvailableTokenAfter     = "available_token_after"
	AttributeKeyDepositDelta            = "deposit_delta"
	AttributeKeyDepositToRemove         = "deposit_to_remove"
	AttributeKeyIncrDeposit             = "incr_deposit"
	AttributeKeyEffectiveDepositAfter   = "effective_deposit_after"
	AttributeKeyIsUnsuspended           = "is_unsuspended"
	AttributeKeyRoot                    = "root"
	AttributeKeyCommitment              = "commitment"

	// pot module ---------------------------------------------------------

	EventTypeVolumeReport      = "stratos.pot.v1.EventVolumeReport"
	EventTypeWithdraw          = "stratos.pot.v1.EventWithdraw"
	EventTypeFoundationDeposit = "stratos.pot.v1.EventFoundationDeposit"
	EventTypeSlashing          = "stratos.pot.v1.EventSlashing"

	AttributeKeyEpoch               = "epoch"
	AttributeKeyReportReference     = "report_reference"
	AttributeKeyAmount              = "amount"
	AttributeKeyWalletAddress       = "wallet_address"
	AttributeKeyLegacyWalletAddress = "legacy_wallet_address"
	AttributeKeyNodeP2PAddress      = "p2p_address"
	AttributeKeySlashingNodeType    = "slashing_type"
	AttributeKeyNodeSuspended       = "suspend"

	// sds module ---------------------------------------------------------

	EventTypeFileUpload       = "stratos.sds.v1.EventFileUpload"
	EventTypePrepay           = "stratos.sds.v1.EventPrePay"
	EventTypeNewFilesUploaded = "stratos.sds.v1.EventNewFilesUploaded"

	AttributeKeyReporter     = "reporter"
	AttributeKeyFileHash     = "file_hash"
	AttributeKeyUploader     = "uploader"
	AttributeKeyPurchasedNoz = "purchased_noz"
	AttributeKeyBeneficiary  = "beneficiary"
	AttributeKeyProofs       = "proofs"
	AttributeKeyHeight       = "height"

	// sdk modules ---------------------------------------------------------

	EventTypeMessage = "message"

	AttributeKeySender = "sender"
	AttributeKeyAction = "action"
)

type EventAttribute struct {
	EventType string
	Attribute string
}

var EvmTxRequiredAttributes map[string][]EventAttribute

func init() {
	EvmTxRequiredAttributes = map[string][]EventAttribute{
		types.MSG_TYPE_PREPAY: {
			{EventTypePrepay, AttributeKeySender},
			{EventTypePrepay, AttributeKeyBeneficiary},
			{EventTypePrepay, AttributeKeyPurchasedNoz},
			{EventTypeMerkleDataUpdated, AttributeKeyRoot},
			{EventTypeMerkleDataUpdated, AttributeKeyCommitment},
		},
	}
}
