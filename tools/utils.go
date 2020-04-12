package tools

import (
    "reflect"
    "strings"
    "regexp"
)

func CleanString(data string) string {
    var d string = data
    d = strings.Replace(d, `\n`, "", -1)
    d = strings.Replace(d, "\n", "", -1)
    space := regexp.MustCompile(`\s+`)
    d = space.ReplaceAllString(d, " ")
    return d
}

func ToGoNameFormat(name string) string {
    var l  []string
    for _, s := range strings.Split(name, "_") {
        l = append(l, strings.Title(s))
    }
    goName := strings.Join(l, "")
    return goName
}

func ToTypeName(name string) string {
    l := strings.Split(name, ".")
    typeName := l[len(l)-1]
    return typeName 
}


/*
This function will help you to convert your object from struct to map[string]interface{} based on your JSON tag in your structs.
Example how to use posted in sample_test.go file.
*/
func StructToMap(item interface{}) map[string]interface{} {

    res := map[string]interface{}{}
    if item == nil {
        return res
    }
    v := reflect.TypeOf(item)
    reflectValue := reflect.ValueOf(item)
    reflectValue = reflect.Indirect(reflectValue)

    if v.Kind() == reflect.Ptr {
        v = v.Elem()
    }
    for i := 0; i < v.NumField(); i++ {
        tag := v.Field(i).Tag.Get("json")
        field := reflectValue.Field(i).Interface()
        if tag != "" && tag != "-" {
            if v.Field(i).Type.Kind() == reflect.Struct {
                res[tag] = StructToMap(field)
            } else {
                res[tag] = field
            }
        }
    }
    return res
}
