package main

import (
   "encoding/json"
   "errors"
   "fmt"
   "os"
   "path/filepath"
   "slices"
   "strings"
   "time"
)

func get_json(name string) error {
   data, err := os.ReadFile(name)
   if err != nil {
      return err
   }
   var value struct {
      ChatBot string
      Developer string
      Model string
      Url string
   }
   err = json.Unmarshal(data, &value)
   if err != nil {
      return err
   }
   fmt.Printf("- URL: %v\n", value.Url)
   fmt.Printf("- chatBot: %v\n", value.ChatBot)
   fmt.Printf("- developer: %v\n", value.Developer)
   fmt.Printf("- model: %v\n", value.Model)
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

func get_md(name string) error {
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
   fmt.Printf("- %v prompts\n", len(durations))
   fmt.Printf("- median is %v\n", get_median(durations))
   fmt.Printf("- sum is %v\n", get_sum(durations))
   return nil
}

func main() {
   names, err := filepath.Glob("*/chatBot.json")
   if err != nil {
      panic(err)
   }
   for i, name := range names {
      if i >= 1 {
         fmt.Println()
      }
      err = get_json(name)
      if err != nil {
         panic(err)
      }
      dir := filepath.Dir(name)
      if get_go(dir + "/chatBot.go") == nil {
         err = get_md(dir + "/readme.md")
         if err != nil {
            panic(err)
         }
      }
   }
}

func get_go(name string) error {
   data, err := os.ReadFile(name)
   if err != nil {
      return err
   }
   var lines int
   for _, line := range strings.Split(string(data), "\n") {
      line = strings.TrimSpace(line)
      if line != ""  {
         if !strings.HasPrefix(line, "//") {
            lines++
         }
      }
   }
   fmt.Printf("- %v LOC\n", lines)
   return nil
}
