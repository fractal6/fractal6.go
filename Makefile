GOFLAGS ?= $(GOFLAGS:) -v
GOFLAGS_PROD ?= $(GOFLAGS:) -mod=vendor
GOBIN := $(PWD)/bin
RELEASE := "fractal6"

# TODO versioning
# LDFLAGS see versioning, hash etc...

.PHONY: build prod

default: build

build:
	go build $(GOFLAGS) -o $(GOBIN)/$(RELEASE) main.go

prod:
	go build -trimpath $(GOFLAGS_PROD) -o $(GOBIN)/$(RELEASE) main.go
