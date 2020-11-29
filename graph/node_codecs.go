package graph

import (
    "fmt"
    "strings"

    "zerogov/fractal6.go/graph/model"
)

//
// Codecs
//

func nodeIdCodec(parentid, targetid string, nodeType model.NodeType) (string, string, error) {
    var nameid string
    rootnameid, err := nid2rootid(parentid)
    if len(strings.Split(targetid, "#")) > 1 {
        return rootnameid, targetid, err
    }
    if nodeType == model.NodeTypeRole {
        if rootnameid == parentid {
            nameid = strings.Join([]string{rootnameid, "", targetid}, "#")
        } else {
            nameid = strings.Join([]string{parentid, targetid}, "#")
        }
    } else if nodeType == model.NodeTypeCircle {
        nameid = strings.Join([]string{rootnameid, targetid}, "#")
    }
    nameid = strings.TrimSuffix(nameid, "#")
    return rootnameid, nameid, err
}

// Get the parent nameid from the given nameid (ROLE)
// @debug nearestCircleId
func nid2pid(nid string) (string, error) {
    var pid string
    parts := strings.Split(nid, "#")

    if len(parts) == 1 || parts[1] == "" {
        pid = parts[0]
    } else if len(parts) == 2 {
        pid = nid
    } else if len(parts) == 3 {
        pid = strings.Join(parts[:len(parts)-1],  "#")
    } else {
        return pid, fmt.Errorf("bad nameid format for nid2pid: " + nid)
    }
    return pid, nil
}

// Get the rootnameid from the given nameid
func nid2rootid(nid string) (string, error) {
    var pid string
    parts := strings.Split(nid, "#")
    if !(len(parts) == 3 || len(parts) == 1 || len(parts) == 2) {
        return pid, fmt.Errorf("bad nameid format for nid2pid: " + nid)
    }

    return parts[0], nil
}

func isCircle(nid string) (bool) {
    parts := strings.Split(nid, "#")
    return len(parts) == 1 || len(parts) == 2
}
func isRole(nid string) (bool) {
    parts := strings.Split(nid, "#")
    return len(parts) == 3
}
