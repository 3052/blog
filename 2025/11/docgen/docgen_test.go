package docgen // Changed from 'docgen_test'

import (
   "fmt"
   "os"
   "path/filepath"
   "testing"
   // No longer need to import "docgen"
)

// TestRunGenerator acts as the main driver for our process.
func TestRunGenerator(t *testing.T) {
   // The path to the package we want to document.
   srcPath := "./samplepkg"

   // Create a temporary directory for the output.
   // t.TempDir() automatically handles creation and cleanup.
   outPath := t.TempDir()

   t.Logf("Source Path: %s", srcPath)
   t.Logf("Output Path: %s", outPath)

   // Call the Run function directly as it's in the same package.
   err := Run(srcPath, outPath)
   if err != nil {
      t.Fatalf("Run() failed: %v", err)
   }

   // Verify that the expected files were created.
   expectedFiles := []string{
      "index.html",
      "type_User.html",
   }

   for _, file := range expectedFiles {
      expectedPath := filepath.Join(outPath, file)
      if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
         t.Errorf("Expected file was not created: %s", expectedPath)
      }
   }

   fmt.Println("\nDocumentation generated successfully in:", outPath)
   fmt.Println("You can open the generated files in your browser.")
}
