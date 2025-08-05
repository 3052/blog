package main

import (
   "errors"
   "fmt"
   "os"
   "path/filepath"
   "slices"
   "strings"
   "time"
)

func main() {
   readmes, err := filepath.Glob("*/readme.md")
   if err != nil {
      panic(err)
   }
   for i, readme := range readmes {
      if i >= 1 {
         fmt.Println()
      }
      err := read_file(readme)
      if err != nil {
         panic(err)
      }
   }
}

func read_file(name string) error {
   data, err := os.ReadFile(name)
   if err != nil {
      return err
   }
   var durations []time.Duration
   for _, line := range strings.Split(string(data), "\n") {
      if strings.HasPrefix(line, "## ") {
         var ok bool
         _, line, ok = strings.Cut(line, ", ")
         if !ok {
            return errors.New("strings.Cut")
         }
         duration, err := time.ParseDuration(line)
         if err != nil {
            return err
         }
         durations = append(durations, duration)
      }
   }
   fmt.Printf("## %v\n", name)
   fmt.Printf("- %v prompts\n", len(durations))
   fmt.Printf("- median is %v\n", get_median(durations))
   fmt.Printf("- sum is %v\n", get_sum(durations))
   return nil
}

func get_sum(values []time.Duration) time.Duration {
   var sum time.Duration
   for _, value := range values {
      sum += value
   }
   return sum
}

func get_median(values []time.Duration) time.Duration {
   // Sort the input slice directly.
   slices.Sort(values)
   size := len(values)
   if size%2 == 0 {
      // Even number of elements, take the average of the two middle values.
      return (values[size/2-1] + values[size/2]) / 2
   }
   // Odd number of elements, take the middle value
   return values[size/2]
}
