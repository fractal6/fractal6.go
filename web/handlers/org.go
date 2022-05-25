package handlers

import (
	//"fmt"
	"encoding/json"
	"net/http"
	"strconv"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph"
	ga "fractale/fractal6.go/graph/auth"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/tools"
	"fractale/fractal6.go/web/auth"
)

// Signup register a new user and gives it a token.
func CreateOrga(w http.ResponseWriter, r *http.Request) {
    // Get user token
    uctx, err := auth.GetUserContextLight(r.Context())
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

    // @TODO
    var userCanJoin bool
    isPersonal := true
    visibility := model.NodeVisibilityPublic
    mode := model.NodeModeCoordinated
    guestCanCreateTension := true
    if visibility == model.NodeVisibilityPublic {
        userCanJoin = true
    } else {
        userCanJoin = false
    }

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
        Visibility: visibility,
        Mode: mode,
        Rights: 0,
        IsArchived: false,
        UserCanJoin: &userCanJoin,
        GuestCanCreateTension: &guestCanCreateTension,
        // Common
        CreatedAt: Now(),
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
    _, err = db.GetDB().Add(db.GetDB().GetRootUctx(), "node", nodeInput)
    if err != nil { http.Error(w, err.Error(), 400); return }

    // Add the root control tension
    tensionInput := graph.MakeNewRootTension(nameid, nodeInput)
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


func SetUserCanJoin(w http.ResponseWriter, r *http.Request) {
    // Get form data
    form := struct {
        Nameid string
        Val bool
    }{}
    err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil { http.Error(w, err.Error(), 400); return }

    // Check if uctx has rights in nameid (is coordo)
    _, uctx, err := auth.GetUserContext(r.Context())
    if err != nil { http.Error(w, err.Error(), 500); return }
    if i := ga.UserIsCoordo(uctx, form.Nameid); i < 0 {
        http.Error(w, "Only coordinators of the root circle can do this.", 400)
        return
    }

    // Set the value
    val := strconv.FormatBool(form.Val)
    err = db.GetDB().SetFieldByEq("Node.nameid", form.Nameid, "Node.userCanJoin", val)
    if err != nil { http.Error(w, err.Error(), 500); return }

    // Maybe Update the circle visibility if userCanJoin is set to True
    if form.Val {
        visibility, err := db.GetDB().GetFieldByEq("Node.nameid", form.Nameid, "Node.visibility")
        if err != nil { http.Error(w, err.Error(), 500); return }
        if visibility.(string) != string(model.NodeVisibilityPublic) {
            // Update Node
            _, err := db.GetDB().Meta("setNodeVisibility", map[string]string{"nameid":form.Nameid, "value":string(model.NodeVisibilityPublic)})
            if err != nil { http.Error(w, err.Error(), 500); return }
        }
    }

    w.Write([]byte(val))
}


func SetGuestCanCreateTension(w http.ResponseWriter, r *http.Request) {
    // Get form data
    form := struct {
        Nameid string
        Val bool
    }{}
    err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil { http.Error(w, err.Error(), 400); return }

    // Check if uctx has rights in nameid (is coordo)
    _, uctx, err := auth.GetUserContext(r.Context())
    if err != nil { http.Error(w, err.Error(), 500); return }
    if i := ga.UserIsCoordo(uctx, form.Nameid); i < 0 {
        http.Error(w, "Only coordinators of the root circle can do this.", 400)
        return
    }

    // Set the value
    val := strconv.FormatBool(form.Val)
    err = db.GetDB().SetFieldByEq("Node.nameid", form.Nameid, "Node.guestCanCreateTension", val)
    if err != nil { http.Error(w, err.Error(), 500); return }

    w.Write([]byte(val))
}
