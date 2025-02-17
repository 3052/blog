package main

import (
   "github.com/BurntSushi/toml"
   "fmt"
   "os"
   "path/filepath"
   "strings"
)

func (u *userinfo) String() string {
   var b strings.Builder
   b.WriteString(u.Username)
   if u.Password != "" {
      b.WriteByte(':') // if this becomes a problem just use tab
      b.WriteString(u.Password)
   }
   return b.String()
}

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
      for key, value := range infos {
         fmt.Println(key, &value)
      }
   }
}

type userinfo struct {
   Password string
   Username string
}
