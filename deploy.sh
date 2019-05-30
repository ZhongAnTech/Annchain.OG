docker build -t og-alpine:latest .
mkdir build
rm -r build/datadir_1
mkdir build/datadir_1
rm -r build/log
mkdir build/log
docker run -p 8000:8000 -p 8001:8001 -p 8002:8002 -p 8003:8003 -v "$PWD/build/datadir_1/:/opt/datadir_1/" -v "$PWD/build/log/:/opt/log/" og-alpine:latest