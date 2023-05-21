/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2023 Fractale Co
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
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/tools"
	"fractale/fractal6.go/web/auth"
	"fractale/fractal6.go/web/email"
)

/*
 *
 * This code manage sending notification
 *
 */

var ctx context.Context = context.Background()

//
// Publisher functions (Redis)
//

// Will trigger Event notifications in cmd/notifier.go
// PublishTensionEvent -> cmd.processTensionNotification -> PushEventNotifications
func PublishTensionEvent(notif model.EventNotif) error {
	payload, _ := json.Marshal(notif)
	if err := cache.Publish(ctx, "api-tension-notification", payload).Err(); err != nil {
		fmt.Printf("Redis publish error: %v", err)
		panic(err)
	}

	return nil
}

// Will trigger Contract notifications in cmd/notifier.go
// PublishContractEvent -> cmd.processContractNotification -> PushContractNotifications
func PublishContractEvent(notif model.ContractNotif) error {
	payload, _ := json.Marshal(notif)
	if err := cache.Publish(ctx, "api-contract-notification", payload).Err(); err != nil {
		fmt.Printf("Redis publish error: %v", err)
		panic(err)
	}

	return nil
}

// Will trigger Notif notifications in cmd/notifier.go
// PublishNotifEvent -> cmd.processNotifNotification -> PushNotifNotifications
func PublishNotifEvent(notif model.NotifNotif) error {
	payload, _ := json.Marshal(notif)
	if err := cache.Publish(ctx, "api-notif-notification", payload).Err(); err != nil {
		fmt.Printf("Redis publish error: %v", err)
		panic(err)
	}

	return nil
}

//
// Notifiers functions
//

/* INTERNAL (websocket, platform notification etc) */

// PushHistory publish event to a tension history.
func PushHistory(notif *model.EventNotif) error {
	var inputs []model.AddEventInput
	for _, e := range notif.History {
		// Build AddtensionInput
		var temp model.AddEventInput
		StructMap(e, &temp)
		temp.Tension = &model.TensionRef{ID: &notif.Tid}

		// Push AddtensionInput
		inputs = append(inputs, temp)
	}
	// Push events
	ids, err := db.GetDB().AddMany(*notif.Uctx, "event", inputs)
	if err != nil {
		return err
	}
	// Set event ids for further notifications
	for i, id := range ids {
		notif.History[i].ID = &id
	}
	return err
}

/* EXTERNAL (email, chat, etc) */

// Notify users for Event events, where events can be batch of event.
func PushEventNotifications(notif model.EventNotif) error {
	// Push event in tension event history
	err := PushHistory(&notif)
	if err != nil {
		return err
	}

	//  Alert and Announcement tensions notification only active
	//  for tensions creation.
	var receiverid string
	var type_ model.TensionType
	var isClosed bool
	if notif.HasEvent(model.TensionEventCreated) {
		if t, err := db.GetDB().GetFieldById(notif.Tid, "Tension.type_ Tension.receiverid Tension.status"); err != nil {
			return err
		} else if t != nil {
			tension := t.(model.JsonAtom)
			type_ = model.TensionType(tension["type_"].(string))
			receiverid = tension["receiverid"].(string)
			isClosed = tension["status"].(string) == string(model.TensionStatusClosed)
		}
	}

	// Get people to notify
	users := make(map[string]model.UserNotifInfo)
	if type_ == model.TensionTypeAlert {
		// Alert tension Notify every members (including Guest)
		if data, err := db.GetDB().GetSubMembers("nameid", receiverid, auth.UserSelection); err == nil {
			for _, n := range data {
				user := *n.FirstLink
				if _, ex := users[user.Username]; ex {
					continue
				}
				users[user.Username] = model.UserNotifInfo{User: user, Reason: model.ReasonIsAlert}
			}
		} else {
			return err
		}
	} else if type_ == model.TensionTypeAnnouncement {
		// Announcement tension Notify all watching users.
		data, err := db.GetDB().Meta("getWatchers", map[string]string{"nameid": receiverid, "user_payload": auth.UserSelection})
		if err != nil {
			return err
		}
		for _, u := range data {
			var user model.User
			if err := Map2Struct(u, &user); err != nil {
				return err
			}
			if _, ex := users[user.Username]; ex {
				continue
			}
			users[user.Username] = model.UserNotifInfo{User: user, Reason: model.ReasonIsAnnouncement}
		}
	} else {
		// Get relevant users to notify for that event.
		users, err = GetUsersToNotify(notif.Tid, true, true, true)
		if err != nil {
			return err
		}
	}
	// +
	// Add mentions and **set tension data**
	if m, err := db.GetDB().Meta("getLastComment", map[string]string{"tid": notif.Tid, "username": notif.Uctx.Username}); err != nil {
		return err
	} else if len(m) > 0 {
		notif.Rootnameid = m[0]["rootnameid"].(string)
		notif.Receiverid = m[0]["receiverid"].(string)
		notif.Title = m[0]["title"].(string)
		notif.Msg, _ = m[0]["message"].(string)

		if notif.Msg != "" && notif.HasEvent(model.TensionEventCommentPushed) {
			// Mentioned users
			err = UpdateWithMentionnedUser(notif.Msg, notif.Receiverid, users)
			if err != nil {
				return err
			}

			// Mentioned tensions
			err = PushMentionedTension(notif)
			if err != nil {
				return err
			}
		}

	} else {
		return fmt.Errorf("tension %s not found.", notif.Tid)
	}

	// Special notifications
	// --
	// User has been kick-out from an organisation
	if notif.HasEvent(model.TensionEventMemberUnlinked) && codec.IsCircle(notif.Receiverid) {
		u := notif.GetExUser()
		if _, ex := users[u]; !ex {
			PushNotifNotifications(model.NotifNotif{
				Uctx: notif.Uctx,
				Tid:  &notif.Tid,
				Cid:  nil,
				Msg:  "You have been removed from this organization",
				To:   []string{u},
			}, false)
		}
	}

	// Only the event with an ID will be notified.
	var eventBatch []*model.EventKindRef
	var createdAt string
	for i, e := range notif.History {
		if i == 0 {
			createdAt = *e.CreatedAt
		}
		if *e.ID != "" {
			eventBatch = append(eventBatch, &model.EventKindRef{EventRef: &model.EventRef{ID: e.ID}})
		}
	}
	if len(eventBatch) == 0 {
		return nil
	}

	// Push user event notification
	for u, ui := range users {
		// Don't self notify.
		if u == notif.Uctx.Username {
			continue
		}
		// Pending user has no history yet
		if ui.IsPending {
			continue
		}

		// Do not publish already closed tension (e.g. role and circle creation)
		if isClosed {
			continue
		}

		// User Event
		var eid string
		if notif.IsNotifiable(ui) {
			eid, err = db.GetDB().Add(db.GetDB().GetRootUctx(), "userEvent", &model.AddUserEventInput{
				User:      &model.UserRef{Username: &u},
				IsRead:    false,
				CreatedAt: createdAt,
				Event:     eventBatch,
			})
			if err != nil {
				return err
			}
		}

		// Email
		if notif.Uctx.Rights.HasEmailNotifications && ui.User.NotifyByEmail && notif.IsEmailable(ui) {
			if eid == "" {
				// @deprected warning: unnecessary/noisy
				//log.Printf("Notification Error: an event is emailable but not notifiable !")
				return nil
			}
			ui.Eid = eid
			err = email.SendEventNotificationEmail(ui, notif)
			if err != nil {
				return err
			}
		}
	}

	return err
}

// Notify users for Contract event.
func PushContractNotifications(notif model.ContractNotif) error {
	// Only the event with an ID will be notified.
	var eventBatch []*model.EventKindRef
	var createdAt string
	if notif.Contract == nil {
		return nil
	}
	createdAt = notif.Contract.CreatedAt
	eventBatch = append(eventBatch, &model.EventKindRef{ContractRef: &model.ContractRef{ID: &notif.Contract.ID}})

	// Get relevant users for the contract
	users, err := GetUsersToNotify(notif.Tid, true, false, false)
	if err != nil {
		return err
	}
	if notif.Contract.ContractType == model.ContractTypeAnyCoordoDual &&
		notif.Contract.Event.EventType == model.TensionEventMoved && notif.Contract.Event.New != nil {
		// The contract is created inside the tension or the node to be moved.
		// But we also need to notidy users in the target circle.
		targetid := *notif.Contract.Event.New
		x, err := db.GetDB().GetSubSubFieldByEq("Node.nameid", targetid, "Node.source", "Blob.tension", "uid")
		if err != nil {
			return err
		}
		targetTid, _ := x.(string)
		users2, err := GetUsersToNotify(targetTid, true, false, false)
		if err != nil {
			return err
		}
		for k, v := range users2 {
			users[k] = v
		}
	}
	// +
	// Add Candidates
	for _, c := range notif.Contract.Candidates {
		if x, _ := db.GetDB().GetFieldByEq("User.username", c.Username, "User.name"); x != nil {
			n := x.(string)
			c.Name = &n
		}
		users[c.Username] = model.UserNotifInfo{
			User:   *c,
			Reason: notif.Contract.Event.EventType.ToContractReason(),
		}
	}
	// +
	// Add Pending Candidates
	for _, c := range notif.Contract.PendingCandidates {
		users[c.Email] = model.UserNotifInfo{
			User:      model.User{Email: c.Email, NotifyByEmail: true},
			Reason:    notif.Contract.Event.EventType.ToContractReason(),
			IsPending: true,
		}
	}
	// +
	// Add Participants
	for _, p := range notif.Contract.Participants {
		if _, ex := users[p.Node.FirstLink.Username]; ex {
			continue
		}
		users[p.Node.FirstLink.Username] = model.UserNotifInfo{User: *p.Node.FirstLink, Reason: model.ReasonIsParticipant}
	}
	// +
	// Add mentionned and **set tension data**
	if m, err := db.GetDB().Meta("getLastContractComment", map[string]string{"cid": notif.Contract.ID, "username": notif.Uctx.Username}); err != nil {
		return err
	} else if len(m) > 0 {
		notif.Rootnameid = m[0]["rootnameid"].(string)
		notif.Receiverid = m[0]["receiverid"].(string)
		notif.Msg, _ = m[0]["message"].(string)
		if notif.Msg != "" {
			err = UpdateWithMentionnedUser(notif.Msg, notif.Receiverid, users)
			if err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("contract %s not found.", notif.Tid)
	}

	// Push user event notification
	for u, ui := range users {
		// Don't self notify.
		if u == notif.Uctx.Username {
			continue
		}

		// User Event
		var eid string
		if ui.IsPending {
			// Update pending users
			err = MaybeSetPendingUserToken(u)
			if err != nil {
				return err
			}
			// Link contract for future push
			err = db.GetDB().Update(db.GetDB().GetRootUctx(), "pendingUser", &model.UpdatePendingUserInput{
				Filter: &model.PendingUserFilter{Email: &model.StringHashFilter{Eq: &u}},
				Set:    &model.PendingUserPatch{Contracts: []*model.ContractRef{&model.ContractRef{ID: &notif.Contract.ID}}},
			})
			if err != nil {
				return err
			}
		} else {
			switch notif.ContractEvent {
			case model.NewContract:
				// Push user event
				eid, err = db.GetDB().Add(db.GetDB().GetRootUctx(), "userEvent", &model.AddUserEventInput{
					User:      &model.UserRef{Username: &u},
					IsRead:    false,
					CreatedAt: createdAt,
					Event:     eventBatch,
				})
				if err != nil {
					return err
				}
			case model.NewComment:
				// Push user notif
				PushNotifNotifications(model.NotifNotif{
					Uctx: notif.Uctx,
					Tid:  &notif.Tid,
					Cid:  &notif.Contract.ID,
					Msg:  "You have a new comment",
					To:   []string{u},
				}, false)

			case model.CloseContract:
				// processed outside the loop, below
			}
		}

		// Email
		if notif.Uctx.Rights.HasEmailNotifications && ui.User.NotifyByEmail && notif.IsEmailable(ui) {
			ui.Eid = eid
			err = email.SendContractNotificationEmail(ui, notif)
			if err != nil {
				return err
			}
		}
	}

	if notif.ContractEvent == model.CloseContract {
		// Push Event History and Notifications
		// Only once because this do not depend
		var event model.EventRef
		StructMap(notif.Contract.Event, &event)
		now := Now()
		event.CreatedAt = &now
		event.CreatedBy = &model.UserRef{Username: &notif.Uctx.Username}
		PushEventNotifications(model.EventNotif{Uctx: notif.Uctx, Tid: notif.Tid, History: []*model.EventRef{&event}})

		// Add a user notif to the candidate user with link to the accepted contract
		// has it won't be notify automatically (not subscrided to the tension yet).
		if *event.EventType == model.TensionEventUserJoined {
			for _, c := range notif.Contract.Candidates {
				isRead := false
				if c.Username == notif.Uctx.Username {
					isRead = true
				}
				PushNotifNotifications(model.NotifNotif{
					Uctx:   notif.Uctx,
					Tid:    &notif.Tid,
					Cid:    &notif.Contract.ID,
					Msg:    "You've joined a new organization.",
					To:     []string{c.Username},
					IsRead: isRead,
				}, true)
			}
		}
	}

	return err
}

// Notify users for Notif events.
func PushNotifNotifications(notif model.NotifNotif, selfNotify bool) error {
	// Only the event with an ID will be notified.
	var eventBatch []*model.EventKindRef
	var createdAt string = Now()
	var tensionRef *model.TensionRef
	var contractRef *model.ContractRef
	if notif.Tid != nil {
		tensionRef = &model.TensionRef{ID: notif.Tid}
	}
	if notif.Cid != nil {
		contractRef = &model.ContractRef{ID: notif.Cid}
	}

	eventBatch = append(eventBatch, &model.EventKindRef{NotifRef: &model.NotifRef{
		CreatedAt: &createdAt,
		CreatedBy: &model.UserRef{Username: &notif.Uctx.Username},
		Message:   &notif.Msg,
		Tension:   tensionRef,
		Contract:  contractRef,
		Link:      notif.Link,
	}})

	// Push user event notification
	for _, u := range notif.To {
		// Notif notification can self-notify !
		if u == notif.Uctx.Username && !selfNotify {
			continue
		}

		// User Event
		_, err := db.GetDB().Add(db.GetDB().GetRootUctx(), "userEvent", &model.AddUserEventInput{
			User:      &model.UserRef{Username: &u},
			IsRead:    notif.IsRead,
			CreatedAt: createdAt,
			Event:     eventBatch,
		})
		if err != nil {
			return err
		}

		// Email
		// No email for this one
	}

	return nil
}

//
// User helpers
//

// GetUserToNotify returns a list of user that should receive notifications upon tension updates.
// Note: order is important as for priority and emailing policy.
func GetUsersToNotify(tid string, withAssignees, withSubscribers, withPeers bool) (map[string]model.UserNotifInfo, error) {
	users := make(map[string]model.UserNotifInfo)

	// Data needed to get the first-link
	m, err := db.GetDB().Meta("getLastBlobTarget", map[string]string{"tid": tid})
	if err != nil {
		return users, err
	}
	if len(m) > 0 && m[0]["receiverid"] != nil && m[0]["nameid"] != nil && m[0]["type_"] != nil {
		// Get First-link
		_, nameid, err := codec.NodeIdCodec(
			m[0]["receiverid"].(string),
			m[0]["nameid"].(string),
			model.NodeType(m[0]["type_"].(string)),
		)
		if err != nil {
			return users, err
		}
		res, err := db.GetDB().GetSubFieldByEq("Node.nameid", nameid, "Node.first_link", auth.UserSelection)
		if err != nil {
			return users, err
		}
		if res != nil {
			var user model.User
			if err := Map2Struct(res.(model.JsonAtom), &user); err == nil {
				if _, ex := users[user.Username]; !ex {
					users[user.Username] = model.UserNotifInfo{User: user, Reason: model.ReasonIsFirstLink}
				}
			}
		}
	}

	if withAssignees {
		// Get Assignees
		res, err := db.GetDB().GetSubFieldById(tid, "Tension.assignees", auth.UserSelection)
		if err != nil {
			return users, err
		}
		if assignees, ok := InterfaceSlice(res); ok {
			for _, u := range assignees {
				var user model.User
				if err := Map2Struct(u.(model.JsonAtom), &user); err == nil {
					if _, ex := users[user.Username]; ex {
						continue
					}
					users[user.Username] = model.UserNotifInfo{User: user, Reason: model.ReasonIsAssignee}
				}
			}
		}
	}

	if withSubscribers {
		// Get Subscribers
		res, err := db.GetDB().GetSubFieldById(tid, "Tension.subscribers", auth.UserSelection)
		if err != nil {
			return users, err
		}
		if subscribers, ok := InterfaceSlice(res); ok {
			for _, u := range subscribers {
				var user model.User
				if err := Map2Struct(u.(model.JsonAtom), &user); err == nil {
					if _, ex := users[user.Username]; ex {
						continue
					}
					users[user.Username] = model.UserNotifInfo{User: user, Reason: model.ReasonIsSubscriber}
				}
			}
		}
	}

	{
		// Get Coordos
		coordos, err := auth.GetCoordosFromTid(tid)
		if err != nil {
			return users, err
		}
		for _, user := range coordos {
			if _, ex := users[user.Username]; ex {
				continue
			}
			users[user.Username] = model.UserNotifInfo{User: user, Reason: model.ReasonIsCoordo}
		}
	}

	if withPeers {
		// Get Peers
		peers, err := auth.GetPeersFromTid(tid)
		if err != nil {
			return users, err
		}
		for _, user := range peers {
			if _, ex := users[user.Username]; ex {
				continue
			}
			users[user.Username] = model.UserNotifInfo{User: user, Reason: model.ReasonIsPeer}
		}
	}

	return users, nil
}

// Update the users map with notified users.
// Note: only add user that are member of the given rootnameid
// @FIX: user inside block code will be notified here...
func UpdateWithMentionnedUser(msg string, receiverid string, users map[string]model.UserNotifInfo) error {
	// Remove code block
	msg = RemoveCodeBlocks(msg)

	rootnameid, err := codec.Nid2rootid(receiverid)
	if err != nil {
		return err
	}

	for i, U := range FindUsernames(msg) {
		u := strings.ToLower(U)
		if _, ex := users[u]; ex {
			user := users[u]
			user.Reason = model.ReasonIsMentionned
			users[u] = user
		} else {
			// Check that user is a member
			filter := `has(Node.first_link) AND NOT eq(Node.role_type, "Pending") AND NOT eq(Node.role_type, "Retired")`
			if ex, _ := db.GetDB().Exists("Node.nameid", codec.MemberIdCodec(rootnameid, u), &filter); !ex {
				continue
			}
			res, err := db.GetDB().GetFieldByEq("User.username", u, auth.UserSelection)
			if err != nil {
				return err
			}
			if res != nil {
				var user model.User
				if err := Map2Struct(res.(model.JsonAtom), &user); err == nil {
					users[u] = model.UserNotifInfo{User: user, Reason: model.ReasonIsMentionned}
				}
			}
		}

		if i > 100 {
			return fmt.Errorf("Too many user mentioned. Please consider using an Alert tension to notify group of users.")
		}
	}
	return nil
}

// Push mentioned tension event
func PushMentionedTension(notif model.EventNotif) error {
	// Remove code block
	msg := RemoveCodeBlocks(notif.Msg)

	rootnameid, err := codec.Nid2rootid(notif.Receiverid)
	if err != nil {
		return err
	}

	createdAt := Now()
	createdBy := model.UserRef{Username: &notif.Uctx.Username}
	var goto_ string
	for _, e := range notif.History {
		goto_ = *e.CreatedAt
		break
	}

	for _, tid := range FindTensions(msg) {
		rid, err := db.GetDB().GetSubFieldById(tid, "Tension.receiver", "Node.rootnameid")
		if err != nil {
			return err
		}
		if rid != nil && rid.(string) == rootnameid && tid != notif.Tid {
			// Push new event...
			_, err := db.GetDB().Add(db.GetDB().GetRootUctx(), "event", &model.AddEventInput{
				CreatedAt: createdAt,
				CreatedBy: &createdBy,
				Tension:   &model.TensionRef{ID: &tid},
				EventType: model.TensionEventMentioned,
				Mentioned: &model.TensionRef{ID: &notif.Tid},
				New:       &goto_,
			})
			if err != nil {
				return err
			}

		}

	}
	return nil
}
