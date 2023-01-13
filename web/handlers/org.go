/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2022 Fractale Co
 *
 * This file is part of Fractale.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Fractale.  If not, see <http://www.gnu.org/licenses/>.
 */

package handlers

import (
	//"fmt"
	"encoding/json"
	"net/http"
	"strconv"
    "strings"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/web/auth"
	. "fractale/fractal6.go/tools"
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
    err = auth.ValidateName(form.Name)
	if err != nil { http.Error(w, err.Error(), 400); return }
    err = auth.ValidateNameid(form.Nameid, form.Nameid)
	if err != nil { http.Error(w, err.Error(), 400); return }
    if strings.Contains(form.Nameid, "#") {
        http.Error(w, "Illegal character '#' in nameid", 400); return
    }

    nameid := form.Nameid
    nidOwner := nameid + "##" + "@" + uctx.Username
    isPersonal := true
    var userCanJoin bool
    var guestCanCreateTension bool = true
    visibility := model.NodeVisibilityPublic
    mode := model.NodeModeCoordinated

    if form.Visibility != nil && form.Visibility.IsValid() {
        visibility = *form.Visibility
    }
    form.Visibility = &visibility

    if visibility == model.NodeVisibilityPublic {
        userCanJoin = true
    } else {
        userCanJoin = false
    }

    // Check plan
    ok, err := auth.CanNewOrga(*uctx, form)
    if err != nil || !ok { http.Error(w, err.Error(), 400); return }

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
        Watchers: []*model.UserRef{&model.UserRef{Username: &uctx.Username}},
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
    owner.About = nil
    owner.Watchers = nil
    nodeInput.Children = []*model.NodeRef{&owner}
    // Gql mutation
    _, err = db.GetDB().Add(db.GetDB().GetRootUctx(), "node", nodeInput)
    if err != nil { http.Error(w, err.Error(), 400); return }

    // Add the root control tension
    about := form.About
    mandate := &model.MandateRef{ Purpose: form.Purpose }
    tensionInput := graph.MakeNewRootTension(nameid, nodeInput, about, mandate)
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
    nameid := form.Nameid
    _, uctx, err := auth.GetUserContext(r.Context())
    if err != nil { http.Error(w, err.Error(), 500); return }
    if i := auth.UserHasCoordoRole(uctx, nameid); i < 0 {
        http.Error(w, "Only coordinators of the circle can do this.", 400)
        return
    }

    // Set the value
    val := strconv.FormatBool(form.Val)
    err = db.GetDB().SetFieldByEq("Node.nameid", nameid, "Node.userCanJoin", val)
    if err != nil { http.Error(w, err.Error(), 500); return }

    // Maybe Update the circle visibility if userCanJoin is set to True
    if form.Val {
        visibility, err := db.GetDB().GetFieldByEq("Node.nameid", nameid, "Node.visibility")
        visibilityPublic := string(model.NodeVisibilityPublic)
        if err != nil { http.Error(w, err.Error(), 500); return }
        if visibility.(string) != visibilityPublic {
            // Update Node
            _, err := db.GetDB().Meta("setNodeVisibility", map[string]string{"nameid":nameid, "value":visibilityPublic})
            if err != nil { http.Error(w, err.Error(), 500); return }
            // Change all role direct children
            err = db.DB.SetChildrenRoleVisibility(nameid, visibilityPublic)
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
    nameid := form.Nameid
    _, uctx, err := auth.GetUserContext(r.Context())
    if err != nil { http.Error(w, err.Error(), 500); return }
    if i := auth.UserHasCoordoRole(uctx, nameid); i < 0 {
        http.Error(w, "Only coordinators of the circle can do this.", 400)
        return
    }

    // Set the value
    val := strconv.FormatBool(form.Val)
    err = db.GetDB().SetFieldByEq("Node.nameid", nameid, "Node.guestCanCreateTension", val)
    if err != nil { http.Error(w, err.Error(), 500); return }

    w.Write([]byte(val))
}
