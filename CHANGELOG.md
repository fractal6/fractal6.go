# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [Unrealeased]

...


## [0.7.6]

- fix count2 query to handle space characters.
- new mockup screenchots


## [0.7.5]

### New
* Pin tension support (schema update + auth rules + custom dql query)

### Changed
* Update CORS to allow staging.fractale.co used for test purpose.

## [0.7.4]

### Changed
- Set/restrict notification policy

### Fixed 
- allow multiple label and assignee events to be published at once
- new orga watchers useless copy to children/
- remove Alert type hint for alert notification.
- fix deep delete mutation
- schema: remove Node.mandate and NodeFragment.children obsolete fields.
- propagate change in tension history when updating labels.


## [0.7.3]

### New
- New schema and query entries to support user reactions/emojis in comments.

### Changed
- add User name and username to tensions_all query results.

### Fixed 
- notification: Fix Mentioned email notification -> user mentioned reason take precedence over other reason.


## [0.7.2]

### New
- Count user notification and asssigned tension with getUserCount.
- Remove assigned tensions of an user that leaves an organization.
- build: add dgraph service files.

### Fixed
- mailer: footpage for Announcement tension with link to unsubscribe.
- mailer: footpage for Alert tension with link to leave organization.
- mailer: mail subject is now quoted to escape special character.
- Make a user watch an organisation when they creates it.
- add some tests/
- fix cookie Expires date and token validity time.
- allow Label edition for labels defined in multiple circles (only editable from the root circle).
- fix bug that prevented user to leave an organization due a bad nameid check.
- fix duplicate email sent for mention user and ref tension.
- fix wrong notification message sent when comments where sent in a short timestamp.


## [0.7]

### New
- schema: new announcement tension type.
- mailer: handle user to notify for announcement tension + queries to get watchers.
- auth: authorization for special tension type (Alert & Announcement).
- cmd: pass optional username argument fort token gen command.

## Changed
- mailer: send valid unsubscribe/unwatch email notifications + email typo fix

### Fixed
- mailer: email formating.
- mailer: email decoding when receiving email in a circle (create a tension).
- auth: panic for isMember check case for non existant user/username.


## [0.6.9]

### New
- Add AGPLv3 License.
- mailer: Mail can be send to organisation to create tension, just use the organisation uri (i.e. f6@fractale.co, f6/tech@fractale.co).
- webhook: add matrix webhook.
- tension: mentioned tensions event capability.
- cmd: adduser command to add user from command line.
- cmd: gentoken to print usable JWT token for the API + Dgraph.
- build: add some go tests.
- build: Makefile rule to build and publish release

### Changed
- dql: returns node attributes along with labels and template roles queries to be able to display to which nodes Labels and Roles template belongs to.
- mailer: refactor code that write email title and body.

### Fixed
- mailer: postal signature validation.
- build: fix schema and build instructions.
- errors: improve error formating in handlers/auth.


## [0.6.6]

### New
- user welcome notification
- email have List-Unsubscrive header (one-click link)
- [schema] add new meta user attribute to count unread notification

### Changed
- upgrade go.mod

### Fixded
- fix double email sent on close contract regression


## [0.6.5]

### New
- Special notification when user join an organisation (with link to contract) or is removed from an organisation.
- Tension node get title automatically updated according to node type and name.

### Changed
- pushHistory is done in the notifications daemon, in the PushEventNotification function.
- dql.Count method return errors instead of exiting.
- [notification] UserCtx now integrate the User.name field, and in email notifications.
- [notification] notifications for contract are now hanledoutside the resolvers.

### Fixed
- [auth] Dgraph token crashed if user was first_link of a circle.
- [auth] HasCoordos ensure that look for coordo role with first_link.
- [auth] protect membership role (Guest, Owner etc) and allow users to be remove from an organisation.
- [auth] changing userCanJoin settings correctly set the children visibility.
- [notification] fixed how we retrive first_link for ReasonIsFirstLink.
- [notification] fix notification reasons order.


## [0.6.4]

### New
- [MAJOR] handle root request trought fileserver to serve frontend and manage i18n preference
- Makefile rules to build frontend with i18n versions
- add restricted usernames to prevent URI unintended effects.
- empty handler `emailing` for http endpoint for postal
### Changed
- cache-control max-age is passed as argument for fileserver
- [uctx] user roles has now color attribute.
- Improve email notifications policies and text.
### Fixed
- authorization check and publication in notification endpoint (web/handlers/notification.go)
- IP restriction to access postal http endpoint.
- Emain invintatino for pending user is now sent.


## v0.6

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

### Removed
- isPrivate extra field for request validation is no longer with @auth schema directives.
