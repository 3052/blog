package main

import (
   "flag"
   "os"
)

func main() {
   var git bool
   flag.BoolVar(&git, "g", false, "install Git")
   var config bool
   flag.BoolVar(&config, "c", false, "install Git config")
   flag.Parse()
   home, err := os.UserHomeDir()
   if err != nil {
      panic(err)
   }
   switch {
   case config:
      err := do_config(home)
      if err != nil {
         panic(err)
      }
   case git:
      err := do_git(home)
      if err != nil {
         panic(err)
      }
   default:
      flag.Usage()
   }
}
