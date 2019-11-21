.PHONY: og  tool dep all
all:og tool dep
og:
	go build  -o ./build/og  ./app
tool :
	go build  -o ./build/ogtool ./client
dep :
	go build  -o ./build/deploy ./deployment
og-solo:
	go build -o ./deployment/solo/og-solo ./app
