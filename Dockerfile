# Build OG from alpine based golang environment
FROM golang:1.12-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

ENV GOPROXY https://goproxy.io
ENV GO111MODULE on

ADD . /go/src/github.com/annchain/OG
# RUN cd /go/src/github.com/annchain/OG && go mod init
RUN cd /go/src/github.com/annchain/OG && make og

# Copy OG into basic alpine image
FROM alpine:latest

RUN apk add --no-cache curl iotop busybox-extras

COPY --from=builder /go/src/github.com/annchain/OG/deployment/config.toml /opt/
COPY --from=builder /go/src/github.com/annchain/OG/deployment/genesis.json /opt/
COPY --from=builder /go/src/github.com/annchain/OG/build/og /opt/

EXPOSE 8000 8001/tcp 8001/udp 8002 8003

WORKDIR /opt

CMD ["./og", "-c", "config.toml", "-m", "-n", "-l", "rw/log/", "run"]



