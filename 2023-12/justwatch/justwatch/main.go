package main

import (
   "flag"
   "time"
)

type flags struct {
   address string
   language string
   sleep time.Duration
}

func main() {
   var f flags
   flag.StringVar(&f.address, "a", "", "address")
   flag.StringVar(&f.language, "lang", "en", "language")
   flag.DurationVar(&f.sleep, "s", 99*time.Millisecond, "sleep")
   flag.Parse()
   if f.address != "" {
      err := f.stream()
      if err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}
