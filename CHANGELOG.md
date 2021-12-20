# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [Unrealeased]

### New
- This changelog file
- [major] add the reset password feature.
- [major]tension and node can be moved (except Root, self-loop and recursion).
- [major] Add contracts object queries.
    - Member of an organisation can add (contract are automatically created for coordo that try to move a blobed tension)
    - Member can update (push or edit on comment) on contract.
    - Author and coordo can delete a contract
- Add more control over authorizations (through tension events)
- [auth] Use dgraph auth rule in schema with JWT token (Root test, Bot Ownership and Memership test)
- [schema] add **Node.rights** and **User.type_** fields
- [schema] add **isContractValidator** directive for Contract "meta" literal.
- [auth] Add auth rules for bot roles
- deepDelete operation implemented. The children node to delete should be definide in `db/dql.go` in the `delete<Type>` entry.
- aggregate queries and fields are not generated thanks to a gqlast update.

### Changed
- [schema] Rename UserJoin event to UserJoined
- [resolver] rename UserCtxFromContext to GetUserContext
- [schema] hook directive renamed with Input suffix for argumants directive and without suffix {query}{Type} for field query/mutation.
- [schema] introduce the @meta directive on field to aggregate count data with DQL requests.

### Removed
- isPrivate extra field for request validation is no longer with @auth schema directives.

### Fixed
- [gqlast.py] gram/graphql.ebnf to handle new directives (@auth and @custom).
- [auth] Bot role type can now only create tension in the parent circle of the bot.
- [auth] add level of authorization and hook based on tension events (see tensionPostHook)
- [auth]improve and strenghen the query authorization for Node Post and Tension through @auth directive and resolver hooks.
- [api] go api now handle graphql variables in queries
- [api] private organisation can now create they label (the one with the "@" in the url (nameid)). Quote was missing inside the DQL eq filter.
- [api] int type in schema are not set with omitempty anymore to prevent losing 0 value (aka false value for bool types). Prior to follow request to the backend, Null value are filtered out after Marshall operation, prior 
- [schema/resolver] directive operating on input object (add/alter/patch) don't need the name of the field anymore in schema.
- [resolver] @unique directive know work on all types.
