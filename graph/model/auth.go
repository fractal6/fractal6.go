package model

// JsonAtom is a general interface for decoing unknonw structure
type JsonAtom = map[string]interface{}


//
// User Auth Data structure
//

// UserCtx are data encoded in the thoken (e.g Jwt claims)
type UserCtx struct {
    Username string  `json:"User.username"`
    Name     *string `json:"User.name"`
    Passwd   string  `json:"User.password"`
	Roles    []Role  `json:"User.roles"`
}
type Role struct {
    Nameid string    `json:"Node.nameid"` // the node identifier
    RoleType string  `json:"Node.role_type"` // the role type
}


// UserCreds are data sink/form for login request
type UserCreds struct {
    Username string  `json:"username"`
    Email    string  `json:"email"`
    Name     *string  `json:"name"`
    Password string  `json:"password"`
}
