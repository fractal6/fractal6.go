/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
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
	"encoding/json"
	"fmt"
	"net/http"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/web/auth"
	"fractale/fractal6.go/web/email"
)

func MakeOwner(w http.ResponseWriter, r *http.Request) {
	// Get form data
	form := struct {
		// User to grant
		Username string
		// Target orga
		Nameid string
	}{}
	err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Check if uctx has rights in nameid (is coordo)
	_, uctx, err := auth.GetUserContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if !codec.IsRoot(form.Nameid) {
		http.Error(w, "The circle must be an anchor circle.", 400)
		return
	}
	if i := auth.UserIsOwner(uctx, form.Nameid); i < 0 {
		http.Error(w, "Only organisation owners can do this.", 400)
		return
	}

	// Upgrade Member
	nid := codec.MemberIdCodec(form.Nameid, form.Username)
	if ex, _ := db.GetDB().Exists("Node.nameid", nid, nil); !ex {
		http.Error(w, "This member does not exist.", 400)
		return
	}
	err = db.GetDB().UpgradeMember(nid, model.RoleTypeOwner)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Send a notification to the new owner.
	var orgName string
	var anchorTid string
	if x, err := db.GetDB().GetFieldByEq("Node.nameid", form.Nameid, "Node.name"); err != nil {
		http.Error(w, err.Error(), 500)
		return
	} else {
		orgName = x.(string)
	}
	if x, err := db.GetDB().GetSubSubFieldByEq("Node.nameid", form.Nameid, "Node.source", "Blob.tension", "uid"); err != nil {
		http.Error(w, err.Error(), 500)
		return
	} else {
		anchorTid = x.(string)
	}
	graph.PushNotifNotifications(model.NotifNotif{
		Uctx: uctx,
		Tid:  &anchorTid,
		Cid:  nil,
		Msg:  fmt.Sprintf("You have been granted owner of an organisation (%s)", orgName),
		To:   []string{form.Username},
	}, false)
	err = email.SendOwnerGrantedEmail(form.Username, form.Nameid, orgName)
	if err != nil {
		panic(err)
	}

	w.Write([]byte("true"))
}
