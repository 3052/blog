package docgen

import (
   "log"
   "os"
   "path/filepath"
   "testing"
)

func TestGenerateDocs(t *testing.T) {
   const inputPath = "./samplepkg"
   const outputPath = "./generated_docs"

   log.Println("--- Starting Documentation Generation ---")
   log.Printf("Input package path: %s", inputPath)
   log.Printf("Output directory: %s", outputPath)

   pkgInfo, err := ParsePackage(inputPath)
   if err != nil {
      t.Fatalf("FATAL: Failed to parse package '%s': %v", inputPath, err)
   }

   if err := os.RemoveAll(outputPath); err != nil {
      t.Logf("Warning: Could not remove old output directory: %v", outputPath)
   }
   if err := os.MkdirAll(outputPath, 0755); err != nil {
      t.Fatalf("FATAL: Failed to create output directory: %v", outputPath)
   }

   err = Generate(pkgInfo, outputPath)
   if err != nil {
      t.Fatalf("FATAL: Failed to generate documentation: %v", err)
   }

   absPath, _ := filepath.Abs(outputPath)
   log.Printf("✅ SUCCESS: Documentation generated in %s", absPath)
}
