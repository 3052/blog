package main

import (
   "errors"
   "go/ast"
   "go/doc"
   "go/parser"
   "go/token"
   "log"
   "os"
   "path/filepath"
   "strings"
   "text/template"
   _ "embed"
   flag "github.com/spf13/pflag"
)

// Simple error to indicate empty folder
var EmptyErr = errors.New("empty folder")

func mustParse(fset *token.FileSet, filename string, src []byte) *ast.File {
   f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
   if err != nil {
      panic(err)
   }
   return f
}

// Parse walks the directory tree rooted at root and parses all .go files
// it returns a [Package] for each directory containing .go files
// or empty [Package] and [EmptyErr]
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
      return Package{}, EmptyErr
   }
   return NewPackage(fset, p, path, pkgs, fnames), nil
}
const version = "0.1.3"

const usage = `usage: gdmd [options] <directory>

go doc markdown generator

options:`

func main() {
   hFlag := flag.BoolP("help", "h", false, "print this help message")
   vFlag := flag.BoolP("version", "v", false, "print version")
   rFlag := flag.BoolP("recursive", "r", false, "walk directories recursively")

   flag.Parse()

   if *hFlag {
      println(usage)
      flag.PrintDefaults()
      return
   }
   if *vFlag {
      println(version)
      return
   }

   root, _ := filepath.Abs(flag.Arg(0))

   _, err := os.Stat(root)
   if err != nil {
      if os.IsNotExist(err) {
         log.Fatalf("directory %s does not exist", root)
      } else {
         log.Fatal(err)
      }
   }

   pkg, err := Parse(root, "", *rFlag)
   if err != nil {
      log.Fatal(err)
   }
   Generate(root, &pkg)
}
//go:embed markdown.tmpl
var templateData string

func generateOne(root string, tmpl *template.Template, pkg *Package) {
   filename := filepath.Join(root, pkg.Dir, "README.md")
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

// Generate creates markdown files for the given [Package] and its nested packages.
func Generate(root string, pkg *Package) {
   funcs := template.FuncMap{
      "ToLower": strings.ToLower,
   }
   tmpl, err := template.
      New("markdown").
      Funcs(funcs).
      Parse(templateData)
   if err != nil {
      log.Fatal(err)
   }
   generateOne(root, tmpl, pkg)
}
