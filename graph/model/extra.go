package model

// JsonAtom is a general interface 
// for decoding unknonw structure
type JsonAtom = map[string]interface{}

type NodeId struct {
    Nameid string `json:"nameid"`
}

type MemberNode struct {
    CreatedAt string    `json:"createdAt"`
    Name string         `json:"name"`
    Nameid string       `json:"nameid"`
    Rootnameid string   `json:"rootnameid"`
	RoleType *RoleType  `json:"role_type,omitempty"`
	FirstLink *User_    `json:"first_link,omitempty"`
	Parent *NodeId      `json:"parent,omitempty"`

}

type User_ struct {
    Username string `json:"username"`
    Name *string    `json:"name,omitempty"`
}

