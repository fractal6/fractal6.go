All contribution, through issues, tensions and merge request are welcome.
Please read this file before contributing.

### Updating Schema

Procedure to update the single source of truth schema:

    cd ..  # behind this repository
    git clone https://github.com/fractal6/fractal6-schema
    # Make your change in fractal6/fractal6-schema/graphql/
    # Propagate changes
    make genall


### Git Branches

- `prod`: Tag tracking release, main branch (see also [CHANGELOG.md](CHANGELOG.md)).
- `dev`: The current development branch. Should only be merged via merge requests.
- `hotfix/*`: A bug fix for production release.
- `fix/*`: A fix an identified bug or issue.
- `feat/*`: A new feature.
- `refactor/*`: Refactoring/Improvements of existing features.


### Git commits

The commit name should starts with a name that identify the **type** of modifications done (e.g. fix, feat, refactor, perf etc), then a **context** that help to determine the scope of the changes (e.g. a file name file modified or a thematic) and finally a short comment that explain, as explicitly as possible, not what modification has been done, but what behaviour has been fixed, added or removed for example.

example: `fix/schema: Add color property to roles.`

Here are some common used for so called semantic commit message:

- feat: A new feature
- fix: A bug fix
- perf: A code change that improves performance
- refactor: A code change that neither fixes a bug nor adds a feature
- typo: Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)
- build: Changes that affect the build system or external dependencies (example scopes: gulp, broccoli, npm)
- test: Adding missing tests or correcting existing tests
- docs: Documentation only changes


### Reporting issues, Questions, Feedback

- Create an issue on the versioning system platform for bug and low-level issues (technical).

- Create a tension on [Fractale](https://fractale.co/o/f6) for questions, feedback and feature requests.

- Chat on Matrix: https://matrix.to/#/#fractal6:matrix.org for support and talking.


### Building (@deprecated)


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
