.PHONY: og  tool all
all:og tool
og:
	go build  -o ./build/og  ./app
tool :
	go build  -o ./build/ogtool ./client
