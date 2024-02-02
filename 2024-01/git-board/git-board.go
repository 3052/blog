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
const (
   fail = "\x1b[30;101m Fail \x1b[m"
   pass = "\x1b[30;102m Pass \x1b[m"
)

const format =
   "{{ .Add_Status }} additions\ttarget:99\tactual:{{ .Add }}\n" +
   "{{ .Delete_Status }} deletions\ttarget:99\tactual:{{ .Delete }}\n" +
   "{{ .Change_Status }} changed files\ttarget:99\tactual:{{ .Change }}\n" +
   "{{ .Date_Status }} last commit\ttarget:{{ .Then }}\tactual:{{ .Now }}\n"

type git_board struct {
   Add int
   Add_Status string
   Delete int
   Delete_Status string
   Change int
   Change_Status string
   Then string
   Now string
   Date_Status string
}

func lines(r rune) bool {
   return r == '\n'
}
