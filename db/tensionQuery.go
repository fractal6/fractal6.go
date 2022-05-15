package db

import (
    "fmt"
    "strings"
    "strconv"

    "fractale/fractal6.go/graph/codec"
    "fractale/fractal6.go/graph/model"

)

type TensionQuery struct {
    Nameids []string 	        `json:"nameids"`
    First int                   `json:"first"`
    Offset int                  `json:"offset"`
    Query *string               `json:"query"`
    Sort *string                `json:"sort"`
    Status *model.TensionStatus `json:"status"`
    Type *model.TensionType     `json:"type_"`
    Authors []string            `json:"authors"`
    Labels []string             `json:"labels"`
    // Protected tensions @auth
    NameidsProtected []string
    Username string
}

// Note: We assumes here all nameids have the same rootnameid.
func FormatTensionIntExtMap(q TensionQuery) (*map[string]string, error) {
    var err error
    /* list format */

    // Nameids
    var nameids []string
    var nameidsString string
    for _, v := range(q.Nameids) {
        nameids = append(nameids, fmt.Sprintf("eq(Node.nameid, \"%s\")", v))
    }

    // Protected Nameids
    var nameidsProtected []string
    var nameidsProtectedString string
    for _, v := range(q.NameidsProtected) {
        nameidsProtected = append(nameidsProtected, fmt.Sprintf("eq(Node.nameid, \"%s\")", v))
    }

    // Authors
    var authors []string
    for _, v := range(q.Authors) {
        authors = append(authors, fmt.Sprintf("eq(User.username, \"%s\")", v))
    }

    // labels
    var labels []string
    for _, v := range(q.Labels) {
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
    if q.Query != nil {
        tf = append(tf, fmt.Sprintf(`anyoftext(Tension.title, "%s")`, *q.Query))
    }
    if len(q.Authors) > 0 {
        tf = append(tf, `has(Post.createdBy)`)
    }
    if len(q.Labels) > 0 {
        tf = append(tf, `has(Tension.labels)`)
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
        if err != nil { return nil, err }
        nameidsString = strings.Join(nameids, " OR ")
    } else if len(q.NameidsProtected) > 0 {
        nameidsString = ""
    }
    // -- Protected circles
    var rootnameidProtected string
    var hasSelf bool
    for _, u := range authors { // @reduce: with generics
        if u == q.Username { hasSelf = true }
    }
    if len(q.NameidsProtected) > 0 && (hasSelf || len(q.Authors) == 0) {
        rootnameidProtected, err = codec.Nid2rootid(q.NameidsProtected[0])
        if err != nil { return nil, err }
        nameidsProtectedString = strings.Join(nameidsProtected, " OR ")
    }

    /* Build template map */
    maps := &map[string]string{
        "first": strconv.Itoa(q.First),
        "offset": strconv.Itoa(q.Offset),
        "rootnameid": rootnameid,
        "nameids": nameidsString,
        "tensionFilter" : tensionFilter,
        "authorsFilter": authorsFilter,
        "labelsFilter": labelsFilter,
        "order": sortFilter,
        // Protected
        "rootnameidProtected": rootnameidProtected,
        "nameidsProtected": nameidsProtectedString,
        "username": q.Username,
    }

    return maps, nil
}

