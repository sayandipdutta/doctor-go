package main

import (
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
