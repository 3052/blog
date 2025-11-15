package doc

import (
   "bytes"
   "fmt"
   "go/ast"
   "go/doc"
   "go/format"
   "go/parser"
   "go/token"
   "html/template"
   "os"
   "path/filepath"
   "strings"
)

// Parse parses the Go package in the given directory and returns a PackageDoc.
func Parse(dir string) (*PackageDoc, error) {
   fset := token.NewFileSet()
   files, err := parseGoFiles(fset, dir)
   if err != nil {
      return nil, err
   }
   if len(files) == 0 {
      return nil, fmt.Errorf("no Go source files found in directory: %s", dir)
   }

   p, err := doc.NewFromFiles(fset, files, "./")
   if err != nil {
      return nil, fmt.Errorf("failed to create doc package: %w", err)
   }

   pkgDoc := &PackageDoc{
      Name: p.Name,
      Doc:  p.Doc,
   }

   for _, f := range p.Funcs {
      if f.Recv == "" {
         sig, err := formatAndHighlight(f.Decl, fset)
         if err != nil {
            return nil, err
         }
         pkgDoc.Functions = append(pkgDoc.Functions, FuncDoc{
            Name:      f.Name,
            Doc:       f.Doc,
            Signature: sig,
         })
      }
   }

   for _, t := range p.Types {
      def, err := formatAndHighlight(t.Decl, fset)
      if err != nil {
         return nil, err
      }
      typeDoc := TypeDoc{
         Name:       t.Name,
         Doc:        t.Doc,
         Definition: def,
      }
      for _, m := range t.Methods {
         sig, err := formatAndHighlight(m.Decl, fset)
         if err != nil {
            return nil, err
         }
         typeDoc.Methods = append(typeDoc.Methods, FuncDoc{
            Name:      m.Name,
            Doc:       m.Doc,
            Signature: sig,
         })
      }
      pkgDoc.Types = append(pkgDoc.Types, typeDoc)
   }

   for _, v := range p.Consts {
      def, err := formatAndHighlight(v.Decl, fset)
      if err != nil {
         return nil, err
      }
      pkgDoc.Constants = append(pkgDoc.Constants, VarDoc{
         Doc:        v.Doc,
         Definition: def,
      })
   }

   for _, v := range p.Vars {
      def, err := formatAndHighlight(v.Decl, fset)
      if err != nil {
         return nil, err
      }
      pkgDoc.Variables = append(pkgDoc.Variables, VarDoc{
         Doc:        v.Doc,
         Definition: def,
      })
   }
   return pkgDoc, nil
}

// formatAndHighlight formats an AST node to a string and then applies syntax highlighting.
func formatAndHighlight(node any, fset *token.FileSet) (template.HTML, error) {
   var buf bytes.Buffer
   if err := format.Node(&buf, fset, node); err != nil {
      return "", fmt.Errorf("failed to format node: %w", err)
   }
   return syntaxHighlight(buf.String())
}

// parseGoFiles reads a directory, parses all non-test .go files,
// and returns them as a slice of *ast.File.
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
      if packageName == "" {
         packageName = file.Name.Name
      } else if file.Name.Name != packageName {
         return nil, fmt.Errorf("multiple package names found in directory: %s and %s", packageName, file.Name.Name)
      }
      files = append(files, file)
   }
   return files, nil
}
