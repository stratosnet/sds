package types

const (
	// ModuleName is the name of the module
	ModuleName = "sds"

	// StoreKey to be used when creating the KVStore
	StoreKey = ModuleName

	// RouterKey to be used for routing msgs
	RouterKey = ModuleName

	// QuerierRoute to be used for querier msgs
	QuerierRoute = ModuleName
)

var (
	// Acc Balance prefix for sds store
	PrepayBalancePrefix = []byte{0x01}
	// FileStorage prefix for sds store
	FileStoreKeyPrefix = []byte{0x02}
)

// PrepayBalanceKey turn an address to key used to get prepaid balance from the sds store
func PrepayBalanceKey(acc []byte) []byte {
	return append(PrepayBalancePrefix, acc...)
}

// FileStoreKey turn an address to key used to get it from the account store
func FileStoreKey(sender []byte) []byte {
	return append(FileStoreKeyPrefix, sender...)
}
