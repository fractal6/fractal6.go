package gql

import (
    "regexp"
    "strings"
    "bytes"
    "text/template"
)

type JsonAtom map[string]interface{}

type Query struct {
    Data string
    Template *template.Template
}

// Init clean the query to be compatible in application/json format.
func (q *Query) Init() {
    d := q.Data

    // Clean the query 
    d = strings.Replace(d, `\n`, "", -1)
    d = strings.Replace(d, "\n", "", -1)
    space := regexp.MustCompile(`\s+`)
    d = space.ReplaceAllString(d, " ")
    q.Data = d

    // Load the template
    // @DEBUG: Do we need a template name ?
    q.Template = template.Must(template.New("a").Parse(q.Data))
}

func (q Query) Format(m JsonAtom) string {
    buf := bytes.Buffer{}
    q.Template.Execute(&buf, m)
    return buf.String()
}
