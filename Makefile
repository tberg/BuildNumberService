all: build
	

build: main.go
	go fmt
	go build

test: build
	./BuildNumbers -config ./conf.yaml
