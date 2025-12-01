package main

import (
   "net/http"
   "net/url"
   "os"
)

func main() {
   var req http.Request
   req.Header = http.Header{}
   req.URL = &url.URL{}
   req.URL.Host = "b42189-p34-h51-v0-afkqiswu-t534310.1.cdn.telefonica.com"
   req.URL.Path = "/_42189/prod/dash/cplus-3427440-md-03_cplus-3427440-mdrm_s4my8zabfhof8ns/manifest.mpd"
   req.URL.Scheme = "https"
   req.Header.Add("X-Tcdn-Token", "eyJhbGciOiJFUzI1NiIsImtpZCI6ImI1OGNhNGM0NGFiOTQ0Y2FiY2U4N2FjNGJmZmI4MDNkIiwidHlwIjoiYXQrand0In0.eyJuYmYiOjE3NTE4NDQ4NDEsImV4cCI6MTc1MTkzMTI0MSwiaXNzIjoiaHR0cHM6Ly9pZHNlcnZlci5kb2Y2LmNvbSIsImF1ZCI6InRjZG4iLCJjbGllbnRfaWQiOiJtb3Zpc3RhcnBsdXMiLCJzdWIiOiI3VTdlN3Y4QjhTOGg4bzlBIiwiYXV0aF90aW1lIjoxNzUxODQ0ODQxLCJpZHAiOiJtb3Zpc3RhcisiLCJ1aWQiOiIvK0drKzRvZFJmc204d01rcFFta2pEczFPUEI0aGxHMTd1SGxiV2tmcWRvPSIsImFjYyI6IndmbWMrWGc2Wk01bExPMHlUSS9oN2x2WHlscDU5Z1U1OFNoN29WQUE3TkU9IiwianRpIjoiRTMwRkQ2MTJDQUJDOTdEMzdERURCODA2QkZENkY1NjkiLCJpYXQiOjE3NTE4NDQ4NDEsInNjb3BlIjoiY2RuIn0.HgmhwthAp3ooc6lq0UvUy8cekKm7VMniCHUJhdg68K-HcsZCV6toPAwiu-3PJ3o8mAWdMxwREfkLRwy7GegH4Q")
   resp, err := http.DefaultClient.Do(&req)
   if err != nil {
      panic(err)
   }
   err = resp.Write(os.Stdout)
   if err != nil {
      panic(err)
   }
}
