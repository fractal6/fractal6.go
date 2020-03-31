GOFLAGS ?= $(GOFLAGS:) -v
GOFLAGS_PROD ?= $(GOFLAGS:) -mod=vendor
GOBIN := $(PWD)/bin
RELEASE := "fractal6"
MOD := "zerogov/fractal6.go"

# TODO versioning
# LDFLAGS see versioning, hash etc...

.ONESHELL:
.PHONY: build prod vendor

default: build

run:
	go run main.go run

build:
	go build $(GOFLAGS) -o $(GOBIN)/$(RELEASE) main.go

prod:
	go build -trimpath $(GOFLAGS_PROD) -ldflags "-X $(MOD)/cmd.buildMode=PROD"  -o $(GOBIN)/$(RELEASE) main.go

vendor:
	go mod vendor

#
# Generate Graphql code
#

generate:
	cd ../schema
	make gqlgen2
	cd -
	go run ./scripts/gqlgen.go
	# Or @DEBUG: why it doesnt work anymore ?
	#go generate ./...  
	# go run github.com/99designs/gqlgen generate
