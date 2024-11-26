package main

import (
   "errors"
   "flag"
   "go/ast"
   "go/doc"
   "go/parser"
   "go/token"
   "os"
   "path/filepath"
   "strings"
   "text/template"
   _ "embed"
)

func main() {
   recurse := flag.Bool("r", false, "walk directories recursively")
   dir := flag.String("d", "", "directory")
   flag.Parse()
   root, err := filepath.Abs(*dir)
   if err != nil {
      panic(err)
   }
   pkg, err := Parse(root, "", *recurse)
   if err != nil {
      panic(err)
   }
   funcs := template.FuncMap{
      "ToLower": func(s string) string {
         return strings.ToLower(strings.ReplaceAll(s, "*", ""))
      },
      "escape": func(s string) string {
         return strings.ReplaceAll(s, "*", `\*`)
      },
   }
   tmpl, err := template.New("markdown").Funcs(funcs).Parse(templateData)
   if err != nil {
      panic(err)
   }
   generateOne(root, tmpl, &pkg)
}

func generateOne(root string, tmpl *template.Template, pkg *Package) {
   filename := filepath.Join(root, pkg.Dir, "docs.md")
   f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
   if err != nil {
      panic(err)
   }
   defer f.Close()

   err = tmpl.Execute(f, pkg)
   if err != nil {
      panic(err)
   }

   for _, nstd := range pkg.Nested {
      generateOne(root, tmpl, &nstd)
   }
}

// Simple error to indicate empty folder
var err_empty = errors.New("empty folder")

func mustParse(fset *token.FileSet, filename string, src []byte) *ast.File {
   f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
   if err != nil {
      panic(err)
   }
   return f
}

// Parse walks the directory tree rooted at root and parses all .go files
// it returns a [Package] for each directory containing .go files
// or empty [Package] and [err_empty]
func Parse(root, path string, recursive bool) (Package, error) {
   entries, _ := os.ReadDir(filepath.Join(root, path))

   fset := token.NewFileSet()

   files := []*ast.File{}

   pkgs := []Package{}
   fnames := []string{}

   for _, e := range entries {
      // Hidden file or directory. The Go compiler behaves consistently across Windows and Posix.
      // It skips files and directories that begin with '.' but ignores hidden attribute in Windows.
      if strings.HasPrefix(e.Name(), ".") {
         continue
      }

      nextPath := filepath.Join(path, e.Name())

      if e.IsDir() && recursive {
         pkg, err := Parse(root, nextPath, recursive)
         if err == nil {
            pkgs = append(pkgs, pkg)
         } // else ignore error
      } else {
         if !strings.HasSuffix(e.Name(), ".go") ||
            strings.HasSuffix(e.Name(), "_test.go") {
            continue
         }
         fnames = append(fnames, e.Name())

         src, _ := os.ReadFile(filepath.Join(root, nextPath))
         files = append(files, mustParse(fset, e.Name(), src))
      }
   }

   p, err := doc.NewFromFiles(fset, files, "example.com")
   if err != nil {
      return Package{}, err
   }
   if len(fnames) == 0 && len(pkgs) == 0 {
      return Package{}, err_empty
   }
   return NewPackage(fset, p, path, pkgs, fnames), nil
}

//go:embed markdown.tmpl
var templateData string
