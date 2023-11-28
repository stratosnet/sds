module github.com/stratosnet/sds/framework

go 1.19

//replace github.com/stratosnet/sds/sds-msg => ../sds-msg // for development on local

require (
	github.com/HuKeping/rbtree v0.0.0-20210106022122-8ad34838eb2b
	github.com/alex023/clock v0.0.0-20191208111215-c265f1b2ab18
	github.com/bgadrian/go-mnemonic v0.0.0-20170924142112-3188dc747a1b
	github.com/bsipos/thist v1.0.0
	github.com/btcsuite/btcd v0.23.4
	github.com/btcsuite/btcd/btcec/v2 v2.3.2
	github.com/btcsuite/btcd/btcutil v1.1.3
	github.com/cosmos/btcutil v1.0.5
	github.com/cosmos/go-bip39 v1.0.0
	github.com/cosmos/gogoproto v1.4.11
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/go-sql-driver/mysql v1.6.0
	github.com/google/uuid v1.3.1
	github.com/hdevalence/ed25519consensus v0.1.0
	github.com/ipfs/go-cid v0.3.2
	github.com/magiconair/properties v1.8.7
	github.com/mattn/go-sqlite3 v1.14.9
	github.com/multiformats/go-multibase v0.1.1
	github.com/multiformats/go-multihash v0.2.1
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646
	github.com/oasisprotocol/ed25519 v0.0.0-20210505154701-76d8c688d86e
	github.com/pborman/uuid v1.2.1
	github.com/pelletier/go-toml/v2 v2.0.8
	github.com/peterh/liner v1.2.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.17.0
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible
	github.com/stratosnet/sds/sds-msg v0.0.0-20231128190750-a3a5ff99118e
	github.com/stretchr/testify v1.8.4
	github.com/tyler-smith/go-bip39 v1.1.0
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	golang.org/x/crypto v0.14.0
	golang.org/x/tools v0.6.0
	google.golang.org/protobuf v1.31.0
	gopkg.in/yaml.v2 v2.4.0
)

require (
	filippo.io/edwards25519 v1.0.0 // indirect
	git.sr.ht/~sbinet/gg v0.3.1 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/ajstarks/svgo v0.0.0-20211024235047-1546f124cd8b // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/btcsuite/btcd/chaincfg/chainhash v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-fonts/liberation v0.2.0 // indirect
	github.com/go-latex/latex v0.0.0-20210823091927-c0d11ff05a81 // indirect
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/go-pdf/fpdf v0.6.0 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-base32 v0.0.3 // indirect
	github.com/multiformats/go-base36 v0.1.0 // indirect
	github.com/multiformats/go-varint v0.0.6 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/gomega v1.20.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	golang.org/x/exp v0.0.0-20230811145659-89c5cff77bcb // indirect
	golang.org/x/image v0.0.0-20220302094943-723b81ca9867 // indirect
	golang.org/x/mod v0.11.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/term v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	gonum.org/v1/gonum v0.12.0 // indirect
	gonum.org/v1/plot v0.10.1 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	lukechampine.com/blake3 v1.1.6 // indirect
)
