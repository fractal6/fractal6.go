package codec

import (
    "fmt"
    "strings"

    "zerogov/fractal6.go/graph/model"
)

// Format the nameid id from its parts
func NodeIdCodec(parentid, targetid string, nodeType model.NodeType) (string, string, error) {
    var nameid string
    rootnameid, err := Nid2rootid(parentid)
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

func MemberIdCodec(rootnameid, username string) (string) {
    nameid := strings.Join([]string{rootnameid, "","@"+ username}, "#")
    return nameid
}

func ContractIdCodec(receiverid string, event_type model.TensionEvent, old, new_ string) (string) {
    nameid := strings.Join([]string{receiverid, string(event_type),old, new_}, "#")
    return nameid
}

func VoteIdCodec(contractid string, nid string) (string) {
    nameid := strings.Join([]string{contractid, nid}, "#")
    return nameid
}

// Get the parent nameid from the given nameid (ROLE)
// @debug nearestCircleId
func Nid2pid(nid string) (string, error) {
    var pid string
    parts := strings.Split(nid, "#")

    if len(parts) == 1 || parts[1] == "" {
        pid = parts[0]
    } else if len(parts) == 2 {
        pid = nid
    } else if len(parts) == 3 {
        pid = strings.Join(parts[:len(parts)-1],  "#")
    } else {
        return pid, fmt.Errorf("bad nameid format for Nid2pid: " + nid)
    }
    return pid, nil
}

// Get the rootnameid from the given nameid
func Nid2rootid(nid string) (string, error) {
    var pid string
    parts := strings.Split(nid, "#")
    if !(len(parts) == 3 || len(parts) == 1 || len(parts) == 2) {
        return pid, fmt.Errorf("bad nameid format for Nid2pid: " + nid)
    }

    return parts[0], nil
}

func IsRoot(nid string) (bool) {
    parts := strings.Split(nid, "#")
    return len(parts) == 1
}

func IsCircle(nid string) (bool) {
    parts := strings.Split(nid, "#")
    return len(parts) == 1 || len(parts) == 2
}
func IsRole(nid string) (bool) {
    parts := strings.Split(nid, "#")
    return len(parts) == 3
}
