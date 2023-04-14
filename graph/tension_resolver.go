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

package graph

import (
    "fmt"
    "context"
    "github.com/99designs/gqlgen/graphql"
    "fractale/fractal6.go/graph/model"
    "fractale/fractal6.go/db"
    "fractale/fractal6.go/web/auth"
    . "fractale/fractal6.go/tools"
)


////////////////////////////////////////////////
// Tension Resolver
////////////////////////////////////////////////

func tensionInputHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    data, err := setUpdateContextInfo(ctx, obj, next)  // for @hasEvent+@isOwner
    if err != nil {
        return data, err
    }

    //newData := data.([]*model.AddContractInput)

    // Set BlobType -- based on Blob.
    b2i := map[bool]int{false:0, true:1}
    switch newData := data.(type) {
    case model.UpdateTensionInput:
        if newData.Set == nil { break }
        input := newData.Set
        if len(input.Blobs) == 0 { break }
        // Blob are update OneByOne
        blob := input.Blobs[0]
        if blob.Node == nil { break }
        // Blob are update OneByOne
        blob_type_lvl := b2i[blob.Node.About != nil] + b2i[blob.Node.Mandate != nil]*2
        var bt model.BlobType
        switch blob_type_lvl {
        case 1:
            bt = model.BlobTypeOnAbout
        case 2:
            bt = model.BlobTypeOnMandate
        case 3:
            bt = model.BlobTypeOnAboutAndMandate
        }
        blob.BlobType = &bt
        return newData, err
    case []*model.AddTensionInput:
        for _, input := range newData {
            if len(input.Blobs) == 0 { break }
            // Blob are update OneByOne
            blob := input.Blobs[0]
            if blob.Node == nil { break }
            bt := model.BlobTypeOnNode
            blob.BlobType = &bt
        }
        return newData, err
    }

    return data, err
}

// Add Tension - Hook
func addTensionHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    ctx, uctx, err := auth.GetUserContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate Input
    inputs := graphql.GetResolverContext(ctx).Args["input"].([]*model.AddTensionInput)
    if len(inputs) != 1 {
        return nil, LogErr("add tension", fmt.Errorf("One and only one tension allowed."))
    }
    if !PayloadContains(ctx, "id") {
        return nil, LogErr("field missing", fmt.Errorf("id field is required in tension payload"))
    }
    input := inputs[0]

    // History and notification Logics --
    // In order to notify user on the given event, we need to know their ids to pass and link them
    // to the notification (UserEvent edge) function. To do so we first cut the history from the original
    // input, and push then the history (see the PushHistory function).
    ctx = context.WithValue(ctx, "cut_history", true) // Used by DgraphQueryResolverRaw
    history := input.History
    input.History = nil
    // Execute query
    data, err := next(ctx)
    if err != nil { return data, err }
    if data.(*model.AddTensionPayload) == nil {
        return nil, LogErr("add tension", fmt.Errorf("no tension added."))
    }
    tension := data.(*model.AddTensionPayload).Tension[0]
    id := tension.ID

    // Validate and process Blob Event
    ok, _,  err := TensionEventHook(uctx, id, history, nil)
    if !ok || err != nil {
        // Delete the tension just added
        e := db.GetDB().DeepDelete("tension", id)
        if e != nil { panic(e) }
    }
    if err != nil {
        return data, err
    }
    if ok {
        PublishTensionEvent(model.EventNotif{Uctx: uctx, Tid: id, History: history})
        return data, err
    }
    return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
}

// Update Tension - Hook
func updateTensionHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    ctx, uctx, err := auth.GetUserContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate input
    input := graphql.GetResolverContext(ctx).Args["input"].(model.UpdateTensionInput)
    ids := input.Filter.ID
    if len(ids) != 1 {
        return nil, LogErr("update tension", fmt.Errorf("One and only one tension allowed."))
    }

    // Validate Event prior the mutation
    var blob *model.BlobRef
    var contract *model.Contract
    var ok bool
    if input.Set != nil {
        if len(input.Set.Blobs) > 0 {
            blob = input.Set.Blobs[0]
        }
        ok, contract, err = TensionEventHook(uctx, ids[0], input.Set.History, blob)
        if err != nil { return nil, err }
        if ok {
            // History and notification Logics --
            // In order to notify user on the given event, we need to know
            // their ids to pass and link them to the user's notifications (UserEvent edge).
            // To do so we first cut the history from the original input,
            // and push then the history (see the [[PushHistory]] function).
            ctx = context.WithValue(ctx, "cut_history", true) // Used by DgraphQueryResolverRaw
            history := input.Set.History
            now := Now()
            input.Set.History = nil
            input.Set.UpdatedAt = &now
            // Execute query
            data, err := next(ctx)
            if err != nil { return data, err }
            PublishTensionEvent(model.EventNotif{Uctx: uctx, Tid: ids[0], History: history})
            return data, err
        } else if contract != nil {
            var t model.UpdateTensionPayload
            t.Tension = []*model.Tension{&model.Tension{
                Contracts: []*model.Contract{contract},
            }}
            return &t, err
        } else {
            return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
        }
    }

    return nil, LogErr("Access denied", fmt.Errorf("Input remove not implemented."))
}


