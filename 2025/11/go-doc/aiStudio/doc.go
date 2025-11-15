package doc

import (
   "os"
   "path/filepath"
)

// Generate creates HTML documentation and a corresponding CSS file for the Go
// package in sourceDir and writes them to outputDir.
func Generate(sourceDir, outputDir string) error {
   pkgDoc, err := Parse(sourceDir)
   if err != nil {
      return err
   }

   // Ensure the output directory exists.
   if err := os.MkdirAll(outputDir, 0755); err != nil {
      return err
   }

   // Render the HTML file.
   templatePath := "template.tmpl"
   htmlOutputPath := filepath.Join(outputDir, "doc.html")
   if err := Render(pkgDoc, templatePath, htmlOutputPath); err != nil {
      return err
   }

   // Copy the CSS file.
   cssSourcePath := "style.css"
   cssDestPath := filepath.Join(outputDir, "style.css")
   return copyFile(cssSourcePath, cssDestPath)
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
   data, err := os.ReadFile(src)
   if err != nil {
      return err
   }
   return os.WriteFile(dst, data, 0644)
}
