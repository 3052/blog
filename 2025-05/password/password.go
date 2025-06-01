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

func (h *hosts) New() error {
   home, err := os.UserHomeDir()
   if err != nil {
      return err
   }
   data, err := os.ReadFile(filepath.Join(home, "password.toml"))
   if err != nil {
      return err
   }
   return toml.Unmarshal(data, h)
}

type hosts map[string][]info

func main() {
   var host_info hosts
   err := host_info.New()
   if err != nil {
      panic(err)
   }
   const day = 24 * time.Hour
   for host, users := range host_info {
      for _, user := range users {
         if time.Since(user.Date) >= 365*day {
            fmt.Println("time.Since(user.Date) >= 365*day")
            fmt.Println(get_lines(host, &user))
            return
         }
      }
   }
   url := flag.String("u", "", "URL")
   userinfo := flag.String("i", "", "userinfo")
   verbose := flag.String("v", "", "verbose")
   flag.Parse()
   switch {
   case *url != "":
      host_info.contains(*url, true)
   case *userinfo != "":
      host_info.equal(*userinfo)
   case *verbose != "":
      host_info.contains(*verbose, false)
   default:
      flag.Usage()
   }
}

func (h hosts) equal(host string) {
   values, ok := h[host]
   if ok {
      fmt.Print(&values[0])
   }
}

func (h hosts) contains(data string, url bool) {
   var line bool
   for host, users := range h {
      if strings.Contains(host, data) {
         for _, user := range users {
            if url {
               fmt.Println(get_line(host, &user))
            } else {
               if line {
                  fmt.Println()
               } else {
                  line = true
               }
               fmt.Println(get_lines(host, &user))
            }
         }
      }
   }
}

func (i *info) String() string {
   var b strings.Builder
   b.WriteString(i.Username)
   b.WriteByte(':')
   b.WriteString(i.Password)
   return b.String()
}

func get_line(host string, user *info) string {
   var b strings.Builder
   b.WriteString("//")
   b.WriteString(user.Username)
   b.WriteByte(':')
   b.WriteString(user.Password)
   b.WriteByte('@')
   b.WriteString(host)
   return b.String()
}

type info struct {
   Date     time.Time
   Password string
   Username string
}

func get_lines(host string, user *info) string {
   var b []byte
   b = append(b, "host = "...)
   b = append(b, host...)
   b = append(b, "\nusername = "...)
   b = append(b, user.Username...)
   b = append(b, "\npassword = "...)
   b = append(b, user.Password...)
   b = append(b, "\ndate = "...)
   b = user.Date.AppendFormat(b, time.DateOnly)
   return string(b)
}
