package doc

import (
   "html/template"
   "os"
   "path/filepath"
)

// Render generates the HTML documentation file.
func Render(pkgDoc *PackageDoc, tmplPath, outputPath string) error {
   tmpl, err := template.ParseFiles(tmplPath)
   if err != nil {
      return err
   }

   if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
      return err
   }

   file, err := os.Create(outputPath)
   if err != nil {
      return err
   }
   defer file.Close()

   return tmpl.Execute(file, pkgDoc)
}
