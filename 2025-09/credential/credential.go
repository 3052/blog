package main

import (
   "encoding/json"
   "fmt"
   "os"
   "path/filepath"
   "slices"
   "strings"
   "time"
)

func (u userinfo) String() string {
   var (
      b strings.Builder
      line bool
   )
   keys := make([]string, 0, len(u))
   for key := range u {
      keys = append(keys, key)
   }
   slices.Sort(keys)
   for _, key := range keys {
      if line {
         b.WriteByte('\n')
      } else {
         line = true
      }
      b.WriteString(key)
      b.WriteString(" = ")
      b.WriteString(u[key])
   }
   return b.String()
}

type userinfo map[string]string

func main() {
   users, err := get_users()
   if err != nil {
      panic(err)
   }
   switch len(os.Args) {
   case 2: // credential google.com
      host := os.Args[1]
      var line bool
      for _, user := range users {
         if strings.Contains(user["host"], host) {
            if line {
               fmt.Println()
            } else {
               line = true
            }
            fmt.Println(user)
         }
      }
   case 3: // credential google.com srpen6@gmail.com
      host, user := os.Args[1], os.Args[2]
      for _, user2 := range users {
         if user2["host"] == host {
            if user2["user"] == user {
               fmt.Print(user2["password"])
               return
            }
         }
      }
   default:
      fmt.Println("credential", "host")
      fmt.Println("credential", "host", "user")
   }
}

func get_users() ([]userinfo, error) {
   home, err := os.UserHomeDir()
   if err != nil {
      return nil, err
   }
   data, err := os.ReadFile(filepath.Join(home, "credential.json"))
   if err != nil {
      return nil, err
   }
   var users []userinfo
   err = json.Unmarshal(data, &users)
   if err != nil {
      return nil, err
   }
   year_ago := time.Now().AddDate(-1, 0, 0).String()
   for _, user := range users {
      if user["date"] < year_ago {
         return nil, fmt.Errorf("%v", user)
      }
   }
   return users, nil
}
