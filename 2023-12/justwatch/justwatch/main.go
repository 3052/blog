package main

import (
   "154.pages.dev/log"
   "flag"
   "time"
)

type flags struct {
   address string
   level log.Level
   sleep time.Duration
}

func main() {
   var f flags
   flag.StringVar(&f.address, "a", "", "address")
   flag.DurationVar(&f.sleep, "s", 99*time.Millisecond, "sleep")
   flag.TextVar(&f.level, "v", f.level, "log level")
   flag.Parse()
   log.Set_Logger(f.level)
   if f.address != "" {
      err := f.stream()
      if err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}
