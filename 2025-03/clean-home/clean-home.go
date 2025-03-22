package main

import (
   "fmt"
   "os"
   "path/filepath"
)

var patterns = []string{
   `C:\ProgramData\Mozilla-*`,
   `C:\Users\Steven\.android`,
   `C:\Users\Steven\.cargo`,
   `C:\Users\Steven\AppData\Local\Android Open Source Project`,
   `C:\Users\Steven\AppData\Local\Android\Sdk\system-images\android-*`,
   `C:\Users\Steven\AppData\Local\Genymobile`,
   `C:\Users\Steven\AppData\Local\Google`,
   `C:\Users\Steven\AppData\Local\Mozilla\Firefox\Profiles\*.*\cache2`,
   `C:\Users\Steven\AppData\Local\go-build`,
   `C:\Users\Steven\AppData\Local\pip`,
   `C:\Users\Steven\go\pkg`,
}

func main() {
   for _, pattern := range patterns {
      matches, err := filepath.Glob(pattern)
      if err != nil {
         panic(err)
      }
      for _, match := range matches {
         fmt.Println(match)
         err := os.RemoveAll(match)
         if err != nil {
            panic(err)
         }
      }
   }
}
