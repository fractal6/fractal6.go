GOFLAGS ?= $(GOFLAGS:) -v
GOFLAGS_PROD ?= $(GOFLAGS:) -v -mod=vendor
GOBIN := $(PWD)/bin

# TODO versioning
# LDFLAGS see versioning, hash etc...

default: build

build:
	go build -trimpath $(GOFLAGS) -o $(GOBIN) ./...

prod:
	go build -trimpath $(GOFLAGS_PROD) ./...
