.ONESHELL:
GOFLAGS ?= $(GOFLAGS:) -v
GOFLAGS_PROD ?= $(GOFLAGS:) -mod=vendor
GOBIN := $(PWD)/bin
RELEASE := fractal6
DGRAPH_RELEASE := v21.12.0
#DGRAPH_RELEASE := v21.03.1
MOD := fractale/fractal6.go
#LANGS := $(shell ls public/index.* | sed -n  "s/.*index\.\([a-z]*\)\.html/\1/p" )
LANGS := $(shell find  public -maxdepth 1  -type d  -printf '%P\n' | xargs | tr " " "_")

# TODO: versioning
# LDFLAGS see versioning, hash etc...

.PHONY: build prod vendor schema
default: build

#
# Build commands
#

run_api:
	go run main.go api

run_notifier:
	go run main.go notifier

build:
	go build $(GOFLAGS) -o $(GOBIN)/$(RELEASE) main.go

prod:
	go build -trimpath $(GOFLAGS_PROD) \
		-ldflags "-X $(MOD)/cmd.buildMode=PROD -X $(MOD)/web/auth.buildMode=PROD -X $(MOD)/db.buildMode=PROD \
		-X $(MOD)/web.langsAvailable=$(LANGS)" \
		-o $(GOBIN)/$(RELEASE) main.go

vendor:
	go mod vendor

test:
	go test ./...

install_client: fetch_client
	# Set the client version in config.toml
	sed -i "s/^client_version\s*=.*$$/client_version = \"$(shell cat public/client_version)\"/" config.toml

fetch_client:
	# Fetch client code
	rm -rf public/ && \
		git clone --depth 1 ssh://git@code.skusku.site:29418/fluid-fractal/public-build.git public/ && \
		rm -rf public/.git

bootstrap:
	# Dgraph
	wget https://github.com/dgraph-io/dgraph/releases/download/$(DGRAPH_RELEASE)/dgraph-linux-amd64.tar.gz
	mkdir -p bin/
	mv dgraph-linux-amd64.tar.gz bin/ && \
		cd bin/ && \
		tar zxvf dgraph-linux-amd64.tar.gz && \
		cd ..


#
# Generate Graphql code and schema
#

genall: dgraph schema generate
gen: schema generate

dgraph: # Do alter Dgraph
	# Requirements:
	# npm install -g get-graphql-schema
	# Alternative: npm install -g graphqurl
	cd ../fractal6-schema
	make dgraph_in
	cd -
	curl -X POST http://localhost:8080/admin/schema --data-binary "@schema/dgraph_schema.graphql" | jq
	mkdir -p schema/
	cp ../fractal6-schema/gen_dgraph_in/schema.graphql schema/dgraph_schema.graphql
	# Used by the `schema` rule, to generate the gqlgen input schema
	get-graphql-schema http://localhost:8080/graphql/graphql > schema/dgraph_out.graphql
	# Alternative: gq http://localhost:8080/graphql -H "Content-Type: application/json" --introspect > schema/schema_out.graphql

schema: # Do not alter Dgraph
	cd ../fractal6-schema
	make gqlgen_in
	cd -
	mkdir -p schema/
	cp ../fractal6-schema/gen/schema.graphql schema/

generate:
	# We add "omitempty" for each generate type's literal except for Bool and Int to prevent
	# loosing data (when literal are set to false/0 values) when marshalling.
	go generate ./... && \
		sed -i "s/\(func.*\)(\([^,]*\),\([^,]*\))/\1(data \2, errors\3)/" graph/schema.resolvers.go && \
		sed -i '/\W\(bool\|int\)\W/I!s/`\w* *json:"\([^`]*\)"`/`json:"\1,omitempty"`/' graph/model/models_gen.go


#
# Utils
#

docs:
	cd ../doc && \
		make quickdoc && \
		cd - && \
		cp ../doc/_data/* data

show_query:
	rg "Gqlgen" graph/schema.resolvers.go -B 2 |grep func |sed "s/^func[^)]*)\W*\([^(]*\).*/\1/" | sort

install:
	# Redis
	#curl https://packages.redis.io/gpg | sudo apt-key add -
	#echo "deb https://packages.redis.io/deb $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/redis.list
	# -- official way
	#curl -fsSL https://packages.redis.io/gpg | sudo gpg --dearmor -o /usr/share/keyrings/redis-archive-keyring.gpg
	#echo "deb [signed-by=/usr/share/keyrings/redis-archive-keyring.gpg] https://packages.redis.io/deb $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/redis.list
	#sudo apt-get update
	sudo apt-get install redis


certs:
	# Dgraph Authorization
	#ssh-keygen -t rsa -P "" -b 2048 -m PEM -f private.pem
	#ssh-keygen -e -m PEM -f jwtRS256.key > public.pem
	openssl genrsa -out private.pem 2048
	openssl rsa -in private.pem -pubout -out public.pem
	# Copy public key for the Dgraph authorization in the schema
	# cat public.pem | sed 's/$/\\\n/' | tr -d "\n" | head -c -2 |  xclip -selection clipboard;
