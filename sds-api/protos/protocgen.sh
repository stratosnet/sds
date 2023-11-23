# https://protobuf.dev/getting-started/gotutorial/
# go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

protoc --go_out=./ *.proto

cp -r github.com/stratosnet/sds-api/* ../

rm -rf github.com