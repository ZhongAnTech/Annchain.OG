rm -r deployment/datadir_1
mkdir deployment/datadir_1
rm -r deployment/log
mkdir deployment/log
docker run -p 8000:8000 -p 8001:8001 -p 8002:8002 -p 8003:8003 \
    -v "$PWD/deployment/datadir_1/:/opt/rw/datadir_1/" \
    -v "$PWD/deployment/log/:/opt/rw/log/" \
    -v "$PWD/deployment/config.toml:/opt/rw/config.toml" \
    -v "$PWD/deployment/genesis.json:/opt/rw/genesis.json" \
    og-alpine:latest