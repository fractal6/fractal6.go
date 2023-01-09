.ONESHELL:
GOFLAGS ?= $(GOFLAGS:) -v
GOFLAGS_PROD ?= $(GOFLAGS:) -mod=vendor
MOD := fractale/fractal6.go
BINARY := f6
DGRAPH_RELEASE := v21.03.1
#DGRAPH_RELEASE := v21.12.0
CLIENT_RELEASE := 0.7.3
$(eval BRANCH_NAME=$(shell git rev-parse --abbrev-ref HEAD))
$(eval COMMIT_NAME=$(shell git rev-parse --short HEAD))
$(eval RELEASE_VERSION=$(shell git tag -l --sort=-creatordate | head -n 1))
NAME := fractal6.go
RELEASE_NAME := fractal6-amd64
RELEASE_DIR := releases/$(BRANCH_NAME)/$(RELEASE_VERSION)
#LANGS := $(shell ls public/index.* | sed -n  "s/.*index\.\([a-z]*\)\.html/\1/p" )
LANGS := $(shell find  public -maxdepth 1  -type d  -printf '%P\n' | xargs | tr " " "_")


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
	go build $(GOFLAGS) -o $(BINARY) main.go

prod:
	go build -trimpath $(GOFLAGS_PROD) \
		-ldflags "-X $(MOD)/cmd.buildMode=PROD -X $(MOD)/web/auth.buildMode=PROD -X $(MOD)/db.buildMode=PROD \
		-X $(MOD)/web.langsAvailable=$(LANGS)" \
		-o $(BINARY) main.go

vendor:
	go mod vendor

test:
	go test ./...

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
	mkdir -p schema/
	cp ../fractal6-schema/gen_dgraph_in/schema.graphql schema/dgraph_schema.graphql
	# Update Dgraph
	curl -X POST http://localhost:8080/admin/schema --data-binary "@schema/dgraph_schema.graphql" | jq
	# Used by the `schema` rule, to generate the gqlgen input schema
	get-graphql-schema http://localhost:8080/graphql > schema/dgraph_out.graphql
	# Alternative: gq http://localhost:8080/graphql -H "Content-Type: application/json" --introspect > schema/dgraph_out.graphql
	# Used by gqlgen_in rule
	cp schema/dgraph_out.graphql ../fractal6-schema/gen_dgraph_out/schema.graphql

schema: # Do not alter Dgraph, just merge schemas...
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
# Publish builds for prod releases
#
#

publish_prod: build_release_prod
	@git push origin prod
	@echo "-- Please upload your release to github: $(RELEASE_DIR)/$(RELEASE_NAME)"

build_release_prod: pre_build_prod install_client_prod extract_client install_dgraph \
	prod copy_config compress_release
	@echo "-- prod release built"

pre_build_prod:
	@if [ "$(BRANCH_NAME)" != "prod" ]; then
		echo "You should be on the 'prod' branch to use this rule."
		exit 1
	fi
	@if [ -d "$(RELEASE_DIR)" ]; then
		echo "$(RELEASE_DIR) does exist, please remove it manually to rebuild this release."
		exit 1
	fi
	echo "Building (or Re-building) release: $(RELEASE_NAME)"
	mkdir -p $(RELEASE_DIR)/$(RELEASE_NAME)

install_client_prod:
	@curl -f -L https://github.com/fractal6/fractal6-ui.elm/releases/download/$(CLIENT_RELEASE)/fractal6-ui.zip \
		-o $(RELEASE_DIR)/$(RELEASE_NAME)/fractal6-ui.zip


#
# Publish builds for op releases
#

publish_op: build_release_op upload_release_op
	@git push f6 op
	@echo "-- op release published"

build_release_op: pre_build_op install_client_op extract_client install_dgraph \
	prod copy_config compress_release
	@echo "-- op release built"

pre_build_op:
	@if [ "$(BRANCH_NAME)" != "op" ]; then
		@echo "You should be on the 'op' branch to use this rule."
		exit 1
	fi
	@if [ -d "$(RELEASE_DIR)" ]; then
		@echo "$(RELEASE_DIR) does exist, please remove it manually to rebuild this release."
		exit 1
	fi
	echo "Building (or Re-building) release: $(RELEASE_NAME)"
	mkdir -p $(RELEASE_DIR)/$(RELEASE_NAME)

install_client_op:
	@curl -f -k -H "Authorization: token $(F6_TOKEN)" \
		https://code.fractale.co/api/packages/fractale/generic/fractal6-ui.elm/$(CLIENT_RELEASE)/fractal6-ui.zip \
		-o $(RELEASE_DIR)/$(RELEASE_NAME)/fractal6-ui.zip

upload_release_op:
	@curl -f -k -H "Authorization: token $(F6_TOKEN)" --progress-bar \
		--upload-file $(RELEASE_DIR)/$(RELEASE_NAME).zip \
		https://code.fractale.co/api/packages/fractale/generic/$(NAME)/$(RELEASE_VERSION)/$(RELEASE_NAME).zip

delete_release_op:
	curl -k -H "Authorization: token $(F6_TOKEN)" -X DELETE \
		https://code.fractale.co/api/packages/fractale/generic/$(NAME)/$(RELEASE_VERSION)/$(RELEASE_NAME).zip

#
# Share release build rules
#

install_dgraph:
	@wget https://github.com/dgraph-io/dgraph/releases/download/$(DGRAPH_RELEASE)/dgraph-linux-amd64.tar.gz \
		-O $(RELEASE_DIR)/$(RELEASE_NAME)/dgraph.tar.gz && \
		cd $(RELEASE_DIR)/$(RELEASE_NAME) && \
		tar zxvf dgraph.tar.gz && \
		rm -f badger && \
		rm -f dgraph.tar.gz && \
		cd -

copy_config:
	@mkdir -p $(addprefix $(RELEASE_DIR)/$(RELEASE_NAME)/, templates schema) && \
		cp templates/config.toml $(RELEASE_DIR)/$(RELEASE_NAME)/templates && \
		sed -i "s/^client_version\s*=.*$$/client_version = \"$(shell cat $(RELEASE_DIR)/$(RELEASE_NAME)/public/client_version)\"/" $(RELEASE_DIR)/$(RELEASE_NAME)/templates/config.toml && \
		cp -r data/ $(RELEASE_DIR)/$(RELEASE_NAME) && \
		cp -r contrib/ $(RELEASE_DIR)/$(RELEASE_NAME) && \
		cp schema/dgraph_schema.graphql $(RELEASE_DIR)/$(RELEASE_NAME)/schema && \
		cp $(BINARY) $(RELEASE_DIR)/$(RELEASE_NAME)

compress_release:
	@(cd $(RELEASE_DIR) && zip -q -r - $(RELEASE_NAME)) > $(RELEASE_NAME).zip && \
		mv $(RELEASE_NAME).zip $(RELEASE_DIR)

extract_client:
	@cd $(RELEASE_DIR)/$(RELEASE_NAME) && \
		unzip fractal6-ui.zip && \
		mv fractal6-ui public && \
		rm -f fractal6-ui.zip && \
		cd -

#
# Install client from source
#

install_client_source: fetch_client_source
	# Set the client version in config.toml
	sed -i "s/^client_version\s*=.*$$/client_version = \"$(shell cat public/client_version)\"/" config.toml

fetch_client_source:
	# Fetch client code
	rm -rf public/ && \
		git clone --depth 1 ssh://git@github.com/fractal6/public-build.git public/ && \
		rm -rf public/.git


#
# Utils
#

docs:
	cd ../docs && \
		make quickdoc && \
		cd - && \
		cp ../docs/_data/* data/

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

tags:
	ctags --exclude=.git  --exclude="public/*" --exclude="releases/*" -R -f .tags
