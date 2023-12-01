package main

import (
   "encoding/json"
   "fmt"
   "os"
   "strings"
)

func main() {
   m := func() map[string]string {
      s, err := os.UserHomeDir()
      if err != nil {
         panic(err)
      }
      b, err := os.ReadFile(s + "/github.json")
      if err != nil {
         panic(err)
      }
      var m map[string]string
      json.Unmarshal(b, &m)
      return m
   }()
   if len(os.Args) == 2 {
      prompt := os.Args[1]
      switch {
      case strings.HasPrefix(prompt, "Username"):
         fmt.Fprintln(os.Stderr, "Username")
         fmt.Println(m["username"])
      case strings.HasPrefix(prompt, "Password"):
         fmt.Fprintln(os.Stderr, "Password")
         fmt.Println(m["password"])
      }
   } else {
      fmt.Println("git-askpass [Username|Password]")
   }
}
