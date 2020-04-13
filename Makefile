.PHONY: all install test lint

# make sure we turn on go modules
export GO111MODULE := on

TOOLS := cmd/collector cmd/api

all: test

install:
	for ex in $(TOOLS); do cd $$ex && make install && cd -; done

build:
	for ex in $(TOOLS); do cd $$ex && make build && cd -; done

test:
	go vet -mod=readonly ./...
	go test -mod=readonly -race ./...

dist:
	for ex in $(TOOLS); do cd $$ex && make dist && cd -; done

lint:
	@go mod vendor
	docker run --rm -it -v $(shell pwd):/go/src/github.com/iov-one/block-metrics="/go/src/github.com/iov-one/block-metrics" golangci/golangci-lint:v1.17.1 golangci-lint run ./...
	@rm -rf vendor
