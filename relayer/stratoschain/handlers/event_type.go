package handlers

import (
	"fmt"
)

const (
	// register module ---------------------------------------------------------

	EventTypeCompleteUnbondingResourceNode = "stratos.register.v1.EventCompleteUnBondingResourceNode"
	EventTypeCompleteUnbondingMetaNode     = "stratos.register.v1.EventCompleteUnBondingMetaNode"

	EventTypeCreateResourceNode                  = "stratos.register.v1.EventCreateResourceNode"
	EventTypeUnbondingResourceNode               = "stratos.register.v1.EventUnBondingResourceNode"
	EventTypeUpdateResourceNode                  = "stratos.register.v1.EventUpdateResourceNode"
	EventTypeUpdateResourceNodeDeposit           = "stratos.register.v1.EventUpdateResourceNodeDeposit"
	EventTypeUpdateEffectiveDeposit              = "stratos.register.v1.EventUpdateEffectiveDeposit"
	EventTypeCreateMetaNode                      = "stratos.register.v1.EventCreateMetaNode"
	EventTypeUnbondingMetaNode                   = "stratos.register.v1.EventUnBondingMetaNode"
	EventTypeUpdateMetaNode                      = "stratos.register.v1.EventUpdateMetaNode"
	EventTypeUpdateMetaNodeDeposit               = "stratos.register.v1.EventUpdateMetaNodeDeposit"
	EventTypeMetaNodeRegistrationVote            = "stratos.register.v1.EventMetaNodeRegistrationVote"
	EventTypeWithdrawMetaNodeRegistrationDeposit = "stratos.register.v1.EventWithdrawMetaNodeRegistrationDeposit"

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

	EventTypeFileUpload = "stratos.sds.v1.EventFileUpload"
	EventTypePrepay     = "stratos.sds.v1.EventPrePay"

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
