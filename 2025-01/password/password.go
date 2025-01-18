package main

import (
   "github.com/BurntSushi/toml"
   "fmt"
   "os"
   "path/filepath"
   "slices"
   "strings"
)

func main() {
   home, err := os.UserHomeDir()
   if err != nil {
      panic(err)
   }
   data, err := os.ReadFile(filepath.Join(home, "password.toml"))
   if err != nil {
      panic(err)
   }
   var infos map[string]userinfo
   err = toml.Unmarshal(data, &infos)
   if err != nil {
      panic(err)
   }
   if len(os.Args) == 2 {
      info := infos[os.Args[1]]
      fmt.Print(&info) // no newline
   } else {
      keys := make([]string, 0, len(infos))
      for key := range infos {
         keys = append(keys, key)
      }
      slices.Sort(keys)
      for _, key := range keys {
         value := infos[key]
         fmt.Println(key, &value)
      }
   }
}

type userinfo struct {
   Password string
   Username string
}

func (u *userinfo) String() string {
   var data strings.Builder
   data.WriteString(u.Username)
   if u.Password != "" {
      data.WriteByte(':')
      data.WriteString(u.Password)
   }
   return data.String()
}
