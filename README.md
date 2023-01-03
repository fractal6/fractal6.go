# Fractale - self-organization for humans

Backend, API, Business-logic layer for [Fractale](https://fractale.co).

**Fractale** is a platform for self-organization. It is designed around the concept that an organization can be represented as a tree and should follow principles of transparency, governance decentralization and authority distribution. A tree divides in branches and form leaves, likewise an organization divides in **Circles** that can have **Roles**. Both, circles and roles have an associated descriptive document, called **Mandate**, intended to define its purpose and operating rules. Finally, the communication inside the organization is done through **Tensions**, and make the link between users and organizations. You can think of a tension as an email, but more structured and more powerful.

Using Fractale for your organization offers the following capabilities and features:
* Interactive tree and graph packing organisation chart
* Organization visibility defined at circles level
* ACL based on member roles and circle governance rules
* Ticketing management through Tensions
* Discussion thread and subscription by tension
* Email notifications broadcast and email reply
* Labels system
* Role templates system
* Journal history of events (including mandate updates!)
* GraphQL API


## Install

#### Requirements

* Redis 4+


#### Setup

Download and extract the given release

    wget https://github.com/fractal6/fractal6.go/releases/download/0.7.2/fractal6-amd64.zip
    unzip fractal6-adm64.zip && mv fractal6-amd64 fractal6
    cd fractal6

> This will install the client built for fractale.co. To point to your own instance, you need to replace the `public/` folder by your own build, otherwise it will query the fractale.co api (for build instruction, see [fractal6-ui.elm](https://github.com/fractal6/fractal6-ui.elm/) and [fractal6-ui.elm#3](https://github.com/fractal6/fractal6-ui.elm/issues/3) ).

Copy the config file template

    cp templates/config.toml .

>
> Edit the config file with your settings.
> Update the `client_version` field if you already have a config file.
>

Generate the certificates to communicate with Dgraph

    openssl genrsa -out private.pem 2048
    openssl rsa -in private.pem -pubout -out public.pem

Copy public key for the Dgraph authorization at the end of the schema

    sed -i '$ d' schema/dgraph_schema.graphql
    cat public.pem | sed 's/$/\\\n/' | tr -d "\n" | head -c -2 | { read PUBKEY; echo "# Dgraph.Authorization {\"Header\":\"X-Frac6-Auth\",\"Namespace\":\"https://YOUR_DOMAIN/jwt/claims\",\"Algo\":\"RS256\",\"VerificationKey\":\"$PUBKEY\"}"; }  >> schema/dgraph_schema.graphql


#### Run

>  Redis needs to be listening at localhost:6379

Launch the following processes:

* `./dgraph zero --config contrib/dgraph/config-zero.yml`
* `./dgraph alpha --config contrib/dgraph/config-alpha.yml`
* `./f6 api`
* `./f6 notifier`

Load up the data schema to Dgraph

    curl -X POST http://localhost:8080/admin/schema --data-binary "@schema/dgraph_schema.graphql"

That's it, Fractale is running \o/ and waiting for connection at the `http://localhost:8888` address.

If it is your first go, you might want to login. But as user registration needs email validation and might not be working if you did not set it up, you can add new user with the CLI:

    ./f6 adduser 


#### Deploy

* setup a reverse proxy to secure connections
* systemd unit files are available in [contrib/systemd](contrib/systemd)


## Contributing

You can open issues for bugs you've found or features you think are missing. You can also submit pull requests to this repository. To get started, take a look at [CONTRIBUTING.md](CONTRIBUTING.md).

You can follow Fractale organisation and roadmap at [o/f6](https://fractale.co/o/f6) and espacially [t/f6/tech](https://fractale.co/t/f6/tech).

**IRC channel**: #fractal6 on matrix.org

## License

Fractale is free, open-source software licensed under AGPLv3.
