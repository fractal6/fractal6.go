.ONESHELL:
SHELL := /bin/bash

#.PHONY: dgraph_in

# Requirements:
# - apt install jq
# - pip install -r requirements.txt
# - npm install -g get-graphql-schema@2.1.2
# Alternative: graphqurl, graphql-cli


# Build the main schema by inserting @auth rule in the root schema
auth_schema:
	@./gqlauth.py auth/ graphql/fractal6.graphql > fractal6-gen.graphql
	echo "Auth rules merged"

# Generate Dgraph input schema
dgraph_in: auth_schema
	@# Generate Dgraph input schema
	./gqlast.py --dgraph <(cat graphql/errors.graphql fractal6-gen.graphql) > dgraph_schema.graphql
	echo "Dgraph input schema generated"

# Update Dgraph
dgraph:
	curl -X POST http://localhost:8080/admin/schema --data-binary "@dgraph_schema.graphql" | jq
	echo "Dgraph schema updated"

# Generate Dgraph output schema
dgraph_out:
	@# Fetch the Dgraph generated schema
	get-graphql-schema http://localhost:8080/graphql > dgraph_out.graphql
	# Alternative: gq http://localhost:8080/graphql -H "Content-Type: application/json" --introspect > dgraph_out.graphql
	echo "Dgraph generated schema retrieved"

# Build final schema by mergin everything
gqlgen_in: auth_schema
	@# Generate Gqlgen compatible GraphQL files with dgraph generated Query and Mutation.
	# Fish shell: use `(cat .. | psub)` instead.
	./gqlast.py <(cat graphql/directives.graphql fractal6-gen.graphql dgraph_out.graphql) > schema.graphql
	echo "Gqlgen input schema generated"

clean:
	rm -f fractal6-gen.graphql
	rm -f dgraph_out.graphql

#
# Build Parser
#

parser:
	# Build the parser from the grammar
	python3 -m tatsu gram/graphql.ebnf -o gram/graphql.py
	# help: gram/graphql.py --help
	# list rule: gram/graphql.py -l
	# parse file rule: gram/graphql.py schema.graphql document

_gram:
	# <!>Warning<!>
	# Get Orinal Grammar
	nop && \
		mkdir -p gram/ && \
		wget https://raw.githubusercontent.com/antlr/grammars-v4/master/graphql/GraphQL.g4 -O gram/graphql.g && \
		python3 -m tatsu.g2e graphql.g > gram/graphql.ebnf
	# Manual modification here to get this work with gqlast.py
