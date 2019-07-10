# Build OG from alpine based golang environment
FROM golang:1.12-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

ENV GOPROXY https://goproxy.io
ENV GO111MODULE on

WORKDIR /go/src/github.com/annchain/OG
COPY go.mod .
COPY go.sum .
# copy self made lib for replacement
COPY common/crypto/dedis/kyber ./common/crypto/dedis/kyber
RUN go mod download

COPY . .
RUN make og

# Copy OG into basic alpine image
FROM alpine:latest

RUN apk add --no-cache curl iotop busybox-extras

COPY --from=builder /go/src/github.com/annchain/OG/deployment/docker_private.toml /opt/config.toml
COPY --from=builder /go/src/github.com/annchain/OG/deployment/genesis.json /opt/
COPY --from=builder /go/src/github.com/annchain/OG/build/og /opt/

# for a temp running folder. This should be mounted from the outside
RUN mkdir /rw

EXPOSE 8000 8001/tcp 8001/udp 8002 8003

WORKDIR /opt

CMD ["./og", "-c", "/opt/config.toml", "-m", "-n", "-l", "/rw/log/", "-d", "/rw/datadir", "run"]



