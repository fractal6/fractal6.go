
package handlers

import (
    //"fmt"
    "time"
    "net/http"
    "encoding/json"

    "zerogov/fractal6.go/db"
    "zerogov/fractal6.go/web/auth"
    "zerogov/fractal6.go/graph"
    "zerogov/fractal6.go/graph/model"
    . "zerogov/fractal6.go/tools"
)


// Signup register a new user and gives it a token.
func CreateOrga(w http.ResponseWriter, r *http.Request) {
    // Get user token
    uctx, err := auth.UserCtxFromContext(r.Context())
    if err != nil { http.Error(w, err.Error(), 500); return }

    // Get request form
    var form model.OrgaForm
	err = json.NewDecoder(r.Body).Decode(&form)
	if err != nil { http.Error(w, err.Error(), 400); return }

    // Check format
    nameid := form.Nameid + "@" + uctx.Username
    err = auth.ValidateNameid(nameid, nameid)
	if err != nil { http.Error(w, err.Error(), 400); return }
    nidOwner := nameid + "##" + "@" + uctx.Username


    // Check plan
    ok, err := auth.CanNewOrga(uctx, form)
    if err != nil || !ok { http.Error(w, err.Error(), 400); return }

    // Create the new node
    isPersonal := true
    userCanJoin := false
    mode := model.NodeModeCoordinated
    nodeInput := model.AddNodeInput{
        // Form
        Name: form.Name,
        Nameid: nameid,
        Rootnameid: nameid,
        About: form.About,
        Mandate: &model.MandateRef{ Purpose: form.Purpose },
        // Default
        Type: model.NodeTypeCircle,
        IsRoot: true,
        IsPersonal: &isPersonal,
        IsPrivate: false,
        IsArchived: false,
        Charac: &model.NodeCharacRef{ UserCanJoin: &userCanJoin, Mode: &mode },
        // Common
        CreatedAt: time.Now().Format(time.RFC3339),
        CreatedBy: &model.UserRef{Username: &uctx.Username},
    }
    // Set Owner
    var owner model.NodeRef
    StructMap(nodeInput, &owner)
    t := model.NodeTypeRole
    rt := model.RoleTypeOwner
    n := string(rt)
    _root := false
    owner.Type = &t
    owner.Nameid = &nidOwner
    owner.Name = &n
    owner.RoleType = &rt
    owner.IsRoot = &_root
    owner.Mandate = nil
    nodeInput.Children = []*model.NodeRef{&owner}
    // Gql mutation
    _, err = db.GetDB().Add("node", nodeInput)
    if err != nil { http.Error(w, err.Error(), 400); return }

    // Add the root control tension
    tensionInput := graph.MakeNewRootTension(uctx, nameid, nodeInput)
    tid, err := db.GetDB().Add("tension", tensionInput)
    if err != nil { http.Error(w, err.Error(), 400); return }

    // Links the source tension
    err = db.GetDB().SetTensionSource(nameid, tid)
    if err != nil { http.Error(w, err.Error(), 400); return }

    // Add the Owner role to the user
    err = auth.AddUserRole(uctx.Username, nidOwner)
    if err != nil { http.Error(w, err.Error(), 400); return }

    // return result on success
    data, _ := json.Marshal(model.NodeId{Nameid: nameid})
    w.Write(data)
}

// AddUserRole add a role to the user roles list
func AddUserRole(username, nameid string) error {
    userInput := model.UpdateUserInput{
        Filter: &model.UserFilter{ Username: &model.StringHashFilter{ Eq: &username } },
        Set: &model.UserPatch{
            Roles: []*model.NodeRef{ &model.NodeRef{ Nameid: &nameid }},
        },
    }
    err := db.GetDB().Update("user", userInput)
    return err
}
