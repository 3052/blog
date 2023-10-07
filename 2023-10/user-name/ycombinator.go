package main

import (
   "bytes"
   "fmt"
   "io"
   "net/http"
   "net/url"
   "os"
   "slices"
   "strings"
   "time"
)

func main() {
   req := new(http.Request)
   req.Header = make(http.Header)
   req.Header["Accept"] = []string{"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"}
   req.Header["Accept-Encoding"] = []string{"identity"}
   req.Header["Accept-Language"] = []string{"en-US,en;q=0.5"}
   req.Header["Connection"] = []string{"keep-alive"}
   req.Header["Content-Type"] = []string{"application/x-www-form-urlencoded"}
   req.Header["Host"] = []string{"news.ycombinator.com"}
   req.Header["Origin"] = []string{"https://news.ycombinator.com"}
   req.Header["Referer"] = []string{"https://news.ycombinator.com/"}
   req.Header["Sec-Fetch-Dest"] = []string{"document"}
   req.Header["Sec-Fetch-Mode"] = []string{"navigate"}
   req.Header["Sec-Fetch-Site"] = []string{"same-origin"}
   req.Header["Sec-Fetch-User"] = []string{"?1"}
   req.Header["Upgrade-Insecure-Requests"] = []string{"1"}
   req.Header["User-Agent"] = []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:101.0) Gecko/20100101 Firefox/101.0"}
   req.Method = "POST"
   req.URL = new(url.URL)
   req.URL.Host = "news.ycombinator.com"
   req.URL.Path = "/login"
   req.URL.Scheme = "https"
   i := 20
   for {
      i++
      before := fmt.Sprint(i)
      after := []byte(before)
      slices.Sort(after)
      after = slices.Compact(after)
      if len(after) < len(before) {
         continue
      }
      {
         s := url.Values{
            "acct":[]string{before}, "creating":[]string{"t"},
            "goto":[]string{"item?id=36700656"}, "pw":[]string{os.Args[1]},
            "switch":[]string{"register"},
         }.Encode()
         req.Body = io.NopCloser(strings.NewReader(s))
      }
      fmt.Println(i)
      res, err := http.DefaultTransport.RoundTrip(req)
      if err != nil {
         panic(err)
      }
      if res.StatusCode != http.StatusOK {
         panic(res.Status)
      }
      body, err := io.ReadAll(res.Body)
      if err != nil {
         panic(err)
      }
      if err := res.Body.Close(); err != nil {
         panic(err)
      }
      if !bytes.Contains(body, []byte("That username is taken")) {
         break
      }
      time.Sleep(time.Second)   
   }
}
