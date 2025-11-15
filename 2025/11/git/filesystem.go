package git

import (
   "bytes"
   "compress/zlib"
   "crypto/sha1"
   "fmt"
   "io"
   "os"
   "path/filepath"
   "strconv"
)

// WriteLooseObject takes raw object data, compresses it, and writes it to the object database.
func WriteLooseObject(directory, objType string, data []byte) (string, error) {
   content := []byte(fmt.Sprintf("%s %d\x00", objType, len(data)))
   content = append(content, data...)

   hash := sha1.Sum(content)
   sha := fmt.Sprintf("%x", hash)

   var b bytes.Buffer
   w := zlib.NewWriter(&b)
   w.Write(content)
   w.Close()

   objPath := filepath.Join(directory, ".git", "objects", sha[:2], sha[2:])
   if err := os.MkdirAll(filepath.Dir(objPath), 0755); err != nil {
      return "", err
   }
   return sha, os.WriteFile(objPath, b.Bytes(), 0644)
}

// ReadLooseObject finds an object by its SHA, reads and decompresses it.
func ReadLooseObject(directory, sha string) (data []byte, objType string, err error) {
   path := filepath.Join(directory, ".git", "objects", sha[:2], sha[2:])
   f, err := os.Open(path)
   if err != nil {
      return nil, "", err
   }
   defer f.Close()

   zr, err := zlib.NewReader(f)
   if err != nil {
      return nil, "", err
   }
   defer zr.Close()

   content, err := io.ReadAll(zr)
   if err != nil {
      return nil, "", err
   }

   nullByteIndex := bytes.IndexByte(content, 0)
   header := content[:nullByteIndex]
   data = content[nullByteIndex+1:]

   var size int
   fmt.Sscanf(string(header), "%s %d", &objType, &size)

   if len(data) != size {
      return nil, "", fmt.Errorf("object size mismatch for %s: expected %d, got %d", sha, size, len(data))
   }

   return data, objType, nil
}
// PktLine represents a single line in the pkt-line format.
type PktLine struct {
   Length  int
   Payload []byte
}

// ReadPktLine reads a single pkt-line from an io.Reader.
func ReadPktLine(r io.Reader) (*PktLine, error) {
   lenBuf := make([]byte, 4)
   if _, err := io.ReadFull(r, lenBuf); err != nil {
      return nil, err
   }

   length, err := strconv.ParseInt(string(lenBuf), 16, 16)
   if err != nil {
      return nil, err
   }

   if length == 0 {
      return &PktLine{Length: 0, Payload: nil}, nil // Flush packet
   }

   payload := make([]byte, length-4)
   if _, err := io.ReadFull(r, payload); err != nil {
      return nil, err
   }

   return &PktLine{
      Length:  int(length),
      Payload: payload,
   }, nil
}
// WriteRepo creates the basic .git directory structure.
func WriteRepo(directory string, headSymRef string, refs map[string]string) error {
   gitDir := filepath.Join(directory, ".git")
   refsDir := filepath.Join(gitDir, "refs")

   if err := os.MkdirAll(filepath.Join(gitDir, "objects"), 0755); err != nil {
      return err
   }
   if err := os.MkdirAll(filepath.Join(refsDir, "heads"), 0755); err != nil {
      return err
   }
   if err := os.MkdirAll(filepath.Join(refsDir, "tags"), 0755); err != nil {
      return err
   }

   // Write HEAD -> points to the default branch (e.g., "ref: refs/heads/master")
   if headSymRef == "" {
      return fmt.Errorf("cannot write HEAD file: symbolic ref is empty")
   }
   headContent := fmt.Sprintf("ref: %s", headSymRef)
   if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte(headContent), 0644); err != nil {
      return err
   }

   // Write refs
   for refName, hash := range refs {
      // Ensure the directory for the ref exists (e.g., for refs/pull/1/head)
      refPath := filepath.Join(gitDir, refName)
      if err := os.MkdirAll(filepath.Dir(refPath), 0755); err != nil {
         return err
      }
      // Write the hash to the ref file. Note: Git also packs refs for efficiency.
      if err := os.WriteFile(refPath, []byte(hash), 0644); err != nil {
         return err
      }
   }

   return nil
}
