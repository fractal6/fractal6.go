.ONESHELL:
GOFLAGS ?= $(GOFLAGS:) -v
GOFLAGS_PROD ?= $(GOFLAGS:) -mod=vendor
GOBIN := $(PWD)/bin
RELEASE := "fractal6"
MOD := "zerogov/fractal6.go"
LANGS := en fr

# TODO: versioning
# LDFLAGS see versioning, hash etc...

.PHONY: build prod vendor schema
default: build gen

#
# Build commands
#

run:
	# DO NOT FORGET TO SET THE FOLLOWING ENV VARIABLE
	# * JWT_SECRET
	# * DGRAPH_SECRET
	# * EMAIL_API_KEY
	# * EMAIL_API_URL
	go run main.go run

build:
	go build $(GOFLAGS) -o $(GOBIN)/$(RELEASE) main.go

prod:
	go build -trimpath $(GOFLAGS_PROD) -ldflags "-X $(MOD)/cmd.buildMode=PROD -X $(MOD)/web/auth.buildMode=PROD"  -o $(GOBIN)/$(RELEASE) main.go

vendor:
	go mod vendor

#
# Generate Graphql code and schema
#

genall: dgraph schema generate
gen: schema generate

dgraph:
	cd ../schema
	make dgraph # Do alter Dgraph
	cd -

schema:
	cd ../schema
	make schema # Do not alter Dgraph
	cd -
	mkdir -p schema/
	cp ../schema/gen/*.graphql schema/

generate:
	# go generate ./... | go run github.com/99designs/gqlgen generate
	# We add "omitempty" for each generate type's literal except for Bool and Int to prevent
	# loosing data (when literal are set to false/0 values) when marshalling.
	go run ./scripts/gqlgen.go && \
		sed -i "s/\(func.*\)(\([^,]*\),\([^,]*\))/\1(data \2, errors\3)/"   graph/schema.resolvers.go && \
		sed -i '/\W\(bool\|int\)\W/I!s/`\w* *json:"\([^`]*\)"`/`json:"\1,omitempty"`/' graph/model/models_gen.go


#
# Generate Data
#

docs: $(LANGS)

$(LANGS):
	# requirements:
	# pip install wildq
	# apt install jq
	wildq -M -i toml -o json '.[] | {name:.name, tasks:.tasks[]|flatten }' ../doc/doc.$@.toml > data/quickdoc.$@.json_
	jq -s "." data/quickdoc.$@.json_ > data/quickdoc.$@.json
	rm -f data/quickdoc.$@.json_

# Utils

show_query:
	rg "Gqlgen" graph/schema.resolvers.go -B 2 |grep func |sed "s/^func[^)]*)\W*\([^(]*\).*/\1/" | sort
