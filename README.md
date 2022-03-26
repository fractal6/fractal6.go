# Build 

Warning, this will depend of files located in th seperated repository `schema` who contains all the graphql schemas.

    make genall

Build only the gqlgen code

    make generate

# Config file

The server need a `config.toml` config file. You can use the folowing template:

```
[server]
host = "localhost"
port = "8888"
prometheus_instrumention = true
prometheus_credentials = my_secret
client_version = "1c555fa"
maintainer_email = "admin@email.com"


[db]
host = "localhost"
port_graphql = "8080"
port_grpc = "9080"
api = "graphql"
admin = "admin"

[graphql]
complexity_limit = 200 # 50
introspection = false
```

# Environement variable

export EMAIL_API_URL=https://postal/api/v1/send/message
export EMAIL_API_KEY=
export JWT_SECRET=
export DGRAPH_PUBLIC_KEY=$(cat public.pem)      # fish: set DGRAPH_PUBLIC_KEY (cat public.pem | string split0)
export DGRAPH_PRIVATE_KEY=$(cat private.pem)    # fish: set DGRAPH_PRIVATE_KEY (cat private.pem | string split0)
environement variable 


