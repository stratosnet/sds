package types

const (
	EventTypeCreateResourceNode = "create_resource_node"
	EventTypeRemoveResourceNode = "remove_resource_node"
	EventTypeCreateIndexingNode = "create_indexing_node"
	EventTypeRemoveIndexingNode = "remove_indexing_node"

	AttributeKeyResourceNode = "resource_node"
	AttributeKeyIndexingNode = "indexing_node"
	AttributeKeyNodeAddress  = "node_address"
	AttributeKeyOwner        = "owner"

	AttributeValueCategory = ModuleName
)
