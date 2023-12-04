package main

import (
   "154.pages.dev/log"
   "flag"
   "time"
)

type flags struct {
   address string
   h log.Handler
   sleep time.Duration
}

func main() {
   var f flags
   flag.StringVar(&f.address, "a", "", "address")
   flag.DurationVar(&f.sleep, "s", 99*time.Millisecond, "sleep")
   flag.TextVar(&f.h.Level, "level", f.h.Level, "level")
   flag.Parse()
   log.Set_Handler(f.h)
   if f.address != "" {
      err := f.stream()
      if err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}
