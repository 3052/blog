package main

import (
   "encoding/json"
   "errors"
   "flag"
   "fmt"
   "os"
   "slices"
   "strings"
   "time"
)

func get_users() ([]userinfo, error) {
   data, err := os.ReadFile(name)
   if err != nil {
      return nil, err
   }
   var users []userinfo
   err = json.Unmarshal(data, &users)
   if err != nil {
      return nil, err
   }
   passwords := map[string]bool{}
   for _, user := range users {
      trial := user["trial"]
      password := user["password"]
      if trial == "true" {
         trial2, ok := passwords[password]
         if ok {
            if !trial2 {
               return nil, errors.New(password)
            }
         } else {
            passwords[password] = true
         }
      } else if trial == "false" {
         _, ok := passwords[password]
         if ok {
            return nil, errors.New(password)
         } else {
            passwords[password] = false
         }
      } else if password != "" {
         return nil, fmt.Errorf("%v", user)
      }
   }
   year_ago := time.Now().AddDate(-1, 0, 0).String()
   for _, user := range users {
      if user["date"] < year_ago {
         return nil, fmt.Errorf("%v", user)
      }
   }
   return users, nil
}

type userinfo map[string]string

const name = `D:\backblaze\largest\credential.json`

func main() {
   key := flag.String("k", "password", "key")
   host := flag.String("h", "", "host")
   user := flag.String("u", "", "user")
   contains := flag.String("c", "", "contains")
   flag.Parse()
   if *key == "password" {
      if *contains == "" {
         if *host == "" {
            flag.Usage()
            return
         }
      }
   }
   users, err := get_users()
   if err != nil {
      panic(err)
   }
   var line bool
   for _, user2 := range users {
      if *contains != "" {
         if strings.Contains(user2.String(), *contains) {
            if line {
               fmt.Println()
            } else {
               line = true
            }
            fmt.Println(user2)
         }
      } else {
         if user2[*key] == "" {
            continue
         }
         if *host != "" {
            if user2["host"] != *host {
               continue
            }
         }
         if *user != "" {
            if user2["user"] != *user {
               continue
            }
         }
         fmt.Print(user2[*key])
         return
      }
   }
}

func (u userinfo) String() string {
   keys := make([]string, 0, len(u))
   for key := range u {
      keys = append(keys, key)
   }
   slices.Sort(keys)
   var b strings.Builder
   for i, key := range keys {
      if i >= 1 {
         b.WriteByte('\n')
      }
      b.WriteString(key)
      b.WriteString(" = ")
      b.WriteString(u[key])
   }
   return b.String()
}
