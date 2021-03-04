package model

//
// Data Form
//

// UserCreds are data sink/form for login request
type UserCreds struct {
    Username string  `json:"username"`
    Email    string  `json:"email"`
    Name     *string `json:"name"`
    Password string  `json:"password"`
}


// OrgaForm are data sink/form for creating new organisation
type OrgaForm struct {
    Name    string   `json:"name"`
    Nameid  string   `json:"nameid"`
    About   *string  `json:"about"`
    Purpose *string  `json:"purpose"`
}

//
// User Auth Data structure
//

// UserCtx are data encoded in the token (e.g Jwt claims)
type UserCtx struct {
    Username string     `json:"username"`
    Name     *string    `json:"name"`
    Passwd   string     `json:"password"` // hash
    Rights   UserRights `json:"rights"`
	Roles    []Role     `json:"roles"`
    Iat      string     // fot token iat (empty when uctx is got from DB)
}
type Role struct {
    Rootnameid string  `json:"rootnameid"`
    Nameid string      `json:"nameid"`
    Name string        `json:"name"`
    RoleType RoleType  `json:"role_type"`
}

//
// Dgraph Payload for Gpm query
//

// UserCtx Payload
var UserCtxPayloadDg string = `{
    User.username
    User.name
    User.password
    User.rights {expand(_all_)}
    User.roles {
        Node.rootnameid
        Node.nameid
        Node.name
        Node.role_type
    }
}`

// NodeCharac Payload
var NodeCharacNF string = "Node.charac"
var NodeCharacPayloadDg string = `{
    Node.charac {
        NodeCharac.userCanJoin
        NodeCharac.mode
    }
}`

