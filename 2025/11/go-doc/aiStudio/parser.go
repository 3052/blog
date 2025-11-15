package doc

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Parse parses the Go package in the given directory and returns a PackageDoc.
func Parse(dir string) (*PackageDoc, error) {
	fset := token.NewFileSet()

	// Manually parse files to avoid using parser.ParseDir, which returns
	// the deprecated ast.Package type.
	files, err := parseGoFiles(fset, dir)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no Go source files found in directory: %s", dir)
	}

	// The final argument for a mode is now removed. The default behavior
	// is to include only exported declarations, which is what is required.
	p, err := doc.NewFromFiles(fset, files, "./")
	if err != nil {
		return nil, fmt.Errorf("failed to create doc package: %w", err)
	}

	pkgDoc := &PackageDoc{
		Name: p.Name,
		Doc:  p.Doc,
	}

	for _, f := range p.Funcs {
		// p.Funcs only contains exported, package-level functions.
		if f.Recv == "" {
			pkgDoc.Functions = append(pkgDoc.Functions, FuncDoc{
				Name: f.Name,
				Doc:  f.Doc,
			})
		}
	}

	for _, t := range p.Types {
		// p.Types only contains exported types.
		typeDoc := TypeDoc{
			Name: t.Name,
			Doc:  t.Doc,
		}
		for _, spec := range t.Decl.Specs {
			if typeSpec, ok := spec.(*ast.TypeSpec); ok {
				typeSpecDoc := TypeSpecDoc{
					Name: typeSpec.Name.Name,
					Doc:  typeSpec.Doc.Text(),
				}
				typeDoc.Specs = append(typeDoc.Specs, typeSpecDoc)
			}
		}
		pkgDoc.Types = append(pkgDoc.Types, typeDoc)
	}

	for _, v := range p.Consts {
		// p.Consts only contains exported constants.
		pkgDoc.Constants = append(pkgDoc.Constants, VarDoc{
			Names: v.Names,
			Doc:   v.Doc,
		})
	}

	for _, v := range p.Vars {
		// p.Vars only contains exported variables.
		pkgDoc.Variables = append(pkgDoc.Variables, VarDoc{
			Names: v.Names,
			Doc:   v.Doc,
		})
	}

	return pkgDoc, nil
}

// parseGoFiles reads a directory, parses all non-test .go files,
// and returns them as a slice of *ast.File. It ensures all
// files belong to the same package.
func parseGoFiles(fset *token.FileSet, dir string) ([]*ast.File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []*ast.File
	var packageName string

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}

		// All files must belong to the same package for doc.NewFromFiles to work.
		if packageName == "" {
			packageName = file.Name.Name
		} else if file.Name.Name != packageName {
			return nil, fmt.Errorf("multiple package names found in directory: %s and %s", packageName, file.Name.Name)
		}

		files = append(files, file)
	}
	return files, nil
}
