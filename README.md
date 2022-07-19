# Fractal6.go

Business-logic layer, API, backend for [Fractale](https;//fractale.co).

The Fractal6 data structures are defined in the **Fractal6 schema**, used as the single source of truth for GraphQL data relations and queries, located in the separate repository (https://code.skusku.site/fractal6/fractal6-schema/-/tree/master/graphql). See the [generate files](#generate-files) section to see how to re-generate code.


## Install and run databases

#### Dgraph

The backend rely on [Dgraph](https://github.com/dgraph-io/dgraph) to store and query data.

    git clone ssh://git@code.skusku.site:29418/fractal6/fractal6-db.git
    cd fractal6-db
    make bootstrap
    ./bin/dgraph zero --config config-zero.yml
    # Open a new terminal and run
    ./bin/dgraph alpha --config config-alpha.yml
    # Setup Dgraph with the GQL schema
    make send_schema


#### Redis

Redis is used as a KV cache store.

    sudo apt-get install redis
    sudo systemctl restart redis-server


## Configure

The server need a `config.toml` config file to run.
You can use the following template:

```
[server]
host = "localhost"
port = "8888"
prometheus_instrumentation = true
prometheus_credentials = "my_secret"
client_version = "git hash used to build the client"
maintainer_email = "admin@mydomain.com"
jwt_secret = "my_secret"
email_api_url = "url_api_email"
email_api_key = "url_api_key"

[db]
host = "localhost"
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


You can generate the key pairs for dgraph as follows

    openssl genrsa -out private.pem 2048
    openssl rsa -in private.pem -pubout -out public.pem


## build

Build for production

    go mod vendor
    # Can take a while
    make prod


## Launch

Open a terminal and run (main server)

    ./bin/fractal6 api

Open a second terminal and run (message passing that manage event notifications)

    ./bin/fractal6 notifier


## Generate files

Generate the gqlgen server:

    make generate

Generate the gqlgen server as well as complete schema completed by Dgraph types and queries:

    make genall

Note: Warning, it depends on files located in the separated repository `schema` who contains all the graphql schemas.


## Environment variable (deprecated)

    export EMAIL_API_URL=https://postal/api/v1/send/message
    export EMAIL_API_KEY=
    export JWT_SECRET=
    export DGRAPH_PUBLIC_KEY=$(cat public.pem)      # fish: set DGRAPH_PUBLIC_KEY (cat public.pem | string split0)
    export DGRAPH_PRIVATE_KEY=$(cat private.pem)    # fish: set DGRAPH_PRIVATE_KEY (cat private.pem | string split0)
    environement variable


## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
