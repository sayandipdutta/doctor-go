package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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

func getDoctypes(path string) ([]DoctypeInfo, error) {
	files := []string{}

	entries, err := os.ReadDir(filepath.Join(path, "/QC"))
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
		if prefix, rest, found := strings.Cut(file, "-"); found {
			if doctype, suffix, found := strings.Cut(rest, "."); found && doctype != last_doctype {
				doctypes = append(doctypes, DoctypeInfo{prefix, doctype, suffix})
				last_doctype = doctype
			}
		}
	}

	return doctypes, nil
}
