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

package model

import "encoding/json"


//
// General
//

// JsonAtom is a general interface
// for decoding unknonw structure
type JsonAtom = map[string]interface{}

// StructMap convert/copy a interface to another
func StructMap(in interface{}, out interface{}) {
    raw, _ := json.Marshal(in)
    json.Unmarshal(raw, &out)
}

//
// Errors
//

type GqlErrors struct {
    Errors []GqlError `json:"errors"`
}

type GqlError struct {
    Location string  `json:"location"`
    Message string   `json:"message"`
}

//
// Notifications
//

// Notification Reason Type enum
type NotifReason int
const (
    ReasonUnknown NotifReason = iota
    ReasonIsInvited
    ReasonIsLinkCandidate
    ReasonIsCandidate
    ReasonIsParticipant
    ReasonIsCoordo
    ReasonIsPeer
    ReasonIsFirstLink
    ReasonIsAssignee
    ReasonIsSubscriber
    ReasonIsMentionned
    ReasonIsAlert
    ReasonIsAnnouncement
)

func (n NotifReason) ToText() string {
    switch n {
    case ReasonIsInvited:
        return `you are invited to join an organization on <a href="https://fractale.co">Fractale</a>`
    case ReasonIsLinkCandidate:
        return "you are invited to play a role"
    case ReasonIsCandidate:
        return "you are candidate"
    case ReasonIsParticipant:
        return "you voted to this contract"
    case ReasonIsCoordo:
        return "you are coordinator in this circle"
    case ReasonIsPeer:
        return "you have role in this circle"
    case ReasonIsFirstLink:
        return "you are lead link of this role"
    case ReasonIsAssignee:
        return "you are assigned to this tension"
    case ReasonIsSubscriber:
        return "you are subscribed to this tension"
    case ReasonIsMentionned:
        return "you have been mentionned"
    case ReasonIsAlert:
        return "you are a member of this organisation"
    case ReasonIsAnnouncement:
        return "you are watching this organisation"
    default:
        return "unknown reason"
    }
}

// Info about user to notify when pushing notification
type UserNotifInfo struct {
    User User
    Reason NotifReason
    Eid string
    IsPending bool
}

// @future: move in schema ?
type ContractEvent int
const (
    NewContract ContractEvent = iota
    NewComment
    CloseContract
)

type EventNotif struct {
    Uctx *UserCtx        `json:"uctx"`
    Tid string           `json:"tid"`
    History []*EventRef  `json:"history"`
    // The following are get after the cache publication
    // to keep the messaging system light and as fast as possible.
    Rootnameid string    `json:"rootnameid"`
    Receiverid string    `json:"receiverid"`
    Title string         `json:"title"`
    Msg string           `json:"msg"`
}

type ContractNotif struct {
    Uctx *UserCtx               `json:"uctx"`
    Tid string                  `json:"tid"`
    Contract *Contract          `json:"contract"`
    ContractEvent ContractEvent `json:"contract_event"`
    // The following are get after the the cache publication
    // to keep the messaging system as fast as possible.
    Rootnameid string           `json:"rootnameid"`
    Receiverid string           `json:"receiverid"`
    Msg string                  `json:"msg"`
}

type NotifNotif struct {
    Uctx *UserCtx  `json:"uctx"`
    Msg string     `json:"msg"`
    Tid *string    `json:"tid"`
    Cid *string    `json:"cid"`
    Link *string   `json:"link"`
    To []string    `json:"to"`
    IsRead bool    `json:"isRead"`
}

//
// EventNotif methods
//


// External Notification Policy
func (notif EventNotif) IsEmailable(ui UserNotifInfo) bool {
    var ok bool = false

    // Mailable events
    for _, e := range notif.History {
        if TensionEventCreated == *e.EventType ||
        TensionEventReopened == *e.EventType ||
        TensionEventClosed == *e.EventType ||
        TensionEventBlobPushed == *e.EventType ||
        TensionEventCommentPushed == *e.EventType ||
        TensionEventUserJoined == *e.EventType ||
        TensionEventUserLeft == *e.EventType ||
        TensionEventMemberLinked == *e.EventType ||
        TensionEventMemberUnlinked == *e.EventType {
            ok = true
        }
    }

    // Emailing Policy
    if ok {
        // PeerReason only for Created tension and Updated mandate.
        if ui.Reason == ReasonIsPeer &&
        !notif.HasEvent(TensionEventCreated) &&
        !notif.HasEvent(TensionEventBlobPushed) {
            ok = false
        }

        // Coordo not for Pushed comment only event.
        if ui.Reason == ReasonIsCoordo && len(notif.History) == 1 &&
        notif.HasEvent(TensionEventCommentPushed) {
            ok = false
        }
    }

    return ok
}

// Internal Notification Policy
func (notif EventNotif) IsNotifiable(ui UserNotifInfo) bool {
    var ok bool = false

    // Policy accept all for
    // - firstlink - assignee - mentionned - alert - announce
    if ui.Reason == ReasonIsFirstLink ||
    ui.Reason == ReasonIsAssignee ||
    ui.Reason == ReasonIsMentionned ||
    ui.Reason == ReasonIsAlert ||
    ui.Reason == ReasonIsAnnouncement {
        ok = true
    }

    // Policy for
    // - subscriber
    if ui.Reason == ReasonIsSubscriber && (
        notif.HasEvent(TensionEventCreated) ||
        notif.HasEvent(TensionEventReopened) ||
        notif.HasEvent(TensionEventClosed) ||
        notif.HasEvent(TensionEventCommentPushed) ||
        notif.HasEvent(TensionEventBlobPushed) ||
        notif.HasEvent(TensionEventBlobArchived) ||
        notif.HasEvent(TensionEventBlobUnarchived) ||
        notif.HasEvent(TensionEventUserJoined) ||
        notif.HasEvent(TensionEventUserLeft) ||
        notif.HasEvent(TensionEventMemberLinked) ||
        notif.HasEvent(TensionEventMemberUnlinked)) {
        ok = true
    }


    // Policy for
    // - coordo
    if ui.Reason == ReasonIsCoordo && (
        notif.HasEvent(TensionEventCreated) ||
        notif.HasEvent(TensionEventBlobPushed) ||
        notif.HasEvent(TensionEventClosed) ||
        notif.HasEvent(TensionEventBlobArchived) ||
        notif.HasEvent(TensionEventBlobUnarchived) ||
        notif.HasEvent(TensionEventUserJoined) ||
        notif.HasEvent(TensionEventUserLeft) ||
        notif.HasEvent(TensionEventMemberLinked) ||
        notif.HasEvent(TensionEventMemberUnlinked) ||
        notif.HasEvent(TensionEventMoved)) {
        ok = true
    }

    // Policy for
    // - peer
    if ui.Reason == ReasonIsPeer && (
        notif.HasEvent(TensionEventCreated) ||
        notif.HasEvent(TensionEventBlobPushed)) {
        ok = true
    }


    return ok
}

func (notif EventNotif) HasEvent(ev TensionEvent) bool {
    for _, e := range notif.History {
        if ev == *e.EventType {
            return true
        }
    }
    return false
}

func (notif EventNotif) GetCreatedAt() string {
    for _, e := range notif.History {
        if e.CreatedAt != nil {
            return *e.CreatedAt
        }
    }
    return ""
}

func (notif EventNotif) GetNewUser() string {
    for _, e := range notif.History {
        if *e.EventType == TensionEventUserJoined || *e.EventType == TensionEventMemberLinked {
            if e.New != nil {
                return *e.New
            }
        }
    }
    return ""
}

func (notif EventNotif) GetExUser() string {
    for _, e := range notif.History {
        if *e.EventType == TensionEventUserLeft || *e.EventType == TensionEventMemberUnlinked {
            if e.Old != nil {
                return *e.Old
            }
        }
    }
    return ""
}

//
// ContractNotif methods
//

func (notif ContractNotif) IsEventEmailable(ui UserNotifInfo) bool {
    ev := EventRef{}
    StructMap(notif.Contract.Event, &ev)
    en := EventNotif{
        Uctx: notif.Uctx,
        Tid: notif.Tid,
        Receiverid: notif.Receiverid,
        History:[]*EventRef{&ev},
    }
    return en.IsEmailable(ui)
}

func (notif ContractNotif) IsEmailable(ui UserNotifInfo) bool {
    return true
}

//
// TensionEvent methods
//

func (e TensionEvent) ToContractText() (t string) {
    switch e {
	case TensionEventUserJoined:
		t = "Invitation"
	case TensionEventMemberLinked:
		t = "Lead link invitation"
	case TensionEventMoved:
		t = "Move tension"
	case TensionEventMemberUnlinked:
		t = "Retired first-link"
	default:
        // Humanize (@debug: cannot import tools because of cycle error.)
		t = string(e)
    }
    return
}

func (e TensionEvent) ToContractReason() (r NotifReason) {
    switch e {
    case TensionEventUserJoined:
        r = ReasonIsInvited
    case TensionEventMemberLinked:
        r = ReasonIsLinkCandidate
    default:
        r = ReasonIsCandidate
    }
    return r
}


