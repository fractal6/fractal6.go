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
- Rename UserJoin event to UserJoined

### Fixed
- gqlast.py gram/graphql.ebnf to handle new directives (@auth and @custom).
- Bot authoration: can now only create tension in the parent circle of the bot.
- add level of authoriszation and hook based on tension events (see tensionPostHook)
- go api now handle graphql variables in queries
- improve and strenghen the query authorization for Node Post and Tension through @auth directive and resolver hooks.

