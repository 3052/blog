package doc

import (
   "os"
   "path/filepath"
   "testing"
)

func TestGenerate(t *testing.T) {
   sourceDir := "example"
   outputDir := "test_output"

   if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
      t.Fatalf("example directory does not exist, please provide it for testing")
   }

   if err := os.RemoveAll(outputDir); err != nil {
      t.Fatalf("failed to remove existing output directory: %v", err)
   }

   if err := Generate(sourceDir, outputDir); err != nil {
      t.Fatalf("Generate() failed: %v", err)
   }

   outputFile := filepath.Join(outputDir, "doc.html")
   if _, err := os.Stat(outputFile); os.IsNotExist(err) {
      t.Errorf("expected output file %s was not created", outputFile)
   }
}
