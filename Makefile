update:
	go mod vendor

build_pp:
	go build -o ./target/ ./pp/

build_sp:
	go build -o ./target/ ./example/sp/src/sp.go
