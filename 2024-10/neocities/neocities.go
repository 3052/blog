package main

import (
   "fmt"
   "io"
   "net/http"
   "net/url"
   "os"
   "strings"
   "time"
)

func main() {
   var req http.Request
   req.Header = http.Header{}
   req.Method = "POST"
   req.ProtoMajor = 1
   req.ProtoMinor = 1
   req.URL = &url.URL{}
   req.URL.Host = "neocities.org"
   req.URL.Path = "/create_validate"
   req.URL.Scheme = "https"
   req.Header["Cookie"] = []string{
      "neocities=BAh7CEkiD3Nlc3Npb25faWQGOgZFVG86HVJhY2s6OlNlc3Npb246OlNlc3Npb25JZAY6D0BwdWJsaWNfaWRJIkU0Y2M4ODFlMjE2NjgxZmNiMGUzYjNmNmQ3NGU0ZjljZmExOWIzNDhhYWJkZDA5MGQ5NTQxNTUxNmIxM2E2ODIzBjsARkkiEF9jc3JmX3Rva2VuBjsARkkiMWMrdEJOVmNVVHlHeXkrR1NWU2drMWR1V2FiUE1nVU9FSHBoa0YxcmtCd1U9BjsARkkiCmZsYXNoBjsARnsA--074c343d7ad303ce0b6ab0a73b6ce4f366a2d97f",
   }
   body := url.Values{
      "field":[]string{"username"},
      "csrf_token":[]string{"c+tBNVcUTyGyy+GSVSgk1duWabPMgUOEHphkF1rkBwU="},
   }
   for i := 0; i <= 99; i++ {
      body.Set("value", fmt.Sprint(i))
      req.Body = io.NopCloser(strings.NewReader(body.Encode()))
      func() {
         resp, err := http.DefaultClient.Do(&req)
         if err != nil {
            panic(err)
         }
         defer resp.Body.Close()
         os.Stdout.ReadFrom(resp.Body)
         fmt.Println(i)
      }()
      time.Sleep(time.Second)
   }
}
