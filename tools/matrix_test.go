package tools

import (
    "fmt"
    "os"
    "testing"
	"github.com/spf13/viper"
)

var matrixPostalRoom string
var matrixToken string
var DOMAIN string

func init() {
    if err := os.Chdir("../"); err != nil {
        panic(err)
    }
    InitViper()
    matrixPostalRoom = viper.GetString("mailer.matrix_postal_room")
    matrixToken = viper.GetString("mailer.matrix_token")
    DOMAIN = viper.GetString("server.domain")
}

func TestMatrixJsonSend(t *testing.T) {
    body := fmt.Sprintf(`"Hi! webhook test for %s"`, DOMAIN)
    err := MatrixJsonSend(string(body), matrixPostalRoom, matrixToken)
    if err != nil {
        t.Errorf("MatrixJsonSend failed: %s", err.Error())
    }
}
