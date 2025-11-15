package doc

import (
   "fmt"
   "go/scanner"
   "go/token"
   "html"
   "html/template"
   "log"
   "strings"
)

// syntaxHighlight takes Go source code, a fileset, and a map of type offsets,
// returning syntax-highlighted HTML with only the correct types linked.
func syntaxHighlight(source string, fset *token.FileSet, typeOffsets map[int]struct{}) (template.HTML, error) {
   if fset == nil {
      fset = token.NewFileSet()
   }
   file := fset.AddFile("", fset.Base(), len(source))

   var s scanner.Scanner
   s.Init(file, []byte(source), nil, scanner.ScanComments)

   var buf strings.Builder
   lastOffset := 0

   for {
      pos, tok, lit := s.Scan()
      if tok == token.EOF {
         break
      }

      offset := file.Offset(pos)
      if lastOffset < offset {
         buf.WriteString(html.EscapeString(source[lastOffset:offset]))
      }

      tokenText := lit
      if tokenText == "" {
         tokenText = tok.String()
      }
      escapedToken := html.EscapeString(tokenText)
      var tokenHTML string

      if tok == token.IDENT {
         _, isTypeOffset := typeOffsets[offset]
         log.Printf("[HIGHLIGHTER] Token: '%s', Offset: %d, IsType: %v", lit, offset, isTypeOffset)
         if isTypeOffset {
            tokenHTML = fmt.Sprintf(`<a href="#%s">%s</a>`, escapedToken, escapedToken)
         } else {
            tokenHTML = escapedToken
         }
      } else if tok.IsKeyword() {
         tokenHTML = fmt.Sprintf(`<span class="keyword">%s</span>`, escapedToken)
      } else if tok == token.COMMENT {
         tokenHTML = fmt.Sprintf(`<span class="comment">%s</span>`, escapedToken)
      } else if tok == token.STRING {
         tokenHTML = fmt.Sprintf(`<span class="string">%s</span>`, escapedToken)
      } else {
         tokenHTML = escapedToken
      }
      buf.WriteString(tokenHTML)

      lastOffset = offset + len(tokenText)
   }

   if lastOffset < len(source) {
      buf.WriteString(html.EscapeString(source[lastOffset:]))
   }

   return template.HTML(buf.String()), nil
}
