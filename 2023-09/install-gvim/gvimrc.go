package main

import "os"

func do_gvimrc(home string) error {
   text, err := os.ReadFile("_gvimrc")
   if err != nil {
      return err
   }
   return os.WriteFile(home + "/_gvimrc", text, 0666)
}
