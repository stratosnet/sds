package handlers

import (
	"fmt"
)

const (

	// register module ---------------------------------------------------------

	EventTypeCompleteUnbondingResourceNode = "complete_unbonding_resource_node"
	EventTypeCompleteUnbondingMetaNode     = "complete_unbonding_meta_node"

	EventTypeCreateResourceNode                  = "create_resource_node"
	EventTypeUnbondingResourceNode               = "unbonding_resource_node"
	EventTypeUpdateResourceNode                  = "update_resource_node"
	EventTypeUpdateResourceNodeDeposit           = "update_resource_node_deposit"
	EventTypeUpdateEffectiveDeposit              = "update_effective_deposit"
	EventTypeCreateMetaNode                      = "create_meta_node"
	EventTypeUnbondingMetaNode                   = "unbonding_Meta_node"
	EventTypeUpdateMetaNode                      = "update_meta_node"
	EventTypeUpdateMetaNodeDeposit               = "update_meta_node_deposit"
	EventTypeMetaNodeRegistrationVote            = "meta_node_reg_vote"
	EventTypeWithdrawMetaNodeRegistrationDeposit = "withdraw_meta_node_reg_deposit"

	AttributeKeyResourceNode            = "resource_node"
	AttributeKeyMetaNode                = "meta_node"
	AttributeKeyNetworkAddress          = "network_address"
	AttributeKeyPubKey                  = "pub_key"
	AttributeKeyCandidateNetworkAddress = "candidate_network_address"
	AttributeKeyVoterNetworkAddress     = "voter_network_address"
	AttributeKeyCandidateStatus         = "candidate_status"

	AttributeKeyUnbondingMatureTime = "unbonding_mature_time"

	AttributeKeyOZoneLimitChanges     = "ozone_limit_changes"
	AttributeKeyInitialDeposit        = "initial_deposit"
	AttributeKeyCurrentDeposit        = "current_deposit"
	AttributeKeyAvailableTokenBefore  = "available_token_before"
	AttributeKeyAvailableTokenAfter   = "available_token_after"
	AttributeKeyDepositDelta          = "deposit_delta"
	AttributeKeyDepositToRemove       = "deposit_to_remove"
	AttributeKeyIncrDeposit           = "incr_deposit"
	AttributeKeyEffectiveDepositAfter = "effective_deposit_after"
	AttributeKeyIsUnsuspended         = "is_unsuspended"

	// pot module ---------------------------------------------------------

	EventTypeVolumeReport      = "volume_report"
	EventTypeWithdraw          = "withdraw"
	EventTypeLegacyWithdraw    = "legacy_withdraw"
	EventTypeFoundationDeposit = "foundation_deposit"
	EventTypeSlashing          = "slashing"

	AttributeKeyEpoch               = "epoch"
	AttributeKeyReportReference     = "report_reference"
	AttributeKeyAmount              = "amount"
	AttributeKeyWalletAddress       = "wallet_address"
	AttributeKeyLegacyWalletAddress = "legacy_wallet_address"
	AttributeKeyNodeP2PAddress      = "p2p_address"
	AttributeKeySlashingNodeType    = "slashing_type"
	AttributeKeyNodeSuspended       = "suspend"

	// sds module ---------------------------------------------------------

	EventTypeFileUpload = "FileUpload"
	EventTypePrepay     = "Prepay"

	AttributeKeyReporter = "reporter"
	AttributeKeyFileHash = "file_hash"
	AttributeKeyUploader = "uploader"

	AttributeKeyPurchasedNoz = "purchased_noz"
	AttributeKeyBeneficiary  = "beneficiary"

	// sdk modules ---------------------------------------------------------

	AttributeKeySender = "sender"
)

func GetEventAttribute(event, attribute string) string {
	return fmt.Sprintf("%s.%s", event, attribute)
}

func GetEventAttributes(event string, attributes ...string) []string {
	result := make([]string, 0)
	for _, attr := range attributes {
		fullAttr := fmt.Sprintf("%s.%s", event, attr)
		result = append(result, fullAttr)
	}
	return result
}
