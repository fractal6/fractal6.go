package tools

import (
    "net/http"
    "bytes"
    "context"
    "log"
    "strings"
    "encoding/json"
    "github.com/spf13/viper"
    "text/template"
    //"io/ioutil"
    
    "github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"
	"google.golang.org/grpc"
)

// Dgraph graphql client from scratch
type Dgraph struct {
    gqlUrl string
    grpcUrl string
    countTmp *template.Template
}


type Resp struct {
	All []map[string]int `json:"all"`
}

type CancelFunc func()


func InitDB() Dgraph {
    HOSTDB := viper.GetString("db.host")
    PORTDB := viper.GetString("db.port_gql")
    PORTGRPC := viper.GetString("db.port_grpc")
    APIDB := viper.GetString("db.api")
    dgraphApiUrl := "http://"+HOSTDB+":"+PORTDB+"/"+APIDB
    grpcUrl := HOSTDB+":"+PORTGRPC

	countQ := `{
        all(func: uid({{.id}})) {
            count({{.field}})
        }
    }`

    return Dgraph{
        gqlUrl: dgraphApiUrl,
        grpcUrl: grpcUrl,
        countTmp: template.Must(template.New("dgraph").Parse(countQ)),
    }
}

/*
*
* Public methods
*
*/


func (d Dgraph) Request(data []byte, res interface{}) error {
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

// Count the number of object in fieldName attribute for given type and id.
// Returns -1 is nothing is found.
func (d Dgraph) Count(id string, typeName string, fieldName string) (int) {
	dg, cancel := d.getDgraphClient()
    defer cancel()
    ctx := context.Background()
    txn := dg.NewTxn()
    defer txn.Discard(ctx)

    // Format Query
    field := strings.Join([]string{typeName, fieldName}, ".")
    buf := bytes.Buffer{}
    d.countTmp.Execute(&buf, map[string]string{"id":id, "field":field})
    q := buf.String()

	res, err := txn.Query(ctx, q)
	if err != nil {
        panic(err)
	}

	var r Resp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		panic(err)
	}

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

