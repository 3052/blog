package docgen

import "fmt"

// Run is the primary entrypoint for the docgen library.
// It parses the Go package at srcPath and generates HTML documentation
// in the specified outPath.
func Run(srcPath, outPath string) error {
   // Step 1: Parse the package.
   pkgInfo, err := ParsePackage(srcPath)
   if err != nil {
      return fmt.Errorf("error parsing package: %w", err)
   }

   // Step 2: Generate HTML files.
   if err := GenerateDocs(pkgInfo, outPath); err != nil {
      return fmt.Errorf("error generating documentation: %w", err)
   }

   return nil
}
