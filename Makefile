build:
	go build -o bin/exchange cmd/exchange/main.go

run: build
	./bin/exchange

proto:	
	protoc --proto_path=proto \
		--go_out=pb --go_opt=paths=source_relative \
		--go-grpc_out=pb --go-grpc_opt=paths=source_relative \
		proto/*.proto

.PHONY: proto run build test

test:
	go test -v ./...