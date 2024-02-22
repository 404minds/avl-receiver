run:
	go run . -port 9000 -datadir ./data

build:
	go build -o ./avl-receiver

docker-build:
	docker build . -t avl-receiver

docker-run:
	docker run -p 9000:9000 -v ./data:/data avl-receiver

proto:
	protoc -I=./protos --go_out=./internal/types --go-grpc_out=./internal/types --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative ./protos/common-types.proto
	protoc -I=./protos --go_out=./internal/store --go-grpc_out=./internal/store --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative ./protos/avl-data-store.proto