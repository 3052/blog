package doc

import (
   "go/ast"
   "go/printer"
   "go/token"
   "strings"
)

// Function represents a function or method.
type Function struct {
   Name      string
   Doc       string
   Signature string
}

func newFunction(funcDecl *ast.FuncDecl) *Function {
   var sb strings.Builder
   printer.Fprint(&sb, token.NewFileSet(), funcDecl.Type)
   return &Function{
      Name:      funcDecl.Name.Name,
      Doc:       funcDecl.Doc.Text(),
      Signature: sb.String(),
   }
}
