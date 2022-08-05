.ONESHELL:
GOFLAGS ?= $(GOFLAGS:) -v
GOFLAGS_PROD ?= $(GOFLAGS:) -mod=vendor
GOBIN := $(PWD)/bin
RELEASE := "fractal6"
MOD := "fractale/fractal6.go"
#LANGS := $(shell ls public/index.* | sed -n  "s/.*index\.\([a-z]*\)\.html/\1/p" )
LANGS := $(shell find  public -maxdepth 1  -type d  -printf '%P\n')

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

build_all: genall build

prod:
	go build -trimpath $(GOFLAGS_PROD) \
		-ldflags "-X $(MOD)/cmd.buildMode=PROD -X $(MOD)/web/auth.buildMode=PROD -X $(MOD)/db.buildMode=PROD" \
		-ldflags "-X $(MOD)/web.langsAvailable=$(LANGS)" \
		-o $(GOBIN)/$(RELEASE) main.go

vendor:
	go mod vendor

install_client: fetch_client
	# Set the client version in config.toml
	sed -i "s/^client_version\s*=.*$$/client_version = \"$(shell cat public/client_version)\"/" config.toml

fetch_client:
	# Fetch client code
	rm -rf public/ && \
		git clone --depth 1 ssh://git@code.skusku.site:29418/fluid-fractal/public-build.git public/ && \
		rm -rf public/.git


#
# Generate Graphql code and schema
#

genall: dgraph schema generate
gen: schema generate

dgraph:
	cd ../fractal6-schema
	make dgraph # Do alter Dgraph
	cd -

schema:
	cd ../fractal6-schema
	make schema # Do not alter Dgraph
	cd -
	mkdir -p schema/
	cp ../fractal6-schema/gen/*.graphql schema/

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


rsa:
	# Dgraph Authorization
	#ssh-keygen -t rsa -P "" -b 2048 -m PEM -f jwtRS256.key
	#ssh-keygen -e -m PEM -f jwtRS256.key > jwtRS256.key.pub
	openssl genrsa -out private.pem 2048
	openssl rsa -in private.pem -pubout -out public.pem
	# Copy public key to the Dgraph authorization object
	# cat public.pem | sed 's/$/\\\n/' | tr -d "\n" |  xclip -selection clipboard;
