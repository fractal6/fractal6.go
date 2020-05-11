package tools

import (
    "fmt"
    "log"
)

func LogErr(module string, reason string, err error) error {
    log.Printf("[%s] %s: %s", module, reason, err.Error())
    return fmt.Errorf("%s: %s", reason, err.Error())
}
