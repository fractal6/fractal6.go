# fractal6.go

Backend, API, Business-logic layer for [Fractale](https://fractale.co).

**Fractale** is a platform for self-organization. It is designed around the concept that an organization can be represented as a tree, and the principles of transparency, governance decentralization and authority distribution. A tree divides in branches and form leaves, likewise an organization divides in **Circles** that can have **Roles**. Both circles and roles have an associated descriptive document, called **Mandate**, intended to define its purpose and operating rules. Finally, the communication inside the organization is done trough **Tensions**, and make the link between users and organizations. You can think of it as an email, but more structured and more powerful.

Using Fractale for your organization offers the following capabilities and feature:
* Tree and Graph-Packing organisation navigation.
* Organization visibility define at circles level.
* ACL based on member role and circle governance rules.
* Ticketing management through Tensions.
* Discussion thread and subscription per tension.
* Journal history of events (including mandate updates).
* Email notifications and response.
* Labels system.
* Role templates system.
* GraphQL API.


## Requirements

* Redis 4+


## Install

#### From source

**Setup**

    git clone -b prod https://github.com/fractal6/fractal6.go
    cd fractal6.go/

    # Install the client UI (Optional)
    # NOTE: This will install the client built for fractale.co.
    #       To point to your own instance, you need to rebuild it (see https://github.com/fractal6/fractal6-ui.elm/)
    #       Otherwise it will query api.fractale.co
    make install_client

    # Setup Dgraph (database)
    make bootstrap
    ./bin/dgraph zero --config contrib/dgraph/config-zero.yml
    # Open a new terminal and run
    ./bin/dgraph alpha --config contrib/dgraph/config-alpha.yml

**Configure**

The server need a `config.toml` config file to run (in the working directory). You can use the following template [templates/config.toml](templates/config.toml)

Finally, generate the certificate for dgraph authorization, and populate the schema:

    # Generate certs
    make certs

	# Copy public key for the Dgraph authorization at the end of the schema
    sed -i '$ d' schema/dgraph_schema.graphql
	cat public.pem | sed 's/$/\\\n/' | tr -d "\n" | head -c -2 | { read PUBKEY; echo "# Dgraph.Authorization {\"Header\":\"X-Frac6-Auth\",\"Namespace\":\"https://YOUR_DOMAIN/jwt/claims\",\"Algo\":\"RS256\",\"VerificationKey\":\"$PUBKEY\"}"; }  >> schema/dgraph_schema.graphql

    # Update Dgraph schema
    curl -X POST http://localhost:8080/admin/schema --data-binary "@schema/dgraph_schema.graphql" | jq


**Launch for production**

    # Build
    go mod vendor
    make prod

    # Open a terminal and run (main server)
    ./f6 api
    # Open a second terminal and run (message passing that manage event notifications)
    ./f6 notifier


**Launch for development**

	go run main.go api
	go run main.go notifier


You can add users in Fractale with the following sub-command :

    ./f6 adduser


Note that this command would be required to add users if the mailer is not enabled as the sign-up process has an email validation step. Once the mailer is setup, new users can be invited to organizations and roles from their email, or from their username if they have already sign-up.


## Contributing

You can open issues for bugs you've found or features you think are missing. You can also submit pull requests to this repository. To get started, take a look at [CONTRIBUTING.md](CONTRIBUTING.md).

You can follow Fractale organisation and roadmap at [o/f6](https://fractale.co/o/f6) and espacially [t/f6/tech](https://fractale.co/t/f6/tech).

IRC channel: #fractal6 on matrix.org

## License

Fractale is free, open-source software licensed under AGPLv3.
