package main

import (
	"flag"
	"fmt"
	"iter"
	"os"
	"sort"
	"strings"
)

func main() {
	batchPath := "batches/Batch_0001"
	if len(os.Args) > 1 {
		batchPath = os.Args[1]
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
