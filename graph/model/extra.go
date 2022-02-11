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

// JsonAtom is a general interface
// for decoding unknonw structure
type JsonAtom = map[string]interface{}
