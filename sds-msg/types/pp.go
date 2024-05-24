package types

type NodeType uint32

const (
	PP_INACTIVE  uint32 = 0
	PP_ACTIVE    uint32 = 1
	PP_UNBONDING uint32 = 2

	STORAGE     NodeType = 4
	DATABASE    NodeType = 2
	COMPUTATION NodeType = 1
)
