package doc

import (
   "os"
   "path/filepath"
   "testing"
)

func TestGenerate(t *testing.T) {
   sourceDir := "example"
   outputDir := "test_output"
   version := "v1.2.3"
   vcs := "git"
   importPath := "example.com/me/repo"
   repoURL := "https://github.com/example/repo"

   if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
      t.Fatalf("example directory does not exist, please provide it for testing")
   }

   if err := os.RemoveAll(outputDir); err != nil {
      t.Fatalf("failed to remove existing output directory: %v", err)
   }

   if err := Generate(sourceDir, outputDir, repoURL, version, importPath, vcs); err != nil {
      t.Fatalf("Generate() failed: %v", err)
   }

   // Check that both the HTML and CSS files were created.
   filesToCheck := []string{"doc.html", "style.css"}
   for _, f := range filesToCheck {
      outputFile := filepath.Join(outputDir, f)
      if _, err := os.Stat(outputFile); os.IsNotExist(err) {
         t.Errorf("expected output file %s was not created", outputFile)
      }
   }
}
