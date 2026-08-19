/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 *
 * This file is part of Fractale.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Fractale.  If not, see <http://www.gnu.org/licenses/>.
 */

package db

var userCtxPayload string = `{
    User.name
    User.username
    User.password
    User.lang
    User.rights {expand(_all_)}
    User.roles {
        Node.nameid
        Node.name
        Node.role_type
        Node.color
    }
}`

var tensionHookPayload string = `
  uid
  Post.createdBy { User.username }
  Tension.emitter {
    Node.nameid
    Node.role_type
    Node.rights
  }
  Tension.receiver {
    Node.nameid
    Node.mode
    Node.visibility
    Node.userCanJoin
  }
  Tension.governed_node {
    uid
    Node.nameid
    Node.type_
    Node.isArchived
    Node.isRootArchived
    Node.first_link { User.username }
  }
`

var tensionBlobHookPayload string = `
  Tension.blobs %s {
    uid
    Blob.node {
      uid
      NodeFragment.type_
      NodeFragment.nameid
      NodeFragment.name
      NodeFragment.about

      NodeFragment.first_link
      NodeFragment.skills
      NodeFragment.role_type
      NodeFragment.role_ext
      NodeFragment.color
      NodeFragment.visibility
      NodeFragment.mode
    }

  }
`

var contractHookPayload string = `{
  uid
  Post.createdAt
  Contract.tension { uid Tension.receiverid }
  Contract.status
  Contract.contract_type
  Contract.event {
    EventFragment.event_type
    EventFragment.old
    EventFragment.new
  }
  Contract.candidates { User.username User.notifyByEmail }
  Contract.pending_candidates { PendingUser.email }
  Contract.participants { Vote.data Vote.node { Node.nameid Node.first_link {User.username User.notifyByEmail} } }
}`
