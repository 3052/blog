package main

import (
   "fmt"
   "time"
)

func main() {
   now := time.Now()
   hours := time.Date(2024, 3, 1, 0, 0, 0, 0, time.Local).Sub(now).Hours()
   days := hours/24
   fmt.Println("days", days)
   weeks := days/7
   fmt.Println("weeks", weeks)
}
