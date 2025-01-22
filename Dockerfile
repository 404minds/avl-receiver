## build environment
FROM golang:1.22.0 as builder

WORKDIR /avl-receiver

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 go build ./cmd/receiver/receiver.go

## execution environment
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /avl-receiver
COPY --from=builder /avl-receiver .

# Expose both TCP and gRPC ports
EXPOSE 21000
EXPOSE 15000

CMD ["./receiver", "-port", "21000", "-grpcPort", "15000", "-remoteStoreAddr", "carbuddy-consumer:9000"]

