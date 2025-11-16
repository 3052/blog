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
func Parse(dir, repoURL, version, importPath, vcs string) (*PackageDoc, error) {
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

   typeNames := make(map[string]struct{})
   for _, t := range p.Types {
      typeNames[t.Name] = struct{}{}
   }

   pkgDoc := &PackageDoc{
      Name:          p.Name,
      RepositoryURL: repoURL,
      Version:       version,
      ImportPath:    importPath,
      VCS:           vcs,
      Doc:           p.Doc,
   }

   process := func(decl ast.Decl) (template.HTML, error) {
      return formatAndHighlight(decl, fset, typeNames)
   }

   for _, f := range p.Funcs {
      if f.Recv == "" {
         sig, err := process(f.Decl)
         if err != nil {
            return nil, err
         }
         pkgDoc.Functions = append(pkgDoc.Functions, FuncDoc{Name: f.Name, Doc: f.Doc, Signature: sig})
      }
   }

   for _, t := range p.Types {
      def, err := process(t.Decl)
      if err != nil {
         return nil, err
      }
      typeDoc := TypeDoc{Name: t.Name, Doc: t.Doc, Definition: def}
      for _, m := range t.Methods {
         sig, err := process(m.Decl)
         if err != nil {
            return nil, err
         }
         typeDoc.Methods = append(typeDoc.Methods, FuncDoc{Name: m.Name, Doc: m.Doc, Signature: sig})
      }
      pkgDoc.Types = append(pkgDoc.Types, typeDoc)
   }

   for _, v := range p.Consts {
      def, err := process(v.Decl)
      if err != nil {
         return nil, err
      }
      pkgDoc.Constants = append(pkgDoc.Constants, VarDoc{Doc: v.Doc, Definition: def})
   }

   for _, v := range p.Vars {
      def, err := process(v.Decl)
      if err != nil {
         return nil, err
      }
      pkgDoc.Variables = append(pkgDoc.Variables, VarDoc{Doc: v.Doc, Definition: def})
   }
   return pkgDoc, nil
}

// collectFromExpr recursively walks a type expression AST node and collects the offsets
// of any identifiers that are known package types.
func collectFromExpr(expr ast.Expr, fset *token.FileSet, typeNames map[string]struct{}, offsets map[int]struct{}) {
   if expr == nil {
      return
   }
   switch x := expr.(type) {
   case *ast.Ident:
      if _, isType := typeNames[x.Name]; isType {
         file := fset.File(x.Pos())
         if file != nil {
            offsets[file.Offset(x.Pos())] = struct{}{}
         }
      }
   case *ast.StarExpr:
      collectFromExpr(x.X, fset, typeNames, offsets)
   case *ast.ArrayType:
      collectFromExpr(x.Elt, fset, typeNames, offsets)
   case *ast.MapType:
      collectFromExpr(x.Key, fset, typeNames, offsets)
      collectFromExpr(x.Value, fset, typeNames, offsets)
   case *ast.ChanType:
      collectFromExpr(x.Value, fset, typeNames, offsets)
   case *ast.FuncType:
      if x.Params != nil {
         for _, field := range x.Params.List {
            collectFromExpr(field.Type, fset, typeNames, offsets)
         }
      }
      if x.Results != nil {
         for _, field := range x.Results.List {
            collectFromExpr(field.Type, fset, typeNames, offsets)
         }
      }
   case *ast.StructType:
      if x.Fields != nil {
         for _, field := range x.Fields.List {
            collectFromExpr(field.Type, fset, typeNames, offsets)
         }
      }
   case *ast.InterfaceType:
      if x.Methods != nil {
         for _, field := range x.Methods.List {
            collectFromExpr(field.Type, fset, typeNames, offsets)
         }
      }
   case *ast.SelectorExpr:
      collectFromExpr(x.X, fset, typeNames, offsets)
   case *ast.IndexExpr:
      collectFromExpr(x.X, fset, typeNames, offsets)
      collectFromExpr(x.Index, fset, typeNames, offsets)
   case *ast.IndexListExpr:
      collectFromExpr(x.X, fset, typeNames, offsets)
      for _, index := range x.Indices {
         collectFromExpr(index, fset, typeNames, offsets)
      }
   }
}

// collectTypeUsageOffsets uses a top-level AST inspection to find all nodes
// that contain type expressions, and then hands them off to the precise recursive walker.
func collectTypeUsageOffsets(rootNode ast.Node, fset *token.FileSet, typeNames map[string]struct{}) map[int]struct{} {
   offsets := make(map[int]struct{})
   ast.Inspect(rootNode, func(n ast.Node) bool {
      if n == nil {
         return false
      }
      switch x := n.(type) {
      case *ast.ValueSpec:
         collectFromExpr(x.Type, fset, typeNames, offsets)
      case *ast.TypeSpec:
         collectFromExpr(x.Type, fset, typeNames, offsets)
      case *ast.Field:
         collectFromExpr(x.Type, fset, typeNames, offsets)
      }
      return true
   })
   return offsets
}

func formatAndHighlight(node ast.Node, fset *token.FileSet, typeNames map[string]struct{}) (template.HTML, error) {
   var buf bytes.Buffer
   if err := format.Node(&buf, fset, node); err != nil {
      return "", fmt.Errorf("failed to format node: %w", err)
   }
   sourceString := buf.String()

   const prefix = "package p\n\n"
   wrappedSource := prefix + sourceString

   fsetForHighlighting := token.NewFileSet()
   astFile, err := parser.ParseFile(fsetForHighlighting, "snippet.go", wrappedSource, 0)
   if err != nil || len(astFile.Decls) == 0 {
      return syntaxHighlight(sourceString, token.NewFileSet(), nil)
   }
   newRootNode := astFile.Decls[0]

   rawOffsets := collectTypeUsageOffsets(newRootNode, fsetForHighlighting, typeNames)

   adjustedOffsets := make(map[int]struct{})
   for offset := range rawOffsets {
      adjustedOffset := offset - len(prefix)
      if adjustedOffset >= 0 {
         adjustedOffsets[adjustedOffset] = struct{}{}
      }
   }
   return syntaxHighlight(sourceString, fsetForHighlighting, adjustedOffsets)
}

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
