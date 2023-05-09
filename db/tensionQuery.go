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

package db

import (
	"fmt"
	"strconv"
	"strings"

	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
)

type TensionQuery struct {
	Nameids []string             `json:"nameids"`
	First   int                  `json:"first"`
	Offset  int                  `json:"offset"`
	Pattern *string              `json:"pattern"`
	Sort    *string              `json:"sort"`
	Status  *model.TensionStatus `json:"status"`
	Type    *model.TensionType   `json:"type_"`
	Authors []string             `json:"authors"`
	Labels  []string             `json:"labels"`
	// Either filter tension that in or NOT in the given project
	InProject bool    `json:in_project`
	Projectid *string `json:"projectid"`
	// Protected tensions @auth
	NameidsProtected []string
	Username         string
}

// Note: We assumes here all nameids have the same rootnameid.
func FormatTensionIntExtMap(q TensionQuery) (*map[string]string, error) {
	var err error
	/* list format */
	preVars := ""

	// Nameids
	var nameids []string
	var nameidsString string
	for _, v := range q.Nameids {
		nameids = append(nameids, fmt.Sprintf("eq(Node.nameid, \"%s\")", v))
	}

	// Protected Nameids
	var nameidsProtected []string
	var nameidsProtectedString string
	for _, v := range q.NameidsProtected {
		nameidsProtected = append(nameidsProtected, fmt.Sprintf("eq(Node.nameid, \"%s\")", v))
	}

	// Authors
	var authors []string
	for _, v := range q.Authors {
		authors = append(authors, fmt.Sprintf("eq(User.username, \"%s\")", v))
	}

	// labels
	var labels []string
	for _, v := range q.Labels {
		labels = append(labels, fmt.Sprintf("eq(Label.name, \"%s\")", v))
	}

	/* Tension filter */
	var tf []string
	var tensionFilter string
	if q.Status != nil {
		tf = append(tf, fmt.Sprintf(`eq(Tension.status, "%s")`, q.Status))
	}
	if q.Type != nil {
		tf = append(tf, fmt.Sprintf(`eq(Tension.type_, "%s")`, q.Type))
	}
	if q.Pattern != nil {
		tf = append(tf, fmt.Sprintf(`anyoftext(Tension.title, "%s")`, *q.Pattern))
	}
	if len(q.Authors) > 0 {
		tf = append(tf, `has(Post.createdBy)`)
	}
	if len(q.Labels) > 0 {
		tf = append(tf, `has(Tension.labels)`)
	}
	if q.Projectid != nil {
		if q.InProject {
			tf = append(tf, `uid_in(Tension.project_statuses, uid(columns))`)
		} else {
			tf = append(tf, `NOT uid_in(Tension.project_statuses, uid(columns))`)
		}
		preVars += fmt.Sprintf(`var(func: uid(%s)) {
            columns as Project.columns
        }`, *q.Projectid)
	}

	if len(tf) > 0 {
		tensionFilter = fmt.Sprintf(
			"@filter(%s)",
			strings.Join(tf, " AND "),
		)
	}

	/* sorting */
	var sortFilter string = "orderdesc"
	if q.Sort != nil {
		if *q.Sort == "oldest" {
			sortFilter = "orderasc"
		}
	}

	/* Sub Tension filter */
	var authorsFilter string
	var labelsFilter string
	if len(q.Authors) > 0 {
		authorsFilter = strings.Join(authors, " OR ")
		authorsFilter = fmt.Sprintf(
			"Post.createdBy @filter(%s)",
			authorsFilter,
		)

	}
	if len(q.Labels) > 0 {
		labelsFilter = strings.Join(labels, " OR ")
		labelsFilter = fmt.Sprintf(
			"Tension.labels @filter(%s)",
			labelsFilter,
		)
	}

	// Rootnameid
	var rootnameid string
	if len(q.Nameids) > 0 {
		rootnameid, err = codec.Nid2rootid(q.Nameids[0])
		if err != nil {
			return nil, err
		}
		nameidsString = strings.Join(nameids, " OR ")
	} else if len(q.NameidsProtected) > 0 {
		nameidsString = ""
	}
	// -- Protected circles
	var rootnameidProtected string
	var hasSelf bool
	for _, u := range authors { // @reduce: with generics
		if u == q.Username {
			hasSelf = true
		}
	}
	if len(q.NameidsProtected) > 0 && (hasSelf || len(q.Authors) == 0) {
		rootnameidProtected, err = codec.Nid2rootid(q.NameidsProtected[0])
		if err != nil {
			return nil, err
		}
		nameidsProtectedString = strings.Join(nameidsProtected, " OR ")
	}

	/* Build template map */
	maps := &map[string]string{
		"first":         strconv.Itoa(q.First),
		"offset":        strconv.Itoa(q.Offset),
		"rootnameid":    rootnameid,
		"nameids":       nameidsString,
		"tensionFilter": tensionFilter,
		"authorsFilter": authorsFilter,
		"labelsFilter":  labelsFilter,
		"order":         sortFilter,
		// Protected
		"rootnameidProtected": rootnameidProtected,
		"nameidsProtected":    nameidsProtectedString,
		"username":            q.Username,
		// Extra
		"extra_pre_vars": preVars,
	}

	return maps, nil
}
