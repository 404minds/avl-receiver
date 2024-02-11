run:
	go run . -port 9000 -datadir ./data

build:
	go build -o ./avl-receiver

docker-build:
	docker build . -t avl-receiver

docker-run:
	docker run -p 9000:9000 -v ./data:/data avl-receiver