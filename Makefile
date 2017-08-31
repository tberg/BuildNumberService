GIT_COMMIT := $(shell git describe --always --long --dirty)
VERSION := 0.0.8
OUT := bns

all: build
	

package: build
	mkdir -p staging/${OUT}-${VERSION}/bin
	cp ${OUT} staging/${OUT}-${VERSION}/bin/
	cp README.md staging/${OUT}-${VERSION}/
	tar czf bns-${VERSION}.tar.gz -C staging ${OUT}-${VERSION}

build: main.go
	go fmt
	go get
	go build -i -v -o ${OUT} -ldflags="-X main.Git=${GIT_COMMIT} -X main.Version=${VERSION}"

test: build
	./${OUT} -config ./conf.yaml

 .PHONY:clean

clean:
	rm -rf ${OUT} bns-${VERSION}.tar.gz staging
