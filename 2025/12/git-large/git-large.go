package main

import (
   "bufio"
   "fmt"
   "io"
   "log"
   "os"
   "os/exec"
   "regexp"
   "sort"
   "strconv"
   "strings"
   "sync"
   "text/tabwriter"
)

// BlobInfo holds information about a Git blob object.
type BlobInfo struct {
   Sha  string
   Size int64
   Path string
}

func main() {
   // --- Step 1: Get all large blob candidates from history. ---
   fmt.Println("Step 1/4: Finding all large file candidates in history...")
   historicalBlobs, err := getAllLargeBlobs()
   if err != nil {
      log.Fatalf("Error finding historical large blobs: %v", err)
   }

   // --- Step 2: Get a set of all paths in the current commit. ---
   fmt.Println("Step 2/4: Getting file paths from the current commit...")
   currentPaths, err := getCurrentPaths()
   if err != nil {
      log.Fatalf("Error getting current paths: %v", err)
   }

   // --- Step 3: Build a reverse-lookup map of renames. ---
   fmt.Println("Step 3/4: Building rename history map...")
   reverseRenameMap, err := buildReverseRenameMap()
   if err != nil {
      log.Fatalf("Error building rename map: %v", err)
   }

   // --- Step 4: Trace the lineage of current files and filter the candidates. ---
   fmt.Println("Step 4/4: Tracing file lineages and identifying orphans...")
   liveLineages := traceLiveLineages(currentPaths, reverseRenameMap)

   var orphanBlobs []BlobInfo
   // We check against the original path of the blob.
   for path, blob := range historicalBlobs {
      if _, isLive := liveLineages[path]; !isLive {
         orphanBlobs = append(orphanBlobs, blob)
      }
   }

   // --- Display the final results ---
   printResults(orphanBlobs)
}

// getAllLargeBlobs uses the fast pipeline approach to find all blob objects >= 100KB.
func getAllLargeBlobs() (map[string]BlobInfo, error) {
   revList := exec.Command("git", "rev-list", "--objects", "--all")
   catFile := exec.Command("git", "cat-file", "--batch-check=%(objecttype) %(objectname) %(objectsize) %(rest)")

   pipeReader, pipeWriter := io.Pipe()
   revList.Stdout = pipeWriter
   catFile.Stdin = pipeReader
   catFileOutput, err := catFile.StdoutPipe()
   if err != nil {
      return nil, err
   }

   if err := revList.Start(); err != nil { return nil, err }
   if err := catFile.Start(); err != nil { return nil, err }

   blobs := make(map[string]BlobInfo)
   re := regexp.MustCompile(`^blob\s+([0-9a-f]{40})\s+(\d+)\s+(.+)`)
   var wg sync.WaitGroup
   wg.Add(1)
   var scannerErr error

   go func() {
      defer wg.Done()
      scanner := bufio.NewScanner(catFileOutput)
      for scanner.Scan() {
         matches := re.FindStringSubmatch(scanner.Text())
         if len(matches) != 4 { continue }
         size, _ := strconv.ParseInt(matches[2], 10, 64)
         if size < 102400 { continue }
         path := matches[3]
         // Keep only the largest version of each blob path found
         if existing, ok := blobs[path]; !ok || size > existing.Size {
            blobs[path] = BlobInfo{Sha: matches[1], Size: size, Path: path}
         }
      }
      scannerErr = scanner.Err()
   }()

   err = revList.Wait()
   pipeWriter.Close()
   if err != nil { return nil, err }
   err = catFile.Wait()
   if err != nil { return nil, err }
   wg.Wait()
   if scannerErr != nil { return nil, scannerErr }

   return blobs, nil
}

// getCurrentPaths returns a set of all file paths in the current HEAD.
func getCurrentPaths() (map[string]struct{}, error) {
   cmd := exec.Command("git", "ls-tree", "-r", "--name-only", "HEAD")
   out, err := cmd.Output()
   if err != nil {
      if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 128 {
         return make(map[string]struct{}), nil
      }
      return nil, fmt.Errorf("git ls-tree failed: %w", err)
   }
   paths := make(map[string]struct{})
   scanner := bufio.NewScanner(strings.NewReader(string(out)))
   for scanner.Scan() {
      paths[scanner.Text()] = struct{}{}
   }
   return paths, scanner.Err()
}

// buildReverseRenameMap uses an optimized git log command to find only renames.
func buildReverseRenameMap() (map[string]string, error) {
   cmd := exec.Command("git", "log", "--all", "-M50%", "--pretty=format:", "--name-status", "--diff-filter=R")
   out, err := cmd.Output()
   if err != nil {
      return nil, fmt.Errorf("git log for renames failed: %w", err)
   }
   renameMap := make(map[string]string)
   scanner := bufio.NewScanner(strings.NewReader(string(out)))
   for scanner.Scan() {
      fields := strings.Split(scanner.Text(), "\t")
      if len(fields) == 3 && strings.HasPrefix(fields[0], "R") {
         oldPath, newPath := fields[1], fields[2]
         renameMap[newPath] = oldPath
      }
   }
   return renameMap, scanner.Err()
}

// traceLiveLineages takes the current files and traces their entire history back.
// THIS FUNCTION IS NOW FIXED TO PREVENT INFINITE LOOPS.
func traceLiveLineages(currentPaths map[string]struct{}, reverseRenameMap map[string]string) map[string]struct{} {
   liveLineages := make(map[string]struct{})
   for path := range currentPaths {
      current := path
      for {
         // THE FIX: If we have already processed this path, we've found a cycle
         // or a common ancestor. We can stop tracing this branch.
         if _, exists := liveLineages[current]; exists {
            break
         }
         liveLineages[current] = struct{}{} // Add the current path to the set

         // Now, look for the older path
         if old, ok := reverseRenameMap[current]; ok {
            current = old // Continue tracing backwards
         } else {
            break // No older path found, end of this historical line.
         }
      }
   }
   return liveLineages
}

// printResults formats and prints the final list.
func printResults(orphanBlobs []BlobInfo) {
   if len(orphanBlobs) == 0 {
      fmt.Println("\nSuccess: No large files were found in your history that do not exist (including renames) in the current commit.")
      return
   }

   sort.Slice(orphanBlobs, func(i, j int) bool {
      return orphanBlobs[i].Path < orphanBlobs[j].Path
   })

   fmt.Println("\nFound large files in history whose lineage does not exist in the current commit:")
   w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
   fmt.Fprintln(w, "SHA\tSIZE (BYTES)\tORIGINAL PATH")
   fmt.Fprintln(w, "---\t------------\t-------------")
   for _, blob := range orphanBlobs {
      fmt.Fprintf(w, "%s\t%d\t%s\n", blob.Sha, blob.Size, blob.Path)
   }
   w.Flush()
}
