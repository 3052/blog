package main

import (
   "fmt"
   "os"
   "strconv"
   "time"
)

func unix(year int) int64 {
   return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
}

func main() {
   if len(os.Args) == 2 {
      year, err := strconv.Atoi(os.Args[1])
      if err != nil {
         panic(err)
      }
      fmt.Println(unix(year))
      fmt.Println(unix(year + 1))
   } else {
      fmt.Println("year [year]")
   }
}
