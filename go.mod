module github.com/stratosnet/sds

go 1.16

replace (
	github.com/99designs/keyring => github.com/cosmos/keyring v1.1.7-0.20210622111912-ef00f8ac3d76
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)

require (
	github.com/HuKeping/rbtree v0.0.0-20210106022122-8ad34838eb2b
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/alex023/clock v0.0.0-20191208111215-c265f1b2ab18
	github.com/bgadrian/go-mnemonic v0.0.0-20170924142112-3188dc747a1b
	github.com/bsipos/thist v1.0.0
	github.com/btcsuite/btcd v0.22.1
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce
	github.com/cosmos/cosmos-sdk v0.45.4
	github.com/cosmos/go-bip39 v1.0.0
	github.com/deckarep/golang-set v1.8.0
	github.com/ethereum/go-ethereum v1.10.16 // indirect
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.5.0
	github.com/ipfs/go-cid v0.1.0
	github.com/mattn/go-sqlite3 v1.14.9
	github.com/multiformats/go-multibase v0.0.3
	github.com/multiformats/go-multihash v0.0.15
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646
	github.com/oasisprotocol/ed25519 v0.0.0-20210505154701-76d8c688d86e
	github.com/pborman/uuid v1.2.1
	github.com/pelletier/go-toml/v2 v2.0.1
	github.com/peterh/liner v1.2.1
	github.com/pkg/errors v0.9.1
	github.com/rs/cors v1.8.2
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible
	github.com/spf13/cobra v1.4.0
	github.com/stratosnet/stratos-chain v0.6.3-0.20220318143013-31934fedd96c
	github.com/tendermint/tendermint v0.34.19
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4
	google.golang.org/protobuf v1.28.0
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce
	gopkg.in/yaml.v2 v2.4.0
)
