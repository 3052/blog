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
	"strings"
)

type DocInfo struct {
	PackageName    string
	PackageComment string
	Structs        []*StructInfo
}

type StructInfo struct {
	Name    string
	Comment string
	Fields  []FieldInfo
	Methods []MethodInfo
}

type FieldInfo struct {
	Name    string
	Type    string
	Comment string
	Tag     string
}

// CORRECTED: MethodInfo now holds the clean signature AND the receiver type for linking.
type MethodInfo struct {
	Comment   string
	Signature string
	RecvType  string
}

func ParsePackage(path string) (*DocInfo, error) {
	fset := token.NewFileSet()
	structMap := make(map[string]*StructInfo)
	docInfo := &DocInfo{}
	var files []*ast.File

	// Step 1: Manually find and parse each .go file.
	log.Println("--- PARSER: Step 1: Parsing Go files ---")
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			filePath := filepath.Join(path, entry.Name())
			log.Printf("Parsing file: %s", filePath)
			file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
			if err != nil {
				return nil, err
			}
			files = append(files, file)
			if docInfo.PackageName == "" {
				docInfo.PackageName = file.Name.Name
				log.Printf("Determined Package Name: '%s'", docInfo.PackageName)
			}
		}
	}

	// Step 2: A single pass to process the ASTs.
	log.Println("--- PARSER: Step 2: Processing AST ---")
	for _, file := range files {
		if file.Doc != nil && docInfo.PackageComment == "" {
			docInfo.PackageComment = file.Doc.Text()
		}

		ast.Inspect(file, func(n ast.Node) bool {
			switch decl := n.(type) {
			case *ast.TypeSpec:
				if s, ok := decl.Type.(*ast.StructType); ok {
					structInfo := &StructInfo{Name: decl.Name.Name, Comment: decl.Doc.Text()}
					// Field parsing is unchanged...
					for _, field := range s.Fields.List {
						fieldType := nodeToString(fset, field.Type)
						if len(field.Names) == 0 {
							fieldInfo := FieldInfo{Name: fieldType, Type: fieldType}
							if field.Doc != nil { fieldInfo.Comment = field.Doc.Text() }
							if field.Tag != nil { fieldInfo.Tag = field.Tag.Value }
							structInfo.Fields = append(structInfo.Fields, fieldInfo)
						} else {
							for _, name := range field.Names {
								fieldInfo := FieldInfo{Name: name.Name, Type: fieldType}
								if field.Doc != nil { fieldInfo.Comment = field.Doc.Text() }
								if field.Tag != nil { fieldInfo.Tag = field.Tag.Value }
								structInfo.Fields = append(structInfo.Fields, fieldInfo)
							}
						}
					}
					docInfo.Structs = append(docInfo.Structs, structInfo)
					structMap[structInfo.Name] = structInfo
				}

			case *ast.FuncDecl:
				if decl.Recv == nil || len(decl.Recv.List) == 0 {
					return true
				}
				recvTypeNode := decl.Recv.List[0].Type
				if star, ok := recvTypeNode.(*ast.StarExpr); ok { recvTypeNode = star.X }
				ident, ok := recvTypeNode.(*ast.Ident)
				if !ok { return true }
				typeName := ident.Name

				if structInfo, ok := structMap[typeName]; ok {
					log.Printf("Found method '%s' for struct '%s'", decl.Name.Name, typeName)
					
					originalBody := decl.Body
					originalDoc := decl.Doc
					decl.Body = nil
					decl.Doc = nil
					signature := nodeToString(fset, decl)
					decl.Body = originalBody
					decl.Doc = originalDoc

					log.Printf(" - Generated Signature: '%s'", signature)
					log.Printf(" - Receiver Type for linking: '%s'", typeName)

					structInfo.Methods = append(structInfo.Methods, MethodInfo{
						Comment:   originalDoc.Text(),
						Signature: signature,
						RecvType:  typeName, // Storing the type name for the template
					})
				}
			}
			return true
		})
	}
	return docInfo, nil
}

func nodeToString(fset *token.FileSet, node ast.Node) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, node)
	return buf.String()
}
