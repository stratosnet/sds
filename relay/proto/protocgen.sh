protoc --go_out=./ relay.proto

cp -r github.com/stratosnet/sds/relay/* ../

rm -rf github.com