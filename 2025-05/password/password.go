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

func main() {
   var user1 user
   err := user1.New()
   if err != nil {
      panic(err)
   }
   for key, value := range user1 {
      const day = 24 * time.Hour
      if time.Since(value.Date) >= 365*day {
         fmt.Println("time.Since(value.Date) >= 365*day")
         fmt.Println(key)
         fmt.Println(&value)
         return
         
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

type info struct {
   Date     time.Time
   Password string
   Username string
}

type user map[string]info

func (u user) equal(data string) {
   info1 := u[data]
   fmt.Print(info1.Username)
   if info1.Password != "" {
      // go.dev/pkg/net/url?m=old#PathEscape
      fmt.Print(":", info1.Password)
   }
}

func (u user) contains(data string) {
   var line bool
   for key, value := range u {
      if strings.Contains(key, data) {
         if line {
            fmt.Println()
         } else {
            line = true
         }
         fmt.Print(key, "\n", &value, "\n")
      }
   }
}
