# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),

## [Unrealeased]

## New
- This changelog file
- add contracts object queries
- use dgraph auth rule in schema with JWT token
- [schema] add **Node.rights** and **User.type_** fields
- add auth rules for bot roles

## Changed
- rename UserJoin event to UserJoined

## [0.4] - ...

### New
- labels (or tags) to assist in categorization/triage/search of tensions
- tensions may have one or more labels
- visual/memory color coding
- description (optional) of the labels
- each circle contains labels that can be associated with tensions. The labels are inherited in the child circles
- small navigation facilitation added in the header where the current path in the organization is displayed.

## Changed
- only the roles of coordinators can edit tensions.

## Fixed
- small bug fix that made us have to update our mdp every day! (updating the token after 30 days today)

## [0.3] - ...

### New
- Quick Help in top menu
- collect user feedback directly in the platform (top menu)
- users can create new (personal) organization
- users can leave their roles
- circle/role can be archived by user with correct rights.
- many bug fixes

