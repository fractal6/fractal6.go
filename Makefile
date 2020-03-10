GOFLAGS ?= $(GOFLAGS:) -v
GOFLAGS_PROD ?= $(GOFLAGS:) -mod=vendor
GOBIN := $(PWD)/bin
RELEASE := "fractal6"
MOD := "zerogov/fractal6.go"

# TODO versioning
# LDFLAGS see versioning, hash etc...

.PHONY: build prod
.ONESHELL:
default: build

run:
	go run main.go run

build:
	go build $(GOFLAGS) -o $(GOBIN)/$(RELEASE) main.go

prod:
	go build -trimpath $(GOFLAGS_PROD) -ldflags "-X $(MOD)/cmd.buildMode=PROD"  -o $(GOBIN)/$(RELEASE) main.go

generate:
	cd ../schema && make gen
	cd -
	go generate ./...
