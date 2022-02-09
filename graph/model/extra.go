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

type Notif struct {
    Tid string          `json:"tid"`
    Uctx *UserCtx       `json:"uctx"`
    History []*EventRef `json:"history"`
}

// JsonAtom is a general interface
// for decoding unknonw structure
type JsonAtom = map[string]interface{}
