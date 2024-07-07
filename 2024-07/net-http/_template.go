package main

import (
   "io"
   "net/http"
   "net/url"
   "os"
   "strings"
)

func main() {
   var req http.Request
   req.Header = make(http.Header)
   {{ range $key, $val := .Header -}}
      req.Header[{{ printf "%q" $key }}] = {{ printf "%#v" $val }}
   {{ end -}}
   req.Method = {{ printf "%q" .Method }}
   req.ProtoMajor = 1
   req.ProtoMinor = 1
   req.URL = new(url.URL)
   req.URL.Host = {{ printf "%q" .URL.Host }}
   req.URL.Path = {{ printf "%q" .URL.Path }}
   req.URL.RawPath = {{ printf "%q" .URL.RawPath }}
   val := make(url.Values)
   {{ range $key, $val := .Query -}}
      val[{{ printf "%q" $key }}] = {{ printf "%#v" $val }}
   {{ end -}}
   req.URL.RawQuery = val.Encode()
   req.URL.Scheme = {{ printf "%q" .URL.Scheme }}
   req.Body = {{ .RequestBody }}
   resp, err := http.DefaultClient.Do(&req)
   if err != nil {
      panic(err)
   }
   defer resp.Body.Close()
   resp.Write(os.Stdout)
}

var body = strings.NewReader({{ .RawBody }})
