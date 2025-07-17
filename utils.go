package main

import (
    "io"
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

func CopyFile(sourcePath string, destPath string) error {
    if reader, err := os.Open(sourcePath); err == nil {
        defer reader.Close()
        if writer, err := os.Create(destPath); err == nil {
            defer writer.Close()
            if _, err := io.Copy(writer, reader); err != nil {
                return fmt.Errorf("could not perform copy operation %w", err)
            }
        } else {
            return fmt.Errorf("could not create dest path %s -> %w", destPath, err)
        }
    } else {
        return fmt.Errorf("could not open source path: %s -> %w", sourcePath, err)
    }
    return nil
}
