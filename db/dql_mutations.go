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

type QueryMut struct {
	Q string // DQL Query
	M []X    // DQL Mutation
}
type X struct {
	S string // set
	D string // delete
	C string // condition
}

var dqlMutations map[string]QueryMut = map[string]QueryMut{
	//
	// Set
	//
	"markAllAsRead": {
		Q: `query {
            var(func: eq(User.username, "{{.username}}")) {
                uids as User.events @filter(eq(UserEvent.isRead, "false")) @cascade {
                    UserEvent.event @filter(NOT type(Contract))
                }
            }
        }`,
		M: []X{{
			S: `uid(uids) <UserEvent.isRead> "true" .`,
		}},
	},
	"markContractAsRead": {
		Q: `query {
            var(func: eq(User.username, "{{.username}}")) {
                uids as User.events @filter(eq(UserEvent.isRead, "false")) @cascade {
                    UserEvent.event @filter(uid({{.id}}))
                }
            }
        }`,
		M: []X{{
			S: `uid(uids) <UserEvent.isRead> "true" .`,
		}},
	},
	"setPendingUserToken": {
		Q: `query {
            var(func: eq(PendingUser.email, "{{.email}}")) @filter(NOT has(PendingUser.token)) {
                u as uid
            }
        }`,
		M: []X{{
			S: `uid(u) <PendingUser.token> "{{.token}}" .`,
		}},
	},
	"setNodeVisibility": {
		Q: `query {
            var(func: eq(Node.nameid, "{{.nameid}}")) {
                n as uid
                Node.source {
                    nf as Blob.node
                }
            }
        }`,
		M: []X{{
			S: `uid(n) <Node.visibility> "{{.value}}" .
                uid(nf) <NodeFragment.visibility> "{{.value}}" .
                `,
		}},
	},
	"movePinnedTension": {
		Q: `query {
            var(func: eq(Node.nameid, "{{.nameid_old}}")) {
                n_old as uid
                Node.pinned @filter(uid({{.tid}})) {
                    t as uid
                }
            }

            n_new as var(func: eq(Node.nameid, "{{.nameid_new}}"))
        }`,
		M: []X{{
			S: `uid(n_new) <Node.pinned> uid(t) . `,
			D: `uid(n_old) <Node.pinned> uid(t) . `,
		}},
	},
	"rewriteLabelEvents": {
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
		M: []X{{
			S: `uid(e_added) <Event.new> "{{.new_name}}" .
                uid(e_removed) <Event.old> "{{.new_name}}" .
                `,
		}},
	},
	"incrementCardPos": {
		Q: `query {
            var(func: uid({{.cardid}})) {
                pos as ProjectCard.pos
                ProjectCard.pc {
                    colid as uid
                }
                ProjectCard.card @filter(type(Tension)) {
                    tid as uid
                }
            }

            var(func: uid(colid)) {
                projectid as ProjectColumn.project
                ProjectColumn.cards @filter(ge(ProjectCard.pos, val(pos)) AND not uid({{.cardid}})) {
                    incrme as uid
                    p1 as ProjectCard.pos
                    new_pos_incr as math(p1 + 1)
                }
            }
        }`,
		M: []X{{
			S: `uid(incrme) <ProjectCard.pos> val(new_pos_incr) .
                uid(colid) <ProjectColumn.tensions> uid(tid) .
                uid(tid) <Tension.project_statuses> uid(colid) .
                uid(projectid) <Project.updatedAt> "{{.now}}" .
                `,
		}},
	},
	"decrementCardPos": {
		Q: `query {
            var(func: uid({{.colid}})) {
                ProjectColumn.cards @filter(gt(ProjectCard.pos, {{.pos}})) {
                    decrme as uid
                    p2 as ProjectCard.pos
                    new_pos_decr as math(p2 - 1)
                }
            }
        }`,
		M: []X{{
			S: `uid(decrme) <ProjectCard.pos> val(new_pos_decr) . `,
			D: `<{{.colid}}> <ProjectColumn.tensions> <{{.tid}}> .
                <{{.tid}}> <Tension.project_statuses> <{{.colid}}> .
                `,
		}},
	},
	"moveCardPos": {
		Q: `query {
            var(func: uid({{.cardid}})) {
                pos as ProjectCard.pos
                ProjectCard.pc {
                    colid as uid
                }
                ProjectCard.card @filter(type(Tension)) {
                    tid as uid
                }
            }

            sameCol as var(func: uid({{.old_colid}})) @filter(uid(colid))

            var(func: uid(colid)) {
                projectid as ProjectColumn.project
                ProjectColumn.cards @filter(ge(ProjectCard.pos, val(pos)) AND not uid({{.cardid}})) {
                    incrme as uid
                    p1 as ProjectCard.pos
                    new_pos_incr as math(p1 + 1)
                }
            }

            var(func: uid({{.old_colid}})) {
                ProjectColumn.cards @filter(gt(ProjectCard.pos, {{.old_pos}}) AND not uid({{.cardid}})) {
                    decrme as uid
                    p2 as ProjectCard.pos
                    new_pos_decr as math(p2 - 1)
                }
            }
        }`,
		M: []X{
			{
				S: `uid(incrme) <ProjectCard.pos> val(new_pos_incr) .
                    uid(decrme) <ProjectCard.pos> val(new_pos_decr) .
                    uid(projectid) <Project.updatedAt> "{{.now}}" .
                    uid(colid) <ProjectColumn.tensions> uid(tid) .
                    uid(tid) <Tension.project_statuses> uid(colid) .
                `,

				D: `<{{.old_colid}}> <ProjectColumn.tensions> uid(tid) .
                    uid(tid) <Tension.project_statuses> <{{.old_colid}}> .
                `,
				// do not set reverse is the column is the same
				// @obsolete (keep it as a example of @if usage)
				C: `@if(NOT eq(len(sameCol), 1))`,
			},
		},
	},
	"moveCardPosUp": {
		Q: `query {
            var(func: uid({{.cardid}})) {
                pos as ProjectCard.pos
                ProjectCard.pc {
                    colid as uid
                }
            }

            var(func: uid(colid)) {
                projectid as ProjectColumn.project
                ProjectColumn.cards @filter(ge(ProjectCard.pos, val(pos)) AND lt(ProjectCard.pos, {{.old_pos}}) AND not uid({{.cardid}})) {
                    incrme as uid
                    p1 as ProjectCard.pos
                    new_pos_incr as math(p1 + 1)
                }
            }
        }`,
		M: []X{
			{
				S: `uid(incrme) <ProjectCard.pos> val(new_pos_incr) .
                    uid(projectid) <Project.updatedAt> "{{.now}}" .
                `,
			},
		},
	},
	"moveCardPosDown": {
		Q: `query {
            var(func: uid({{.cardid}})) {
                pos as ProjectCard.pos
                ProjectCard.pc {
                    colid as uid
                }
            }

            var(func: uid(colid)) {
                projectid as ProjectColumn.project
                ProjectColumn.cards @filter(gt(ProjectCard.pos, {{.old_pos}}) AND le(ProjectCard.pos, val(pos)) AND not uid({{.cardid}})) {
                    decrme as uid
                    p1 as ProjectCard.pos
                    new_pos_decr as math(p1 - 1)
                }
            }
        }`,
		M: []X{
			{
				S: `uid(decrme) <ProjectCard.pos> val(new_pos_decr) .
                    uid(projectid) <Project.updatedAt> "{{.now}}" .
                `,
			},
		},
	},
	"upsertActivity": {
		Q: `query {
            v as var(func: eq(Activity.activityid, "{{.activityid}}")) {
                c as Activity.count
                next as math(c + 1)
            }
        }`,
		M: []X{
			{
				C: `@if(gt(len(v), 0))`,
				S: `uid(v) <Activity.count> val(next) .`,
			},
			{
				C: `@if(eq(len(v), 0))`,
				S: `_:new <dgraph.type> "Activity" .
                    _:new <Activity.activityid> "{{.activityid}}" .
                    _:new <Activity.ownerid> "{{.ownerid}}" .
                    _:new <Activity.date> "{{.date}}" .
                    _:new <Activity.count> "1" .`,
			},
		},
	},
	//
	// Delete
	//
	"removeAssignedTension": {
		Q: `query {
            var(func: eq(User.username, "{{.username}}")) {
                u as uid
                t as User.tensions_assigned @cascade {
                    Tension.receiver @filter(eq(Node.rootnameid, "{{.rootnameid}}"))
                }
            }
        }`,
		M: []X{{
			D: `uid(u) <User.tensions_assigned> uid(t) .
                uid(t) <Tension.assignees> uid(u) .
                `,
		}},
	},
	"deleteCardDraft": {
		Q: `query {
            var(func: uid({{.cardid}})) {
                c as uid
                cc as ProjectCard.card
                v as ProjectCard.values
            }
        }`,
		M: []X{{
			D: `uid(v) * *  .
                uid(cc) * * .
                uid(c) * * .
               `,
		}},
	},
	"deleteComment": {
		Q: `query {
			t as var(func: uid({{.tid}}))
            var(func: uid({{.cid}})) {
                c as uid
                reactions as Comment.reactions
                files as Comment.files
            }
        }`,
		M: []X{{
			D: `uid(t) <Tension.comments> uid(c) .
				uid(reactions) * *  .
				uid(files) * * .
				uid(c) * * .
				`,
		}},
	},
	// addFileToComment creates a File node attached to a comment.
	// Inputs (all required): cid, username, filename, contentType, sizeStr (decimal),
	//                        storageKey, now (RFC3339).
	// Caller has already verified comment and user exist via getCommentAuth + JWT,
	// so no @if guard is needed.
	"addFileToComment": {
		Q: `query {
            var(func: uid({{.cid}})) { c as uid }
            var(func: eq(User.username, "{{.username}}")) { u as uid }
        }`,
		M: []X{{
			S: `_:f <dgraph.type> "File" .
				_:f <File.storageKey> "{{.storageKey}}" .
				_:f <File.filename> "{{.filename}}" .
				_:f <File.contentType> "{{.contentType}}" .
				_:f <File.size> "{{.sizeStr}}" .
				_:f <File.createdAt> "{{.now}}" .
				_:f <File.createdBy> uid(u) .
				_:f <File.comment> uid(c) .
				uid(c) <Comment.files> _:f .
				`,
		}},
	},
	// deleteFile removes a File node by uid. The reverse edge from the parent
	// comment is dropped automatically by Dgraph when the node is deleted.
	"deleteFile": {
		Q: `query {
            var(func: uid({{.id}})) { f as uid }
        }`,
		M: []X{{
			D: `uid(f) * * .`,
		}},
	},
	// Deleting user by replacing its authoring by the ghost user.
	// * [x] Unnassigned his tension
	// * [x] Unlink his roles
	// * [x] Delete his membership
	// * [x] Delete his reactions
	// * [x] Delete his watched organisations
	// * [x] Remove his contracts (candidates)
	// * [x] Delete his UserEvents (subDelete Notif!)
	"deleteUser": {
		Q: `query {
            var(func: eq(User.username, "{{.username}}")) {
                u as uid
                ur as User.UserRights
                assigned as User.tensions_assigned
                roles as User.roles @filter(not eq(Node.role_type, ["Guest", "Member", "Owner", "Retired", "Pending"])) {
                    Node.source { Blob.node { frag as NodeFragment.first_link }}
                }
                reactions as User.reactions {
                    comments as Reaction.comment
                }
                watched as User.watching
                contracts as User.contracts
                ue as User.events {
                    notifs as UserEvent.event @filter(type(Notif))
                }
            }
            var(func: eq(User.username, "{{.username}}")) {
                membership as User.roles @filter(eq(Node.role_type, ["Guest", "Member", "Owner", "Retired", "Pending"]))
            }

            posts as var(func: has(Post.createdBy)) @cascade {
                Post.createdBy @filter(eq(User.username, "{{.username}}"))
            }
            node_created as var(func: has(Node.createdBy)) @cascade {
                Node.createdBy @filter(eq(User.username, "{{.username}}"))
            }
            project_created as var(func: has(Project.createdBy)) @cascade {
                Project.createdBy @filter(eq(User.username, "{{.username}}"))
            }
        }`,
		M: []X{{
			S: `
        uid(posts) <Post.createdBy> <{{.ghostid}}> .
        uid(node_created) <Node.createdBy> <{{.ghostid}}> .
        uid(project_created) <Project.createdBy> <{{.ghostid}}> .
        `,
			D: `
        uid(ur) * *  .
        uid(assigned) <Tension.assignees> uid(u) .
        uid(roles) <Node.first_link> * .
        uid(membership) * * .
        uid(frag) <NodeFragment.first_link> * .
        uid(comments) <Comment.reactions> uid(reactions) .
        uid(reactions) * * .
        uid(watched) <Node.watchers> uid(u) .
        uid(contracts) <Contract.candidates> uid(u) .
        uid(notifs) * * .
        uid(ue) * * .
        uid(u) * * .
        `,
		}},
	},

	//
	// --- Converted ad-hoc mutations ---
	//

	"setFieldById": {
		Q: `query {
            node as var(func: uid({{.id}}))
        }`,
		M: []X{{
			S: `uid(node) <{{.predicate}}> "{{.value}}" .`,
		}},
	},
	"setFieldByEq": {
		Q: `query {
            node as var(func: eq({{.fieldid}}, "{{.objid}}"))
        }`,
		M: []X{{
			S: `uid(node) <{{.predicate}}> "{{.value}}" .`,
		}},
	},
	"setSubFieldByEq": {
		Q: `query {
            var(func: eq({{.fieldid}}, "{{.objid}}")) {
                x as {{.predicate1}}
            }
        }`,
		M: []X{{
			S: `uid(x) <{{.predicate2}}> "{{.value}}" .`,
		}},
	},
	"setPushedFlagBlob": {
		Q: `query {
            obj as var(func: uid({{.bid}}))
        }`,
		M: []X{{
			S: `uid(obj) <Blob.pushedFlag> "{{.flag}}" .
                <{{.tid}}> <Tension.action> "{{.action}}" .`,
			D: `uid(obj) <Blob.archivedFlag> * .`,
		}},
	},
	"setArchivedFlagBlob": {
		Q: `query {
            obj as var(func: uid({{.bid}}))
        }`,
		M: []X{{
			S: `uid(obj) <Blob.archivedFlag> "{{.flag}}" .
                <{{.tid}}> <Tension.action> "{{.action}}" .`,
		}},
	},
	"setNodeSource": {
		Q: `query {
            node as var(func: eq(Node.nameid, "{{.nameid}}"))
        }`,
		M: []X{{
			S: `uid(node) <Node.source> <{{.bid}}> .`,
		}},
	},
	"setChildrenRoleVisibility": {
		Q: `query {
            var(func: eq(Node.nameid, "{{.nameid}}")) {
                c as Node.children @filter(eq(Node.type_, "Role"))
            }
        }`,
		M: []X{{
			S: `uid(c) <Node.visibility> "{{.value}}" .`,
		}},
	},
	"patchNameid": {
		Q: `query {
            node as var(func: eq(Node.nameid, "{{.nameid_old}}")) {
                tin as Node.tensions_in
                tout as Node.tensions_out
            }
        }`,
		M: []X{{
			S: `uid(node) <Node.nameid> "{{.nameid_new}}" .
                uid(tin) <Tension.receiverid> "{{.nameid_new}}" .
                uid(tout) <Tension.emitterid> "{{.nameid_new}}" .`,
		}},
	},
	"upgradeMember": {
		Q: `query {
            node as var(func: eq(Node.nameid, "{{.nameid}}"))
        }`,
		M: []X{{
			S: `uid(node) <Node.role_type> "{{.roleType}}" .
                uid(node) <Node.name> "{{.roleType}}" .`,
		}},
	},
	"rewriteContractId": {
		Q: `query {
            var(func: uid({{.cid}})) {
                cuid as uid
                Contract.participants {
                    vuid as uid
                }
            }
        }`,
		M: []X{{
			S: `uid(cuid) <Contract.contractid> "" .
                uid(vuid) <Vote.voteid> "" .`,
		}},
	},
}
