package tools

import (
    "os"
    "testing"
	"github.com/spf13/viper"
)

var matrixPostalRoom string
var matrixToken string

func init() {
    if err := os.Chdir("../"); err != nil {
        panic(err)
    }
    InitViper()
    matrixPostalRoom = viper.GetString("emailing.matrix_postal_room")
    matrixToken = viper.GetString("emailing.matrix_token")
}

func TestMatrixJsonSend(t *testing.T) {
    body := `"Hi! webhook test for fractale.co"`
    err := MatrixJsonSend(string(body), matrixPostalRoom, matrixToken)
    if err != nil {
        t.Errorf("MatrixJsonSend failed: %s", err.Error())
    }
}
