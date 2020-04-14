package tools

import (
    "fmt"
    "reflect"
    "strings"
    "regexp"
    "unicode"
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

// CleanAliasedMap copy the input map by renaming all the keys
// recursively by removing trailing integers.
// @DEBUG: Assume either all keys or no one are aliased.
func CleanAliasedMap(m map[string]interface{})  map[string]interface{} {
    
    out := make(map[string]interface{}, len(m))
    for k, v := range m {
        var nk string
        if IsDigit(k[len(k)-1]) {
            endDigits := regexp.MustCompile(`[0-9]+$`)
            nk = endDigits.ReplaceAllString(k, "")
        } else {
            panic(fmt.Errorf("map should be aliased."))
        }

        var nv interface{}
        switch t := v.(type) {
        case map[string]interface{}:
            nv = CleanAliasedMap(t)
        default:
            nv = t
        }
        out[nk] =  nv
    }

    return out

}

func IsDigit(s byte) bool {
    return unicode.IsDigit(rune(s))
}
