package docgen

import (
   "embed"
   "fmt"
   "html/template"
   "log"
   "os"
   "path/filepath"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// GenerateDocs is an exported function that generates the HTML files.
func GenerateDocs(pkg *PackageInfo, outDir string) error {
   // Create the output directory if it doesn't exist.
   if err := os.MkdirAll(outDir, 0755); err != nil {
      return fmt.Errorf("failed to create output directory: %w", err)
   }

   // Parse all templates from the embedded filesystem.
   tmpl, err := template.ParseFS(templateFS, "templates/*.tmpl")
   if err != nil {
      return fmt.Errorf("failed to parse templates: %w", err)
   }

   // --- Generate index.html ---
   indexPath := filepath.Join(outDir, "index.html")
   indexFile, err := os.Create(indexPath)
   if err != nil {
      return fmt.Errorf("failed to create index.html: %w", err)
   }
   defer indexFile.Close()

   // Execute the index template with the package data.
   err = tmpl.ExecuteTemplate(indexFile, "index.html.tmpl", pkg)
   if err != nil {
      return fmt.Errorf("failed to execute index template: %w", err)
   }
   log.Printf("Generated: %s", indexPath)

   // --- Generate a file for each type ---
   for _, t := range pkg.Types {
      // Create a file named `type_TypeName.html`.
      typeFilename := fmt.Sprintf("type_%s.html", t.Name)
      typePath := filepath.Join(outDir, typeFilename)
      typeFile, err := os.Create(typePath)
      if err != nil {
         return fmt.Errorf("failed to create file for type %s: %w", t.Name, err)
      }
      defer typeFile.Close()

      // Execute the type template with the specific type's data.
      err = tmpl.ExecuteTemplate(typeFile, "type.html.tmpl", t)
      if err != nil {
         return fmt.Errorf("failed to execute template for type %s: %w", t.Name, err)
      }
      log.Printf("Generated: %s", typePath)
   }

   return nil
}
