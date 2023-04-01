package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/fatih/camelcase"
	"golang.org/x/tools/go/packages"
)

const srcHeader = `// This is generated code. DO NOT EDIT
package main

import (
	"golang.org/x/exp/shiny/materialdesign/icons"
)

const numEntries = %d

var allEntries = [%d]iconEntry{
`

func readAndSortNames() ([]string, error) {
	names := make([]string, 0, 1000)
	cfg := packages.Config{
		Mode: packages.NeedTypes | packages.NeedSyntax,
	}
	pkgs, err := packages.Load(&cfg, "golang.org/x/exp/shiny/materialdesign/icons")
	if err != nil {
		return nil, fmt.Errorf("loading icons package: %w", err)
	}
	iconsPkg := pkgs[0]
	for _, f := range iconsPkg.Syntax {
		for _, obj := range f.Scope.Objects {
			names = append(names, obj.Name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func main() {
	names, err := readAndSortNames()
	if err != nil {
		log.Fatalf("reading and sorting icon names: %v", err)
	}

	out, err := os.OpenFile("./data.go", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		log.Fatalf("opening out file: %v", err)
	}
	defer out.Close()

	count := len(names)
	if _, err = fmt.Fprintf(out, srcHeader, count, count); err != nil {
		log.Fatalf("writing source header: %v", err)
	}
	for _, name := range names {
		nameWithSpaces := strings.Join(camelcase.Split(name), " ")
		fmt.Fprintf(out, "\t{%q, %q, %q, mi(icons.%s)},\n", nameWithSpaces, name, strings.ToLower(name), name)
	}
	if _, err = out.WriteString("}\n"); err != nil {
		log.Fatalf("writing last curly bracket: %v", err)
	}
}
