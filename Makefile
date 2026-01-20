BINARY_NAME ?= dshell
GOOS ?= linux
GOARCH ?= amd64

.PHONY: build
build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -o $(BINARY_NAME) .
