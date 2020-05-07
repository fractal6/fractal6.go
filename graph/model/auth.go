package model

// JsonAtom is a general interface 
// for decoding unknonw structure
type JsonAtom = map[string]interface{}


//
// User Auth Data structure
//


// UserCreds are data sink/form for login request
type UserCreds struct {
    Username string  `json:"username"`
    Email    string  `json:"email"`
    Name     *string `json:"name"`
    Password string  `json:"password"`
}

// UserCtx are data encoded in the thoken 
// (e.g Jwt claims)
type UserCtx struct {
    Username string  `json:"username"`
    Name     *string `json:"name"`
    Passwd   string  `json:"password"` // hash
	Roles    []Role  `json:"roles"`
}
type Role struct {
    Rootnamid string    `json:"rootnameid"` 
    Nameid string    `json:"nameid"` 
    RoleType string  `json:"role_type"` 
}

//
// Dgraph copy with their own json tag
//
//
type UserCtxDg struct {
    Username string  `json:"User.username"`
    Name     *string `json:"User.name"`
    Passwd   string  `json:"User.password"` // hash
	Roles    []Role  `json:"User.roles"`
}
type RoleDg struct {
    Nameid string    `json:"Node.nameid"` 
    RoleType string  `json:"Node.role_type"` 
}
