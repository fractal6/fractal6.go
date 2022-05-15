package auth

import (
    "fmt"
    "fractale/fractal6.go/graph/model"
    "fractale/fractal6.go/graph/codec"
    "fractale/fractal6.go/db"
    . "fractale/fractal6.go/tools"
)

// Inherits node properties
func InheritNodeCharacDefault(node *model.NodeFragment, parent *model.Node) {
    if node.Mode == nil {
        node.Mode = &parent.Mode
    }
    if node.Visibility == nil {
        node.Visibility = &parent.Visibility
    }
}

////////////////////////////////////////////////
// Base authorization methods
// @future: GBAC authorization with @auth directive (DGraph)
////////////////////////////////////////////////

//
// Getters
//

func HasCoordoRole(uctx *model.UserCtx, nameid string, mode *model.NodeMode) (bool, error) {
    // Get the node mode eventually
    if mode == nil {
        mode_, err := db.GetDB().GetFieldByEq("Node.nameid", nameid, "Node.mode")
        if err != nil { return false, LogErr("Internal error", err) }
        mode = mode_.(*model.NodeMode)
    }

    // Check user rights
    ok, err := CheckUserRights(uctx, nameid, *mode)
    if err != nil { return ok, LogErr("Internal error", err) }

    // Check if user has rights in any parents if the node has no Coordo role.
    if !ok && !db.GetDB().HasCoordos(nameid) {
        ok, err = CheckUpperRights(uctx, nameid, *mode)
    }
    return ok, err
}

func GetCoordosFromTid(tid string) ([]model.User, error) {
    var coordos []model.User

    // Check user rights
    nodes, err := db.GetDB().Meta("getCoordosFromTid", map[string]string{"tid":tid})
    if err != nil { return coordos, LogErr("Internal error", err) }

    // Return direct coordos
    if len(nodes) > 0 {
        for _, c := range nodes {
            var coordo model.User
            if err := Map2Struct(c, &coordo); err != nil {
                return coordos, err
            }
            coordos = append(coordos, coordo)
        }
        return coordos, err
    }

    // Return first met parent coordos
    var parents []string
    node, err := db.GetDB().Meta("getParentFromTid", map[string]string{"tid":tid})
    if err != nil { return coordos, LogErr("Internal Error", err) }
    if len(node) == 0 || node[0]["parent"] == nil {
        return coordos, err
    }
    // @debug: dql decoding !
    if nodes := node[0]["parent"].([]interface{}); len(nodes) > 0 {
        if nids, ok :=  nodes[0].(model.JsonAtom)["nameid"]; ok && nids != nil {
            switch x := nids.(type) {
            case []interface{}:
                for _, v := range x {
                    parents = append(parents, v.(string))
                }
            case string:
                parents = append(parents, x)
            }
        }
    }

    for _, nameid := range parents {
        res, err := db.GetDB().Meta("getCoordos2", map[string]string{"nameid": nameid})
        if err != nil { return coordos, LogErr("Internal error", err) }

        // stop at the first circle with coordos
        if len(res) > 0 {
            for _, c := range res {
                var coordo model.User
                if err := Map2Struct(c, &coordo); err != nil {
                    return coordos, err
                }
                coordos = append(coordos, coordo)
            }
            return coordos, err
        }
    }

    return coordos, err
}

//
// Checkers
//

// ChechUserRight return true if the user has access right (e.g. Coordo) on the given node
func CheckUserRights(uctx *model.UserCtx, nameid string, mode model.NodeMode) (bool, error) {
    var ok bool = false
    var err error

    // Get the nearest circle
    nid, err := codec.Nid2pid(nameid)
    if err != nil { return ok, err }

    // Escape if the user is an owner
    if UserIsOwner(uctx, nid) >= 0 { return true, err }

    if mode == model.NodeModeAgile {
        ok = UserHasRole(uctx, nid) >= 0
    } else if mode == model.NodeModeCoordinated {
        ok = UserIsCoordo(uctx, nid) >= 0
    }

    return ok, err
}

// chechUpperRight return true if the user has access right (e.g. Coordo) on any on its parents.
func CheckUpperRights(uctx *model.UserCtx, nameid string, mode model.NodeMode) (bool, error) {
    var ok bool = false
    parents, err := db.GetDB().GetParents(nameid)
    if err != nil { return ok, LogErr("Internal Error", err) }

    for _, p := range(parents) {
        ok, err = CheckUserRights(uctx, p, mode)
        if err != nil { return ok, LogErr("Internal error", err) }
        if ok { break }
    }

    return ok, err
}

//
// Sanitize TensionQuery
//

func QueryAuthFilter(uctx model.UserCtx, q *db.TensionQuery) error {
    if q == nil { return fmt.Errorf("Empty query") }

    res, err := db.GetDB().Query(uctx, "node", "nameid", q.Nameids, "nameid visibility")
    if err != nil { return err }

    // For circle with visibility right
    var nameids []string
    // For circle with restricted visibility right
    var nameidsProtected []string

    for _, r := range res {
        nameid := r["nameid"]
        visibility := r["visibility"]

        // Get the nearest circle
        nid, err := codec.Nid2pid(r["nameid"])
        if err != nil { return err }

        if visibility == string(model.NodeVisibilityPrivate) && UserIsMember(&uctx, nid) < 0 {
            // If Private & non Member
            nameidsProtected = append(nameidsProtected, nameid)
        } else if visibility == string(model.NodeVisibilitySecret) && UserHasRole(&uctx, nid) < 0 {
            // If Secret & non Peer
            nameidsProtected = append(nameidsProtected, nameid)
        } else {
            // else (Public or with right)
            nameids = append(nameids, nameid)
        }
    }

    q.Nameids = nameids
    q.NameidsProtected = nameidsProtected
    q.Username = uctx.Username
    // add NameidsProtected attribute in TensionQuery
    if len(nameids) + len(nameidsProtected) == 0 {
		return fmt.Errorf("error: no node name given (nameid empty)")
    }

    return nil
}
