module github.com/stratosnet/sds

go 1.19

//replace (
//	github.com/stratosnet/sds/framework => ./framework  // for development on local
//	github.com/stratosnet/sds/sds-msg => ./sds-msg  // for development on local
//	github.com/stratosnet/sds/tx-client => ./tx-client  // for development on local
//)

require (
	cosmossdk.io/api v0.7.2
	cosmossdk.io/math v1.2.0
	github.com/alecthomas/units v0.0.0-20211218093645-b94a6e3cc137
	github.com/alex023/clock v0.0.0-20191208111215-c265f1b2ab18
	github.com/deckarep/golang-set v1.8.0
	github.com/glendc/go-external-ip v0.1.0
	github.com/google/uuid v1.3.1
	github.com/gorilla/websocket v1.5.0
	github.com/ipfs/go-cid v0.3.2
	github.com/klauspost/compress v1.17.2
	github.com/multiformats/go-multibase v0.2.0
	github.com/multiformats/go-multihash v0.2.3
	github.com/pelletier/go-toml v1.9.5
	github.com/pkg/errors v0.9.1
	github.com/rs/cors v1.8.2
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible
	github.com/spf13/cobra v1.6.1
	github.com/stratosnet/sds/framework v0.0.0-20231128191014-169cb82668b8
	github.com/stratosnet/sds/sds-msg v0.0.0-20231128190750-a3a5ff99118e
	github.com/stratosnet/sds/tx-client v0.0.0-20231128191210-a8ded32412c4
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d
	google.golang.org/protobuf v1.31.0
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce
)

require (
	filippo.io/edwards25519 v1.0.0 // indirect
	github.com/HuKeping/rbtree v0.0.0-20210106022122-8ad34838eb2b // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bgadrian/go-mnemonic v0.0.0-20170924142112-3188dc747a1b // indirect
	github.com/btcsuite/btcd v0.23.4 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.2 // indirect
	github.com/btcsuite/btcd/btcutil v1.1.3 // indirect
	github.com/btcsuite/btcd/chaincfg/chainhash v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cosmos/btcutil v1.0.5 // indirect
	github.com/cosmos/cosmos-proto v1.0.0-beta.3 // indirect
	github.com/cosmos/go-bip39 v1.0.0 // indirect
	github.com/cosmos/gogoproto v1.4.11 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0 // indirect
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/hdevalence/ed25519consensus v0.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/klauspost/cpuid/v2 v2.1.2 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-base32 v0.1.0 // indirect
	github.com/multiformats/go-base36 v0.1.0 // indirect
	github.com/multiformats/go-varint v0.0.6 // indirect
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646 // indirect
	github.com/oasisprotocol/ed25519 v0.0.0-20210505154701-76d8c688d86e // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pelletier/go-toml/v2 v2.0.8 // indirect
	github.com/peterh/liner v1.2.1 // indirect
	github.com/prometheus/client_golang v1.17.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stratosnet/stratos-chain/api v0.0.0-20231113204325-6de660f174b5 // indirect
	github.com/tendermint/go-amino v0.16.0 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/tyler-smith/go-bip39 v1.1.0 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20231016165738-49dd2c1f3d0b // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20231012201019-e917dd12ba7a // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231030173426-d783a09b4405 // indirect
	google.golang.org/grpc v1.59.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	lukechampine.com/blake3 v1.1.7 // indirect
)
