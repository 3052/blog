package doc

import (
   "fmt"
   "go/scanner"
   "go/token"
   "html"
   "html/template"
   "strings"
)

// syntaxHighlight takes a string of Go source code and returns it as
// syntax-highlighted HTML. It preserves all whitespace, operators, and braces.
func syntaxHighlight(source string) (template.HTML, error) {
   fset := token.NewFileSet()
   // Add the source code as a new file to the fileset.
   // Using a base of 1 is standard.
   file := fset.AddFile("", 1, len(source))

   var s scanner.Scanner
   s.Init(file, []byte(source), nil, scanner.ScanComments)

   var buf strings.Builder
   lastOffset := 0 // Tracks the 0-based offset of the end of the last token

   for {
      pos, tok, lit := s.Scan()
      if tok == token.EOF {
         break
      }

      // Use file.Offset to get the correct 0-based offset for the token.
      // This is the robust way to prevent panics.
      offset := file.Offset(pos)

      // Append any text (whitespace, operators) between the end of the last
      // token and the beginning of the current one.
      if lastOffset < offset {
         buf.WriteString(html.EscapeString(source[lastOffset:offset]))
      }

      // The scanner doesn't provide a literal for operators (like braces),
      // so we use tok.String() as a fallback.
      tokenText := lit
      if tokenText == "" {
         tokenText = tok.String()
      }

      escapedToken := html.EscapeString(tokenText)
      switch {
      case tok.IsKeyword():
         fmt.Fprintf(&buf, `<span class="keyword">%s</span>`, escapedToken)
      case tok == token.COMMENT:
         fmt.Fprintf(&buf, `<span class="comment">%s</span>`, escapedToken)
      case tok == token.STRING:
         fmt.Fprintf(&buf, `<span class="string">%s</span>`, escapedToken)
      default:
         buf.WriteString(escapedToken)
      }

      // Update the last offset to be the position right after the current token.
      lastOffset = offset + len(tokenText)
   }

   // Append any trailing text (like a final newline) after the very last token.
   if lastOffset < len(source) {
      buf.WriteString(html.EscapeString(source[lastOffset:]))
   }

   return template.HTML(buf.String()), nil
}
