package docgen

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

var funcMap = template.FuncMap{
	"nl2br": func(text string) template.HTML {
		return template.HTML(strings.ReplaceAll(strings.TrimSpace(text), "\n", "<br>"))
	},
	"cleantag": func(tag string) string {
		return strings.Trim(tag, "`")
	},
	// RESTORED: This function creates an HTML link for the receiver type.
	"linkifyRecv": func(signature, recvType string) template.HTML {
		// Create the hyperlink HTML
		link := fmt.Sprintf(`<a href="#%s">%s</a>`, recvType, recvType)

		// Replace the first occurrence of the receiver type name in the signature.
		// We use strings.Replace with a count of 1 to avoid mangling return types.
		// This trusts that the receiver is the first instance of the type name.
		linkedSignature := strings.Replace(signature, recvType, link, 1)

		return template.HTML(linkedSignature)
	},
}

// Generate function is unchanged from the last working version.
func Generate(info *DocInfo, outputPath string) error {
	tmpl, err := template.New("").Funcs(funcMap).ParseFiles("template.html")
	if err != nil {
		return fmt.Errorf("FATAL: failed to parse template.html: %w", err)
	}

	fileName := fmt.Sprintf("%s.html", info.PackageName)
	filePath := filepath.Join(outputPath, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", filePath, err)
	}
	defer file.Close()

	if err := tmpl.ExecuteTemplate(file, "template.html", info); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
