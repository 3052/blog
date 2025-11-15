package docgen

import (
   "bytes"
   "go/ast"
   "go/parser"
   "go/printer"
   "go/token"
   "log"
   "os"
   "path/filepath"
   "sort"
   "strings"
)

// DocInfo, StructInfo, MethodInfo are unchanged

type DocInfo struct {
   PackageName    string
   PackageComment string
   Structs        []*StructInfo
   StructNames    []string
}
type StructInfo struct {
   Name       string
   Comment    string
   Definition string
   Methods    []MethodInfo
}
type MethodInfo struct {
   Comment   string
   Signature string
}

func ParsePackage(path string) (*DocInfo, error) {
   fset := token.NewFileSet()
   structMap := make(map[string]*StructInfo)
   docInfo := &DocInfo{}
   var files []*ast.File

   // Step 1 is unchanged
   log.Println("--- PARSER: Step 1: Parsing Go files ---")
   entries, err := os.ReadDir(path)
   if err != nil {
      return nil, err
   }
   for _, entry := range entries {
      if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
         filePath := filepath.Join(path, entry.Name())
         file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
         if err != nil {
            return nil, err
         }
         files = append(files, file)
         if docInfo.PackageName == "" {
            docInfo.PackageName = file.Name.Name
         }
      }
   }

   // Step 2 is updated to filter unexported fields
   log.Println("--- PARSER: Step 2: Processing AST ---")
   for _, file := range files {
      if file.Doc != nil && docInfo.PackageComment == "" {
         docInfo.PackageComment = file.Doc.Text()
      }
      for _, decl := range file.Decls {
         switch d := decl.(type) {
         case *ast.GenDecl:
            if d.Tok == token.TYPE {
               for _, spec := range d.Specs {
                  typeSpec, ok := spec.(*ast.TypeSpec)
                  if !ok {
                     continue
                  }

                  if s, ok := typeSpec.Type.(*ast.StructType); ok {
                     // --- CORRECTED: Filter unexported fields ---
                     originalFields := s.Fields
                     exportedFields := &ast.FieldList{Opening: originalFields.Opening}

                     for _, field := range originalFields.List {
                        isExported := false
                        if len(field.Names) == 0 { // Embedded Field
                           if ast.IsExported(nodeToString(fset, field.Type)) {
                              isExported = true
                           }
                        } else { // Named Field
                           if ast.IsExported(field.Names[0].Name) {
                              isExported = true
                           }
                        }

                        if isExported {
                           exportedFields.List = append(exportedFields.List, field)
                        } else {
                           log.Printf("FILTERED unexported field from struct '%s'", typeSpec.Name.Name)
                        }
                     }
                     s.Fields = exportedFields // Temporarily swap with filtered list

                     originalDoc := d.Doc
                     d.Doc = nil
                     definition := nodeToString(fset, d)
                     d.Doc = originalDoc

                     s.Fields = originalFields // Restore original fields
                     // --- End of correction ---

                     comment := ""
                     if typeSpec.Doc != nil {
                        comment = typeSpec.Doc.Text()
                     }
                     structInfo := &StructInfo{
                        Name:       typeSpec.Name.Name,
                        Comment:    comment,
                        Definition: definition,
                     }
                     docInfo.Structs = append(docInfo.Structs, structInfo)
                     structMap[structInfo.Name] = structInfo
                  }
               }
            }

         case *ast.FuncDecl: // This case is unchanged
            if d.Recv == nil || len(d.Recv.List) == 0 {
               continue
            }
            recvTypeNode := d.Recv.List[0].Type
            if star, ok := recvTypeNode.(*ast.StarExpr); ok {
               recvTypeNode = star.X
            }
            ident, ok := recvTypeNode.(*ast.Ident)
            if !ok {
               continue
            }
            typeName := ident.Name

            if structInfo, ok := structMap[typeName]; ok {
               originalBody, originalDoc := d.Body, d.Doc
               d.Body, d.Doc = nil, nil
               signature := nodeToString(fset, d)
               d.Body, d.Doc = originalBody, originalDoc

               structInfo.Methods = append(structInfo.Methods, MethodInfo{
                  Comment:   originalDoc.Text(),
                  Signature: signature,
               })
            }
         }
      }
   }

   // Steps 3 and 4 are unchanged
   log.Println("--- PARSER: Step 3: Collecting all struct names for linking ---")
   for _, s := range docInfo.Structs {
      docInfo.StructNames = append(docInfo.StructNames, s.Name)
   }

   log.Println("--- PARSER: Step 4: Sorting structs alphabetically ---")
   sort.Slice(docInfo.Structs, func(i, j int) bool {
      return docInfo.Structs[i].Name < docInfo.Structs[j].Name
   })

   return docInfo, nil
}

// nodeToString is unchanged
func nodeToString(fset *token.FileSet, node ast.Node) string {
   var buf bytes.Buffer
   printer.Fprint(&buf, fset, node)
   return buf.String()
}
