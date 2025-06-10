package main

import (
	"flag"
	"fmt"
	"io"
	"iter"
	"os"
	"sort"
	"strings"
	"sync"
)

var wg sync.WaitGroup

func main() {
	batchPath := "batches/Batch_0001"
	destPath := "./output"
	if len(os.Args) > 2 {
		batchPath = os.Args[1]
		destPath = os.Args[2]
	} else {
		panic("Not enought arguments")
	}

	if err := os.Mkdir(destPath, 0o777); err != nil {
		if _, ok := err.(*os.PathError); !ok {
			panic(err)
		}
	}

	withIndex := flag.Bool("withindex", true, "If given, take files from QC, else from Scan")
	flag.Parse()

	doctypes := traverseBatch(batchPath)

	var parent, fullPath string

	for deedPath, doctype := range doctypes {
		if *withIndex {
			parent = "QC"
			fullPath = strings.Join([]string{deedPath, parent, doctype.IndexedName()}, "/")
		} else {
			parent = "Scan"
			fullPath = strings.Join([]string{deedPath, parent, doctype.Name()}, "/")
		}
		println(fullPath)
	}

	for i, deedPath := range iterDeeds(batchPath) {
		wg.Add(1)
		go CopyStartingDoctypesPerDeed(deedPath, destPath, *withIndex)
		println(i)
	}
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

func traverseBatch(batchPath string) iter.Seq2[string, *DoctypeInfo] {
	return func(yield func(string, *DoctypeInfo) bool) {
		if deeds, err := os.ReadDir(batchPath); err == nil {
			for _, deed := range deeds {
				if deed.IsDir() {
					if doctypes, err := getDoctypes(batchPath + "/" + deed.Name()); err == nil {
						for _, doctype := range doctypes {
							if !yield(deed.Name(), &doctype) {
								return
							}
						}
					}
				}
			}
		}
	}
}

func iterDeeds(batchPath string) iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		deeds, err := os.ReadDir(batchPath)
		if err != nil {
			return
		}
		for i, deed := range deeds {
			deedPath := batchPath + "/" + deed.Name()
			if !yield(i, deedPath) {
				return
			}
		}
	}
}

func CopyStartingDoctypesPerDeed(deedPath string, dest string, withIndex bool) error {
	defer wg.Done()
	doctypes, err := getDoctypes(deedPath)
	if err != nil {
		return err
	}
	var sourcePath, destPath string
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
				println(destPath)
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
