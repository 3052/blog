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
   "{{ .AddStatus }} additions\ttarget:99\tactual:{{ .Add }}\n" +
   "{{ .DeleteStatus }} deletions\ttarget:99\tactual:{{ .Delete }}\n" +
   "{{ .ChangeStatus }} changed files\ttarget:99\tactual:{{ .Change }}\n" +
   "{{ .DateStatus }} last commit\ttarget:{{ .Then }}\tactual:{{ .Now }}\n"

type git_board struct {
   Add int
   AddStatus string
   Delete int
   DeleteStatus string
   Change int
   ChangeStatus string
   Then string
   Now string
   DateStatus string
}

func lines(r rune) bool {
   return r == '\n'
}
