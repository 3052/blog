package main

import (
   "flag"
   "fmt"
   "github.com/BurntSushi/toml"
   "os"
   "path/filepath"
   "strings"
   "time"
)

func (u *user) New() error {
   home, err := os.UserHomeDir()
   if err != nil {
      return err
   }
   data, err := os.ReadFile(filepath.Join(home, "password.toml"))
   if err != nil {
      return err
   }
   return toml.Unmarshal(data, u)
}

func (i *info) String() string {
   b := []byte("username = ")
   b = append(b, i.Username...)
   b = append(b, "\npassword = "...)
   b = append(b, i.Password...)
   b = append(b, "\ndate = "...)
   b = i.Date.AppendFormat(b, time.DateOnly)
   return string(b)
}

func main() {
   var user1 user
   err := user1.New()
   if err != nil {
      panic(err)
   }
   const day = 24 * time.Hour
   for key, values := range user1 {
      for _, value := range values {
         if time.Since(value.Date) >= 365*day {
            fmt.Println("time.Since(value.Date) >= 365*day")
            fmt.Println(key)
            fmt.Println(&value)
            return
         }
      }
   }
   contains := flag.String("c", "", "contains")
   equal := flag.String("e", "", "equal")
   flag.Parse()
   switch {
   case *contains != "":
      user1.contains(*contains)
   case *equal != "":
      user1.equal(*equal)
   default:
      flag.Usage()
   }
}

func (u user) contains(data string) {
   var line bool
   for key, values := range u {
      if strings.Contains(key, data) {
         for _, value := range values {
            if line {
               fmt.Println()
            } else {
               line = true
            }
            fmt.Print(key, "\n", &value, "\n")
         }
      }
   }
}

type info struct {
   Date     time.Time
   Password string
   Username string
}

type user map[string][]info

func (u user) equal(data string) {
   values, ok := u[data]
   if ok {
      value := values[0]
      fmt.Print(value.Username)
      if value.Password != "" {
         // go.dev/pkg/net/url?m=old#PathEscape
         fmt.Print(":", value.Password)
      }
   }
}
