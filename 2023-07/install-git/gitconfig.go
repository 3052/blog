package main

import "os"

func do_config(home string) error {
   text, err := os.ReadFile(".gitconfig")
   if err != nil {
      return err
   }
   return os.WriteFile(home + "/.gitconfig", text, 0666)
}
