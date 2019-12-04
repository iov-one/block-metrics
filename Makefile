.PHONY: all install test

# make sure we turn on go modules
export GO111MODULE := on

all: test install

install:
	go install ./cmd/collector

build:
	cd cmd/collector && $(MAKE) build

test:
	@# customd binary is required by some tests. In order to not skip them, ensure customd binary is provided and in the latest version.
	go vet -mod=readonly ./...
	go test -mod=readonly -race ./...
