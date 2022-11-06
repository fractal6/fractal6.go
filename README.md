# Fractal6.go

Business-logic layer, API, backend for [Fractale](https;//fractale.co).

The Fractal6 data structures are defined in the **Fractal6 schema** and represent the single source of truth for GraphQL data relations and queries.
It is located in the separate repository (https://code.skusku.site/fractal6/fractal6-schema/-/tree/master/graphql).
See the [Generate files](#generate-files) section to see how to re-generate code.


## Install and run databases

#### 1) DB - Dgraph

The backend rely on [Dgraph](https://github.com/dgraph-io/dgraph) to store and query data.

    git clone ssh://git@code.skusku.site:29418/fractal6/fractal6-db.git
    cd fractal6-db
    make bootstrap
    ./bin/dgraph zero --config config-zero.yml
    # Open a new terminal and run
    ./bin/dgraph alpha --config config-alpha.yml
    cd -


#### 2) KV Cache - Redis

Redis is used as a KV cache store.

    sudo apt-get install redis
    sudo systemctl restart redis-server  # or "systemctl restart redis" depending on your version

## Install client

Copy the client code to the `public/` folder:

    make install_client

## Configure

The server need a `config.toml` config file to run (in the project's root folder, i.e `fractal6.go/`).
You can use the following template:

```config.toml
[server]
instance_name = "Fractale"
domain = "fractale.co"
hostname = "localhost"
port = "8888"
jwt_secret = "my_jwt_secret"
prometheus_instrumentation = true
prometheus_credentials = "my_prom_secret"
client_version = "git hash used to build the client"

[mailer]
admin_email = "admin@mydomain.com"
# URL API
email_api_url = "https://..."
email_api_key = "..."
# SMTP api
# ...todo...
# Postal validation creds
# postal default-dkim-record: Just the p=... part of the TXT record (without the semicolon at the end)
dkim_key = "..."
# webhook redirection for Postal alert.
matrix_postal_room = "!...:matrix.org"
matrix_token = "..."

[db]
hostname = "localhost"
port_graphql = "8080"
port_grpc = "9080"
api = "graphql"
admin = "admin"
dgraph_public_key = "public.pem"
dgraph_private_key = "private.pem"

[graphql]
complexity_limit = 200 # 50
introspection = false
```


Finally, generate the certificate for dgraph authorization, and populate the schema:

    # Generate certs
    make certs

	# Copy public key for the Dgraph authorization at the end of the schema
    sed -i '$ d' fractal6-db/schema/schema_in.graphql
	cat public.pem | sed 's/$/\\\n/' | tr -d "\n" | head -c -2 | { read my; echo "# Dgraph.Authorization {\"Header\":\"X-Frac6-Auth\",\"Namespace\":\"https://YOUR_DOMAIN/jwt/claims\",\"Algo\":\"RS256\",\"VerificationKey\":\"$PUBKEY\"}"; }  >> fractal6-db/schema/schema_in.graphql
    
    # Setup Dgraph with the GQL schema
    cd fractal6-db/
    make send_schema


## Launch for dev

    make build
    make run_api


## Launch for production

Build

    go mod vendor
    # Can take a while
    make prod

Open a terminal and run (main server)

    ./bin/fractal6 api

Open a second terminal and run (message passing that manage event notifications)

    ./bin/fractal6 notifier


## (Re)Generate files

Generate the gqlgen server:

    make generate

Generate the gqlgen server as well as complete schema completed by Dgraph types and queries:

    make genall

Note: Warning, it depends on files located in the separated repository `fractal6-schema` who contains all the graphql schemas.


## Environment variable (@deprecated)

    export EMAIL_API_URL=https://postal/api/v1/send/message
    export EMAIL_API_KEY=
    export JWT_SECRET=
    export DGRAPH_PUBLIC_KEY=$(cat public.pem)      # fish: set DGRAPH_PUBLIC_KEY (cat public.pem | string split0)
    export DGRAPH_PRIVATE_KEY=$(cat private.pem)    # fish: set DGRAPH_PRIVATE_KEY (cat private.pem | string split0)
    environement variable


## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
