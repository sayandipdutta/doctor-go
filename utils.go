package main

import (
    "os"
    "strings"
    "fmt"
)

func IsDirectory(path string) bool {
    fileinfo, err := os.Stat(path)
    if err != nil { return false }
    return fileinfo.IsDir()
}


func PrettyFormatMap(m map[string]int) string {
    var sb strings.Builder;

    for k, v := range m {
        s := fmt.Sprintf("%s: %d\n", k, v)
        sb.WriteString(s)
    }
    return sb.String()
}
