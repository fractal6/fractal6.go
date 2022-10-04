All contribution, through issues, tensions and merge request are welcome.
Please read this file before contributing.


### Git Branches

- `master`: Tag tracking (see also [CHANGELOG.md](CHANGELOG.md)).
- `dev`: The current development branch. Should only be merged via merge requests.
- `release-*`: A Production release (no more features added).
- `hotfix/*`: A bug fix for production release.
- `fix/*`: A fix an identified bug or issue.
- `feat/*`: A new feature.
- `codefactor/*`: Refactoring/Improvements of existing features.


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


## Reporting issues, Questions, Feedback

- Issue on the versioning system platform for bug and low-level issues (technical).

- Tension on [Fractale](https://fractale.co/o/f6) for all the rest.

- Chat on Matrix: https://matrix.to/#/#fractal6:matrix.org
