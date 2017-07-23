all: build
	

build: main.go
	go fmt
	go get
	go build

test: build
	./BuildNumberService -config ./conf.yaml
