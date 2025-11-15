package doc

import (
   "go/ast"
   "go/parser"
   "go/token"
   "html/template"
   "io"
   "os"
   "path/filepath"
)

// Generate parses the Go files in a given directory and generates HTML documentation.
func Generate(writer io.Writer, path string) error {
   pkg, err := parsePackage(path)
   if err != nil {
      return err
   }

   tmpl, err := template.ParseFiles("package.tmpl", "functions.tmpl", "types.tmpl")
   if err != nil {
      return err
   }

   return tmpl.ExecuteTemplate(writer, "package.tmpl", pkg)
}

func parsePackage(path string) (*Package, error) {
   fset := token.NewFileSet()
   pkgs, err := parser.ParseDir(fset, path, func(fi os.FileInfo) bool {
      return !fi.IsDir() && filepath.Ext(fi.Name()) == ".go"
   }, parser.ParseComments)

   if err != nil {
      return nil, err
   }

   for name, astPkg := range pkgs {
      pkg := &Package{
         Name: name,
      }
      for _, file := range astPkg.Files {
         if file.Doc != nil {
            pkg.Doc = file.Doc.Text()
         }
         for _, decl := range file.Decls {
            switch d := decl.(type) {
            case *ast.FuncDecl:
               if d.Recv == nil {
                  fn := newFunction(d)
                  pkg.Functions = append(pkg.Functions, fn)
               } else {
                  typeName := getTypeName(d.Recv.List[0].Type)
                  for i, t := range pkg.Types {
                     if t.Name == typeName {
                        pkg.Types[i].Methods = append(pkg.Types[i].Methods, newFunction(d))
                     }
                  }
               }
            case *ast.GenDecl:
               for _, spec := range d.Specs {
                  if ts, ok := spec.(*ast.TypeSpec); ok {
                     typ := &Type{
                        Name: ts.Name.Name,
                        Doc:  ts.Doc.Text(),
                     }
                     pkg.Types = append(pkg.Types, typ)
                  }
               }
            }
         }
      }
      return pkg, nil
   }

   return nil, nil
}

func getTypeName(expr ast.Expr) string {
   switch e := expr.(type) {
   case *ast.StarExpr:
      return e.X.(*ast.Ident).Name
   case *ast.Ident:
      return e.Name
   }
   return ""
}
