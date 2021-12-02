package handlers

import (
    //"fmt"
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
    uctx, err := auth.GetUserContext(r.Context())
    if err != nil { http.Error(w, err.Error(), 500); return }

    // Get request form
    var form model.OrgaForm
	err = json.NewDecoder(r.Body).Decode(&form)
	if err != nil { http.Error(w, err.Error(), 400); return }

    // Check format
    nameid := form.Nameid + "@" + uctx.Username // is personal namespace
    err = auth.ValidateNameid(nameid, nameid)
	if err != nil { http.Error(w, err.Error(), 400); return }
    nidOwner := nameid + "##" + "@" + uctx.Username

    // Check plan
    ok, err := auth.CanNewOrga(*uctx, form)
    if err != nil || !ok { http.Error(w, err.Error(), 400); return }

    // @debug; temporary hack, see issue here: https://discuss.dgraph.io/t/create-child-nodes-with-addparent/11311/13
    uid_, _ := db.GetDB().GetFieldByEq("User.username", uctx.Username, "uid")
    uid := uid_.(string)

    // @TODO
    userCanJoin := true
    isPersonal := true
    visibility := model.NodeVisibilityPublic
    mode := model.NodeModeCoordinated

    // Create the new node
    nodeInput := model.AddNodeInput{
        // Form
        Name: form.Name,
        Nameid: nameid,
        Rootnameid: nameid,
        About: form.About,
        // Default
        Type: model.NodeTypeCircle,
        IsRoot: true,
        IsPersonal: &isPersonal,
        Mandate: &model.MandateRef{ Purpose: form.Purpose },
        // Permission
        UserCanJoin: &userCanJoin,
        Visibility: visibility,
        Mode: mode,
        Rights: 0,
        IsArchived: false,
        // Common
        CreatedAt: Now(),
        CreatedBy: &model.UserRef{ID: &uid},
        //CreatedBy: &model.UserRef{Username: &uctx.Username},
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
    _, err = db.GetDB().Add(db.GetDB().GetRootUctx(), "node", nodeInput)
    if err != nil { http.Error(w, err.Error(), 400); return }

    // Add the root control tension
    tensionInput := graph.MakeNewRootTension(uctx, nameid, nodeInput)
    tid, err := db.GetDB().Add(db.GetDB().GetRootUctx(), "tension", tensionInput)
    if err != nil { http.Error(w, err.Error(), 400); return }

    // Links the source tension
    bid := db.GetDB().GetLastBlobId(tid)
    err = db.GetDB().SetNodeSource(nameid, *bid)
    if err != nil { http.Error(w, err.Error(), 400); return }

    // Add the Owner role to the user
    err = db.GetDB().AddUserRole(uctx.Username, nidOwner)
    if err != nil { http.Error(w, err.Error(), 400); return }

    // return result on success
    data, _ := json.Marshal(model.Node{Nameid: nameid})
    w.Write(data)
}

