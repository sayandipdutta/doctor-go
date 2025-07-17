package main

import (
    "archive/zip"
    "flag"
    "fmt"
    "io"
    "iter"
    "os"
    "path/filepath"
    "strings"
    "sync"
)

const (
    INDEXED_FOLDER = "QC"
    SCANNED_FOLDER = "Scan"
)

var wg sync.WaitGroup

func main() {
    var sourcePath, destPath, taskType string

    flag.StringVar(&sourcePath, "source", "", "Source path")
    flag.StringVar(&destPath, "dest", "", "Destination path")
    flag.StringVar(&taskType, "task", "doctype", "Type of task. Choices: doctype, topsheet")
    stats := flag.Bool("stats", false, "If given, print doctype distribution and exit")
    withIndex := flag.Bool("withindex", false, "If given, take files from QC, else from Scan")
    withBatch := flag.Bool("withbatch", false, "If given, copy deeds under their respective batch names")
    shouldZip := flag.Bool("zip", false, "If given, create zip archive from the output")
    flag.Parse()

    if sourcePath == "" {
        panic("Invalid source path")
    }

    if *stats {
        ComputeDistribution(sourcePath)
        return
    }

    if destPath == "" {
        panic("Invalid dest path")
    }
	if err := os.Mkdir(filepath.Clean(destPath), 0o777); err != nil {
		if _, ok := err.(*os.PathError); !ok {
			panic(err)
		}
	}
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		fmt.Println("Source path does not exist:", sourcePath)
		return
	}

	var taskFn func(string, string, bool, bool) error
	switch taskType {
	case "doctype":
		taskFn = CopyStartingDoctypesPerDeed
	case "topsheet":
		taskFn = CopyTopsheetPerDeed
	default:
		{
			fmt.Println("unknown taskType:", taskType)
			return
		}
	}
	println()
	i := 1
	for deedPath := range iterDeeds(sourcePath) {
		wg.Add(1)
		go taskFn(deedPath, destPath, *withIndex, *withBatch)
		fmt.Printf("\r%d", i)
		i++
	}
	println()
	wg.Wait()

    if *shouldZip {
        zipDestPath := filepath.Dir(destPath)
        err := zipDoctypes(destPath, zipDestPath)
        if err != nil {
            fmt.Printf("Could not zip %s: %v\n", destPath, err)
            return
        }
    }
	println("Done!")
}

func isDeed(path string) bool {
    children, err := os.ReadDir(path)
    if err != nil {
        return false
    }
    for _, child := range children {
        if child.IsDir() && (strings.Contains(child.Name(), INDEXED_FOLDER)) {
            return true
        }
    }
    return false
}

func iterDeeds(rootPath string) iter.Seq[string] {
    return func(yield func(string) bool) {
        children, err := os.ReadDir(rootPath)
        if err != nil {
            return
        }
        for _, child := range children {
            childPath := filepath.Join(rootPath, child.Name())
            if isDeed(childPath) {
                if !yield(childPath) {
                    return
                }
            } else {
                if !IsDirectory(childPath) { continue }
                for deedPath := range iterDeeds(childPath) {
                    if !yield(deedPath) {
                        return
                    }
                }
            }
        }
    }
}

func CopyStartingDoctypesPerDeed(deedPath string, dest string, withIndex bool, withBatch bool) error {
	defer wg.Done()
	doctypes, err := getDoctypes(deedPath)
	if err != nil {
		return fmt.Errorf("could not get doctype %w", err)
	}
	var sourcePath, destPath string
	if withBatch {
		dest = filepath.Join(dest, filepath.Base(filepath.Dir(deedPath)))
		if _, err := os.Stat(dest); err != nil {
			os.Mkdir(dest, 0o777)
		}
	}
	for _, doctype := range doctypes {
		if withIndex {
			sourcePath = filepath.Join(deedPath, INDEXED_FOLDER, doctype.IndexedName())
			destPath = filepath.Join(dest, doctype.IndexedName())
		} else {
			sourcePath = filepath.Join(deedPath, SCANNED_FOLDER, doctype.Name())
			destPath = filepath.Join(dest, doctype.Name())
		}
        err = CopyFile(sourcePath, destPath)
        if err != nil {
            fmt.Printf("Could not copy %s to %s", sourcePath, destPath)
        }
	}
	return nil
}

func CopyTopsheetPerDeed(deedPath string, dest string, withIndex bool, withBatch bool) error {
	defer wg.Done()
	if withBatch {
		dest = filepath.Join(dest, filepath.Base(filepath.Dir(deedPath)))
		if _, err := os.Stat(dest); err != nil {
			os.Mkdir(dest, 0o777)
		}
	}
	entries, err := os.ReadDir(filepath.Join(deedPath, INDEXED_FOLDER))
	if err != nil {
		return fmt.Errorf("could not read directory %s: %w", deedPath, err)
	}
	topsheetName := ""
	for _, entry := range entries {
		desirableSuffix := fmt.Sprintf("-Others%s", filepath.Ext(entry.Name()))
		undesirableSuffix := fmt.Sprintf("_A-Others%s", filepath.Ext(entry.Name()))
		if !strings.HasSuffix(entry.Name(), undesirableSuffix) && strings.HasSuffix(entry.Name(), desirableSuffix) {
			topsheetName = entry.Name()
		}
	}
	if topsheetName == "" {
		return fmt.Errorf("topsheet not found in path %s", deedPath)
	}
	var sourcePath string
	if withIndex {
		sourcePath = filepath.Join(deedPath, INDEXED_FOLDER, topsheetName)
	} else {
		topsheetName = strings.Replace(topsheetName, "-Others.", ".", 1)
		sourcePath = filepath.Join(deedPath, SCANNED_FOLDER, topsheetName)
	}
	destPath := filepath.Join(dest, topsheetName)
    err = CopyFile(sourcePath, destPath)
    return err
}


func ComputeDistribution(sourcePath string) {
    allDoctypes := make(map[string]int)
    ch := make(chan map[string]int)
    i := 1
    go func() {
        for deedPath := range iterDeeds(sourcePath) {
            wg.Add(1)
            go getDoctypesCount(deedPath, ch)
            fmt.Printf("\r%d", i)
            i++
        }
        println()
        wg.Wait()
        close(ch)
    }()

    for doctypeCount := range ch {
        for doctype, count := range doctypeCount {
            i, ok := allDoctypes[doctype]
            if !ok {
                i = 0
            }
            allDoctypes[doctype] = i + count;
        }
    }
    fmt.Println("count: ")
    fmt.Println(PrettyFormatMap(allDoctypes))
}

func getDoctypesCount(deedPath string, ch chan<- map[string]int) {
    defer wg.Done()
    doctypeMap := make(map[string]int)
    doctypes, err := getDoctypes(deedPath); if err == nil {
        for _, doctypeInfo := range doctypes {
            i, ok := doctypeMap[doctypeInfo.doctype]
            if !ok {
                i = 0;
            }
            doctypeMap[doctypeInfo.doctype] = i + 1;
        }
    }
    ch <- doctypeMap
}

func zipDoctypes(dirToZipPath string, dest string) error {
    zipfileName := filepath.Base(dirToZipPath) + ".zip"
    zipFile, err := os.Create(filepath.Join(dest, zipfileName))
    if err != nil {
        return fmt.Errorf("Failed to create zipfile %v", err);
    }
    defer zipFile.Close()

    zipWriter := zip.NewWriter(zipFile);
    defer zipWriter.Close()

	entries, err := os.ReadDir(dirToZipPath)
    if err != nil {
        return fmt.Errorf("Could not read directory %v", err)
    }
    for _, fileName := range entries {
        fileToZip, err := os.Open(filepath.Join(dirToZipPath, fileName.Name()))
        if err != nil {
            return fmt.Errorf("Could not open file %v", err)
        }
        defer fileToZip.Close()
        writer, err := zipWriter.Create(fileName.Name())
        if err != nil {
            return fmt.Errorf("Could not create zip entry for file %v", err)
        }

        _, err = io.Copy(writer, fileToZip)
        if err != nil {
            return fmt.Errorf("Copy failed! %v", err)
        }
    }
    return nil
}
