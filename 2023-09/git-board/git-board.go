package main

import (
   "fmt"
   "os"
   "os/exec"
   "strings"
   "text/template"
   "time"
)

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

func new_git_board() (*git_board, error) {
   var board git_board
   err := exec.Command("git", "add", ".").Run()
   if err != nil {
      return nil, err
   }
   buf, err := exec.Command("git", "diff", "--cached", "--numstat").Output()
   if err != nil {
      return nil, err
   }
   // split fails on empty string
   for _, line := range strings.FieldsFunc(string(buf), lines) {
      var add, del int
      // binary files will be "- - hello.txt", so ignore error
      fmt.Sscan(line, &add, &del)
      // Add
      board.Add += add
      // Delete
      board.Delete += del
      // Change
      board.Change++
   }
   if board.Add >= 99 {
      board.Add_Status = pass
   } else {
      board.Add_Status = fail
   }
   if board.Delete >= 99 {
      board.Delete_Status = pass
   } else {
      board.Delete_Status = fail
   }
   if board.Change >= 99 {
      board.Change_Status = pass
   } else {
      board.Change_Status = fail
   }
   // Then
   then, err := get_then()
   if err != nil {
      return nil, err
   }
   board.Then = then
   // Now
   board.Now = time.Now().AddDate(0, 0, -1).String()[:10]
   if board.Then <= board.Now {
      board.Date_Status = pass
   } else {
      board.Date_Status = fail
   }
   return &board, nil
}

func get_then() (string, error) {
   buf, err := exec.Command("git", "log", "-1", "--format=%cI").Output()
   if err != nil {
      return "", err
   }
   if len(buf) >= 11 {
      buf = buf[:10]
   }
   return string(buf), nil
}
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
