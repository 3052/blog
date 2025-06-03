package main

import (
   "bytes"
   "fmt"
   "io/fs"
   "os"
   "path/filepath"
)

func walk_dir(path string, _ fs.DirEntry, err error) error {
   if err != nil {
      return err
   }
   if filepath.Ext(path) != ".go" {
      return nil
   }
   fmt.Printf("%q\n", path)
   data, err := os.ReadFile(path)
   if err != nil {
      return err
   }
   data = bytes.ReplaceAll(
      data, []byte{'\t'}, []byte("   "),
   )
   return os.WriteFile(path, data, os.ModePerm)
}

func main() {
   err := filepath.WalkDir(".", walk_dir)
   if err != nil {
      panic(err)
   }
}
