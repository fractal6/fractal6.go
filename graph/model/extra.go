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
    ReasonIsCoordo
    ReasonIsFirstLink
    ReasonIsAssignee
    ReasonIsSubscriber
)

func (n NotifReason) ToText() string {
    switch n {
    case ReasonIsCandidate:
        return "candidate"
    case ReasonIsCoordo:
        return "coordinator"
    case ReasonIsFirstLink:
        return "first-link"
    case ReasonIsAssignee:
        return "assignee"
    case ReasonIsSubscriber:
        return "subscriber"
    default:
        return "unknown"
    }
}

// User info when pushing notification
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


// JsonAtom is a general interface
// for decoding unknonw structure
type JsonAtom = map[string]interface{}
