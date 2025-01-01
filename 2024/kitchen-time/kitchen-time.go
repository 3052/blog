package main

import (
   "flag"
   "fmt"
   "time"
)

type kitchen struct {
   time time.Time
}

func (k kitchen) String() string {
   return k.time.Format(time.Kitchen)
}

func (k *kitchen) Set(data string) error {
   var err error
   k.time, err = time.Parse(time.Kitchen, data)
   if err != nil {
      return err
   }
   return nil
}

func main() {
   duration := 15 * time.Minute
   flag.DurationVar(&duration, "d", duration, "duration")
   var from kitchen
   from.time = time.Now()
   flag.Var(&from, "f", "from")
   flag.Parse()
   fmt.Println(from)
   var to kitchen
   to.time = from.time.Add(duration)
   fmt.Println(to)
}
