package db

import (
    "fmt"
    "net/http"
    "bytes"
    "context"
    "log"
    "strings"
    "encoding/json"
    "text/template"
    //"io/ioutil"
    "github.com/spf13/viper"
	"google.golang.org/grpc"
    "github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"

	"zerogov/fractal6.go/tools"
)

// Dgraph graphql client from scratch
type Dgraph struct {
    gqlUrl string
    grpcUrl string
    countTemplate *template.Template
}


type RespCount struct {
	All []map[string]int `json:"all"`
}

type RespGet struct {
	All []map[string]interface{} `json:"all"`
}

type CancelFunc func()


// Database client
var DB *Dgraph

// Initialize
func init () {
    DB = initDB()
}


func GetDB() *Dgraph {
    return DB
}

func initDB() *Dgraph {
    tools.InitViper()

    HOSTDB := viper.GetString("db.host")
    PORTDB := viper.GetString("db.port_graphql")
    PORTGRPC := viper.GetString("db.port_grpc")
    APIDB := viper.GetString("db.api")
    dgraphApiUrl := "http://"+HOSTDB+":"+PORTDB+"/"+APIDB
    grpcUrl := HOSTDB+":"+PORTGRPC

    if HOSTDB == "" {
        panic("viper error: not host found.")
    } else {
        fmt.Println("Dgraph Graphql addr:", dgraphApiUrl)
        fmt.Println("Dgraph Grpc addr:", grpcUrl)
    }

	countQ := `{
        all(func: uid({{.id}})) {
            count({{.field}})
        }
    }`

    return &Dgraph{
        gqlUrl: dgraphApiUrl,
        grpcUrl: grpcUrl,
        countTemplate: template.Must(template.New("dgraph").Parse(countQ)),
    }
}

/*
*
* Public methods
*
*/


// Post send a post request to the Graphql client.
func (d Dgraph) Post(data []byte, res interface{}) error {
    fmt.Println(d.gqlUrl)
    fmt.Println(d.grpcUrl)
    req, err := http.NewRequest("POST", d.gqlUrl, bytes.NewBuffer(data))
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Get the string/byte response
    //res, _ := ioutil.ReadAll(resp.Body)
    return json.NewDecoder(resp.Body).Decode(res)
}

func (d Dgraph) GetUserGql(fieldid string, userid string, user interface{}) error {

    // Format Query
     var _q string
     if fieldid == "username" {
         _q = `{
             "query":  getUser({{.fieldid}}: "{{.userid}}") {
                 User.username
                 User.password
                 User.roles {
                     Node.nameid
                     Node.role_type
                 }
             }
         }`
     } else if fieldid == "email" {
         return fmt.Errorf("GetUser by email not implemented, user username instead.")
     } else {
         return fmt.Errorf("User fieldid unknown %s", fieldid)
     }

    buf := bytes.Buffer{}
    qTemplate := template.Must(template.New("dgraph").Parse(_q))
    qTemplate.Execute(&buf, map[string]string{
        "fieldid":fieldid, 
        "userid":userid})
    q := buf.String()

    // Send Request
    err := d.Post([]byte(q), user)
    return err
}

func (d Dgraph) GetUser(fieldid string, userid string, user interface{}) error {
    // init client
	dg, cancel := d.getDgraphClient()
    defer cancel()
    ctx := context.Background()
    txn := dg.NewTxn()
    defer txn.Discard(ctx)

    // Format Query
    _q := `{
        all(func: eq(User.{{.fieldid}}, {{.userid}})) {
            User.username
            User.name
            User.password
            User.roles {
                Node.nameid
                Node.role_type
           }
        }
    }`

    buf := bytes.Buffer{}
    qTemplate := template.Must(template.New("dgraph").Parse(_q))
    qTemplate.Execute(&buf, map[string]string{
        "fieldid":fieldid, 
        "userid":userid})
    q := buf.String()

    // Send Request
	res, err := txn.Query(ctx, q)
	if err != nil {
        return err
	}

    // Decode response
    var r RespGet
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
        return err
	}

    if len(r.All) > 1 {
        return fmt.Errorf("Got multiple user with same @id: %s, %s", fieldid, userid)
    } else if len(r.All) == 1 {
        rRaw, err := json.Marshal(r.All[0])
        if err != nil {
            return err
        }
        json.Unmarshal(rRaw, user)
    }
    return nil
}

// Count count the number of object in fieldName attribute for given type and id,
// by using the gprc/Grapql+- client.
// Returns: int or -1 if nothing is found.
func (d Dgraph) Count(id string, typeName string, fieldName string) int {
    // init client
	dg, cancel := d.getDgraphClient()
    defer cancel()
    ctx := context.Background()
    txn := dg.NewTxn()
    defer txn.Discard(ctx)

    // Format Query
    field := strings.Join([]string{typeName, fieldName}, ".")

    buf := bytes.Buffer{}
    d.countTemplate.Execute(&buf, map[string]string{"id":id, "field":field})
    q := buf.String()

    // Send Request
	res, err := txn.Query(ctx, q)
	if err != nil {
        panic(err)
	}

    // Decode response
	var r RespCount
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		panic(err)
	}

    // Extract result
    
    if len(r.All) == 0 {
        return -1
    }

	values := make([]int, 0, len(r.All[0]))
	for _, v := range r.All[0] {
		values = append(values, v)
	}
    return values[0]
}

/*
*
* private methods
*
*/

func (d Dgraph) getDgraphClient() (*dgo.Dgraph, CancelFunc) {
	conn, err := grpc.Dial(d.grpcUrl, grpc.WithInsecure())
	if err != nil {
		log.Fatal("While trying to dial gRPC")
	}

	dc := api.NewDgraphClient(conn)
	dg := dgo.NewDgraphClient(dc)
	//ctx := context.Background()

	//// Perform login call. If the Dgraph cluster does not have ACL and
	//// enterprise features enabled, this call should be skipped.
	//for {
	//	// Keep retrying until we succeed or receive a non-retriable error.
	//	err = dg.Login(ctx, "groot", "password")
	//	if err == nil || !strings.Contains(err.Error(), "Please retry") {
	//		break
	//	}
	//	time.Sleep(time.Second)
	//}
	//if err != nil {
	//	log.Fatalf("While trying to login %v", err.Error())
	//}

	return dg, func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error while closing connection:%v", err)
		}
	}
}

