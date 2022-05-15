package tools

import (
    "fmt"
    "log"
    "runtime"
)

func LogErr(reason string, err error) error {
    // Get trace information
    pc := make([]uintptr, 10)  // at least 1 entry needed
    runtime.Callers(2, pc) // Skip 2 levels to get the caller
    f := runtime.FuncForPC(pc[0])
    fname := f.Name()
    //file, line := f.FileLine(pc[0])

    log.Printf("[@%s] %s: %s", fname, reason, err.Error())
    return fmt.Errorf("%s: %s", reason, err.Error())
}
