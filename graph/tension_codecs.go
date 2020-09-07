package graph

import (
    "fmt"

    "zerogov/fractal6.go/graph/model"
)

// Action Type enum
type actionType string
const (
    NewAction actionType = "new"
    EditAction actionType = "edit"
    ArchiveAction actionType = "archive"
)

// Doc Type enum
type docType string
const (
    NodeDoc docType = "node"
    MdDoc docType = "md"
)

// Tension Action information
type TensionCharac struct {
    ActionType actionType
    DocType docType
}

// Create new TensionCharac from a TensionAction type.
func (TensionCharac) New(action model.TensionAction) (*TensionCharac, error) {
    var l []string
    var err error

    switch action {
    case model.TensionActionNewRole:
        l = append(l, "new", "node")
    case model.TensionActionNewCircle:
        l = append(l, "new", "node")
    case model.TensionActionNewMd:
        l = append(l, "new", "md")
    case model.TensionActionEditRole:
        l = append(l, "edit", "node")
    case model.TensionActionEditCircle:
        l = append(l, "edit", "node")
    case model.TensionActionEditMd:
        l = append(l, "edit", "md")
    default:
        err = fmt.Errorf("Tension Action type unknown: " + string(action))
    }

    tc := &TensionCharac{
        ActionType: actionType(l[0]),
        DocType: docType(l[1]),
    }
    return tc, err
}

func (tc TensionCharac) EditAction(t *model.NodeType) (model.TensionAction) {
    var a model.TensionAction
    switch tc.DocType {
    case MdDoc:
        a = model.TensionActionEditMd
    case NodeDoc:
        switch *t {
        case model.NodeTypeRole:
            a = model.TensionActionEditRole
        case model.NodeTypeCircle:
            a = model.TensionActionEditCircle
        }
    }
    return a
}
