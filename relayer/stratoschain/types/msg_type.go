package types

const (
	MSG_TYPE_CREATE_RESOURCE_NODE             = "/stratos.register.v1.MsgCreateResourceNode"
	MSG_TYPE_UPDATE_RESOURCE_NODE             = "/stratos.register.v1.MsgUpdateResourceNode"
	MSG_TYPE_UPDATE_RESOURCE_NODE_DEPOSIT     = "/stratos.register.v1.MsgUpdateResourceNodeDeposit"
	MSG_TYPE_REMOVE_RESOURCE_NODE             = "/stratos.register.v1.MsgRemoveResourceNode"
	MSG_TYPE_CREATE_META_NODE                 = "/stratos.register.v1.MsgCreateMetaNode"
	MSG_TYPE_UPDATE_META_NODE                 = "/stratos.register.v1.MsgUpdateMetaNode"
	MSG_TYPE_UPDATE_META_NODE_DEPOSIT         = "/stratos.register.v1.MsgUpdateMetaNodeDeposit"
	MSG_TYPE_REMOVE_META_NODE                 = "/stratos.register.v1.MsgRemoveMetaNode"
	MSG_TYPE_META_NODE_REG_VOTE               = "/stratos.register.v1.MsgMetaNodeRegistrationVote"
	MSG_TYPE_UPDATE_EFFECTIVE_DEPOSIT         = "/stratos.register.v1.MsgUpdateEffectiveDeposit"
	MSG_TYPE_UNBONDING_RESOURCE_NODE          = "/stratos.register.v1.EventUnBondingResourceNode"
	MSG_TYPE_COMPLETE_UNBONDING_RESOURCE_NODE = "/stratos.register.v1.EventCompleteUnBondingResourceNode"
	MSG_TYPE_UNBONDING_META_NODE              = "/stratos.register.v1.EventUnBondingMetaNode"
	MSG_TYPE_COMPLETE_UNBONDING_META_NODE     = "/stratos.register.v1.EventCompleteUnBondingMetaNode"

	MSG_TYPE_PREPAY      = "/stratos.sds.v1.MsgPrepay"
	MSG_TYPE_FILE_UPLOAD = "/stratos.sds.v1.MsgFileUpload"

	MSG_TYPE_VOLUME_REPORT          = "/stratos.pot.v1.MsgVolumeReport"
	MSG_TYPE_WITHDRAW               = "/stratos.pot.v1.MsgWithdraw"
	MSG_TYPE_SLASHING_RESOURCE_NODE = "/stratos.pot.v1.MsgSlashingResourceNode"

	MSG_TYPE_EVM_TX = "/stratos.evm.v1.MsgEthereumTx"

	MSG_TYPE_SEND = "/cosmos.bank.v1beta1.MsgSend"
)
