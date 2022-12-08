package types

type TxFee struct {
	Fee      Coin
	Gas      uint64
	Simulate bool
}
