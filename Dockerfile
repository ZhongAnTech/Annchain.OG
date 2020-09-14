# Build OG from alpine based golang environment
FROM golang:1.13-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

ENV GOPROXY https://goproxy.cn
ENV GO111MODULE on

ADD . /OG
WORKDIR /OG
RUN make og

# Copy OG into basic alpine image
FROM alpine:latest

RUN apk add --no-cache curl iotop busybox-extras

COPY --from=builder OG/deployment/config.toml /data
COPY --from=builder OG/deployment/genesis.json /data
COPY --from=builder OG/build/og /data

# for a temp running folder. This should be mounted from the outside
RUN mkdir /data

EXPOSE 8000 8001/tcp 8001/udp 8002 8003

WORKDIR /data

CMD ["./og", "--config", "/data/config.toml", "--multifile_by_level", "--log_line_number", "--log_dir", "/data/log/", "--datadir", "/data", "--genkey", "run"]



