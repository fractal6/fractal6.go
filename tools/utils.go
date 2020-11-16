package tools

import (
    "time"
    "reflect"
    "regexp"
    "strings"
    "encoding/json"
    "github.com/spf13/viper"
)

func Now() string {
    return time.Now().UTC().Format(time.RFC3339)
}

func IsOlder(d1, d2 string) bool {
    date1, _ := time.Parse(time.RFC3339, d1)
    date2, _ := time.Parse(time.RFC3339, d2)
    return  date1.Before(date2)
}

//
// Read Config file
//
func InitViper() {
	configName := "config"
	viper.AddConfigPath("./")
	viper.SetConfigName(configName) // name of config file (without extension)
	//viper.AutomaticEnv() // read in environment variables that match
    if err := viper.ReadInConfig(); err != nil {
        // Panic on config reading error
        panic(err)
    }

}


// This function will help you to convert your object from struct to map[string]interface{} based on your JSON tag in your structs.
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

// StructMap convert/copy a interface to another
func StructMap(in interface{}, out interface{}) {
    raw, _ := json.Marshal(in)
    json.Unmarshal(raw, &out)
}


// CleanAliasedMap copy the input map by renaming all the keys
// recursively by removing trailing integers.
// @DEBUG: how to better handle aliasing (check gqlgen)
func CleanAliasedMap(m map[string]interface{})  map[string]interface{} {
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

