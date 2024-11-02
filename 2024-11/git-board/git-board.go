package main

import (
   "fmt"
   "os"
   "os/exec"
   "strings"
   "text/template"
   "time"
)

func main() {
   board, err := new_git_board()
   if err != nil {
      panic(err)
   }
   temp, err := new(template.Template).Parse(format)
   if err != nil {
      panic(err)
   }
   if err := temp.Execute(os.Stdout, board); err != nil {
      panic(err)
   }
}

func get_then() (string, error) {
   cmd := exec.Command("git", "log", "-1", "--format=%cI")
   fmt.Println(cmd.Args)
   text, err := cmd.Output()
   if err != nil {
      return "", err
   }
   if len(text) >= 11 {
      text = text[:10]
   }
   return string(text), nil
}

func new_git_board() (*git_board, error) {
   cmd := exec.Command("git", "add", ".")
   fmt.Println(cmd.Args)
   err := cmd.Run()
   if err != nil {
      return nil, err
   }
   cmd = exec.Command("git", "diff", "--cached", "--numstat")
   fmt.Println(cmd.Args)
   text, err := cmd.Output()
   if err != nil {
      return nil, err
   }
   var board git_board
   // split fails on empty string
   for _, line := range strings.FieldsFunc(string(text), lines) {
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
      board.AddStatus = pass
   } else {
      board.AddStatus = fail
   }
   if board.Delete >= 99 {
      board.DeleteStatus = pass
   } else {
      board.DeleteStatus = fail
   }
   if board.Change >= 99 {
      board.ChangeStatus = pass
   } else {
      board.ChangeStatus = fail
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
      board.DateStatus = pass
   } else {
      board.DateStatus = fail
   }
   return &board, nil
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
