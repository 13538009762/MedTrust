package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type DeclItem struct {
	Pkg  string
	Kind string
	Name string
}

func main() {
	rootDir := "."
	if len(os.Args) > 1 {
		rootDir = os.Args[1]
	}

	fset := token.NewFileSet()
	var items []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			base := info.Name()
			if base == "vendor" || base == ".git" || base == "tools" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		node, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}

		pkgName := node.Name.Name
		for _, decl := range node.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				recv := ""
				if d.Recv != nil && len(d.Recv.List) > 0 {
					switch t := d.Recv.List[0].Type.(type) {
					case *ast.StarExpr:
						if ident, ok := t.X.(*ast.Ident); ok {
							recv = "(*" + ident.Name + ")."
						}
					case *ast.Ident:
						recv = t.Name + "."
					}
				}
				items = append(items, fmt.Sprintf("%s: FUNC %s%s", pkgName, recv, d.Name.Name))
			case *ast.GenDecl:
				kind := d.Tok.String() // type, const, var
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						items = append(items, fmt.Sprintf("%s: TYPE %s", pkgName, s.Name.Name))
					case *ast.ValueSpec:
						for _, name := range s.Names {
							items = append(items, fmt.Sprintf("%s: %s %s", pkgName, strings.ToUpper(kind), name.Name))
						}
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Walk error: %v\n", err)
		os.Exit(1)
	}

	sort.Strings(items)
	for _, it := range items {
		fmt.Println(it)
	}
}
