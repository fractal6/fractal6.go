package tools

import (
    //"fmt"
    "time"
    "reflect"
    "regexp"
    "strings"
    "encoding/json"
    "github.com/spf13/viper"
)

//Now returns the current time formated with RFC3339
func Now() string {
    return time.Now().UTC().Format(time.RFC3339)
}

//IsOlder returns trus if d2 is older than d1.
func IsOlder(d1, d2 string) bool {
    date1, _ := time.Parse(time.RFC3339, d1)
    date2, _ := time.Parse(time.RFC3339, d2)
    return date1.Before(date2)
}

//InitViper Read the config file
func InitViper() {
	viper.AddConfigPath("./")
	viper.SetConfigName("config") // name of config file (without extension)
	//viper.AutomaticEnv() // read in environment variables that match
    if err := viper.ReadInConfig(); err != nil {
        // Panic on config reading error
        panic(err)
    }
}

// StructMap convert/copy a interface to another
func StructMap(in interface{}, out interface{}) {
    raw, _ := json.Marshal(in)
    json.Unmarshal(raw, &out)
}

//StructToMap convert a struct to a map[string]interface{} based on your JSON tag in your structs.
// Use mapstructure instead ?
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

// Use Marshaling and Unmarshaling to convert an object to a map of string.
// @DEBUG: mapstructure seems to not decode enum (RoleType) !
// @see also: https://stackoverflow.com/questions/23589564/function-for-converting-a-struct-to-map-in-golang#25117810
func Struct2Map(item interface{}) map[string]interface{} {
    var amap map[string]interface{}
    itemRaw, _ := json.Marshal(item)
    json.Unmarshal(itemRaw, &amap)
    return amap
}

//MarshalWithoutNil marshal an struct but removed all empty (null) edges.
func MarshalWithoutNil(item interface{}) ([]byte, error) {
    m := CleanNilMap(Struct2Map(item))
    return json.Marshal(m)
}

//CleanNilMap remove all empty (nil) edges recursively.
func CleanNilMap(m map[string]interface{}) map[string]interface{} {
    out := make(map[string]interface{}, len(m))
    for k, v := range m {
        if v == nil { continue }

        var nv interface{}
        switch t := v.(type) {
        case map[string]interface{}:
            nv = CleanNilMap(t)
        default:
            nv = t
        }
        out[k] = nv
    }

    return out
}

// CleanAliasedMap copy the input map by renaming all the keys
// recursively by removing trailing integers.
// @DEBUG: how to better handle aliasing (check gqlgen)
func CleanAliasedMap(m map[string]interface{}) map[string]interface{} {
    out := make(map[string]interface{}, len(m))
    for k, v := range m {
        var nk string
        if IsDigit(k[len(k)-1]) {
            endDigits := regexp.MustCompile(`[0-9]+$`)
            nk = endDigits.ReplaceAllString(k, "")
        } else {
            nk = k
        }

        var nv interface{}
        switch t := v.(type) {
        case map[string]interface{}:
            nv = CleanAliasedMap(t)
        default:
            nv = t
        }
        out[nk] = nv
    }

    return out
}

// Keep the last key string when separated by dot (eg [a.key.name: 10] -> [name: 10])
// + replace uid field with ID (dgraph to gqlgen compatibility
func CleanCompositeName(m map[string]interface{}) map[string]interface{} {
    out := make(map[string]interface{}, len(m))
    for k, v := range m {
        ks := strings.Split(k, ".")
        nk := ks[len(ks) -1]

        if nk == "uid" {
          nk = "id"
        }

        var nv interface{}
        switch t := v.(type) {
        case map[string]interface{}:
            nv = CleanAliasedMap(t)
        default:
            nv = t
        }
        out[nk] = nv
    }

    return out
}

//InterfaceSlice tries to convert an interface to a Slice of interface.
// see stackoverflow.com/a/12754757/4223749
func InterfaceSlice(slice interface{}) []interface{} {
    s := reflect.ValueOf(slice)
    if s.Kind() != reflect.Slice {
        panic("InterfaceSlice() given a non-slice type")
    }

    // Keep the distinction between nil and empty slice input
    if s.IsNil() {
        return nil
    }

    ret := make([]interface{}, s.Len())

    for i:=0; i<s.Len(); i++ {
        ret[i] = s.Index(i).Interface()
    }

    return ret
}
