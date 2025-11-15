package docgen

import (
   "fmt"
   "go/scanner"
   "go/token"
   "html/template"
   "os"
   "path/filepath"
   "strings"
)

var funcMap = template.FuncMap{
   "nl2br": func(text string) template.HTML {
      return template.HTML(strings.ReplaceAll(strings.TrimSpace(text), "\n", "<br>"))
   },
   "cleantag": func(tag string) string {
      return strings.Trim(tag, "`")
   },
   // CORRECTED: This function now preserves whitespace and newlines.
   "renderCode": func(code string, knownTypes []string) template.HTML {
      var highlighted strings.Builder
      var s scanner.Scanner
      fset := token.NewFileSet()
      file := fset.AddFile("", fset.Base(), len(code))
      s.Init(file, []byte(code), nil, scanner.ScanComments)

      var lastPos int // Keep track of position in original string

      // 1. Scan the code, preserving whitespace and wrapping comments.
      for {
         pos, tok, lit := s.Scan()
         if tok == token.EOF {
            break
         }

         offset := file.Offset(pos)

         // Append the whitespace/newlines between the last token and this one.
         if offset > lastPos {
            highlighted.WriteString(template.HTMLEscapeString(code[lastPos:offset]))
         }

         // Append the token itself, wrapping comments in a span.
         if tok == token.COMMENT {
            highlighted.WriteString(`<span class="comment">`)
            highlighted.WriteString(template.HTMLEscapeString(lit))
            highlighted.WriteString(`</span>`)
         } else {
            highlighted.WriteString(template.HTMLEscapeString(lit))
         }

         // Update the last position.
         lastPos = offset + len(lit)
      }

      // Append any trailing whitespace after the final token.
      if lastPos < len(code) {
         highlighted.WriteString(template.HTMLEscapeString(code[lastPos:]))
      }

      // 2. Link any known types in the highlighted code.
      linked := highlighted.String()
      for _, typeName := range knownTypes {
         escapedTypeName := template.HTMLEscapeString(typeName)
         link := fmt.Sprintf(`<a href="#%s">%s</a>`, escapedTypeName, escapedTypeName)
         linked = strings.ReplaceAll(linked, escapedTypeName, link)
      }

      return template.HTML(linked)
   },
}

// Generate function is unchanged.
func Generate(info *DocInfo, outputPath string) error {
   tmpl, err := template.New("").Funcs(funcMap).ParseFiles("template.html")
   if err != nil {
      return fmt.Errorf("FATAL: failed to parse template.html: %w", err)
   }

   htmlFileName := fmt.Sprintf("%s.html", info.PackageName)
   htmlFilePath := filepath.Join(outputPath, htmlFileName)
   htmlFile, err := os.Create(htmlFilePath)
   if err != nil {
      return fmt.Errorf("failed to create output file %s: %w", htmlFilePath, err)
   }
   defer htmlFile.Close()

   if err := tmpl.ExecuteTemplate(htmlFile, "template.html", info); err != nil {
      return fmt.Errorf("failed to execute template: %w", err)
   }

   cssSrcPath := "style.css"
   cssDestPath := filepath.Join(outputPath, "style.css")
   cssData, err := os.ReadFile(cssSrcPath)
   if err != nil {
      return fmt.Errorf("failed to read source css file %s: %w", cssSrcPath, err)
   }
   err = os.WriteFile(cssDestPath, cssData, 0644)
   if err != nil {
      return fmt.Errorf("failed to write destination css file %s: %w", cssDestPath, err)
   }

   return nil
}
