package main

import (
   "embed"
   "fmt"
   "html/template"
   "net/http"
   "net/url"
   "strings"
   "time"
)

func (v values) day(s string) []string {
   value, ok := v.Values[s]
   if ok {
      return value
   }
   return make([]string, 4)
}

//go:embed time-clock.html
var content embed.FS

func main() {
   index, err := template.ParseFS(content, "time-clock.html")
   if err != nil {
      panic(err)
   }
   http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
      if req.URL.Path != "/" {
         http.NotFound(w, req)
      } else {
         w.Header().Set("Content-Type", "text/html")
         switch req.Method {
         case "GET":
            form := new_values()
            for _, cookie := range req.Cookies() {
               form.Values[cookie.Name] = strings.Split(cookie.Value, ",")
            }
            err := index.Execute(w, form)
            if err != nil {
               fmt.Println(err)
            }
         case "POST":
            req.ParseForm()
            for key, values := range req.Form {
               w.Header().Add(
                  "Set-Cookie", key + "=" + strings.Join(values, ","),
               )
            }
            err := index.Execute(w, values{req.Form})
            if err != nil {
               fmt.Println(err)
            }
         }
      }
   })
   fmt.Println("localhost:99")
   http.ListenAndServe(":99", nil)
}

func punch(s string) (time.Time, error) {
   return time.Parse("15:04", s)
}

type values struct {
   url.Values
}

func new_values() values {
   var v values
   v.Values = make(url.Values)
   return v
}

func (v values) Friday() []string {
   return v.day("fri")
}

func (v values) Monday() []string {
   return v.day("mon")
}

func (v values) Thursday() []string {
   return v.day("thu")
}

func (v values) Total() time.Duration {
   var dur time.Duration
   for _, punches := range v.Values {
      if punches[0] != "" {
         in_day, err := punch(punches[0])
         if err != nil {
            return 0
         }
         out_day, err := punch(punches[3])
         if err != nil {
            return 0
         }
         if punches[1] != "" {
            out_lunch, err := punch(punches[1])
            if err != nil {
               return 0
            }
            in_lunch, err := punch(punches[2])
            if err != nil {
               return 0
            }
            dur += out_lunch.Sub(in_day)
            dur += out_day.Sub(in_lunch)
         } else {
            dur += out_day.Sub(in_day)
         }
      }
   }
   return dur
}

func (v values) Tuesday() []string {
   return v.day("tue")
}

func (v values) Wednesday() []string {
   return v.day("wed")
}
