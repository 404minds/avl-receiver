## build environment

FROM golang:1.21.5 as builder

WORKDIR /avl-receiver

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 go build -o ./avl-receiver

## execution environment

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /avl-receiver
COPY --from=builder /avl-receiver/avl-receiver .

CMD ["./avl-receiver", "-port", "9000", "-datadir", "/data"]

EXPOSE 9000

