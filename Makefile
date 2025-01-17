run:
	go run cmd/receiver/receiver.go -port 21000 -remoteStoreAddr localhost:8000

run-buddy:
	go run cmd/receiver/receiver.go -port 21000 -remoteStoreAddr localhost:9000

run-server:
	go run cmd/testRpcStore/testRpcStore.go -port 8080

build:
	go build cmd/receiver/receiver.go
	go build cmd/testRpcStore/testRpcStore.go

docker-build:
	docker build . -t avl-receiver

docker-run:
	docker run -p 9000:9000 -v ./data:/data avl-receiver

compose-up:
	docker-compose up

proto:
	protoc -I=./protos --go_out=./internal/types --go-grpc_out=./internal/types --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative ./protos/common-types.proto
	protoc -I=./protos --go_out=./internal/store --go-grpc_out=./internal/store --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative ./protos/avl-data-store.proto
    protoc -I=./protos --go_out=./internal/store --go-grpc_out=./internal/store --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative ./protos/avl-service.proto
