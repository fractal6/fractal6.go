package model

//
// User Auth Data structure
//

// UserCtx are data encoded in the token (e.g Jwt claims)
// @DEBUG: see emcapsulation issue: https://github.com/golang/go/issues/9859
type UserCtx struct {
    Username       string     `json:"username"`
    Name           *string    `json:"name"`
    Password       string     `json:"password"` // hash
    Rights         UserRights `json:"rights"`
	Roles          []*Node    `json:"roles"`
    // fot token iat (empty when uctx is got from DB)
    // limit the DB hit by keeping nodes checked for iat
    // number of time the userctx iat is checked
    Iat            string
    CheckedNameid  []string // keep the nameid checked for context session to limit the db requests.
    Hit            int
    NoCache        bool
}

//
// Data Form
//

// UserCreds are data sink/form for login request
type UserCreds struct {
    Username   string  `json:"username"`
    Email      string  `json:"email"`
    Name       *string `json:"name"`
    Password   string  `json:"password"`
    Puid       *string `json:"puid"`
    EmailToken *string `json:"email_token"`
}


// OrgaForm are data sink/form for creating new organisation
type OrgaForm struct {
    Name    string  `json:"name"`
    Nameid  string  `json:"nameid"`
    About   *string `json:"about"`
    Purpose *string `json:"purpose"`
}

//
// Data Patch
//

// Prevent Auth properties to be changed from blob pushes
// as unentended update can occurs as Peer role can pushed blob.
// It means, that each of the properties below should have their own events
type NodePatchFromFragment struct {
	Name       *string            `json:"name,omitempty"`
	About      *string            `json:"about,omitempty"`
	Mandate    *MandateRef        `json:"mandate,omitempty"`
	Skills     []string           `json:"skills,omitempty"`
	Children   []*NodeFragmentRef `json:"children,omitempty"`
}

