package main

import (
   "encoding/json"
   "log"
   "os"
   "path/filepath"
   "strings"
)

// FileData struct uses the clear "name" and "data" keys.
type FileData struct {
   Name string `json:"name"`
   Data string `json:"data"`
}

func main() {
   // 1. Get the target folder path from the command-line arguments.
   if len(os.Args) < 2 {
      log.Println("Error: You must provide a folder path.")
      log.Println("Usage: go run combine.go <path_to_folder>")
      os.Exit(1)
   }
   inputDir := os.Args[1]

   // Validate that the provided path is a valid directory.
   info, err := os.Stat(inputDir)
   if os.IsNotExist(err) {
      log.Fatalf("Error: The folder '%s' does not exist.", inputDir)
   }
   if !info.IsDir() {
      log.Fatalf("Error: The path '%s' is not a directory.", inputDir)
   }

   // 2. Find all processable files in that directory.
   sourceFiles, err := findSourceFiles(inputDir)
   if err != nil {
      log.Fatalf("Error finding files in '%s': %v", inputDir, err)
   }
   if len(sourceFiles) == 0 {
      log.Printf("Warning: No processable files were found in '%s'.", inputDir)
      return
   }

   log.Printf("Found %d files in '%s' to process...", len(sourceFiles), inputDir)

   // 3. Read the content of each file.
   var fileDataList []FileData
   for _, filename := range sourceFiles {
      // Construct the full path for each file before reading.
      fullPath := filepath.Join(inputDir, filename)
      content, err := os.ReadFile(fullPath)
      if err != nil {
         log.Fatalf("Error reading file %s: %v", fullPath, err)
      }

      // Convert bytes to string and remove carriage returns.
      contentString := string(content)
      cleanedContent := strings.ReplaceAll(contentString, "\r", "")

      // Append the struct using the clean field names.
      fileDataList = append(fileDataList, FileData{Name: filename, Data: cleanedContent})
   }

   // 4. Generate the JSON output.
   output, err := generateJSON(fileDataList)
   if err != nil {
      log.Fatalf("Error generating JSON output: %v", err)
   }

   // 5. Write the output file inside the target directory.
   outputFilename := filepath.Join(inputDir, "combined.json")
   err = os.WriteFile(outputFilename, []byte(output), 0644)
   if err != nil {
      log.Fatalf("Error writing to file %s: %v", outputFilename, err)
   }

   log.Printf("Success! Output has been saved to %s", outputFilename)
}

// findSourceFiles searches a directory for all files, excluding itself and its output.
func findSourceFiles(targetDir string) ([]string, error) {
   var files []string
   err := filepath.WalkDir(targetDir, func(path string, d os.DirEntry, err error) error {
      if err != nil {
         return err
      }
      // Only process files in the top-level of the target directory (no subdirectories).
      if !d.IsDir() && filepath.Dir(path) == targetDir {
         // Exclude the script itself and its potential output file.
         if d.Name() != "combine.go" && d.Name() != "combined.json" {
            files = append(files, d.Name())
         }
      }
      return nil
   })
   return files, err
}

// --- MODIFICATION IS HERE ---
// generateJSON converts the file data into a compact, single-line JSON string.
func generateJSON(data []FileData) (string, error) {
   // Use json.Marshal for compact output without indentation.
   bytes, err := json.Marshal(data)
   if err != nil {
      return "", err
   }
   return string(bytes), nil
}
