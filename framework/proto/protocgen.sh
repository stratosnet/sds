protoc --go_out=./ ./framework/crypto/secp256k1/keys.proto ./framework/crypto/ed25519/keys.proto

cp -r github.com/stratosnet/framework/* ../

rm -rf github.com