package main

import (
   "encoding/csv"
   "encoding/json"
   "errors"
   "fmt"
   "os"
   "path/filepath"
   "slices"
   "strings"
   "time"
)

const pattern = "internal/*/chatBot.json"

func main() {
   names, err := filepath.Glob(pattern)
   if err != nil {
      panic(err)
   }
   file, err := os.Create("chatBot.csv")
   if err != nil {
      panic(err)
   }
   write := csv.NewWriter(file)
   err = write.Write(new(chatBot).header())
   if err != nil {
      panic(err)
   }
   for _, name := range names {
      var bot chatBot
      err = bot.get_json(name)
      if err != nil {
         panic(err)
      }
      if bot.Ok {
         dir := filepath.Dir(name)
         err = bot.get_md(dir + "/readme.md")
         if err != nil {
            panic(err)
         }
         err = bot.get_go(dir + "/chatBot.go")
         if err != nil {
            panic(err)
         }
      }
      err = write.Write(bot.record())
      if err != nil {
         panic(err)
      }
   }
   write.Flush()
   err = write.Error()
   if err != nil {
      panic(err)
   }
}

func (c *chatBot) get_md(name string) error {
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
   c.median = get_median(durations)
   c.prompts = len(durations)
   c.sum = get_sum(durations)
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

func (c *chatBot) get_json(name string) error {
   data, err := os.ReadFile(name)
   if err != nil {
      return err
   }
   return json.Unmarshal(data, c)
}

func (c *chatBot) get_go(name string) error {
   data, err := os.ReadFile(name)
   if err != nil {
      return err
   }
   var lines int
   for _, line := range strings.Split(string(data), "\n") {
      line = strings.TrimSpace(line)
      if line != "" {
         if !strings.HasPrefix(line, "//") {
            lines++
         }
      }
   }
   c.loc = lines
   return nil
}

type chatBot struct {
   Developer string
   ChatBot   string
   Model     string
   Url       string
   Ok        bool
   prompts   int
   median    time.Duration
   sum       time.Duration
   loc       int
}

func (*chatBot) header() []string {
   return []string{
      "developer",
      "chatbot",
      "model",
      "URL",
      "OK",
      "prompts",
      "median",
      "sum",
      "LOC",
   }
}

func (c *chatBot) record() []string {
   return []string{
      c.Developer,
      c.ChatBot,
      c.Model,
      c.Url,
      fmt.Sprint(c.Ok),
      fmt.Sprint(c.prompts),
      fmt.Sprint(c.median),
      fmt.Sprint(c.sum),
      fmt.Sprint(c.loc),
   }
}
