package graph

import (
    "bytes"
    "text/template"

	"zerogov/fractal6.go/tools"
)

type JsonAtom = map[string]interface{}

type GqlRes struct {
	Data   JsonAtom `json:"data"`
	Errors []JsonAtom  `json:"errors"` // message, locations, path, extensions
}

type Query struct {
    Data string
    Template *template.Template
}

// Init clean the query to be compatible in application/json format.
func (q *Query) Init() {
    d := q.Data

    q.Data = tools.CleanString(d, false)

    // Load the template
    // @DEBUG: Do we need a template name ?
    q.Template = template.Must(template.New("graphql").Parse(q.Data))
}

func (q Query) Format(m JsonAtom) string {
    buf := bytes.Buffer{}
    q.Template.Execute(&buf, m)
    return buf.String()
}
