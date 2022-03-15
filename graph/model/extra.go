package model

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
    ReasonIsCandidate
    ReasonIsPendingCandidate
    ReasonIsParticipant
    ReasonIsCoordo
    ReasonIsFirstLink
    ReasonIsAssignee
    ReasonIsSubscriber
)

func (n NotifReason) ToText() string {
    switch n {
    case ReasonIsCandidate:
        return "you are invited"
    case ReasonIsCoordo:
        return "you are coordinator"
    case ReasonIsFirstLink:
        return "you are first-link"
    case ReasonIsAssignee:
        return "you are assigned to this tension"
    case ReasonIsSubscriber:
        return "you are subscribed to this tension"
    default:
        return "unknown reason"
    }
}

// Info about user to notify when pushing notification
type UserNotifInfo struct {
    User User
    Reason NotifReason
    Eid string
}


type EventNotif struct {
    Uctx *UserCtx        `json:"uctx"`
    Tid string           `json:"tid"`
    History []*EventRef  `json:"history"`
}

type ContractNotif struct {
    Uctx *UserCtx       `json:"uctx"`
    Tid string          `json:"tid"`
    Contract *Contract  `json:"contract"`
}

type NotifNotif struct {
    Uctx *UserCtx  `json:"uctx"`
    Msg string     `json:"msg"`
    Tid *string    `json:"tid"`
    Cid *string    `json:"cid"`
    To []string    `json:"to"`
}

//
// Object methods
//

func (notif EventNotif) IsEmailable() bool {
    for _, e := range notif.History {
        if TensionEventCreated == *e.EventType ||
        TensionEventCommentPushed == *e.EventType ||
        TensionEventReopened == *e.EventType ||
        TensionEventClosed == *e.EventType ||
        TensionEventMemberUnlinked == *e.EventType ||
        TensionEventUserLeft == *e.EventType {
            return true
        }
    }
    return false
}

func (notif ContractNotif) IsEmailable() bool {
    e := notif.Contract.Event
    if TensionEventCreated == e.EventType ||
    TensionEventCommentPushed == e.EventType ||
    TensionEventReopened == e.EventType ||
    TensionEventClosed == e.EventType ||
    TensionEventMemberUnlinked == e.EventType ||
    TensionEventUserLeft == e.EventType {
        return true
    }
    return false
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

// Event methods

func (e TensionEvent) ToContractText() (t string) {
    switch e {
	case TensionEventMoved:
		t = "Move tension"

	case TensionEventMemberLinked:
		t = "New first-link"

	case TensionEventMemberUnlinked:
		t = "Retired first-link"

	case TensionEventUserJoined:
		t = "New member"

	default:
        // Humanize (@debug: cannot import tools because of cycle error.)
		t = string(e)
    }
    return
}


// JsonAtom is a general interface
// for decoding unknonw structure
type JsonAtom = map[string]interface{}
