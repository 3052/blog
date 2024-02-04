package main

import (
   "fmt"
   "os"
   "strings"
)

func main() {
   if len(os.Args) == 2 {
      prompt := os.Args[1]
      switch {
      case strings.HasPrefix(prompt, "Username"):
         fmt.Fprintln(os.Stderr, "Username")
         fmt.Println(os.Getenv("github_username"))
      case strings.HasPrefix(prompt, "Password"):
         fmt.Fprintln(os.Stderr, "Password")
         fmt.Println(os.Getenv("github_password"))
      }
   } else {
      fmt.Println("git-askpass [Username|Password]")
   }
}
