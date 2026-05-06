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

package graph

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strconv"

	"github.com/99designs/gqlgen/graphql"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/internal/tools"
	"fractale/fractal6.go/web/auth"
)

// ProjectCard Resolver
// --
// We update the card position in list when thery are
// moved to respect the shifting.

type ProjectCardLoc struct {
	ID          string `json:"id"`
	Colid       string
	Colname     string
	Colcolor    string
	Projectid   string
	Projectname string
	Pos         int
	Contentid   string
	Receiverid  string
	Typenames   []string
}

var QueryCardLoc db.QueryMut = db.QueryMut{
	Q: `query {
            all(func: uid({{.cardid}})) @normalize {
                uid
                ProjectCard.pc {
                    colid: uid
                    colname: ProjectColumn.name
                    colcolor: ProjectColumn.color
                    ProjectColumn.project {
                        projectid: uid
                        projectname: Project.name
                    }
                }
                pos: ProjectCard.pos
                ProjectCard.card {
                    contentid: uid
                    receiverid: Tension.receiverid
                    typenames: dgraph.type
                }
            }
        }`,
}

type ProjectColumnDesc struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// projectCardMoveLoc is the decoded record of QueryCardLocAndNewCol. The query
// returns two normalized rows over the same `all` block — one for the card uid
// (with ProjectCard.* fields populated, NewColname/NewColcolor empty) and one
// for the new column uid (only NewColname/NewColcolor populated). They are
// split apart by isCardRow.
type projectCardMoveLoc struct {
	ProjectCardLoc
	NewColname  string `json:"new_colname,omitempty"`
	NewColcolor string `json:"new_colcolor,omitempty"`
}

// isCardRow reports whether a decoded row corresponds to the ProjectCard uid
// (vs. the new column uid). Cards always have a ProjectCard.card link, so
// Contentid is set; columns have no such field.
func (l projectCardMoveLoc) isCardRow() bool { return l.Contentid != "" }

// QueryCardLocAndNewCol fetches both the old ProjectCardLoc and the new column
// descriptor in a single DQL request. Used by updateProjectCardHook when the
// card is being moved between columns.
var QueryCardLocAndNewCol db.QueryMut = db.QueryMut{
	Q: `query {
            all(func: uid({{.cardid}}, {{.new_colid}})) @normalize {
                uid
                ProjectCard.pc {
                    colid: uid
                    colname: ProjectColumn.name
                    colcolor: ProjectColumn.color
                    ProjectColumn.project {
                        projectid: uid
                        projectname: Project.name
                    }
                }
                pos: ProjectCard.pos
                ProjectCard.card {
                    contentid: uid
                    receiverid: Tension.receiverid
                    typenames: dgraph.type
                }
                new_colname: ProjectColumn.name
                new_colcolor: ProjectColumn.color
            }
        }`,
}

// fetchCardAndNewCol runs QueryCardLocAndNewCol and splits the two normalized
// rows into the card loc and the new column descriptor.
func fetchCardAndNewCol(cardid, newColid string) (ProjectCardLoc, ProjectColumnDesc, error) {
	rows, err := db.Gamma[projectCardMoveLoc](QueryCardLocAndNewCol, map[string]string{
		"cardid":     cardid,
		"new_colid":  newColid,
	})
	if err != nil {
		return ProjectCardLoc{}, ProjectColumnDesc{}, err
	}
	var card ProjectCardLoc
	var newCol ProjectColumnDesc
	newCol.ID = newColid
	for _, r := range rows {
		if r.isCardRow() {
			card = r.ProjectCardLoc
		} else {
			newCol.Name = r.NewColname
			newCol.Color = r.NewColcolor
		}
	}
	return card, newCol, nil
}

// projectDescriptor returns "{projectid}§{projectname}§" for use in Event.old/new.
func projectDescriptor(id, name string) string {
	return id + "§" + name + "§"
}

// goTensionEvent runs a tension-history write in the background. Errors are
// logged; panics are recovered. Used by the ProjectCard hooks so the API
// response is not blocked on the secondary event write.
func goTensionEvent(label string, fn func() error) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("async %s panic: %v", label, r)
			}
		}()
		if err := fn(); err != nil {
			log.Printf("async %s failed: %v", label, err)
		}
	}()
}

// columnDescriptor returns "{colid}§{colname}§{colcolor}§{projectid}" for use in Event.old/new.
func columnDescriptor(id, name, color, projectid string) string {
	return id + "§" + name + "§" + color + "§" + projectid
}

// isTensionCard reports whether a ProjectCard's referenced card is a Tension
// (not a ProjectDraft). Project events are only emitted for tension cards.
func isTensionCard(typenames []string) bool {
	return slices.Contains(typenames, "Tension")
}

// pushTensionProjectEvent appends a project-related Event to tension.history and
// bumps Tension.updatedAt in a single updateTension mutation. Post.updatedAt is
// the order key for sort=activity in db/tensionQuery.go.
// Auth is already enforced upstream (CheckProjectAuth in the ProjectCard hook),
// so we skip EMAP and write directly with the root uctx.
func pushTensionProjectEvent(uctx *model.UserCtx, tid, receiverNameid string, et model.TensionEvent, oldVal, newVal string) error {
	now := Now()
	event := &model.EventRef{
		CreatedAt: &now,
		CreatedBy: &model.UserRef{Username: &uctx.Username},
		EventType: &et,
	}
	if oldVal != "" {
		event.Old = &oldVal
	}
	if newVal != "" {
		event.New = &newVal
	}
	if err := db.GetDB().Update(db.GetDB().GetRootUctx(), "tension", &model.UpdateTensionInput{
		Filter: &model.TensionFilter{ID: []string{tid}},
		Set: &model.TensionPatch{
			UpdatedAt: &now,
			History:   []*model.EventRef{event},
		},
	}); err != nil {
		return err
	}
	if receiverNameid != "" {
		trackActivity(uctx.Username, receiverNameid, et)
	}
	return nil
}

// PushProjectAdded writes a ProjectAdded event for a freshly-added ProjectCard.
// No-op if the card is a ProjectDraft.
func PushProjectAdded(uctx *model.UserCtx, cardID string) error {
	loc, err := First(db.Gamma[ProjectCardLoc](QueryCardLoc, map[string]string{"cardid": cardID}))
	if err != nil {
		return err
	}
	if !isTensionCard(loc.Typenames) {
		return nil
	}
	return pushTensionProjectEvent(
		uctx, loc.Contentid, loc.Receiverid, model.TensionEventProjectAdded,
		"", projectDescriptor(loc.Projectid, loc.Projectname),
	)
}

// PushProjectRemoved writes a ProjectRemoved event for a ProjectCard about to be deleted.
// `loc` must be captured BEFORE the card is removed. No-op for drafts.
func PushProjectRemoved(uctx *model.UserCtx, loc ProjectCardLoc) error {
	if !isTensionCard(loc.Typenames) {
		return nil
	}
	return pushTensionProjectEvent(
		uctx, loc.Contentid, loc.Receiverid, model.TensionEventProjectRemoved,
		projectDescriptor(loc.Projectid, loc.Projectname), "",
	)
}

// PushProjectColumnMoved writes a ProjectColumnMoved event for a card moved between columns.
// `oldLoc` must reflect the card state BEFORE the move; `newCol` is the destination
// column descriptor (callers typically already have it on hand). No-op for drafts
// or pure in-column position shuffles.
func PushProjectColumnMoved(uctx *model.UserCtx, oldLoc ProjectCardLoc, newCol ProjectColumnDesc) error {
	if newCol.ID == oldLoc.Colid {
		return nil
	}
	if !isTensionCard(oldLoc.Typenames) {
		return nil
	}
	return pushTensionProjectEvent(
		uctx, oldLoc.Contentid, oldLoc.Receiverid, model.TensionEventProjectColumnMoved,
		columnDescriptor(oldLoc.Colid, oldLoc.Colname, oldLoc.Colcolor, oldLoc.Projectid),
		columnDescriptor(newCol.ID, newCol.Name, newCol.Color, oldLoc.Projectid),
	)
}

// Add "ProjectCard"
func addProjectCardHook(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	// Pre-processing:
	// - Auth

	// Get User context
	ctx, uctx, err := auth.GetUserContext(ctx)
	if err != nil {
		return nil, LogErr("Access denied", err)
	}

	// Validate input
	var inputs []model.AddProjectCardInput
	ExtractInputs(ctx, &inputs)
	isDraft := make([]bool, len(inputs))
	for i, input := range inputs {
		x, err := db.GetDB().GetByUid(*input.Pc.ID, "ProjectColumn.project", "uid")
		if err != nil {
			return nil, err
		}
		if x == nil {
			return nil, fmt.Errorf("project not found for column %s", *input.Pc.ID)
		}
		projectid := x.(string)

		// Check project auth
		if err = auth.Authorize(auth.CheckProjectAuth(uctx, projectid)); err != nil {
			return nil, err
		}

		isDraft[i] = input.Card != nil && input.Card.ProjectDraftRef != nil
	}

	// Forward query
	data, err := next(ctx)
	if err != nil {
		return data, err
	}
	d := data.(*model.AddProjectCardPayload)
	if d == nil {
		return nil, LogErr("add ProjectCard", fmt.Errorf("silent error: no card added"))
	}

	// Post-processing:
	// - Shift card position in columns list
	// - Emit ProjectAdded for tension cards (drafts have no tension to attach to).

	for i, card := range d.ProjectCard {
		if card.ID == "" {
			return data, fmt.Errorf("id payload required for project card mutation")
		}
		if _, err := db.GetDB().Meta("incrementCardPos", map[string]string{"cardid": card.ID, "now": Now()}); err != nil {
			return data, err
		}
		if i < len(isDraft) && isDraft[i] {
			continue
		}
		cardID := card.ID
		goTensionEvent("PushProjectAdded", func() error {
			return PushProjectAdded(uctx, cardID)
		})
	}

	return data, err
}

// Add "ProjectCard"
func deleteProjectCardHook(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	// Pre-processing:
	// - get values prior mutations
	// - Auth

	// Get User context
	ctx, uctx, err := auth.GetUserContext(ctx)
	if err != nil {
		return nil, LogErr("Access denied", err)
	}

	// Get input
	var filter model.ProjectCardFilter
	ExtractFilter(ctx, &filter)
	if len(filter.ID) == 0 {
		return nil, fmt.Errorf("Query requires id filters.")
	}
	// Prior to remove, get information about that object for post-processing
	oldCards := []ProjectCardLoc{}
	for _, uid := range filter.ID {
		card, err := First(db.Gamma[ProjectCardLoc](QueryCardLoc, map[string]string{"cardid": uid}))
		if err != nil {
			return nil, err
		}
		oldCards = append(oldCards, card)

		// Check project auth
		if err = auth.Authorize(auth.CheckProjectAuth(uctx, card.Projectid)); err != nil {
			return nil, err
		}
	}

	// Forward query
	data, err := next(ctx)
	if err != nil {
		return data, err
	}
	d := data.(*model.DeleteProjectCardPayload)
	if d == nil {
		return nil, LogErr("delete ProjectCard", fmt.Errorf("silent error: no card deleted"))
	}

	// Post-processing:
	// - shift card positions in columns list
	// - eventually delete draft

	for _, card := range d.ProjectCard {
		if card.ID == "" {
			return data, fmt.Errorf("id payload required for project card mutation")
		}
		// Search for ids that as been actually deleted
		cardLoc, ok := Find(oldCards, func(c ProjectCardLoc) bool {
			return c.ID == card.ID
		})
		if !ok {
			log.Printf("Error: ProjectCard loc not found for card: %s", card.ID)
			continue
		}
		// Shift card position
		_, err := db.GetDB().Meta("decrementCardPos", map[string]string{
			"pos":   strconv.Itoa(card.Pos),
			"colid": cardLoc.Colid,
			"tid":   cardLoc.Contentid,
		})
		if err != nil {
			return data, err
		}
		// Push ProjectRemoved to tension history (skip drafts).
		loc := cardLoc
		goTensionEvent("PushProjectRemoved", func() error {
			return PushProjectRemoved(uctx, loc)
		})
		if l := slices.Index(cardLoc.Typenames, "ProjectDraft"); l >= 0 {
			// Delete draft
			_, err := db.GetDB().Meta("deleteCardDraft", map[string]string{"cardid": card.ID})
			if err != nil {
				return data, err
			}
		}
	}

	return data, err
}

// Update "ProjectCard"
// @warning: update of card position only supported for one card at a time.
func updateProjectCardHook(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	// Pre-processing:
	// - Auth

	// Get User context
	ctx, uctx, err := auth.GetUserContext(ctx)
	if err != nil {
		return nil, LogErr("Access denied", err)
	}

	// Get input
	var input model.UpdateProjectCardInput
	ExtractInput(ctx, &input)
	isMoved := false
	oldCard := ProjectCardLoc{}
	newCol := ProjectColumnDesc{}
	if input.Set != nil && len(input.Filter.ID) == 1 {
		id := input.Filter.ID[0]
		isMoved = input.Set.Pos != nil && input.Set.Pc != nil
		if isMoved {
			// Single DQL request returns both old card loc and new col descriptor.
			oldCard, newCol, err = fetchCardAndNewCol(id, *input.Set.Pc.ID)
		} else {
			oldCard, err = First(db.Gamma[ProjectCardLoc](QueryCardLoc, map[string]string{"cardid": id}))
		}
		if err != nil {
			return nil, err
		}
		if oldCard.Projectid == "" {
			return nil, fmt.Errorf("project not found for card %s", id)
		}

		// Check project auth
		if err = auth.Authorize(auth.CheckProjectAuth(uctx, oldCard.Projectid)); err != nil {
			return nil, err
		}
	} else {
		// Review Auth + auto increment
		return nil, fmt.Errorf("Not implemented")
	}

	if input.Remove != nil {
		return nil, fmt.Errorf("remove is not allowed for this mutation")
	}

	// Forward query
	data, err := next(ctx)
	if err != nil {
		return data, err
	}

	// Post-processing:
	// - shift card position in columns list
	// - emit ProjectColumnMoved when the column actually changed

	// Auto increment card position only when updating a single card,
	// otherwise, assume that user know what they are doing.
	if isMoved {
		newPos := *input.Set.Pos
		newColid := newCol.ID
		q := "moveCardPos"
		if newColid == oldCard.Colid {
			if oldCard.Pos > newPos {
				q = "moveCardPosUp"
			} else if newPos > oldCard.Pos {
				q = "moveCardPosDown"
			}
		}

		_, err := db.GetDB().Meta(q, map[string]string{
			"cardid":    oldCard.ID,
			"old_pos":   strconv.Itoa(oldCard.Pos),
			"old_colid": oldCard.Colid,
			"now":       Now(),
		})
		if err != nil {
			return data, err
		}

		// PushProjectColumnMoved skips drafts and same-column shuffles internally.
		loc := oldCard
		col := newCol
		goTensionEvent("PushProjectColumnMoved", func() error {
			return PushProjectColumnMoved(uctx, loc, col)
		})
	}

	return data, err
}
