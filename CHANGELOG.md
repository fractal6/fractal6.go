# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [Unrealeased]

### New
- This changelog file
- Add contracts object queries.
- Add more control over authorizations (through tension events)
- Use dgraph auth rule in schema with JWT token
- [schema] add **Node.rights** and **User.type_** fields
- Add auth rules for bot roles

### Changed
- [schema] Rename UserJoin event to UserJoined

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
