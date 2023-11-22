protoc --go_out=./ ./tx-client/crypto/secp256k1/keys.proto ./tx-client/crypto/ed25519/keys.proto

cp -r github.com/stratosnet/tx-client/* ../

rm -rf github.com