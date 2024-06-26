package main

import (
   "fmt"
   "os/exec"
   "strings"
   "time"
)

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
