package github

import (
   "encoding/json"
   "os"
   "testing"
)

func Test_User(t *testing.T) {
   home, err := os.UserHomeDir()
   if err != nil {
      t.Fatal(err)
   }
   m, err := user_info(home + "/github.json")
   if err != nil {
      t.Fatal(err)
   }
   u := user{
      bio: "email srpen6@gmail.com, Discord srpen6",
      company: "looking for work",
      location: "Dallas",
      website: "https://discord.com/invite/WWq6rFb8Rf",
   }
   if err := u.update(m); err != nil {
      t.Fatal(err)
   }
}

func user_info(name string) (map[string]string, error) {
   b, err := os.ReadFile(name)
   if err != nil {
      return nil, err
   }
   var m map[string]string
   if err := json.Unmarshal(b, &m); err != nil {
      return nil, err
   }
   return m, nil
}
