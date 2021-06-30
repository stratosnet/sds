package types

// pot module event types
const (
	EventTypeVolumeReport = "volume_report"
	EventTypeWithdraw     = "withdraw"

	AttributeKeyEpoch           = "report_epoch"
	AttributeKeyReportReference = "report_reference"
	AttributeKeyAmount          = "amount"
	AttributeKeyNodeAddress     = "node_address"
	AttributeKeyOwnerAddress    = "owner_address"

	AttributeValueCategory = ModuleName
)
