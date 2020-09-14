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

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache curl iotop busybox-extras

RUN mkdir /pod-data

COPY --from=builder OG/deployment/config.toml /
COPY --from=builder OG/deployment/genesis.json /
COPY --from=builder OG/build/og /

EXPOSE 8000
EXPOSE 8001/tcp
EXPOSE 8001/udp
EXPOSE 8002
EXPOSE 8003

WORKDIR /

CMD ["./og", "--config", "/config.toml", "--multifile_by_level", "--log_line_number", "--log_dir", "/data/log/", "--datadir", "/data", "--genkey", "run"]



