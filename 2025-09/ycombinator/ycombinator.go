package main

import (
   "flag"
   "net/http"
   "net/url"
   "os"
   "strconv"
)

func post(acct int, pw string) error {
   resp, err := http.PostForm("https://news.ycombinator.com/login", url.Values{
      "acct":     {strconv.Itoa(acct)},
      "creating": {"t"},
      "pw":       {pw},
   })
   if err != nil {
      return err
   }
   return resp.Write(os.Stdout)
}

func main() {
   acct := flag.Int("a", 0, "acct")
   pw := flag.String("p", "", "pw")
   flag.Parse()
   var ok bool
   if *acct >= 1 {
      if *pw != "" {
         err := post(*acct, *pw)
         if err != nil {
            panic(err)
         }
         ok = true
      }
   }
   if !ok {
      flag.Usage()
   }
}
