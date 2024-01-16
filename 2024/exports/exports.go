package exports

import (
   "fmt"
   "go/ast"
   "go/parser"
   "go/token"
   "io/fs"
   "strings"
)

// github.com/dominikh/go-tools/issues/1416
func (e Export) Format(f fmt.State, verb rune) {
   switch verb {
   case 'p':
      if e.Recv != nil {
         fmt.Fprint(f, ".", e.Recv.Name, ".")
      }
   case 'v':
      if e.Recv != nil {
         fmt.Fprint(f, e.Recv.Name, ".")
      }
   case 'u':
      fmt.Fprint(f, "_")
   }
   fmt.Fprint(f, e.Name.Name)
}

type Export struct {
   Recv *ast.Ident
   Name *ast.Ident
}

func Exports(dir string) ([]Export, error) {
   set := token.NewFileSet()
   filter := func(fi fs.FileInfo) bool {
      return !strings.HasSuffix(fi.Name(), "_test.go")
   }
   packs, err := parser.ParseDir(set, dir, filter, 0) // ast.Package
   if err != nil {
      return nil, err
   }
   var exps []Export
   for _, pack := range packs { // ast.Package
      for _, file := range pack.Files { // ast.File
         for _, decl := range file.Decls { // ast.Decl
            switch decl := decl.(type) {
            default:
               panic("default")
            case *ast.FuncDecl:
               if decl.Name.IsExported() {
                  var exp Export
                  if decl.Recv != nil {
                     for _, recv := range decl.Recv.List {
                        switch recv := recv.Type.(type) {
                        default:
                           panic("default")
                        case *ast.Ident:
                           exp.Recv = recv
                        case *ast.StarExpr:
                           switch expr := recv.X.(type) {
                           default:
                              panic("default")
                           case *ast.Ident:
                              exp.Recv = expr
                           }
                        }
                     }
                  }
                  exp.Name = decl.Name
                  exps = append(exps, exp)
               }
            case *ast.GenDecl:
               for _, spec := range decl.Specs { // ast.Spec
                  switch spec := spec.(type) {
                  default:
                     panic("default")
                  case *ast.ImportSpec:
                  case *ast.TypeSpec:
                     if spec.Name.IsExported() {
                        exps = append(exps, Export{Name: spec.Name})
                     }
                     switch typ := spec.Type.(type) {
                     default:
                        panic(typ)
                     case *ast.ArrayType:
                     case *ast.FuncType:
                     case *ast.Ident:
                     case *ast.InterfaceType:
                        exps = field(exps, spec.Name, typ.Methods)
                     case *ast.MapType:
                     case *ast.StructType:
                        exps = field(exps, spec.Name, typ.Fields)
                     }
                  case *ast.ValueSpec:
                     for _, name := range spec.Names {
                        if name.IsExported() {
                           exps = append(exps, Export{Name: name})
                        }
                     }
                  }
               }
            }
         }
      }
   }
   return exps, nil
}

func field(exps []Export, recv *ast.Ident, f *ast.FieldList) []Export {
   for _, field := range f.List {
      if field.Names != nil {
         for _, name := range field.Names {
            if name.IsExported() {
               exps = append(exps, Export{
                  Recv: recv,
                  Name: name,
               })
            }
         }
      } else {
         switch field := field.Type.(type) {
         default:
            panic("default")
         case *ast.Ident:
         case *ast.SelectorExpr:
            exps = append(exps, Export{
               Recv: recv,
               Name: field.Sel,
            })
         case *ast.StarExpr:
            switch expr := field.X.(type) {
            case *ast.Ident:
               exps = append(exps, Export{
                  Recv: recv,
                  Name: expr,
               })
            case *ast.SelectorExpr:
            default:
               panic(expr)
            }
         }
      }
   }
   return exps
}
