module github.com/stratosnet/sds/tx-client

go 1.19

//replace (
//	github.com/stratosnet/sds/framework => ../framework // for development on local
//)

require (
	cosmossdk.io/api v0.3.1
	cosmossdk.io/math v1.2.0
	github.com/cosmos/cosmos-proto v1.0.0-beta.3
	github.com/cosmos/gogoproto v1.4.11
	github.com/pkg/errors v0.9.1
	github.com/stratosnet/sds/framework v0.0.0-20231128191014-169cb82668b8
	github.com/stratosnet/stratos-chain/api v0.0.0-20231113204325-6de660f174b5
	github.com/stretchr/testify v1.8.4
	github.com/tendermint/go-amino v0.16.0
	google.golang.org/grpc v1.57.0
	google.golang.org/protobuf v1.31.0
)

require (
	filippo.io/edwards25519 v1.0.0 // indirect
	github.com/btcsuite/btcd v0.23.4 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.2 // indirect
	github.com/btcsuite/btcd/btcutil v1.1.3 // indirect
	github.com/btcsuite/btcd/chaincfg/chainhash v1.0.1 // indirect
	github.com/cosmos/btcutil v1.0.5 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/hdevalence/ed25519consensus v0.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.11.0 // indirect
	github.com/tyler-smith/go-bip39 v1.1.0 // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/exp v0.0.0-20230811145659-89c5cff77bcb // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/genproto v0.0.0-20230803162519-f966b187b2e5 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230726155614-23370e0ffb3e // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230815205213-6bfd019c3878 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
