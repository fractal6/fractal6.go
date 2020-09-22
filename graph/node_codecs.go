package graph

import (
    "fmt"
    "strings"

    "zerogov/fractal6.go/graph/model"
)

//
// Codecs
//

func nodeIdCodec(parentid string, targetid string,  nodeType model.NodeType) (string, string, error) {
    var nameid string
    rootnameid, err := nid2rootid(parentid)
    if nodeType == model.NodeTypeRole {
        if rootnameid == parentid {
            nameid = strings.Join([]string{rootnameid, "", targetid}, "#")
        } else {
            nameid = strings.Join([]string{parentid, targetid}, "#")
        }
    } else if nodeType == model.NodeTypeCircle {
        nameid = strings.Join([]string{rootnameid, targetid}, "#")
    }
    return rootnameid, nameid, err
}

// Get the parent nameid from the given nameid (ROLE)
func nid2pid(nid string) (string, error) {
    var pid string
    parts := strings.Split(nid, "#")
    if !(len(parts) == 3 || len(parts) == 1 || len(parts) == 2) {
        return pid, fmt.Errorf("bad nameid format for nid2pid: " + nid)
    }

    if len(parts) == 1 || parts[1] == "" {
        pid = parts[0]
    } else {
        pid = strings.Join(parts[:len(parts)-1],  "#")
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
