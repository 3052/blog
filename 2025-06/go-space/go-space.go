package main

import (
   "bytes"
   "fmt"
   "os"
   "path/filepath"
)

func main() {
   names, err := filepath.Glob("*.go")
   if err != nil {
      panic(err)
   }
   for _, name := range names {
      fmt.Printf("%q\n", name)
      data, err := os.ReadFile(name)
      if err != nil {
         panic(err)
      }
      data = bytes.ReplaceAll(
         data, []byte{'\t'}, []byte("   "),
      )
      err = os.WriteFile(name, data, os.ModePerm)
      if err != nil {
         panic(err)
      }
   }
}
