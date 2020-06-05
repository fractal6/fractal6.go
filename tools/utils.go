package tools

import (
    "reflect"
    "regexp"
    "strings"
    "github.com/spf13/viper"
)

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
// Example how to use posted in sample_test.go file.
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
        out[nk] =  nv
    }

    return out

}

// Keep the last key string when separated by dot (eg [a.key.name: 10] -> [name: 10])
func CleanCompositeName(m map[string]interface{})  map[string]interface{} {
    
    out := make(map[string]interface{}, len(m))
    for k, v := range m {
        ks := strings.Split(k, ".")
        nk := ks[len(ks) -1]

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

