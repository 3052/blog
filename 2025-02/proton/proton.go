package main

import (
   "flag"
   "fmt"
   "io"
   "net/http"
   "net/url"
   "strconv"
)

func main() {
   name := flag.Int("n", 0, "name")
   flag.Parse()
   if *name <= 0 {
      flag.Usage()
      return
   }
   var req http.Request
   req.Header = http.Header{}
   req.URL = &url.URL{}
   req.URL.Host = "account.proton.me"
   req.URL.Path = "/api/core/v4/users/available"
   value := url.Values{}
   req.URL.Scheme = "http"
   req.Header["X-Pm-Appversion"] = []string{"web-account@5.0.212.3"}
   req.Header["X-Pm-Uid"] = []string{"r6a7hvax2jkxgqgnggf5u43sogmxxml5"}
   req.Header["Cookie"] = []string{
      "AUTH-r6a7hvax2jkxgqgnggf5u43sogmxxml5=dfwm7bv2q5lybkvrbovkwi3eayjiqwhb",
   }
   value["ParseDomain"] = []string{"1"}
   for {
      value["Name"] = []string{
         strconv.Itoa(*name) + "@proton.me",
      }
      req.URL.RawQuery = value.Encode()
      resp := func() *http.Response {
         resp, err := http.DefaultClient.Do(&req)
         if err != nil {
            panic(err)
         }
         defer resp.Body.Close()
         _, err = io.Copy(io.Discard, resp.Body)
         if err != nil {
            panic(err)
         }
         return resp
      }()
      fmt.Println(resp.Status, *name)
      if resp.StatusCode != 409 {
         break
      }
      *name++
   }
}
