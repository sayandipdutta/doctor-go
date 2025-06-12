package main

import (
	"flag"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var wg sync.WaitGroup

func main() {
	var batchPath, destPath string
	if len(os.Args) > 2 {
		batchPath = os.Args[1]
		destPath = os.Args[2]
	} else {
		panic("Not enought arguments")
	}

	if err := os.Mkdir(filepath.Clean(destPath), 0o777); err != nil {
		if _, ok := err.(*os.PathError); !ok {
			panic(err)
		}
	}

	withIndex := flag.Bool("withindex", true, "If given, take files from QC, else from Scan")
	withBatch := flag.Bool("withbatch", true, "If given, copy deeds under their respective batch names")
	flag.Parse()

	println()
	i := 1
	for deedPath := range iterDeeds(batchPath) {
		wg.Add(1)
		go CopyStartingDoctypesPerDeed(deedPath, destPath, *withIndex, *withBatch)
		fmt.Printf("\r%d", i)
		i++
	}
	println()
	wg.Wait()
}

type DoctypeInfo struct {
	prefix  string
	doctype string
	suffix  string
}

func (entry *DoctypeInfo) Name() string {
	var sb strings.Builder
	sb.WriteString(entry.prefix)
	sb.WriteString(".")
	sb.WriteString(entry.suffix)
	return sb.String()
}

func (entry *DoctypeInfo) IndexedName() string {
	var sb strings.Builder
	sb.WriteString(entry.prefix)
	sb.WriteString("-")
	sb.WriteString(entry.doctype)
	sb.WriteString(".")
	sb.WriteString(entry.suffix)
	return sb.String()
}

func isDeed(path string) bool {
	children, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, child := range children {
		if child.IsDir() && (strings.Contains(child.Name(), "QC")) {
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
			childPath := rootPath + "/" + child.Name()
			if isDeed(childPath) {
				if !yield(childPath) {
					return
				}
			} else {
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
		return err
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
			sourcePath = strings.Join([]string{deedPath, "QC", doctype.IndexedName()}, "/")
			destPath = strings.Join([]string{dest, doctype.IndexedName()}, "/")
		} else {
			sourcePath = strings.Join([]string{deedPath, "Scan", doctype.Name()}, "/")
			destPath = strings.Join([]string{dest, doctype.Name()}, "/")
		}
		if reader, err := os.Open(sourcePath); err == nil {
			defer reader.Close()
			if writer, err := os.Create(destPath); err == nil {
				defer writer.Close()
				if _, err := io.Copy(writer, reader); err != nil {
					return err
				}
			} else {
				panic(err)
			}
		} else {
			return err
		}
	}
	return nil
}

func getDoctypes(path string) ([]DoctypeInfo, error) {
	files := []string{}

	entries, err := os.ReadDir(path + "/QC")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading directory:", err.Error())
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, entry.Name())
	}

	sort.Strings(files)

	doctypes := []DoctypeInfo{}
	last_doctype := ""

	for _, file := range files {
		if prefix, after, found := strings.Cut(file, "-"); found {
			if doctype, suffix, found := strings.Cut(after, "."); found && doctype != last_doctype {
				doctypes = append(doctypes, DoctypeInfo{prefix, doctype, suffix})
				last_doctype = doctype
			}
		}
	}

	return doctypes, nil
}
