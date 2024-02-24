package main

import (
   "flag"
   "fmt"
   "time"
)

func main() {
   duration := 15 * time.Minute
   flag.DurationVar(&duration, "d", duration, "duration")
   var raw_from string
   flag.StringVar(&raw_from, "f", "", "from")
   flag.Parse()
   from, err := func() (time.Time, error) {
      if raw_from != "" {
         return time.Parse(time.Kitchen, raw_from)
      }
      return time.Now(), nil
   }()
   if err != nil {
      panic(err)
   }
   to := from.Add(duration)
   fmt.Println(from.Format(time.Kitchen))
   fmt.Println(to.Format(time.Kitchen))
}
