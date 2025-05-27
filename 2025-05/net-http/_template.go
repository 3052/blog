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
   req.Header = http.Header{}
   {{ range $key, $value := .Header -}}
      req.Header[{{ printf "%q" $key }}] = {{ printf "%#v" $value }}
   {{ end -}}
   req.Method = {{ printf "%q" .Method }}
   req.ProtoMajor = 1
   req.ProtoMinor = 1
   req.URL = &url.URL{}
   req.URL.Host = {{ printf "%q" .URL.Host }}
   req.URL.Path = {{ printf "%q" .URL.Path }}
   req.URL.RawPath = {{ printf "%q" .URL.RawPath }}
   value := url.Values{}
   {{ range $key, $value := .Query -}}
      value[{{ printf "%q" $key }}] = {{ printf "%#v" $value }}
   {{ end -}}
   req.URL.RawQuery = value.Encode()
   req.URL.Scheme = {{ printf "%q" .URL.Scheme }}
   req.Body = {{ .RequestBody }}
   resp, err := http.DefaultClient.Do(&req)
   if err != nil {
      panic(err)
   }
   err = resp.Write(os.Stdout)
   if err != nil {
      panic(err)
   }
}

var body = strings.NewReader({{ .RawBody }})
