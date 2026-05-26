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
	// deleteTension cascades a tension and everything it owns: comments
	// (with reactions + files), blobs (with NodeFragment + Mandate),
	// contracts (with events, votes, comments, reactions, files), history
	// events, and mention edges. Reverse edges Node.tensions_out and
	// Node.tensions_in are dropped manually; @hasInverse is a GraphQL-side
	// concept Dgraph won't auto-clean from a DQL delete.
	//
	// The `all` block surfaces File.storageKey for every comment attachment
	// (tension- and contract-level) so callers can fire-and-forget S3 GC
	// without a second round-trip.
	//
	// Known limitation: cascaded contract uids dangle in their reverse edges
	// (User.contracts, PendingUser.contracts, Node.contracts) — fixing this
	// requires walking the contracts before delete.
	"deleteTension": {
		Q: `query {
            id as var(func: uid({{.id}})) {
              rid_emitter as Tension.emitter
              rid_receiver as Tension.receiver
              comments as Tension.comments {
                reactions as Comment.reactions
                files as Comment.files
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
                    files2 as Comment.files
                  }
              }
              events as Tension.history
              mentions as Tension.mentions
            }
            var(func: uid(id,comments,reactions,events,mentions,b,bn,m,c,e,votes,comments2,reactions2,files,files2)) {
                all_ids as uid
            }
            all(func: uid(files,files2)) {
                File.storageKey
            }
        }`,
		M: []X{{
			D: `uid(rid_emitter) <Node.tensions_out> uid(id) .
                uid(rid_receiver) <Node.tensions_in> uid(id) .
                uid(all_ids) * * .
                `,
		}},
	},
	// deleteContract cascades a contract and its participants/comments/
	// reactions, plus User.events rows whose `event` edge points back at
	// the contract (the desync_events case). Reverse edges from Tension,
	// User, PendingUser, Node are dropped manually — same @hasInverse
	// caveat as deleteTension. No storage keys today.
	"deleteContract": {
		Q: `query {
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

            var(func: uid(id,a,votes,c,r)) {
                all_ids as uid
            }
        }`,
		M: []X{{
			D: `uid(rid) <Tension.contracts> uid(id) .
                uid(candidates) <User.contracts> uid(id) .
                uid(user_pending) <PendingUser.contracts> uid(id) .
                uid(nodes) <Node.contracts> uid(votes) .
                uid(members) <User.events> uid(desync_events) .
                uid(desync_events) * * .
                uid(all_ids) * * .
                `,
		}},
	},
	"deleteComment": {
		// The `all` block returns the storage keys of the files attached to
		// the comment so the caller can GC the S3 objects without a second
		// round-trip. Upsert evaluates the query before applying mutations,
		// so the keys reflect the pre-delete state.
		Q: `query {
			t as var(func: uid({{.tid}}))
            var(func: uid({{.cid}})) {
                c as uid
                reactions as Comment.reactions
                files as Comment.files
            }
            all(func: uid(files)) {
                File.storageKey
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
	// addCommentFile inserts a File anchored to a comment, with `tension`
	// denormalised. The caller (web/handlers/files.go) has already verified that
	// uctx.Username may post in the tension and that cid belongs to tid.
	// File.embedded is set to false here; a follow-up call to
	// embedCommentMessage flips it to true and rewrites the message when the
	// upload was an inline screenshot paste.
	//
	// Inputs (all required): cid, tid, username, filename, contentType,
	//                        sizeStr (decimal), storageKey, now (RFC3339).
	"addCommentFile": {
		Q: `query {
            var(func: uid({{.cid}})) { c as uid }
            var(func: uid({{.tid}})) { t as uid }
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
				_:f <File.tension> uid(t) .
				_:f <File.embedded> "false" .
				uid(c) <Comment.files> _:f .
				`,
		}},
	},
	// embedCommentMessage rewrites Comment.message and flips File.embedded=true
	// on zero-or-more fids in a single upsert. Last-writer-wins on
	// Comment.message: if two uploads race against the same comment, the second
	// overwrite can drop the first's rewrite. Acceptable trade-off — the File
	// rows themselves are independent and both stay queryable; the UI is
	// already lenient about embedded=true files whose URL no longer appears in
	// the message (rendered as a regular attachment).
	//
	// Inputs (all required): cid, newMessage, embeddedTriples. The caller
	// (db.EmbedCommentMessage) builds embeddedTriples from the fid list — empty
	// string when no fids need flipping, just the message swap fires then.
	"embedCommentMessage": {
		Q: `query {
            var(func: uid({{.cid}})) { c as uid }
        }`,
		M: []X{{
			S: `uid(c) <Post.message> "{{.newMessage}}" .
                {{.embeddedTriples}}
                `,
		}},
	},
	// replaceUserAvatar inserts a new avatar File for user `username` and drops
	// the previous one (if any). The mutation returns the previous storageKey
	// so the caller can fire-and-forget the S3 GC.
	//
	// Inputs (all required): username, filename, contentType, sizeStr,
	//                        storageKey, now.
	"replaceUserAvatar": {
		Q: `query {
            var(func: eq(User.username, "{{.username}}")) {
                u as uid
                User.avatar {
                    old as uid
                }
            }
            all(func: uid(old)) {
                File.storageKey
            }
        }`,
		M: []X{
			{
				C: `@if(eq(len(old), 1))`,
				D: `uid(old) * * .
                    uid(u) <User.avatar> uid(old) .
                    `,
			},
			{
				S: `_:f <dgraph.type> "File" .
                    _:f <File.storageKey> "{{.storageKey}}" .
                    _:f <File.filename> "{{.filename}}" .
                    _:f <File.contentType> "{{.contentType}}" .
                    _:f <File.size> "{{.sizeStr}}" .
                    _:f <File.createdAt> "{{.now}}" .
                    _:f <File.createdBy> uid(u) .
                    _:f <File.user> uid(u) .
                    uid(u) <User.avatar> _:f .
                    `,
			},
		},
	},
	// replaceNodeAvatar mirrors replaceUserAvatar for org (root Node) avatars.
	//
	// Inputs (all required): nameid (Node.nameid), username (uploader),
	//                        filename, contentType, sizeStr, storageKey, now.
	"replaceNodeAvatar": {
		Q: `query {
            var(func: eq(Node.nameid, "{{.nameid}}")) {
                n as uid
                Node.avatar {
                    old as uid
                }
            }
            var(func: eq(User.username, "{{.username}}")) {
                u as uid
            }
            all(func: uid(old)) {
                File.storageKey
            }
        }`,
		M: []X{
			{
				C: `@if(eq(len(old), 1))`,
				D: `uid(old) * * .
                    uid(n) <Node.avatar> uid(old) .
                    `,
			},
			{
				S: `_:f <dgraph.type> "File" .
                    _:f <File.storageKey> "{{.storageKey}}" .
                    _:f <File.filename> "{{.filename}}" .
                    _:f <File.contentType> "{{.contentType}}" .
                    _:f <File.size> "{{.sizeStr}}" .
                    _:f <File.createdAt> "{{.now}}" .
                    _:f <File.createdBy> uid(u) .
                    _:f <File.node> uid(n) .
                    uid(n) <Node.avatar> _:f .
                    `,
			},
		},
	},
	// deleteFile removes a File node by uid and drops the reverse edges from
	// any anchor (Comment.files, User.avatar, Node.avatar). DQL doesn't auto-
	// maintain @hasInverse, so each candidate parent is named explicitly — the
	// `uid(parent)` references are no-ops when the binding is empty.
	"deleteFile": {
		Q: `query {
            var(func: uid({{.id}})) {
                f as uid
                File.comment { commentParent as uid }
                File.user    { userParent as uid }
                File.node    { nodeParent as uid }
            }
        }`,
		M: []X{{
			D: `uid(commentParent) <Comment.files> uid(f) .
                uid(userParent)    <User.avatar>   uid(f) .
                uid(nodeParent)    <Node.avatar>   uid(f) .
                uid(f) * * .
                `,
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
                avatar as User.avatar
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
            all(func: uid(avatar)) {
                File.storageKey
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
        uid(avatar) * * .
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

// collectStorageKeys pulls "storageKey" strings out of a Meta() response.
// Templates project them under an `all` block as `File.storageKey`;
// CleanDqlMap strips the type prefix so the cleaned key is just "storageKey".
func collectStorageKeys(resp []map[string]any) []string {
	if len(resp) == 0 {
		return nil
	}
	out := make([]string, 0, len(resp))
	for _, m := range resp {
		if k, ok := m["storageKey"].(string); ok && k != "" {
			out = append(out, k)
		}
	}
	return out
}

// DeleteCommentDeep removes a comment (see "deleteComment" template) and
// fires async S3 cleanup for every attached file. Per-key S3 failures are
// logged in the goroutine. Auth is the caller's responsibility.
func (dg Dgraph) DeleteCommentDeep(tid, cid string) error {
	resp, err := dg.Meta("deleteComment", map[string]string{"tid": tid, "cid": cid})
	if err != nil {
		return err
	}
	if keys := collectStorageKeys(resp); len(keys) > 0 {
		deleteStorageKeysAsync(keys)
	}
	return nil
}

// DeleteTensionDeep cascades a tension delete (see "deleteTension" template)
// and fires async S3 cleanup for every comment attachment carried by the
// tension or its contracts. Per-key S3 failures are logged in the goroutine.
func (dg Dgraph) DeleteTensionDeep(id string) error {
	resp, err := dg.Meta("deleteTension", map[string]string{"id": id})
	if err != nil {
		return err
	}
	if keys := collectStorageKeys(resp); len(keys) > 0 {
		deleteStorageKeysAsync(keys)
	}
	return nil
}

// DeleteContractDeep cascades a contract delete (see "deleteContract").
// No S3 GC: contract comments don't carry files in the current schema.
func (dg Dgraph) DeleteContractDeep(id string) error {
	_, err := dg.Meta("deleteContract", map[string]string{"id": id})
	return err
}

// DeleteUser runs the deleteUser cascade and fires async S3 cleanup of the
// removed user's avatar (if any). The Dgraph mutation always runs; per-key
// S3 failures are logged in the goroutine and do not surface as errors.
func (dg Dgraph) DeleteUser(username, ghostid string) error {
	resp, err := dg.Meta("deleteUser", map[string]string{
		"username": username,
		"ghostid":  ghostid,
	})
	if err != nil {
		return err
	}
	if keys := collectStorageKeys(resp); len(keys) > 0 {
		deleteStorageKeysAsync(keys)
	}
	return nil
}
