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

The GraphQL schema represent the single source of truth to generate the GraphQL server.


Generate only the gqlgen code

    make generate

Generate all from the initial schema definition

    make genall

Note: Warning, it depends on files located in the separated repository `schema` who contains all the graphql schemas.


## Environment variable (deprecated)

    export EMAIL_API_URL=https://postal/api/v1/send/message
    export EMAIL_API_KEY=
    export JWT_SECRET=
    export DGRAPH_PUBLIC_KEY=$(cat public.pem)      # fish: set DGRAPH_PUBLIC_KEY (cat public.pem | string split0)
    export DGRAPH_PRIVATE_KEY=$(cat private.pem)    # fish: set DGRAPH_PRIVATE_KEY (cat private.pem | string split0)
    environement variable
