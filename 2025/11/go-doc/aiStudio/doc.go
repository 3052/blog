package doc

import (
	"path/filepath"
)

// Generate creates HTML documentation for the Go package in sourceDir
// and writes it to outputDir.
func Generate(sourceDir, outputDir string) error {
	pkgDoc, err := Parse(sourceDir)
	if err != nil {
		return err
	}

	templatePath := "template.tmpl"
	outputPath := filepath.Join(outputDir, "doc.html")

	return Render(pkgDoc, templatePath, outputPath)
}
