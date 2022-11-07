# fractal6.go

Backend, API, Business-logic layer for [Fractale](https://fractale.co).

**Fractale** is a platform for self-organization. It provides a secure space shared by the members of any organisation that features:
* Tree and Graph-Packing organisation navigation (Circle are branches, Roles are leaves).
* Circle mandate, governance and visibility.
* ACL based on member Role.
* Ticketing management through [Tensions](https://doc.fractale.co/tension/).
* Journal history of events (including mandate updates).
* Email notifications.
* GraphQL API.


## Requirements

* Redis 4+


## Install

#### From source

**Setup**

    git clone -b prod https://github.com/fractal6/fractal6.go
    cd fractal6.go

    # Install the client UI
    make install_client

    # Start Redis (KV cache store)
    sudo systemctl restart redis-server  # or "systemctl restart redis" depending on your version

    # Setup Dgraph (database)
    make bootstrap
    ./bin/dgraph zero --config contrib/dgraph/config-zero.yml
    # Open a new terminal and run
    ./bin/dgraph alpha --config contrib/dgraph/config-alpha.yml

**Configure**

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
    sed -i '$ d' schema/dgraph_schema.graphql
	cat public.pem | sed 's/$/\\\n/' | tr -d "\n" | head -c -2 | { read my; echo "# Dgraph.Authorization {\"Header\":\"X-Frac6-Auth\",\"Namespace\":\"https://YOUR_DOMAIN/jwt/claims\",\"Algo\":\"RS256\",\"VerificationKey\":\"$PUBKEY\"}"; }  >> schema/dgraph_schema.graphql

    # Update Dgraph schema
    curl -X POST http://localhost:8080/admin/schema --data-binary "@schema/dgraph_schema.graphql" | jq


**Launch for production**

    # Build
    go mod vendor
    make prod

    # Open a terminal and run (main server)
    ./bin/fractal6 api
    # Open a second terminal and run (message passing that manage event notifications)
    ./bin/fractal6 notifier


**Launch for development**

	go run main.go api
	go run main.go notifier


## Contributing

Fractale is free, open-source software licensed under AGPLv3.

You can open issues for bugs you've found or features you think are missing. You can also submit pull requests to this repository. To get started, take a look at [CONTRIBUTING.md](CONTRIBUTING.md).

You can follow Fractale organisation and roadmap at [o/f6](https://fractale.co/o/f6).

IRC channel: #fractal6 on matrix.org
