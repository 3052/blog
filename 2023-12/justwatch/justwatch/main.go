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
   flag.TextVar(&f.h.Level, "v", f.h.Level, "log level")
   flag.Parse()
   log.Set_Logger(f.h.Level)
   if f.address != "" {
      err := f.stream()
      if err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}
