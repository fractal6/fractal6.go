/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2022 Fractale Co
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

import (
	"fmt"
    "log"
	"context"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/dgraph-io/dgo/v200/protos/api"
	"github.com/mitchellh/mapstructure"

	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/tools"
)

// @refactor: modularize generic function (GetFilterBy*) (returns (interface{}, error}
//            as Traverse([list of key to traver], [list of payload to get])
//  Use Meta() for all other queries... (return []interface{})

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
  Tension.action
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
`
var tensionBlobHookPayload string = `
  Tension.blobs %s {
    uid
    Blob.blob_type
    Blob.md
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

type QueryMut struct {
    Q string  // query
    S string  // set
    D string  // delete
}

// @uture: with Go.18 rewrite this module with generics
//
// make QueryMut and Queries/Mutations {
//      Q string // query blocks
//      S string // mutations block (for dql mutaitons)
//      T expected type/
// }
//
// rewrite Meta and Meta_patch for as the main generic function to uses the librairies of queries,
// replacing all the singular functions here.

// GPRC/DQL Request Template
var dqlQueries map[string]string = map[string]string{
    // Count objects
    "count": `{
        all(func: uid("{{.id}}")) {
            count({{.fieldName}})
        }
    }`,
    "count2": `{
        all(func: eq({{.f1}}, "{{.v1}}")) @filter(eq({{.f2}}, "{{.v2}}")) {
            count({{.fieldName}})
        }
    }`,
    "countHas": `{
        all(func: has({{.fieldName}})) {
            count(uid)
        }
    }`,
    "countHas2": `{
        all(func: has({{.fieldName}})) @filter(eq({{.f2}}, "{{.v2}}")) {
            count(uid)
        }
    }`,
    "count_open_contracts_from_node": `{
        var(func: eq(Node.nameid, "{{.nameid}}")) {
            Node.source {
                Blob.tension {
                    Tension.contracts @filter(eq(Contract.status, "Open")) {
                        c as count(uid)
                    }
                }
            }
        }
        all() {
            n_open_contracts: val(c)
        }
    }`,
    "count_open_contracts_from_tension": `{
        var(func: uid({{id}})) {
            Tension.contracts @filter(eq(Contract.status, "Open")) {
                c as count(uid)
            }
        }
        all() {
            n_open_contracts: val(c)
        }
    }`,
    "getOrgaAgg": `{
        var(func: eq(Node.nameid, "{{.nameid}}")) {
            Node.children @filter(eq(Node.role_type, "Guest")) {
                guest as count(uid)
            }
        }
        var(func: eq(Node.nameid, "{{.nameid}}")) {
            Node.children @filter(eq(Node.role_type, "Member") OR eq(Node.role_type, "Owner")) {
                member as count(uid)
            }
        }
        all() {
            n_members: sum(val(member))
            n_guests: sum(val(guest))
        }
    }`,
    "getNodeHistory": `{
        var(func: eq(Node.nameid, "{{.nameid}}")) {
            n1 as uid
            n2 as Node.children
        }

        var(func: uid(n1, n2)) {
            Node.tensions_in {
                h as Tension.history
            }
        }

        all(func: uid(h), first:25, orderdesc: Post.createdAt) @filter(NOT eq(Event.event_type, "BlobCreated")) {
            Post.createdAt
            Post.createdBy { User.username }
            Event.event_type
            Event.tension {
                uid
                Tension.title
                Tension.receiver { Node.name Node.nameid }
            }
        }
    }`,
    // Get the total number of roles and circle recursively
    //    var(func: eq(Node.nameid, "{{.nameid}}")) @recurse {
    //        c as Node.children @filter(eq(Node.isArchived, false))
    //    }
    //    var(func: uid(c)) @filter(eq(Node.type_, "Circle")) {
    //        circle as count(uid)
    //    }
    //    var(func: uid(c)) @filter(eq(Node.type_, "Role")) {
    //        role as count(uid)
    //    }

    //    all() {
    //        n_member: sum(val(member))
    //        n_guest: sum(val(guest))
    //        n_role: sum(val(role))
    //        n_circle: sum(val(circle))
    //    }
    //}`,
    // Get literal value
    "getID": `{
        all(func: eq({{.fieldName}}, "{{.value}}")) {{.filter}} { uid }
    }`,
    "getFieldById": `{
        all(func: uid("{{.id}}")) {
            {{.fieldName}}
        }
    }`,
    "getFieldByEq": `{
        all(func: eq({{.fieldid}}, "{{.value}}")) {
            {{.fieldName}}
        }
    }`,
    "getSubFieldById": `{
        all(func: uid("{{.id}}")) {
            {{.fieldNameSource}} {
                {{.fieldNameTarget}}
            }
        }
    }`,
    "getSubFieldByEq": `{
        all(func: eq({{.fieldid}}, "{{.value}}")) {
            {{.fieldNameSource}} {
                {{.fieldNameTarget}}
            }
        }
    }`,
    "getSubSubFieldById": `{
        all(func: uid({{.id}})) {
            {{.fieldNameSource}} {
                {{.fieldNameTarget}} {
                    {{.subFieldNameTarget}}
                }
            }
        }
    }`,
    "getSubSubFieldByEq": `{
        all(func: eq({{.fieldid}}, "{{.value}}")) {
            {{.fieldNameSource}} {
                {{.fieldNameTarget}} {
                    {{.subFieldNameTarget}}
                }
            }
        }
    }`,
    "getUser": `{
        all(func: eq(User.{{.fieldid}}, "{{.userid}}"))
        {{.payload}}
    }`,
    "getUserRoles": `{
        var(func: eq(User.username, "{{.userid}}")) {
            r as User.roles
        }

        all(func: uid(r)) {
            Node.nameid
            Node.name
            Node.role_type
        }

    }`,
    "getPendingUser": `{
        all(func: eq(PendingUser.{{.k}}, "{{.v}}")) {
            PendingUser.username
            PendingUser.email
            PendingUser.password
            PendingUser.updatedAt
            PendingUser.subscribe
        }
    }`,
    "getNode": `{
        all(func: eq(Node.{{.fieldid}}, "{{.objid}}"))
        {{.payload}}
    }`,
    "getNodes": `{
        all(func: regexp(Node.nameid, /{{.regex}}/))
        {{.payload}}
    }`,
    "getNodesRoot": `{
        all(func: regexp(Node.nameid, /{{.regex}}/)) @filter(eq(Node.isRoot, true))
        {{.payload}}
    }`,
    "getTensionHook": `{
        all(func: uid("{{.id}}"))
        {{.payload}}
    }`,
    "getTensionSimple": `{
        all(func: uid("{{.id}}")) {
            uid
            Tension.receiver {
                Node.nameid
                Node.mode
                Node.visibility
            }
        }
    }`,
    "getContractHook": `{
        all(func: uid("{{.id}}"))
        {{.payload}}
    }`,
    "getContractHook2": `{
        all(func: eq(Contract.contractid, "{{.id}}"))
        {{.payload}}
    }`,
    // Boolean
    "exists": `{
        all(func: eq({{.fieldName}}, "{{.value}}")) {{.filter}} { uid }
    }`,
    "isChild": `{
        var(func: eq(Node.nameid, "{{.parent}}")) @recurse {
            uid
            u as Node.children @filter(eq(Node.nameid, "{{.child}}"))
        }
        all(func: uid(u)) { uid }
    }`,
    // Get multiple objects
    "getChildren": `{
        all(func: eq(Node.nameid, "{{.nameid}}"))  {
            Node.children @filter(eq(Node.isArchived, false)) {
                Node.nameid
            }
        }
    }`,
    "getCoordos": `{
        all(func: eq(Node.nameid, "{{.nameid}}")) {
            Node.children @filter((eq(Node.role_type, "Coordinator") OR eq(Node.role_type, "Owner"))
                AND eq(Node.isArchived, false) AND has(Node.first_link)) { uid }
        }
    }`,
    "getCoordos2": `{
        var(func: eq(Node.nameid, "{{.nameid}}")) {
            Node.children @filter((eq(Node.role_type, "Coordinator") OR eq(Node.role_type, "Owner")) AND eq(Node.isArchived, false)) {
                u as Node.first_link
            }
        }

        all(func: uid(u)) {
            {{.user_payload}}
        }
    }`,
    "getCoordosFromTid": `{
        var(func: uid({{.tid}})) {
            Tension.receiver {
                Node.children @filter((eq(Node.role_type, "Coordinator") OR eq(Node.role_type, "Owner")) AND eq(Node.isArchived, false)) {
                    u as Node.first_link
                }
            }
        }

        all(func: uid(u)) {
            {{.user_payload}}
        }
    }`,
    "getPeersFromTid": `{
        var(func: uid({{.tid}})) {
            Tension.receiver {
                Node.children @filter(eq(Node.role_type, "Peer") AND eq(Node.isArchived, false)) {
                    u as Node.first_link
                }
            }
        }

        all(func: uid(u)) {
            {{.user_payload}}
        }
    }`,
    "getParents": `{
        all(func: eq(Node.nameid, "{{.nameid}}")) @recurse {
            Node.parent @normalize
            Node.nameid
        }
    }`,
    "getParentFromTid": `{
        var(func: uid({{.tid}})) {
            n as Tension.receiver
        }

        all(func: uid(n)) @recurse {
            Node.parent @normalize
            Node.nameid
        }
    }`,
    "getWatchers": `{
        var(func: eq(Node.nameid, "{{.nameid}}")) {
            u as Node.watchers
        }

        all(func: uid(u)) {
            {{.user_payload}}
        }
    }`,
    "getLastComment": `{
        all(func: uid({{.tid}})) @normalize {
            title: Tension.title
            Tension.receiver {
                rootnameid: Node.rootnameid
                receiverid: Node.nameid
            }
            Tension.comments(first:1, orderdesc: Post.createdAt) @cascade {
                message: Post.message
                Post.createdBy @filter(eq(User.username, "{{.username}}"))
            }
        }
    }`,
    "getLastContractComment": `{
        all(func: uid({{.cid}})) @normalize {
            Contract.tension {
                Tension.receiver {
                    rootnameid: Node.rootnameid
                    receiverid: Node.nameid
                }
            }
            Contract.comments(first:1, orderdesc: Post.createdAt) @cascade {
                message: Post.message
                Post.createdBy @filter(eq(User.username, "{{.username}}"))
            }
        }
    }`,
    "getLastBlobTarget": `{
        all(func: uid({{.tid}})) @normalize {
            receiverid: Tension.receiverid
            Tension.blobs(first:1, orderdesc: Post.createdAt) @filter(has(Blob.pushedFlag)) {
                Blob.node {
                    nameid: NodeFragment.nameid
                    type_: NodeFragment.type_
                }
            }

        }
    }`,
    "getSubNodes": `{
        var(func: eq(Node.{{.fieldid}}, "{{.objid}}")) @recurse {
            o as Node.children
        }

        all(func: uid(o)) @filter(eq(Node.isArchived, false)) {
            Node.{{.fieldid}}
        }
    }`,
    "getSubMembers": `{
        var(func: eq(Node.{{.fieldid}}, "{{.objid}}")) @recurse {
            o as Node.children
        }

        all(func: uid(o)) @filter(has(Node.first_link) AND has(Node.role_type) AND eq(Node.isArchived, false)
                           AND NOT eq(Node.role_type, "Pending") AND NOT eq(Node.role_type, "Retired")
            ) {
            Node.createdAt
            Node.name
            Node.nameid
            Node.role_type
            Node.color
            Node.first_link { {{.user_payload}} }
            Node.parent {
                Node.nameid
            }
        }
    }`,
    "getTopLabels": `{
        var(func: eq(Node.{{.fieldid}}, "{{.objid}}")) @recurse {
            o as uid
            Node.parent @normalize
        }

        var(func: uid(o)) @filter(eq(Node.isArchived, false) AND NOT eq(Node.{{.fieldidinclude}}, "{{.objid}}")) {
            l as Node.labels
        }

        all(func: uid(l)){
            uid
            Label.name
            Label.color
            Label.nodes { Node.nameid }
        }
    }`,
    "getSubLabels": `{
        var(func: eq(Node.{{.fieldid}}, "{{.objid}}")) @recurse {
            o as Node.children
        }

        var(func: uid(o)) @filter(eq(Node.isArchived, false)) {
            l as Node.labels
        }

        all(func: uid(l)){
            uid
            Label.name
            Label.color
            Label.nodes { Node.nameid }
        }
    }`,
    "getTopRoles": `{
        var(func: eq(Node.{{.fieldid}}, "{{.objid}}")) @recurse {
            o as uid
            Node.parent @normalize
        }

        var(func: uid(o)) @filter(eq(Node.isArchived, false) AND NOT eq(Node.{{.fieldid}}, "{{.objid}}")) {
            l as Node.roles
        }

        all(func: uid(l)){
            uid
            RoleExt.name
            RoleExt.color
            RoleExt.role_type
            RoleExt.nodes { Node.nameid }
        }
    }`,
    "getSubRoles": `{
        var(func: eq(Node.{{.fieldid}}, "{{.objid}}")) @recurse {
            o as Node.children
        }

        var(func: uid(o)) @filter(eq(Node.isArchived, false)) {
            l as Node.roles
        }

        all(func: uid(l)){
            uid
            RoleExt.name
            RoleExt.color
            RoleExt.role_type
            RoleExt.nodes { Node.nameid }
        }
    }`,
    "getTensionInt": `{
        var(func: eq(Node.rootnameid, "{{.rootnameid}}")) @filter({{.nameids}}) {
            tensions as Node.tensions_in {{.tensionFilter}} @cascade {
                {{.authorsFilter}}
                {{.labelsFilter}}
            }
        }

        var(func: eq(Node.rootnameid, "{{.rootnameidProtected}}")) @filter({{.nameidsProtected}}) {
            tensionsProtected as Node.tensions_in {{.tensionFilter}} @cascade {
                Post.createdBy @filter(eq(User.username, "{{.username}}")),
                {{.labelsFilter}}
            }
        }

        all(func: uid(tensions, tensionsProtected), first:{{.first}}, offset:{{.offset}}, {{.order}}: Post.createdAt) {
            uid
            Post.createdAt
            Post.createdBy { User.username }
            Tension.receiver { Node.nameid Node.name Node.role_type }
            Tension.emitter { Node.nameid Node.name Node.role_type }
            Tension.title
            Tension.status
            Tension.type_
            Tension.action
            Tension.labels { uid Label.name Label.color }
            n_comments: count(Tension.comments)
        }
    }`,
    "getTensionExt": `{
        var(func: eq(Node.rootnameid, "{{.rootnameid}}")) @filter({{.nameids}}) {
            tensions_in as Node.tensions_in {{.tensionFilter}} @cascade {
                Tension.emitter @filter(NOT ({{.nameids}}))
                {{.authorsFilter}}
                {{.labelsFilter}}
            }
            tensions_out as Node.tensions_out {{.tensionFilter}} @cascade {
                Tension.receiver @filter(NOT ({{.nameids}}))
                {{.authorsFilter}}
                {{.labelsFilter}}
            }
        }

        all(func: uid(tensions_in, tensions_out), first:{{.first}}, offset:{{.offset}}, {{.order}}: Post.createdAt) {
            uid
            Post.createdAt
            Post.createdBy { User.username }
            Tension.receiver { Node.nameid Node.name Node.role_type }
            Tension.emitter { Node.nameid Node.name Node.role_type }
            Tension.title
            Tension.status
            Tension.type_
            Tension.action
            Tension.labels { uid Label.name Label.color }
            n_comments: count(Tension.comments)
        }
    }`,
    "getTensionAll": `{
        var(func: eq(Node.rootnameid, "{{.rootnameid}}")) @filter({{.nameids}}) {
            tensions as Node.tensions_in {{.tensionFilter}} @cascade {
                {{.authorsFilter}}
                {{.labelsFilter}}
            }
        }

        var(func: eq(Node.rootnameid, "{{.rootnameidProtected}}")) @filter({{.nameidsProtected}}) {
            tensionsProtected as Node.tensions_in {{.tensionFilter}} @cascade {
                Post.createdBy @filter(eq(User.username, "{{.username}}")),
                {{.labelsFilter}}
            }
        }

        all(func: uid(tensions, tensionsProtected), first:{{.first}}, offset:{{.offset}}, {{.order}}: Post.createdAt) {
            uid
            Post.createdAt
            Post.createdBy { User.username }
            Tension.receiver { Node.nameid Node.name Node.role_type }
            Tension.emitter { Node.nameid Node.name Node.role_type }
            Tension.title
            Tension.status
            Tension.type_
            Tension.action
            Tension.labels { uid Label.name Label.color }
            n_comments: count(Tension.comments)

            Tension.assignees { User.username User.name }
        }
    }`,
    "getTensionCount": `{
        var(func: eq(Node.rootnameid, "{{.rootnameid}}")) @filter({{.nameids}}) {
            tensions as Node.tensions_in {{.tensionFilter}} @cascade {
                {{.authorsFilter}}
                {{.labelsFilter}}
            }
        }

        var(func: eq(Node.rootnameid, "{{.rootnameidProtected}}")) @filter({{.nameidsProtected}}) {
            tensionsProtected as Node.tensions_in {{.tensionFilter}} @cascade {
                Post.createdBy @filter(eq(User.username, "{{.username}}")),
                {{.labelsFilter}}
            }
        }

        all(func: uid(tensions, tensionsProtected)) @filter(eq(Tension.status, "Open")) {
            count: count(uid)
        }
        all2(func: uid(tensions, tensionsProtected)) @filter(eq(Tension.status, "Closed")) {
            count: count(uid)
        }
    }`,
    "getEventCount": `{
		var(func: eq(User.username, "{{.username}}")) {
			User.events @filter(eq(UserEvent.isRead, "false")) {
				ev as UserEvent.event(first:1)
			}
            User.tensions_assigned @filter(eq(Tension.status, "Open")) {
                t as count(uid)
            }
		}
		var(func: uid(ev)) @filter(NOT type(Contract)) {
			e as count(uid)
		}
		var(func: uid(ev)) @filter(type(Contract)) {
			c as count(uid)
		}

		all() {
			unread_events: sum(val(e))
			pending_contracts: sum(val(c))
            assigned_tensions: sum(val(t))
		}
    }`,
    "getMembers": `{
        all(func: eq(Node.nameid, "{{.nameid}}")) @filter({{.nameids}}) @normalize {
            Node.children @filter(eq(Node.role_type, "Owner") OR eq(Node.role_type, "Member") OR eq(Node.role_type, "Guest")) {
                Node.first_link {
                    username: User.username
                }
            }
        }
    }`,
    // Deletion - Used by DeepDelete
    "deleteTension": `{
        id as var(func: uid({{.id}})) {
          rid_emitter as Tension.emitter
          rid_receiver as Tension.receiver
          comments as Tension.comments {
            reactions as Comment.reactions
          }
          b as Tension.blobs {
              bn as Blob.node {
                  m as NodeFragment.mandate
              }
          }
          c as Tension.contracts {
              e as Contract.event
              votes as Contract.participants
              comments2 as Contract.comments {
                reactions2 as Comment.reactions
              }
          }
          events as Tension.history
          mentions as Tension.mentions
        }
        all(func: uid(id,comments,reactions,events,mentions,b,bn,m,c,e,votes,comments2,reactions2)) {
            all_ids as uid
        }
    }`,
    "deleteContract": `{
        id as var(func: uid({{.id}})) {
          rid as Contract.tension
          a as Contract.event
          b as Contract.participants
          c as Contract.comments {
            r as Comment.reactions
          }
        }
        all(func: uid(id,a,b,c,r)) {
            all_ids as uid
        }
    }`,
}

var dqlMutations map[string]QueryMut = map[string]QueryMut{
    // Set
    "markAllAsRead": QueryMut{
        Q: `query {
            var(func: eq(User.username, "{{.username}}")) {
                uids as User.events @filter(eq(UserEvent.isRead, "false")) @cascade {
                    UserEvent.event @filter(NOT type(Contract))
                }
            }
        }`,
        S: `uid(uids) <UserEvent.isRead> "true" .`,
    },
    "markContractAsRead": QueryMut{
        Q: `query {
            var(func: eq(User.username, "{{.username}}")) {
                uids as User.events @filter(eq(UserEvent.isRead, "false")) @cascade {
                    UserEvent.event @filter(uid({{.id}}))
                }
            }
        }`,
        S: `uid(uids) <UserEvent.isRead> "true" .`,
    },
    "setPendingUserToken": QueryMut{
        Q: `query {
            var(func: eq(PendingUser.email, "{{.email}}")) @filter(NOT has(PendingUser.token)) {
                u as uid
            }
        }`,
        S: `uid(u) <PendingUser.token> "{{.token}}" .`,
    },
    "setNodeVisibility": QueryMut{
        Q: `query {
            var(func: eq(Node.nameid, "{{.nameid}}")) {
                n as uid
                Node.source {
                    nf as Blob.node
                }
            }
        }`,
        S: `
        uid(n) <Node.visibility> "{{.value}}" .
        uid(nf) <NodeFragment.visibility> "{{.value}}" .
        `,
    },
    "movePinnedTension": QueryMut{
        Q: `query {
            var(func: eq(Node.nameid, "{{.nameid_old}}")) {
                n_old as uid
                Node.pinned @filter(uid({{.tid}})) {
                    t as uid
                }
            }

            n_new as var(func: eq(Node.nameid, "{{.nameid_new}}"))
        }`,
        S: `
        uid(n_new) <Node.pinned> uid(t) .
        `,
        D:`
        uid(n_old) <Node.pinned> uid(t) .
        `,
    },
    "rewriteLabelEvents": QueryMut{
        Q: `query {
            var(func: eq(Node.rootnameid, "{{.rootnameid}}")) {
                Node.tensions_in {
                    events as Tension.history @filter(eq(Event.event_type, ["LabelAdded", "LabelRemoved"]))
                }
            }

            var(func: uid(events)) @filter(eq(Event.event_type, "LabelAdded") AND eq(Event.new, "{{.old_name}}")) {
                e_added as uid
            }
            var(func: uid(events)) @filter(eq(Event.event_type, "LabelRemoved") AND eq(Event.old, "{{.old_name}}")) {
                e_removed as uid
            }

        }`,
        S: `
        uid(e_added) <Event.new> "{{.new_name}}" .
        uid(e_removed) <Event.old> "{{.new_name}}" .
        `,

    },
    // Delete
    "removeAssignedTension": QueryMut{
        Q: `query {
            var(func: eq(User.username, "{{.username}}")) {
                u as uid
                t as User.tensions_assigned @cascade {
                    Tension.receiver @filter(eq(Node.rootnameid, "{{.rootnameid}}"))
                }
            }
        }`,
        D: `
        uid(u) <User.tensions_assigned> uid(t) .
        uid(t) <Tension.assignees> uid(u) .
        `,

    },
}

//
// Gprc/DQL requests
//

// Count count the number of object in fieldName attribute for given type and id
// Returns: int or -1 if nothing is found.
func (dg Dgraph) Count(id string, fieldName string) int {
    // Format Query
    maps := map[string]string{
        "id":id, "fieldName":fieldName,
    }
    // Send request
    res, err := dg.QueryDql("count", maps)
    if err != nil { log.Printf("Error in db.Count: %v", err); return -1 }

    // Decode response
    var r DqlRespCount
    err = json.Unmarshal(res.Json, &r)
    if err != nil { log.Printf("Error in db.Count: %v", err); return -1 }

    // Extract result
    if len(r.All) == 0 { return -1 }

    values := make([]int, 0, len(r.All[0]))
    for _, v := range r.All[0] {
        values = append(values, v)
    }

    return values[0]
}
func (dg Dgraph) Count2(f1, v1, f2, v2, fieldName string) int {
    // Format Query
    maps := map[string]string{
        "f1": f1,
        "v1": v1,
        "f2": f2,
        "v2": v2,
        "fieldName": fieldName,
    }
    // Send request
    res, err := dg.QueryDql("count2", maps)
    if err != nil { log.Printf("Error in db.Count2: %v", err); return -1 }

    // Decode response
    var r DqlRespCount
    err = json.Unmarshal(res.Json, &r)
    if err != nil { log.Printf("Error in db.Count2: %v", err); return -1 }

    // Extract result
    if len(r.All) == 0 { return -1 }

    values := make([]int, 0, len(r.All[0]))
    for _, v := range r.All[0] {
        values = append(values, v)
    }

    return values[0]
}

func (dg Dgraph) CountHas(fieldName string) int {
    // Format Query
    maps := map[string]string{
        "fieldName":fieldName,
    }
    // Send request
    res, err := dg.QueryDql("countHas", maps)
    if err != nil { log.Printf("Error in db.CountHas: %v", err); return -1 }

    // Decode response
    var r DqlRespCount
    err = json.Unmarshal(res.Json, &r)
    if err != nil { log.Printf("Error in db.CountHas: %v", err); return -1 }

    // Extract result
    if len(r.All) == 0 { return -1 }

    values := make([]int, 0, len(r.All[0]))
    for _, v := range r.All[0] {
        values = append(values, v)
    }

    return values[0]
}
func (dg Dgraph) CountHas2(fieldName, f2, v2 string) int {
    // Format Query
    maps := map[string]string{
        "fieldName":fieldName,
        "f2": f2,
        "v2": v2,
    }
    // Send request
    res, err := dg.QueryDql("countHas2", maps)
    if err != nil { log.Printf("Error in db.CountHas2: %v", err); return -1 }

    // Decode response
    var r DqlRespCount
    err = json.Unmarshal(res.Json, &r)
    if err != nil { log.Printf("Error in db.CountHas2: %v", err); return -1 }

    // Extract result
    if len(r.All) == 0 { return -1 }

    values := make([]int, 0, len(r.All[0]))
    for _, v := range r.All[0] {
        values = append(values, v)
    }

    return values[0]
}

func (dg Dgraph) Meta(f string, maps map[string]string) ([]map[string]interface{}, error) {
    var res *api.Response
    var err error

    if _, ok := dqlQueries[f]; ok { // Query Case
        // Send request
        res, err = dg.QueryDql(f, maps)
        if err != nil { return nil, err }
    } else { // Mutation Case
        // Send request
        //err := dg.MutateWithQueryDql(query, mutation)
        // @codefactor: unify api...
        res, err = dg.MutateWithQueryDql2(f, maps)
        return nil, err
    }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    // Extract result
    x := make([]map[string]interface{}, len(r.All))
    for i, s := range r.All {
        y := make(map[string]interface{}, len(s))
        for n, m := range CleanCompositeName(s, true) {
            y[n] = m
        }
        x[i] = y
    }
    return x, err
}

func (dg Dgraph) Meta1(f string, maps map[string]string, data interface{}) (error) {
	x, err := dg.Meta(f, maps)
    if err != nil { return err }
    if len(x) > 0 {
        Map2Struct(x[0], data)
    }
    return nil
}

// Probe if an object exists.
func (dg Dgraph) Exists(fieldName string, value string, filter *string) (bool, error) {
    // Format Query
    maps := map[string]string{
        "fieldName": fieldName,
        "value": value,
        "filter": "",
    }
    if filter != nil {
        maps["filter"] = fmt.Sprintf(`@filter(%s)`, *filter )
    }
    // Send request
    res, err := dg.QueryDql("exists", maps)
    if err != nil { return false, err }
    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return false, err }
    return len(r.All) > 0, nil
}

//IsChild returns true is a node parent has the given child.
func (dg Dgraph) IsChild(parent, child string) (bool, error) {
    // Format Query
    maps := map[string]string{
        "parent": parent,
        "child": child,
    }
    // Send request
    res, err := dg.QueryDql("isChild", maps)
    if err != nil { return false, err }
    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return false, err }
    return len(r.All) > 0, nil
}

// Returns the uids of the objects if found.
func (dg Dgraph) GetIDs(fieldName string, value string, filterName, filterValue *string) ([]string, error) {
    result := []string{}
    // Format Query
    maps := map[string]string{
        "fieldName":fieldName,
        "value": value,
        "filter": "",
    }
    if filterName != nil {
        maps["filter"] = fmt.Sprintf(`@filter(eq(%s, "%s"))`, *filterName, *filterValue )
    }
    // Send request
    res, err := dg.QueryDql("getID", maps)
    if err != nil { return result, err }
    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return result, err }
    for _, x := range r.All {
        result = append(result, x["uid"].(string))
    }
    return result, nil
}

// Returns a field from id
func (dg Dgraph) GetFieldById(id string, fieldName string) (interface{}, error) {
    // Format Query
    maps := map[string]string{
        "id": id,
        "fieldName": fieldName,
    }
    // Send request
    res, err := dg.QueryDql("getFieldById", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    fields := strings.Split(strings.Trim(fieldName, " "), " ")

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in DQL query: %s %s", fieldName, id)
    } else if len(r.All) == 1 {
        if len(fields) > 1 {
            return CleanCompositeName(r.All[0], true), nil
        } else {
            return r.All[0][fieldName], nil
        }
    }
    return nil, err
}

// Returns a field from objid
func (dg Dgraph) GetFieldByEq(fieldid string, objid string, fieldName string) (interface{}, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "value": objid,
        "fieldName": fieldName,
    }
    // Send request
    res, err := dg.QueryDql("getFieldByEq", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    fields := strings.Split(strings.Trim(fieldName, " "), " ")

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in DQL query: %s %s", fieldName, objid)
    } else if len(r.All) == 1 {
        if len(fields) > 1 {
            return CleanCompositeName(r.All[0], true), nil
        } else {
            return r.All[0][fieldName], nil
        }
    }
    return nil, err
}

// Returns a subfield from uid
func (dg Dgraph) GetSubFieldById(id string, fieldNameSource string, fieldNameTarget string) (interface{}, error) {
    // Format Query
    maps := map[string]string{
        "id": id,
        "fieldNameSource": fieldNameSource,
        "fieldNameTarget": fieldNameTarget,
    }
    // Send request
    res, err := dg.QueryDql("getSubFieldById", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    fields := strings.Split(strings.Trim(fieldNameTarget, " "), " ")

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in DQL query")
    } else if len(r.All) == 1 {
        switch x := r.All[0][fieldNameSource].(type) {
        case model.JsonAtom:
            if x != nil {
                if len(fields) > 1 {
                    return CleanCompositeName(x, true), nil
                } else {
                    return x[fieldNameTarget], nil
                }
            }
        case []interface{}:
            if x != nil {
                var y []interface{}
                for _, v := range(x) {
                    if len(fields) > 1 {
                        y = append(y, CleanCompositeName(v.(model.JsonAtom), true))
                    } else {
                        y = append(y, v.(model.JsonAtom)[fieldNameTarget])
                    }
                }
                return y, nil
            }
        default:
            return nil, fmt.Errorf("Decode type unknonwn: %T", x)
        }
    }
    return nil, err
}

// Returns a subfield from Eq
func (dg Dgraph) GetSubFieldByEq(fieldid string, value string, fieldNameSource string, fieldNameTarget string) (interface{}, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "value": value,
        "fieldNameSource": fieldNameSource,
        "fieldNameTarget": fieldNameTarget,
    }
    // Send request
    res, err := dg.QueryDql("getSubFieldByEq", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    fields := strings.Split(strings.Trim(fieldNameTarget, " "), " ")

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in DQL query")
    } else if len(r.All) == 1 {
        switch x := r.All[0][fieldNameSource].(type) {
        case model.JsonAtom:
            if x != nil {
                if len(fields) > 1 {
                    return CleanCompositeName(x, true), nil
                } else {
                    return x[fieldNameTarget], nil
                }
            }
        case []interface{}:
            if x != nil {
                var y []interface{}
                for _, v := range(x) {
                    if len(fields) > 1 {
                        y = append(y, CleanCompositeName(v.(model.JsonAtom), true))
                    } else {
                        y = append(y, v.(model.JsonAtom)[fieldNameTarget])
                    }
                }
                return y, nil
            }
        default:
            return nil, fmt.Errorf("Decode type unknonwn: %T", x)
        }
    }
    return nil, err
}

// Returns a subsubfield from uid
func (dg Dgraph) GetSubSubFieldById(id string, fieldNameSource string, fieldNameTarget string, subFieldNameTarget string) (interface{}, error) {
    // Format Query
    maps := map[string]string{
        "id": id,
        "fieldNameSource": fieldNameSource,
        "fieldNameTarget": fieldNameTarget,
        "subFieldNameTarget": subFieldNameTarget,
    }
    // Send request
    res, err := dg.QueryDql("getSubSubFieldById", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    fields := strings.Split(strings.Trim(subFieldNameTarget, " "), " ")

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in DQL query")
    } else if len(r.All) == 1 {
        x := r.All[0][fieldNameSource].(model.JsonAtom)
        if x != nil {
            y := x[fieldNameTarget].(model.JsonAtom)
            if y != nil {
                if len(fields) > 1 {
                    return CleanCompositeName(y, true), nil
                } else {
                    return y[subFieldNameTarget], nil
                }
            }
        }
    }
    return nil, err
}

// Returns a subsubfield from Eq
func (dg Dgraph) GetSubSubFieldByEq(fieldid string, value string, fieldNameSource string, fieldNameTarget string, subFieldNameTarget string) (interface{}, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "value": value,
        "fieldNameSource": fieldNameSource,
        "fieldNameTarget": fieldNameTarget,
        "subFieldNameTarget": subFieldNameTarget,
    }
    // Send request
    res, err := dg.QueryDql("getSubSubFieldByEq", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    fields := strings.Split(strings.Trim(subFieldNameTarget, " "), " ")

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in DQL query")
    } else if len(r.All) == 1 {
        x := r.All[0][fieldNameSource].(model.JsonAtom)
        if x != nil {
            y := x[fieldNameTarget].(model.JsonAtom)
            if y != nil {
                if len(fields) > 1 {
                    return CleanCompositeName(y, true), nil
                } else {
                    return y[subFieldNameTarget], nil
                }
            }
        }
    }
    return nil, err
}

// Returns the user context
func (dg Dgraph) GetUctxFull(fieldid string, userid string) (*model.UserCtx, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "userid": userid,
        "payload": userCtxPayload,
    }
    // Send request
    res, err := dg.QueryDql("getUser", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var user model.UserCtx
    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple user with same @id: %s, %s", fieldid, userid)
    } else if len(r.All) == 1 {
        config := &mapstructure.DecoderConfig{
            Result: &user,
            TagName: "json",
            DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
                if to == reflect.Struct {
                    nv := CleanCompositeName(v.(map[string]interface{}), false)
                    return nv, nil
                }
                return v, nil
            },
        }
        decoder, err := mapstructure.NewDecoder(config)
        if err != nil { return nil, err }
        err = decoder.Decode(r.All[0])
        if err != nil { return nil, err }
    }
    user.Hit++ // Avoid reloading user during the session context
    return &user, err
}

// Returns the user roles
func (dg Dgraph) GetUserRoles(userid string) ([]*model.Node, error) {
    // Format Query
    maps := map[string]string{
        "userid": userid,
    }
    // Send request
    res, err := dg.QueryDql("getUserRoles", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data []*model.Node
    config := &mapstructure.DecoderConfig{
        Result: &data,
        TagName: "json",
        DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
            if to == reflect.Struct {
                nv := CleanCompositeName(v.(map[string]interface{}), false)
                return nv, nil
            }
            return v, nil
        },
    }
    decoder, err := mapstructure.NewDecoder(config)
    if err != nil { return nil, err }
    err = decoder.Decode(r.All)
    return data, err
}

// Returns matching User. Never return nil user without an error.
func (dg Dgraph) GetUctx(fieldid string, userid string) (*model.UserCtx, error) {
    user, err := dg.GetUctxFull(fieldid, userid)
    if err != nil { return user, nil }
    if user == nil || user.Username == "" {
        return nil, fmt.Errorf("User not found for '%s': %s", fieldid, userid)
    }
    // @deprecated: special role are processed in web/auth
    // Filter special roles
    //for i := 0; i < len(user.Roles); i++ {
    //    if *user.Roles[i].RoleType == model.RoleTypeRetired ||
    //    *user.Roles[i].RoleType == model.RoleTypeMember ||
    //    *user.Roles[i].RoleType == model.RoleTypePending {
    //        user.Roles = append(user.Roles[:i], user.Roles[i+1:]...)
    //        i--
    //    }
    //}
    return user, err
}

// Returns the matching nodes
func (dg Dgraph) GetNodes(regex string, isRoot bool) ([]model.Node, error) {
    // Format Query
    maps := map[string]string{
        "regex": regex,
        "payload": `{
            Node.nameid
            Node.visibility
        }`,
    }

    // Send request
    var res *api.Response
    var err error
    if isRoot {
        res, err = dg.QueryDql("getNodesRoot", maps)
    } else {
        res, err = dg.QueryDql("getNodes", maps)
    }
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data []model.Node
    config := &mapstructure.DecoderConfig{
        Result: &data,
        TagName: "json",
        DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
            if to == reflect.Struct {
                nv := CleanCompositeName(v.(map[string]interface{}), false)
                return nv, nil
            }
            return v, nil
        },
    }
    decoder, err := mapstructure.NewDecoder(config)
    if err != nil { return nil, err }
    err = decoder.Decode(r.All)
    return data, err
}

// Returns the tension hook content
func (dg Dgraph) GetTensionHook(tid string, withBlob bool, bid *string) (*model.Tension, error) {
    // Format Query
    var maps map[string]string
    if withBlob {
        var blobFilter string
        if bid == nil {
            blobFilter = "(orderdesc: Post.createdAt, first: 1)"
        } else {
            blobFilter = fmt.Sprintf(`@filter(uid(%s))`, *bid)
        }
        maps = map[string]string{
            "id": tid,
            "payload": "{" + tensionHookPayload + fmt.Sprintf(tensionBlobHookPayload, blobFilter) + "}",
        }
    } else {
        maps = map[string]string{
            "id": tid,
            "payload": "{" + tensionHookPayload + "}",
        }
    }

    // Send request
    res, err := dg.QueryDql("getTensionHook", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var obj model.Tension
    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple tension for @uid: %s", tid)
    } else if len(r.All) == 1 {
        config := &mapstructure.DecoderConfig{
            Result: &obj,
            TagName: "json",
            DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
                if to == reflect.Struct {
                    nv := CleanCompositeName(v.(map[string]interface{}), false)
                    return nv, nil
                }
                return v, nil
            },
        }
        decoder, err := mapstructure.NewDecoder(config)
        if err != nil { return nil, err }
        err = decoder.Decode(r.All[0])
        if err != nil { return nil, err }
    }

    // Assume that tension does not exists if receiver is empty
    // This is because DQL returns an object with t uid even is non existent.
    if obj.Receiver == nil { return nil, err }
    return &obj, err
}

// Returns the contract hook content
func (dg Dgraph) GetContractHook(cid string) (*model.Contract, error) {
    // Format Query
    maps := map[string]string{
        "id": cid,
        "payload": "{" + contractHookPayload + "}",
    }

    // Send request
    var q string
    if strings.Contains(cid, "#") {
        q = "getContractHook2"
    } else {
        q = "getContractHook"
    }
    res, err := dg.QueryDql(q, maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var obj model.Contract
    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple contract for @uid: %s", cid)
    } else if len(r.All) == 1 {
        config := &mapstructure.DecoderConfig{
            Result: &obj,
            TagName: "json",
            DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
                if to == reflect.Struct {
                    nv := CleanCompositeName(v.(map[string]interface{}), false)
                    return nv, nil
                }
                return v, nil
            },
        }
        decoder, err := mapstructure.NewDecoder(config)
        if err != nil { return nil, err }
        err = decoder.Decode(r.All[0])
        if err != nil { return nil, err }
    }
    return &obj, err
}

// Get all sub children
func (dg Dgraph) GetSubNodes(fieldid string, objid string) ([]model.Node, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "objid": objid,
    }
    // Send request
    res, err := dg.QueryDql("getSubNodes", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data []model.Node
    config := &mapstructure.DecoderConfig{
        Result: &data,
        TagName: "json",
        DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
            if to == reflect.Struct {
                nv := CleanCompositeName(v.(map[string]interface{}), false)
                return nv, nil
            }
            return v, nil
        },
    }
    decoder, err := mapstructure.NewDecoder(config)
    if err != nil { return nil, err }
    err = decoder.Decode(r.All)
    return data, err
}

// Get all sub members
func (dg Dgraph) GetSubMembers(fieldid, objid, user_payload string) ([]model.Node, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "objid": objid,
        "user_payload": user_payload,
    }
    // Send request
    res, err := dg.QueryDql("getSubMembers", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data []model.Node
    config := &mapstructure.DecoderConfig{
        Result: &data,
        TagName: "json",
        DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
            if to == reflect.Struct {
                nv := CleanCompositeName(v.(map[string]interface{}), false)
                return nv, nil
            }
            return v, nil
        },
    }
    decoder, err := mapstructure.NewDecoder(config)
    if err != nil { return nil, err }
    err = decoder.Decode(r.All)
    return data, err
}

// Get all top labels
func (dg Dgraph) GetTopLabels(fieldid string, objid string, includeSelf bool) ([]model.Label, error) {
    // Format Query
    var fieldinclude string
    if includeSelf {
        fieldinclude = fieldid
    } else {
        fieldinclude = fieldid + "_IGNORE"
    }
    maps := map[string]string{
        "fieldid": fieldid,
        "objid": objid,
        "fieldidinclude": fieldinclude,
    }
    // Send request
    res, err := dg.QueryDql("getTopLabels", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data_dup []model.Label
    config := &mapstructure.DecoderConfig{
        Result: &data_dup,
        TagName: "json",
        DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
            if to == reflect.Struct {
                nv := CleanCompositeName(v.(map[string]interface{}), false)
                return nv, nil
            }
            return v, nil
        },
    }
    decoder, err := mapstructure.NewDecoder(config)
    if err != nil { return nil, err }
    err = decoder.Decode(r.All)
    // Remove duplicate based on Label.name
    data := []model.Label{}
    check := make(map[string]bool)
    for _, d := range data_dup {
        if _, v := check[d.Name]; !v {
            check[d.Name] = true
            data = append(data, d)
        }
    }
    return data, err
}

// Get all sub labels
func (dg Dgraph) GetSubLabels(fieldid string, objid string) ([]model.Label, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "objid": objid,
    }
    // Send request
    res, err := dg.QueryDql("getSubLabels", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data_dup []model.Label
    config := &mapstructure.DecoderConfig{
        Result: &data_dup,
        TagName: "json",
        DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
            if to == reflect.Struct {
                nv := CleanCompositeName(v.(map[string]interface{}), false)
                return nv, nil
            }
            return v, nil
        },
    }
    decoder, err := mapstructure.NewDecoder(config)
    if err != nil { return nil, err }
    err = decoder.Decode(r.All)
    // Remove duplicate based on Label.name
    data := []model.Label{}
    check := make(map[string]bool)
    for _, d := range data_dup {
        if _, v := check[d.Name]; !v {
            check[d.Name] = true
            data = append(data, d)
        }
    }
    return data, err
}

// Get all top roles
func (dg Dgraph) GetTopRoles(fieldid string, objid string) ([]model.RoleExt, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "objid": objid,
    }
    // Send request
    res, err := dg.QueryDql("getTopRoles", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data_dup []model.RoleExt
    config := &mapstructure.DecoderConfig{
        Result: &data_dup,
        TagName: "json",
        DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
            if to == reflect.Struct {
                nv := CleanCompositeName(v.(map[string]interface{}), false)
                return nv, nil
            }
            return v, nil
        },
    }
    decoder, err := mapstructure.NewDecoder(config)
    if err != nil { return nil, err }
    err = decoder.Decode(r.All)
    // Remove duplicate based on Label.name
    data := []model.RoleExt{}
    check := make(map[string]bool)
    for _, d := range data_dup {
        if _, v := check[d.Name]; !v {
            check[d.Name] = true
            data = append(data, d)
        }
    }
    return data, err
}

// Get all sub labels
func (dg Dgraph) GetSubRoles(fieldid string, objid string) ([]model.RoleExt, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "objid": objid,
    }
    // Send request
    res, err := dg.QueryDql("getSubRoles", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data_dup []model.RoleExt
    config := &mapstructure.DecoderConfig{
        Result: &data_dup,
        TagName: "json",
        DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
            if to == reflect.Struct {
                nv := CleanCompositeName(v.(map[string]interface{}), false)
                return nv, nil
            }
            return v, nil
        },
    }
    decoder, err := mapstructure.NewDecoder(config)
    if err != nil { return nil, err }
    err = decoder.Decode(r.All)
    // Remove duplicate based on Label.name
    data := []model.RoleExt{}
    check := make(map[string]bool)
    for _, d := range data_dup {
        if _, v := check[d.Name]; !v {
            check[d.Name] = true
            data = append(data, d)
        }
    }
    return data, err
}

func (dg Dgraph) GetTensions(q TensionQuery, type_ string) ([]model.Tension, error) {
    // Format Query
    maps, err := FormatTensionIntExtMap(q)
    if err != nil { return nil, err }
    // Send request
    var op string
    if type_ == "int" {
        op = "getTensionInt"
    } else if type_ == "ext" {
        op = "getTensionExt"
    } else if type_ == "all" {
        op = "getTensionAll"
    } else {
        panic("Unknow type (tension query)")
    }
    res, err := dg.QueryDql(op, *maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data []model.Tension
    config := &mapstructure.DecoderConfig{
        Result: &data,
        TagName: "json",
        DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
            if to == reflect.Struct {
                nv := CleanCompositeName(v.(map[string]interface{}), false)
                return nv, nil
            }
            return v, nil
        },
    }
    decoder, err := mapstructure.NewDecoder(config)
    if err != nil { return nil, err }
    err = decoder.Decode(r.All)
    return data, err
}

func (dg Dgraph) GetTensionsCount(q TensionQuery) (map[string]int, error) {
    // Format Query
    maps, err := FormatTensionIntExtMap(q)
    if err != nil { return nil, err }
    // Send request
    res, err := dg.QueryDql("getTensionCount", *maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlRespCount
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    if len(r.All) > 0 && len(r.All2) > 0 {
        v := map[string]int{"open":r.All[0]["count"], "closed":r.All2[0]["count"]}
        return v, err
    }
    return nil, err
}

func (dg Dgraph) GetLastBlobId(tid string) (*string) {
    // init client
    dgc, cancel := dg.getDgraphClient()
    defer cancel()
    ctx := context.Background()
    txn := dgc.NewTxn()
    defer txn.Discard(ctx)

    q := fmt.Sprintf(`{all(func: uid(%s))  {
        Tension.blobs (orderdesc: Post.createdAt, first: 1) { uid }
    }}`, tid)
    // Send request
    res, err := txn.Query(ctx, q)
    if err != nil { return nil}

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil}

    var bid string
    if len(r.All) > 1 {
        return nil
    } else if len(r.All) == 1 {
        blobs := r.All[0]["Tension.blobs"].([]interface{})
        if len(blobs) > 0 {
            bid = blobs[0].(model.JsonAtom)["uid"].(string)
        }
    }

    return &bid
}

// Get all coordo roles in the given circle with an user linked.
func (dg Dgraph) HasCoordos(nameid string) (bool) {
    // Format Query
    maps := map[string]string{
        "nameid": nameid,
    }
    // Send request
    res, err := dg.QueryDql("getCoordos", maps)
    if err != nil { return false }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return false }

    var ok bool = false
    if len(r.All) > 1 {
        return ok
    } else if len(r.All) == 1 {
        c := r.All[0]["Node.children"]
        if c != nil && len(c.([]interface{})) > 0 {
            ok = true
        }
    }
    return ok
}

// Get children
func (dg Dgraph) GetChildren(nameid string) ([]string, error) {
    // Format Query
    maps := map[string]string{
        "nameid" :nameid,
    }
    // Send request
    res, err := dg.QueryDql("getChildren", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data []string
    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple object for term: %s", nameid)
    } else if len(r.All) == 1 {
        c := r.All[0]["Node.children"].([]interface{})
        for _, x := range(c) {
            data = append(data, x.(model.JsonAtom)["Node.nameid"].(string))
        }
    }
    return data, err
}

// Get path to root
func (dg Dgraph) GetParents(nameid string) ([]string, error) {
    // Format Query
    maps := map[string]string{
        "nameid" :nameid,
    }
    // Send request
    res, err := dg.QueryDql("getParents", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data []string
    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple object for term: %s", nameid)
    } else if len(r.All) == 1 {
        // f%$*µ%ing decoding
        parents := r.All[0]["Node.parent"]
        if parents == nil {return data, err}
        switch p := parents.([]interface{})[0].(model.JsonAtom)["Node.nameid"].(type) {
        case []interface{}:
            for _, x := range(p) {
                data = append(data, x.(string))
            }
        case string:
            data = append(data, p)
        }
    }
    return data, err
}

// DQL Mutations

// SetFieldById set a predicate for the given node in the DB
func (dg Dgraph) SetFieldById(objid string, predicate string, val string) error {
    query := fmt.Sprintf(`query {
        node as var(func: uid(%s))
    }`, objid)

    mu := fmt.Sprintf(`
        uid(node) <%s> "%s" .
    `, predicate, val)

    mutation := &api.Mutation{
        SetNquads: []byte(mu),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}
// SetFieldByEq set a predicate for the given node in the DB
func (dg Dgraph) SetFieldByEq(fieldid string, objid string, predicate string, val string) error {
    query := fmt.Sprintf(`query {
        node as var(func: eq(%s, "%s"))
    }`, fieldid, objid)

    mu := fmt.Sprintf(`
        uid(node) <%s> "%s" .
    `, predicate, val)

    mutation := &api.Mutation{
        SetNquads: []byte(mu),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}
//SetSubFieldByEq set a predicate for the given node in the DB
func (dg Dgraph) SetSubFieldByEq(fieldid string, objid string, predicate1, predicate2 string, val string) error {
    query := fmt.Sprintf(`query {
        var(func: eq(%s, "%s")) {
            x as %s
        }
    }`, fieldid, objid, predicate1)

    mu := fmt.Sprintf(`
        uid(x) <%s> "%s" .
    `, predicate2, val)

    mutation := &api.Mutation{
        SetNquads: []byte(mu),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}

// Set the blob pushedFlag and the tension action
func (dg Dgraph) SetPushedFlagBlob(bid string, flag string, tid string, action model.TensionAction) error {
    query := fmt.Sprintf(`query {
        obj as var(func: uid(%s))
    }`, bid)

    mu := fmt.Sprintf(`
        uid(obj) <Blob.pushedFlag> "%s" .
        <%s> <Tension.action> "%s" .
    `, flag, tid, action)
    muDel := `uid(obj) <Blob.archivedFlag> * . `

    mutation := &api.Mutation{
        SetNquads: []byte(mu),
        DelNquads: []byte(muDel),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}

// Set the blob pushedFlag and the tension action
func (dg Dgraph) SetArchivedFlagBlob(bid string, flag string, tid string, action model.TensionAction) error {
    query := fmt.Sprintf(`query {
        obj as var(func: uid(%s))
    }`, bid)

    mu := fmt.Sprintf(`
        uid(obj) <Blob.archivedFlag> "%s" .
        <%s> <Tension.action> "%s" .
    `, flag, tid, action)

    mutation := &api.Mutation{
        SetNquads: []byte(mu),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}

func (dg Dgraph) SetNodeSource(nameid string, bid string) error {
    query := fmt.Sprintf(`query {
        node as var(func: eq(Node.nameid, "%s"))
    }`, nameid)

    mu := fmt.Sprintf(`
        uid(node) <Node.source> <%s> .
    `, bid)

    mutation := &api.Mutation{
        SetNquads: []byte(mu),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}

func (dg Dgraph) SetChildrenRoleVisibility(nameid string, value string) error {
    query := fmt.Sprintf(`query {
        var(func: eq(Node.nameid, "%s")) {
            c as Node.children @filter(eq(Node.type_, "Role"))
        }
    }`, nameid)

    mu := fmt.Sprintf(`
        uid(c) <Node.visibility> "%s" .
    `, value)

    mutation := &api.Mutation{
        SetNquads: []byte(mu),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}
// PatchNameid is used to update a node nameid
func (dg Dgraph) PatchNameid(nameid_old string, nameid_new string) error {
    query := fmt.Sprintf(`query {
        node as var(func: eq(Node.nameid, "%s")) {
            tin as Node.tensions_in
            tout as Node.tensions_out
        }
    }`, nameid_old)

    mu := fmt.Sprintf(`
        uid(node) <Node.nameid> "%s" .
        uid(tin) <Tension.receiverid> "%s" .
        uid(tout) <Tension.emitterid> "%s" .
    `, nameid_new, nameid_new, nameid_new)

    mutation := &api.Mutation{
        SetNquads: []byte(mu),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}

// UpdateRoleType update the role of a node given the nameid using upsert block.
func (dg Dgraph) UpgradeMember(nameid string, roleType model.RoleType) error {
    query := fmt.Sprintf(`query {
        node as var(func: eq(Node.nameid, "%s"))
    }`, nameid)

    mu := fmt.Sprintf(`
        uid(node) <Node.role_type> "%s" .
        uid(node) <Node.name> "%s" .
    `, roleType, roleType)

    mutation := &api.Mutation{
        SetNquads: []byte(mu),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}

// Remove the user link in the last blob if user match
func (dg Dgraph) MaybeDeleteFirstLink(tid, username string) error {
    query := fmt.Sprintf(`query {
        var(func: uid(%s)) {
          Tension.blobs (orderdesc: Post.createdAt, first: 1) {
            n as Blob.node @filter(eq(NodeFragment.first_link, "%s"))
          }
        }
    }`, tid, username)

    muDel := `uid(n) <NodeFragment.first_link> * .`

    mutation := &api.Mutation{
        DelNquads: []byte(muDel),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}

// Rewrite Contractid and voteid as they are use to upsert contracts and votes.
func (dg Dgraph) RewriteContractId(cid string) error {
    query := fmt.Sprintf(`query {
        var(func: uid(%s)) {
            cuid as uid
            Contract.participants {
                vuid as uid
            }
        }
    }`, cid)

    mu := `
        uid(cuid) <Contract.contractid> "" .
        uid(vuid) <Vote.voteid> "" .
    `

    mutation := &api.Mutation{
        SetNquads: []byte(mu),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}

// Deletions

// DeepDelete delete edges recursively for type {t} and id {id}.
// If {rid} is given, it represents the reverse node that should be cleaned up.
func (dg Dgraph) DeepDelete(t string, id string) (error) {
    var reverse string
    var query string

    switch t {
    case "tension":
        reverse = fmt.Sprintf(`
            uid(rid_emitter) <Node.tensions_out> uid(id) .
            uid(rid_receiver) <Node.tensions_in> uid(id) .
        `)
    case "contract":
        reverse = fmt.Sprintf(`uid(rid) <Tension.contracts> uid(id) .`)
    default:
        return fmt.Errorf("delete query not implemented for this type %s", t)
    }

    maps := map[string]string{"id": id}
    query = dg.getDqlQuery("delete" + strings.Title(t), maps)
    mu := fmt.Sprintf(`
        %s
        uid(all_ids) * * .
    `, reverse)

    mutation := &api.Mutation{
        DelNquads: []byte(mu),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}


