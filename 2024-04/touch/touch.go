package main

import (
   "fmt"
   "os"
   "time"
)

func main() {
   if len(os.Args) == 2 {
      now := time.Now()
      err := os.Chtimes(os.Args[1], now, now)
      if err != nil {
         panic(err)
      }
   } else {
      fmt.Println("touch [entry]")
   }
}
