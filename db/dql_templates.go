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
	"getNodeHistory": `{
        var(func: eq(Node.nameid, "{{.nameid}}")) {
            n1 as uid
            n2 as Node.children
        }

        var(func: uid(n1, n2)) {
            Node.tensions_in {{if .query}}@filter(anyoftext(Tension.title, "{{.query}}") OR anyoftext(Post.message, "{{.query}}")){{end}} {
                h_in as Tension.history
            }
            Node.tensions_out {{if .query}}@filter(anyoftext(Tension.title, "{{.query}}") OR anyoftext(Post.message, "{{.query}}")){{end}} {
                h_out as Tension.history
            }
        }

        all(func: uid(h_in, h_out), first:25, orderdesc: Post.createdAt) @filter(NOT eq(Event.event_type, "BlobCreated")) @cascade {
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
	"getSubFieldByEq2": `{
        all(func: eq({{.fieldid}}, "{{.value}}")) @filter(eq({{.f2}}, "{{.v2}}")) {
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
	"getShortestPath": `{
        A as var(func: eq(Node.nameid, "{{.from}}"))
        B as var(func: eq(Node.nameid, "{{.to}}"))

        path as shortest(from: uid(A), to: uid(B)) {
            Node.children
        }

        all(func:uid(path)) {
            weight: count(uid)
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
            PendingUser.lang
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
            o as uid
            Node.children
        }

        all(func: uid(o)) @filter(eq(Node.isArchived, false){{if .excludeSelf}} AND NOT eq(Node.{{.fieldid}}, "{{.objid}}"){{end}}) {
            Node.{{.fieldid}}
        }
    }`,
	"getSubMembers": `{
        var(func: eq(Node.{{.fieldid}}, "{{.objid}}")) @recurse {
            o as uid
            Node.children
        }

        all(func: uid(o)) @filter(has(Node.first_link) AND has(Node.role_type) AND eq(Node.isArchived, false)
                           AND NOT eq(Node.role_type, "Pending") AND NOT eq(Node.role_type, "Retired")
                           {{if .excludeSelf}}AND NOT eq(Node.{{.fieldid}}, "{{.objid}}"){{end}}
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

        var(func: uid(o)) @filter(eq(Node.isArchived, false){{if .excludeSelf}} AND NOT eq(Node.{{.fieldid}}, "{{.objid}}"){{end}}) {
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
            o as uid
            Node.children
        }

        var(func: uid(o)) @filter(eq(Node.isArchived, false){{if .excludeSelf}} AND NOT eq(Node.{{.fieldid}}, "{{.objid}}"){{end}}) {
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

        var(func: uid(o)) @filter(eq(Node.isArchived, false){{if .excludeSelf}} AND NOT eq(Node.{{.fieldid}}, "{{.objid}}"){{end}}) {
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
            o as uid
            Node.children
        }

        var(func: uid(o)) @filter(eq(Node.isArchived, false){{if .excludeSelf}} AND NOT eq(Node.{{.fieldid}}, "{{.objid}}"){{end}}) {
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
	"getSubProjects": `{
        var(func: eq(Node.{{.fieldid}}, "{{.objid}}")) @recurse {
            o as uid
            Node.children
        }

        var(func: uid(o)) @filter(eq(Node.isArchived, false){{if .excludeSelf}} AND NOT eq(Node.{{.fieldid}}, "{{.objid}}"){{end}}) {
            p as Node.projects
        }

        all(func: uid(p)) @filter(eq(Project.status, "Open")) {
            uid
            Project.updatedAt
            Project.name
            Project.description
            Project.nodes { Node.nameid Node.name }
            Project.collaborators { User.username User.name }
        }
    }`,
	"getTensionInt": `{
        {{.extra_pre_vars}}
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

        all(func: uid(tensions, tensionsProtected), first:{{.first}}, offset:{{.offset}}, {{.order}}: {{.orderBy}}) {
            {{.payload}}
        }
    }`,
	"getTensionExt": `{
        {{.extra_pre_vars}}
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

        all(func: uid(tensions_in, tensions_out), first:{{.first}}, offset:{{.offset}}, {{.order}}: {{.orderBy}}) {
            {{.payload}}
        }
    }`,
	"getTensionAll": `{
        {{.extra_pre_vars}}
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

        all(func: uid(tensions, tensionsProtected), first:{{.first}}, offset:{{.offset}}, {{.order}}: {{.orderBy}}) {
            {{.payload}}
        }
    }`,
	"getTensionCount": `{
        {{.extra_pre_vars}}
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
	"getOwners": `{
        all(func: eq(Node.nameid, "{{.nameid}}")) @normalize {
            Node.children @filter(eq(Node.role_type, "Owner")) {
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
          candidates as Contract.candidates
          user_pending as Contract.pending_candidates
          a as Contract.event
          votes as Contract.participants {
            nodes as Vote.node {
                Node.parent {
                    Node.children @filter(eq(Node.type_, "Role")) {
                        members as Node.first_link
                    }
                }
            }
          }
          c as Contract.comments {
            r as Comment.reactions
          }
        }

        var(func:uid(members)) @cascade {
            desync_events as User.events {
              UserEvent.event @filter(uid({{.id}}))
            }
        }

        all(func: uid(id,a,votes,c,r)) {
            all_ids as uid
        }
    }`,
	"getUserActivity": `{
        all(func: eq(Activity.ownerid, "u#{{.username}}"), orderdesc: Activity.date, first: 366)
        {{if .from}}@filter(between(Activity.date, "{{.from}}", "{{.to}}")){{end}}
        {
            activityid: Activity.activityid
            count: Activity.count
            date: Activity.date
        }
    }`,
	"getNodeActivity": `{
        all(func: eq(Activity.ownerid, "o#{{.rootnameid}}"), orderdesc: Activity.date, first: 366)
        {{if .from}}@filter(between(Activity.date, "{{.from}}", "{{.to}}")){{end}}
        {
            activityid: Activity.activityid
            count: Activity.count
            date: Activity.date
        }
    }`,
	"getLastBlobId": `{
        all(func: uid({{.tid}})) {
            Tension.blobs (orderdesc: Post.createdAt, first: 1) { uid }
        }
    }`,
	"getTensionSearchData": `{
        all(func: uid({{.tid}})) {
            Tension.labels { Label.name }
            Tension.comments(first:1, orderasc: Post.createdAt) {
                message: Post.message
            }
        }
    }`,
}
