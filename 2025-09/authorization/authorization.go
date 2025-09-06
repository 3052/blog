package main

import (
   "flag"
   "fmt"
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
   data, err := os.ReadFile(filepath.Join(home, "authorization.json"))
   if err != nil {
      return err
   }
   return json.Unmarshal(data, h)
}

type hosts map[string][]userinfo

type userinfo map[string]string

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
   host := flag.String("h", "", "host")
   flag.Parse()
   if *host != "" {
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
   } else {
      flag.Usage()
   }
}
