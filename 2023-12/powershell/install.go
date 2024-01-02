package main

import (
   "os"
   "path/filepath"
)

var names = []string{
   `C:\Users\Steven\ripgrep.txt`,
   `C:\Users\Steven\AppData\Local\Microsoft\Windows Terminal\settings.json`,
   `C:\Users\Steven\Documents\PowerShell\Microsoft.PowerShell_profile.ps1`,
}

func main() {
   for _, name := range names {
      text, err := os.ReadFile(filepath.Base(name))
      if err != nil {
         panic(err)
      }
      if err := os.WriteFile(name, text, os.ModePerm); err != nil {
         panic(err)
      }
   }
}
