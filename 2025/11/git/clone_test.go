package git

import (
   "os"
   "path/filepath"
   "strings"
   "testing"
)

func TestCloneCreatesObjects(t *testing.T) {
   repoURL := "https://github.com/git-fixtures/basic.git"
   outputDir := "test-clone-output-full"

   t.Logf("Cleaning up previous test directory: %s", outputDir)
   if err := os.RemoveAll(outputDir); err != nil {
      t.Fatalf("Failed to clean up directory %s: %v", outputDir, err)
   }

   // Execute the simplified clone function.
   err := Clone(repoURL, outputDir)
   if err != nil {
      t.Fatalf("Clone function failed: %v", err)
   }

   // Verification steps
   gitDir := filepath.Join(outputDir, ".git")
   masterRefFile := filepath.Join(gitDir, "refs", "heads", "master")
   hashBytes, err := os.ReadFile(masterRefFile)
   if err != nil {
      t.Fatalf("Could not read master ref file: %v", err)
   }
   headCommitSHA := strings.TrimSpace(string(hashBytes))

   objectPath := filepath.Join(gitDir, "objects", headCommitSHA[:2], headCommitSHA[2:])
   _, err = os.Stat(objectPath)
   if os.IsNotExist(err) {
      t.Fatalf("FATAL: HEAD commit object '%s' was NOT created at '%s'", headCommitSHA, objectPath)
   }

   t.Logf("SUCCESS: HEAD commit object '%s' found.", headCommitSHA)
}
