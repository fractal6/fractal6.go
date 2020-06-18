package model

// JsonAtom is a general interface 
// for decoding unknonw structure
type JsonAtom = map[string]interface{}

type NodeId struct {
    Nameid string `json:"nameid"`
}

