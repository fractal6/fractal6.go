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

package graph

import (
	"fmt"
	"strings"
	"time"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/internal/tools"
	"fractale/fractal6.go/web/auth"
)

var (
	EMAP              EventsMap
	SubscribingEvents map[model.TensionEvent]bool
)

func init() {
	EMAP = EventsMap{
		model.TensionEventCreated: EventMap{
			Auth: MemberStrictHook,
		},
		model.TensionEventCommentPushed: EventMap{
			Auth: MemberHook | AuthorHook,
		},
		model.TensionEventCommentDeleted: EventMap{
			Auth:   MemberHook | AuthorHook,
			Action: RemoveComment,
		},
		model.TensionEventBlobCreated: EventMap{
			Auth: MemberStrictHook,
		},
		model.TensionEventBlobCommitted: EventMap{
			Auth: MemberStrictHook,
		},
		model.TensionEventTitleUpdated: EventMap{
			Auth:      SourceCoordoHook | TargetCoordoHook | AuthorHook | AssigneeHook,
			Propagate: "title",
		},
		model.TensionEventTypeUpdated: EventMap{
			Auth:      SourceCoordoHook | TargetCoordoHook | AuthorHook | AssigneeHook,
			Propagate: "type_",
		},
		model.TensionEventReopened: EventMap{
			Auth:      SourceCoordoHook | TargetCoordoHook | AuthorHook | AssigneeHook,
			Propagate: "status",
		},
		model.TensionEventClosed: EventMap{
			Auth:      SourceCoordoHook | TargetCoordoHook | AuthorHook | AssigneeHook,
			Propagate: "status",
		},
		model.TensionEventLabelAdded: EventMap{
			Auth: TargetCoordoHook | AuthorHook | AssigneeHook,
		},
		model.TensionEventLabelRemoved: EventMap{
			Auth: TargetCoordoHook | AuthorHook | AssigneeHook,
		},
		model.TensionEventAssigneeAdded: EventMap{
			Auth: TargetCoordoHook,
			Restrict: []RestrictValue{
				UserNewIsMemberRestrict,
			},
		},
		model.TensionEventAssigneeRemoved: EventMap{
			Auth: TargetCoordoHook,
		},
		model.TensionEventPinned: EventMap{
			Auth:   TargetCoordoHook,
			Action: PinTension,
		},
		model.TensionEventUnpinned: EventMap{
			Auth:   TargetCoordoHook,
			Action: UnpinTension,
		},
		// --- Trigger Action ---
		model.TensionEventBlobPushed: EventMap{
			Auth:   TargetCoordoHook | AssigneeHook,
			Action: PushBlob,
		},
		model.TensionEventBlobArchived: EventMap{
			Auth:   TargetCoordoHook | AssigneeHook,
			Action: ChangeArchiveBlob,
		},
		model.TensionEventBlobUnarchived: EventMap{
			Auth:   TargetCoordoHook | AssigneeHook,
			Action: ChangeArchiveBlob,
		},
		model.TensionEventAuthority: EventMap{
			Auth:   TargetCoordoHook,
			Action: ChangeAuhtority,
		},
		model.TensionEventVisibility: EventMap{
			Auth:   TargetCoordoHook,
			Action: ChangeVisibility,
		},
		model.TensionEventMoved: EventMap{
			Validation: model.ContractTypeAnyCoordoDual,
			Auth:       AuthorHook | SourceCoordoHook | TargetCoordoHook | AssigneeHook,
			Action:     MoveTension,
			Restrict: []RestrictValue{
				UserIsMemberRestrict,
			},
		},
		model.TensionEventMemberLinked: EventMap{
			Validation: model.ContractTypeAnyCandidates,
			// @DEBUG: auth, can a user open a contract in a private Circle ???
			// if yes, constraint the candidateHook to Public circle only.
			Auth:   TargetCoordoHook | AssigneeHook | CandidateHook,
			Action: ChangeFirstLink,
		},
		model.TensionEventMemberUnlinked: EventMap{
			Auth:   TargetCoordoHook | AssigneeHook,
			Action: ChangeFirstLink,
		},
		model.TensionEventUserJoined: EventMap{
			// @FIXFEAT: Either Check Receiver NodeCharac or contract value to check that user has been invited !
			Validation: model.ContractTypeAnyCandidates,
			Auth:       TargetCoordoHook | AssigneeHook | CandidateHook,
			Action:     UserJoin,
		},
		model.TensionEventUserLeft: EventMap{
			// Authorisation is done in the method for now ("FirstLinkHook").
			Auth:   PassingHook,
			Action: UserLeave,
		},
		// Project-related events are emitted internally by the ProjectCard hooks
		// (graph/card_resolver.go), never via updateTension(history:...).
		// PassingHook lets Check() succeed so the rejecting Action runs and
		// returns a clear error.
		model.TensionEventProjectAdded: EventMap{
			Auth:   PassingHook,
			Action: RejectInternalEvent,
		},
		model.TensionEventProjectRemoved: EventMap{
			Auth:   PassingHook,
			Action: RejectInternalEvent,
		},
		model.TensionEventProjectColumnMoved: EventMap{
			Auth:   PassingHook,
			Action: RejectInternalEvent,
		},
	}

	SubscribingEvents = map[model.TensionEvent]bool{
		model.TensionEventCreated:       true,
		model.TensionEventCommentPushed: true,
		model.TensionEventReopened:      true,
		model.TensionEventClosed:        true,
	}
}

// tensionEventHook is applied for addTension and updateTension query directives.
// Take action based on the given Event. The targeted tension is fetch (see TensionHookPayload) with
// All events in History must pass.
func TensionEventHook(uctx *model.UserCtx, tid string, events []*model.EventRef, blob *model.BlobRef) (bool, *model.Contract, error) {
	var ok bool = true
	var addSubscriber bool
	var err error
	var tension *model.Tension
	var contract *model.Contract
	if events == nil {
		return false, nil, LogErr("Access denied", fmt.Errorf("No event given."))
	}
	if tid == "" {
		return false, nil, LogErr("Value error", fmt.Errorf("Tension ID is mandatory."))
	}

	for _, event := range events {
		if tension == nil { // don't fetch if there is no events (Comment updated...)
			// Fetch Tension, target Node and blob charac (last if blob if nil)
			var bid *string
			if blob != nil {
				bid = blob.ID
			}
			tension, err = db.GetDB().GetTensionHook(tid, true, bid)
			if err != nil {
				return false, nil, LogErr("Access denied", err)
			}
		}

		// Process event
		ok, contract, err = ProcessEvent(uctx, tension, event, blob, nil, true, true)
		if !ok || err != nil {
			break
		}

		// Check if event make a new subscriber
		addSubscriber = addSubscriber || SubscribingEvents[*event.EventType]
	}

	// Add subscriber
	// @performance: @defer this with Redis
	if addSubscriber && ok && err == nil {
		err = db.GetDB().Update(*uctx, "tension", &model.UpdateTensionInput{
			Filter: &model.TensionFilter{ID: []string{tension.ID}},
			Set:    &model.TensionPatch{Subscribers: []*model.UserRef{{Username: &uctx.Username}}},
		})
	}

	return ok, contract, err
}

func ProcessEvent(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, blob *model.BlobRef, contract *model.Contract,
	doCheck, doProcess bool,
) (bool, *model.Contract, error) {
	var ok bool
	var err error

	if tension == nil {
		return ok, contract, LogErr("Access denied", fmt.Errorf("tension not found."))
	}
	if event == nil || event.EventType == nil {
		return false, contract, fmt.Errorf("event type is required")
	}

	em, hasEvent := EMAP[*event.EventType]
	if !hasEvent { // Minimum level of authorization
		return false, nil, LogErr("Access denied", fmt.Errorf("Event not implemented."))
	}

	// Check Authorization (optionally generate a contract)
	if doCheck {
		ok, contract, err = em.Check(uctx, tension, event, contract)
		if !ok || err != nil {
			return ok, contract, err
		}
	}

	// act is false if contract is cancelled for example !
	act := contract == nil || contract.Status == model.ContractStatusClosed

	// Trigger Action
	if act && doProcess {
		if em.Propagate != "" {
			v, err := CheckEvent(tension, event)
			if err != nil {
				return ok, contract, err
			}
			err = db.GetDB().UpdateValue(*uctx, "tension", tension.ID, em.Propagate, v)
			if err != nil {
				return ok, contract, err
			}
		}
		if em.Action != nil {
			ok, err = em.Action(uctx, tension, event, blob)
			if !ok || err != nil {
				return ok, contract, err
			}
		}

		// leave trace
		go leaveTrace(uctx, tension, *event.EventType)
	}

	// Set contract status if any
	if contract != nil && doProcess {
		err = db.GetDB().SetFieldById(contract.ID, "Contract.status", string(contract.Status))
		if err != nil {
			return false, contract, err
		}

		// Assumes contract is either closed or cancelled.
		_, err = db.GetDB().Meta("rewriteContractId", map[string]string{"cid": contract.ID})
		if err != nil {
			return false, contract, err
		}
	}

	return ok, contract, err
}

// GetBlob returns the first blob found in the given tension.
func GetBlob(tension *model.Tension) *model.Blob {
	if tension.Blobs != nil {
		return tension.Blobs[0]
	}
	return nil
}

// nodeUpdatingEvents are the events whose action writes the governed Node itself.
// Other events (comment, label, assignee...) only touch the tension.
var nodeUpdatingEvents = map[model.TensionEvent]bool{
	model.TensionEventBlobPushed:     true,
	model.TensionEventBlobArchived:   true,
	model.TensionEventBlobUnarchived: true,
	model.TensionEventAuthority:      true,
	model.TensionEventVisibility:     true,
	model.TensionEventMemberLinked:   true,
	model.TensionEventMemberUnlinked: true,
	model.TensionEventUserLeft:       true,
	model.TensionEventMoved:          true,
}

func leaveTrace(uctx *model.UserCtx, tension *model.Tension, et model.TensionEvent) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("error: leaveTrace panic: %v\n", r)
		}
	}()

	if tension.GovernedNode != nil && nodeUpdatingEvents[et] {
		// Set the Update time into the affected node.
		if err := db.GetDB().SetFieldByEq("Node.nameid", tension.GovernedNode.Nameid, "Node.updatedAt", Now()); err != nil {
			fmt.Printf("error: leaveTrace node update: %v\n", err)
		}
	}
	// Set the Update time of its parent node (tension.receiver)
	if err := db.GetDB().SetFieldByEq("Node.nameid", tension.Receiver.Nameid, "Node.updatedAt", Now()); err != nil {
		fmt.Printf("error: leaveTrace receiver update: %v\n", err)
	}

	// Track activity
	trackActivity(uctx.Username, tension.Receiver.Nameid, et)
}

// trackActivity increments the daily activity counter for both the user
// and the root organisation. Noise events (not in trackedEvents) are skipped.
func trackActivity(username, receiverNameid string, et model.TensionEvent) {
	if !isTrackedEvent(et) {
		return
	}
	today := time.Now().UTC().Format("2006-01-02")
	todayISO := today + "T00:00:00Z"

	// User activity
	_, err := db.GetDB().Meta("upsertActivity", map[string]string{
		"activityid": "u#" + username + "#" + today,
		"ownerid":    "u#" + username,
		"date":       todayISO,
	})
	if err != nil {
		fmt.Printf("error: trackActivity user: %v\n", err)
	}

	// Org activity
	rootid, err := codec.Nid2rootid(receiverNameid)
	if err != nil {
		fmt.Printf("error: trackActivity Nid2rootid: %v\n", err)
		return
	}
	_, err = db.GetDB().Meta("upsertActivity", map[string]string{
		"activityid": "o#" + rootid + "#" + today,
		"ownerid":    "o#" + rootid,
		"date":       todayISO,
	})
	if err != nil {
		fmt.Printf("error: trackActivity org: %v\n", err)
	}
}

//
// Event Actions
//

// Add or Update the governed Node
// --
// 1. no governed Node yet -> create it from the blob fragment and link it to the tension
// 2. governed Node linked  -> update that Node only (never create another one)
// - copy the Blob data in the target Node.source (Uses GQL requests)
// - update the blob pushedFlag
func PushBlob(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
	subject, err := resolveGovernanceSubject(tension, governancePublish)
	if err != nil {
		return false, err
	}

	var ok bool
	if subject.create {
		var nid string
		ok, nid, err = TryAddNode(uctx, tension, subject.fragment, &subject.blob.ID)
		if err == nil && ok {
			err = db.GetDB().LinkGovernedNode(tension.ID, nid, subject.blob.ID)
		}
	} else {
		ok, err = TryUpdateNode(tension, subject.fragment, subject.node, &subject.blob.ID)
	}
	if err != nil || !ok {
		return ok, err
	}
	return ok, db.GetDB().SetPushedFlagBlob(subject.blob.ID, Now())
}

// Archived/Unarchive the governed Node
// - set Node.isArchived (lifecycle source of truth) and the blob archive/push flag
// - unlink first-link on archive, once the archive is persisted
func ChangeArchiveBlob(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
	if event == nil || event.EventType == nil {
		return false, fmt.Errorf("archive event type is required")
	}
	operation := governanceArchive
	archived := true
	switch *event.EventType {
	case model.TensionEventBlobArchived:
	case model.TensionEventBlobUnarchived:
		operation = governanceUnarchive
		archived = false
	default:
		return false, fmt.Errorf("bad tension event %q", *event.EventType)
	}

	subject, err := resolveGovernanceSubject(tension, operation)
	if err != nil {
		return false, err
	}
	return TryChangeArchiveNode(subject.fragment, subject.node, subject.blob.ID, archived)
}

// ChangeAuthory
// - If Circle : change mode on pointed node
// - If Role : change role_type on the pointed node (on Node + Node.RoleExt)
// - Don't touch the current blob as we do not use "authority" properties at the moment (just when adding node)
func ChangeAuhtority(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
	if event == nil || event.New == nil {
		return false, fmt.Errorf("authority event value is required")
	}
	subject, err := resolveGovernanceSubject(tension, governanceUpdate)
	if err != nil {
		return false, err
	}
	return TryChangeAuthority(subject.fragment, subject.node, *event.New)
}

// ChangeVisibility
// - Change the visiblity of the node
// - Don't touch the current blob as we do not use "authority" properties at the moment (just when adding node)
func ChangeVisibility(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
	if event == nil || event.New == nil {
		return false, fmt.Errorf("visibility event value is required")
	}
	subject, err := resolveGovernanceSubject(tension, governanceUpdate)
	if err != nil {
		return false, err
	}
	return TryChangeVisibility(subject.fragment, subject.node, *event.New)
}

// ChangeFirstLink
// - ensure first_link is free on link
// - Link/unlink user
// - Only Guest can be link/unlink in unsafe mode
func ChangeFirstLink(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
	if event == nil || event.EventType == nil {
		return false, fmt.Errorf("membership event type is required")
	}
	switch *event.EventType {
	case model.TensionEventMemberLinked:
		if event.New == nil {
			return false, fmt.Errorf("linked member is required")
		}
	case model.TensionEventMemberUnlinked:
		if event.Old == nil || event.New == nil {
			return false, fmt.Errorf("unlinked member and role type are required")
		}
	default:
		return false, fmt.Errorf("bad membership event %q", *event.EventType)
	}
	subject, err := resolveGovernanceSubject(tension, governanceUpdate)
	if err != nil {
		return false, err
	}
	var ok bool
	var unsafe bool
	node := subject.fragment

	if *node.Type == model.NodeTypeCircle {
		if event.Old == nil {
			return false, fmt.Errorf("previous circle member is required")
		}
		// A membership node wants to leave.
		// Auth: only Guest user can be detached (Retired)
		// --
		rootid, err := codec.Nid2rootid(tension.Receiver.Nameid)
		if err != nil {
			return ok, err
		}
		nid := codec.MemberIdCodec(rootid, *event.Old)
		n, err := db.GetDB().GetByEq("Node.nameid", nid, "Node.name Node.nameid Node.type_ Node.role_type")
		if err != nil {
			return ok, err
		}
		nf := StructMap[model.NodeFragment](n)
		if nf.RoleType == nil {
			return false, fmt.Errorf("guest membership role not found")
		}
		if *nf.RoleType != model.RoleTypeGuest {
			return false, LogErr("access denied", fmt.Errorf("You cannot detach this role (%s) like this.", string(*nf.RoleType)))
		}
		nf.FirstLink = event.Old
		node = &nf
		unsafe = true
	}

	ok, err = TryUpdateLink(node, subject.node, event, unsafe)

	return ok, err
}

func MoveTension(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
	if event == nil || event.Old == nil || event.New == nil {
		return false, fmt.Errorf("old and new event data must be defined.")
	}
	subject, err := resolveGovernanceSubject(tension, governanceUpdate)
	if err != nil {
		return false, err
	}
	if *event.Old != tension.Receiver.Nameid {
		return false, LogErr("access denied", fmt.Errorf("Contract outdated: event source (%s) and actual source (%s) differ. Please, refresh or remove this contract.", *event.Old, tension.Receiver.Nameid))
	}

	receiverid_old := *event.Old // == tension.Receiverid
	receiverid_new := *event.New
	nameid_old := subject.node.Nameid
	parts := strings.Split(nameid_old, "#")
	localNameid := parts[len(parts)-1]
	_, nameid_new, err := codec.NodeIdCodec(receiverid_new, localNameid, subject.node.Type)
	if err != nil {
		return false, err
	}

	if codec.IsRoot(nameid_old) {
		return false, fmt.Errorf("You can't move the root node.")
	}
	if receiverid_new == nameid_new {
		return false, fmt.Errorf("A node cannot be its own parent.")
	}
	isChild, err := db.GetDB().IsChild(nameid_old, receiverid_new)
	if err != nil {
		return false, err
	}
	if isChild {
		return false, fmt.Errorf("You can't move a node in their children.")
	}
	if codec.IsRole(receiverid_new) {
		return false, fmt.Errorf("You can't move a node in a Role.")
	}

	nodeInput := model.UpdateNodeInput{
		Filter: &model.NodeFilter{Nameid: &model.StringHashFilterStringRegExpFilter{Eq: &nameid_old}},
		Set:    &model.NodePatch{Parent: &model.NodeRef{Nameid: &receiverid_new}},
	}
	if err = db.GetDB().Update(db.GetDB().GetRootUctx(), "node", nodeInput); err != nil {
		return false, err
	}
	if nameid_old != nameid_new {
		if _, err = db.GetDB().Meta("patchNameid", map[string]string{"nameid_old": nameid_old, "nameid_new": nameid_new}); err != nil {
			return false, err
		}
		subject.node.Nameid = nameid_new
	}

	// tension input
	tensionInput := model.UpdateTensionInput{
		Filter: &model.TensionFilter{ID: []string{tension.ID}},
		Set: &model.TensionPatch{
			Receiver:   &model.NodeRef{Nameid: &receiverid_new},
			Receiverid: &receiverid_new,
		},
	}

	// update tension
	err = db.GetDB().Update(db.GetDB().GetRootUctx(), "tension", tensionInput)
	if err != nil {
		return false, err
	}

	// Update tension pin
	_, err = db.GetDB().Meta("movePinnedTension", map[string]string{"nameid_old": receiverid_old, "nameid_new": receiverid_new, "tid": tension.ID})

	return true, err
}

func UserJoin(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
	var ok bool

	// Only root node can be joined
	// --
	username := *event.New
	rootid, err := codec.Nid2rootid(tension.Receiver.Nameid)
	if err != nil {
		return ok, err
	}

	// Validate
	// --
	// check the invitation if a hash is given
	// * orga invitation ? <> user invitation hash ?
	// * else check if User Can Join Organisation
	// @debug: this should be done before the contract creation.
	guestid := codec.MemberIdCodec(rootid, username)
	// Pending node as been created at invitation
	// ex, err :=  db.GetDB().Exists("Node.nameid", guestid, nil, nil)
	// if err != nil { return ok, err }
	err = LinkUser(rootid, guestid, username)
	if err != nil {
		return ok, err
	}

	// Make user watch that organisation.
	err = db.GetDB().Update(*uctx, "user", &model.UpdateUserInput{
		Filter: &model.UserFilter{Username: &model.StringHashFilterStringRegExpFilter{Eq: &username}},
		Set:    &model.UserPatch{Watching: []*model.NodeRef{{Nameid: &rootid}}},
	})

	return true, err
}

// UserLeave  remove user reference
// - remove User role
// - update user membership
func UserLeave(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
	if event == nil || event.New == nil {
		return false, fmt.Errorf("membership role type is required")
	}
	roleType := model.RoleType(*event.New)
	if codec.IsMembershipRoleType(roleType) {
		uctx.NoCache = true
		membershipNode := auth.GetMembershipRole(uctx, tension.Emitter.Nameid)
		if membershipNode == nil || membershipNode.RoleType == nil {
			return false, fmt.Errorf("membership role not found")
		}
		if roleType != *membershipNode.RoleType {
			return false, LogErr("access denied", fmt.Errorf("You must have the same membership as the one given in the event."))
		}
		nf := StructMap[model.NodeFragment](membershipNode)
		nf.FirstLink = &uctx.Username
		nodeType := model.NodeTypeRole
		nf.Type = &nodeType
		return LeaveRole(uctx, &nf, nil)
	}

	subject, err := resolveGovernanceSubject(tension, governanceUpdate)
	if err != nil {
		return false, err
	}
	return LeaveRole(uctx, subject.fragment, subject.node)
}

func PinTension(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
	tid := tension.ID
	nameid := tension.Receiver.Nameid
	// node input
	nodeInput := model.UpdateNodeInput{
		Filter: &model.NodeFilter{Nameid: &model.StringHashFilterStringRegExpFilter{Eq: &nameid}},
		Set: &model.NodePatch{
			Pinned: []*model.TensionRef{{ID: &tid}},
		},
	}
	// update node
	err := db.GetDB().Update(db.GetDB().GetRootUctx(), "node", nodeInput)
	return true, err
}

func UnpinTension(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
	tid := tension.ID
	nameid := tension.Receiver.Nameid
	// node input
	nodeInput := model.UpdateNodeInput{
		Filter: &model.NodeFilter{Nameid: &model.StringHashFilterStringRegExpFilter{Eq: &nameid}},
		Remove: &model.NodePatch{
			Pinned: []*model.TensionRef{{ID: &tid}},
		},
	}
	// update node
	err := db.GetDB().Update(db.GetDB().GetRootUctx(), "node", nodeInput)
	return true, err
}

// RejectInternalEvent is the Action used by EMAP entries for events that must
// only be emitted from internal hooks (e.g. ProjectAdded/Removed/ColumnMoved
// from the ProjectCard mutations in graph/card_resolver.go). Any caller that
// reaches this through TensionEventHook gets a clear error instead of the
// generic "Event not implemented" fallback.
func RejectInternalEvent(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
	return false, fmt.Errorf("%s is emitted internally and cannot be set via updateTension.", *event.EventType)
}

func RemoveComment(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
	tid := tension.ID
	cid := *event.Old

	// Check that the user is the author of the comment
	res, err := db.GetDB().GetByUid(cid, "Post.createdBy", "User.username")
	if err != nil {
		return false, err
	}
	if res == nil || res.(string) != uctx.Username {
		return false, LogErr("Access denied", fmt.Errorf("Only the author of the comment can delete it."))
	}

	if err := db.GetDB().DeleteCommentDeep(tid, cid); err != nil {
		return false, err
	}
	return true, nil
}

//
// Utilities
//

// Check event before propagation. Should be defined in directives,
// those are transactionned from event.
func CheckEvent(t *model.Tension, e *model.EventRef) (string, error) {
	if e.New == nil || *e.New == "" {
		return "", fmt.Errorf("Event new field must be given.")
	}

	b := GetBlob(t)
	v := *e.New
	var err error

	switch *e.EventType {
	case model.TensionEventTypeUpdated:
		if b != nil && b.Node != nil && *b.Node.Type == model.NodeTypeCircle {
			err = fmt.Errorf("The type of tensions with circle attached cannot be changed.")
		} else if b != nil && b.Node != nil && *b.Node.Type == model.NodeTypeRole {
			err = fmt.Errorf("The type of tensions with role attached cannot be changed.")
		}
	default:
		// pass
	}

	return v, err
}
