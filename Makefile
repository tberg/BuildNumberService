GIT_COMMIT := $(shell git describe --always --long --dirty)
VERSION := 0.0.7
OUT := bns

all: build
	

build: main.go
	go fmt
	go get
	go build -i -v -o ${OUT} -ldflags="-X main.Git=${GIT_COMMIT} -X main.Version=${VERSION}"

test: build
	./${OUT} -config ./conf.yaml
