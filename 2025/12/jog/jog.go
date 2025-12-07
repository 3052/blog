package main

import (
   "fmt"
   "time"
)

var events = []jog{
   {
      days_before: 9,
      day_of: date(2025, 12, 16),
      ok: nil,
   },
   {
      days_before: 8,
      day_of: date(2025, 12, 7),
      ok: nil,
   },
   {
      days_before: 7,
      day_of: date(2025, 11, 29),
      ok: some(false),
   },
}

func main() {
   for i, event := range events {
      if i >= 1 {
         fmt.Println()
      }
      fmt.Println(&event)
   }
}

func (j *jog) String() string {
   var data []byte
   data = fmt.Appendln(data, "days before =", j.days_before)
   data = fmt.Appendln(data, "day of =", j.day_of.Format(time.DateOnly))
   if j.ok != nil {
      data = fmt.Append(data, "ok = ", *j.ok)
   } else {
      data = append(data, "ok = nil"...)
   }
   return string(data)
}

type jog struct {
   days_before int
   day_of      time.Time
   ok          *bool
}

func date(year int, month time.Month, day int) time.Time {
   return time.Date(year, month, day, 0, 0, 0, 0, time.Local)
}

func some(value bool) *bool {
   return &value
}
