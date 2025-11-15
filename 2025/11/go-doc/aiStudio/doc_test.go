package doc

import (
   "os"
   "testing"
)

func TestGenerate(t *testing.T) {
   file, err := os.Create("example.html")
   if err != nil {
      t.Fatalf("Failed to create output file: %v", err)
   }
   defer file.Close()

   if err := Generate(file, "example"); err != nil {
      t.Fatalf("Generate() failed: %v", err)
   }
}
