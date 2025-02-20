package main

import (
   "flag"
   "fmt"
   "io"
   "net/http"
   "net/url"
   "strconv"
   "strings"
)

func main() {
   name := flag.Int64("n", 0, "name")
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
   req.URL.Scheme = "https"
   req.Header["X-Pm-Appversion"] = []string{"web-account@5.0.212.3"}
   req.Header["X-Pm-Uid"] = []string{"u6y2uibht2f2c7qwbkxpiwos5vtbcqer"}
   req.Header["Cookie"] = []string{
      "AUTH-u6y2uibht2f2c7qwbkxpiwos5vtbcqer=ui7oc7wqxb3zubbd5lonreojsnisfom7",
   }
   value["ParseDomain"] = []string{"1"}
   for {
      name_hex := strconv.FormatInt(*name, 16)
      if valid(name_hex) {
         name_hex += "@proton.me"
         value["Name"] = []string{name_hex}
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
         fmt.Println(resp.Status, name_hex, *name)
         if resp.StatusCode != 409 {
            break
         }
      }
      *name++
   }
}

func valid(name_hex string) bool {
   if strings.Contains(name_hex, "a") {
      return true
   }
   if strings.Contains(name_hex, "b") {
      return true
   }
   if strings.Contains(name_hex, "c") {
      return true
   }
   if strings.Contains(name_hex, "d") {
      return true
   }
   if strings.Contains(name_hex, "e") {
      return true
   }
   if strings.Contains(name_hex, "f") {
      return true
   }
   return false
}
