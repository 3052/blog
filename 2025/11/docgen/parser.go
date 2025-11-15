package docgen

import (
   "bytes"
   "fmt"
   // "go/ast" is no longer used and has been removed.
   "go/doc"
   "go/printer"
   "go/token"
   "log"
   "strings"

   "golang.org/x/tools/go/packages"
)

// PackageInfo holds all documentation for a single package.
type PackageInfo struct {
   Name   string
   Doc    string
   Consts *ValueInfo
   Vars   *ValueInfo
   Types  []*TypeInfo
   Funcs  []*FuncInfo
}

// TypeInfo holds documentation for a single type.
type TypeInfo struct {
   Name    string
   Doc     string
   Decl    string
   Funcs   []*FuncInfo // Associated functions
   Methods []*FuncInfo // Methods on this type
}

// FuncInfo holds documentation for a single function or method.
type FuncInfo struct {
   Name      string
   Doc       string
   Recv      string // Receiver, e.g., "u *User"
   Signature string
}

// ValueInfo holds documentation for a block of consts or vars.
type ValueInfo struct {
   Doc  string
   Decl []string
}

// formatNode converts an AST node back to a string using the correct context.
func formatNode(fset *token.FileSet, node any) string {
   var buf bytes.Buffer
   // Pass the original FileSet to the printer.
   err := printer.Fprint(&buf, fset, node)
   if err != nil {
      // Log the detailed error when formatting fails.
      log.Printf("failed to format node: %v", err)
      return ""
   }
   return buf.String()
}

// ParsePackage parses a Go package directory and returns structured doc info.
func ParsePackage(path string) (*PackageInfo, error) {
   cfg := &packages.Config{
      Mode: packages.LoadSyntax,
   }

   pkgs, err := packages.Load(cfg, path)
   if err != nil {
      return nil, fmt.Errorf("failed to load package %s: %w", path, err)
   }
   if len(pkgs) != 1 {
      return nil, fmt.Errorf("expected 1 package, but found %d", len(pkgs))
   }
   pkg := pkgs[0]

   // Check for errors during parsing.
   if len(pkg.Errors) > 0 {
      // --- FIX IS HERE ---
      // The error message now wraps the underlying error correctly.
      return nil, fmt.Errorf("encountered parse error: %w", pkg.Errors[0])
   }

   docPkg, err := doc.NewFromFiles(pkg.Fset, pkg.Syntax, pkg.PkgPath, doc.Mode(0))
   if err != nil {
      return nil, fmt.Errorf("failed to create doc package: %w", err)
   }

   p := &PackageInfo{
      Name: docPkg.Name,
      Doc:  docPkg.Doc,
   }

   // Extract Constants
   if len(docPkg.Consts) > 0 {
      p.Consts = &ValueInfo{Doc: docPkg.Consts[0].Doc}
      for _, c := range docPkg.Consts {
         for _, spec := range c.Decl.Specs {
            p.Consts.Decl = append(p.Consts.Decl, formatNode(pkg.Fset, spec))
         }
      }
   }

   // Extract Variables
   if len(docPkg.Vars) > 0 {
      p.Vars = &ValueInfo{Doc: docPkg.Vars[0].Doc}
      for _, v := range docPkg.Vars {
         for _, spec := range v.Decl.Specs {
            p.Vars.Decl = append(p.Vars.Decl, formatNode(pkg.Fset, spec))
         }
      }
   }

   // Extract top-level Functions
   for _, f := range docPkg.Funcs {
      p.Funcs = append(p.Funcs, &FuncInfo{
         Name:      f.Name,
         Doc:       f.Doc,
         Signature: formatNode(pkg.Fset, f.Decl.Type),
      })
   }

   // Extract Types and their Methods/Functions
   for _, t := range docPkg.Types {
      typeInfo := &TypeInfo{
         Name: t.Name,
         Doc:  t.Doc,
         Decl: formatNode(pkg.Fset, t.Decl),
      }

      // Functions that return an instance of the type
      for _, f := range t.Funcs {
         typeInfo.Funcs = append(typeInfo.Funcs, &FuncInfo{
            Name:      f.Name,
            Doc:       f.Doc,
            Signature: formatNode(pkg.Fset, f.Decl.Type),
         })
      }
      // Methods on the type
      for _, m := range t.Methods {
         typeInfo.Methods = append(typeInfo.Methods, &FuncInfo{
            Name:      m.Name,
            Doc:       m.Doc,
            Recv:      strings.Trim(formatNode(pkg.Fset, m.Recv), "()"),
            Signature: formatNode(pkg.Fset, m.Decl.Type),
         })
      }
      p.Types = append(p.Types, typeInfo)
   }

   return p, nil
}
