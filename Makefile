.PHONY: all install test

# make sure we turn on go modules
export GO111MODULE := on

TOOLS := cmd/collector

all: test install

install:
	for ex in $(TOOLS); do cd $$ex && make install && cd -; done

build:
	for ex in $(TOOLS); do cd $$ex && make build && cd -; done

test:
	go vet -mod=readonly ./...
	go test -mod=readonly -race ./...

dist:
	for ex in $(TOOLS); do cd $$ex && make dist && cd -; done
