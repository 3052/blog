package main

import (
   "os"
   "text/template"
)

func main() {
   board, err := new_git_board()
   if err != nil {
      panic(err)
   }
   tem, err := new(template.Template).Parse(format)
   if err != nil {
      panic(err)
   }
   if err := tem.Execute(os.Stdout, board); err != nil {
      panic(err)
   }
}
