package tools

import (
	"testing"
	"reflect"
	"fractale/fractal6.go/graph/model"
)

func TestStructMap(t *testing.T) {

    var nodeFragment *model.NodeFragment
    var nodeInput model.AddNodeInput

    name := "name"
    nameid := "nameid"
    username := "username"
    nodeFragment = &model.NodeFragment{
        Name: &name,
        Nameid: &nameid,
        FirstLink: &username,
    }

    StructMap(nodeFragment, &nodeInput)

    // FirstLink cannot be added by adding a node !
    want := model.AddNodeInput{
        Name: name,
        Nameid: nameid,
    }

    if reflect.DeepEqual(nodeInput, want) {
        t.Errorf("StructMap error, want: %v, got: %v", want, nodeInput)
    }
}

