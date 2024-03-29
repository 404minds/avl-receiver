## build environment

FROM golang:1.21.5 as builder

WORKDIR /avl-receiver

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 go build ./cmd/receiver/receiver.go
RUN CGO_ENABLED=0 go build ./cmd/testRpcStore/testRpcStore.go

## execution environment

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /avl-receiver
COPY --from=builder /avl-receiver .

EXPOSE 9000
EXPOSE 8080

CMD ["./receiver", "-port", "9000", "-remoteStoreAddr", "localhost:8080"]

