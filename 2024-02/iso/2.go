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
   req.Header["Accept"] = []string{"*/*"}
   req.Header["Accept-Encoding"] = []string{"gzip, deflate, br"}
   req.Header["Accept-Language"] = []string{"en-US,en;q=0.5"}
   req.Header["Content-Length"] = []string{"303"}
   req.Header["Content-Type"] = []string{"application/x-www-form-urlencoded"}
   req.Header["Cookie"] = []string{"JSESSIONID=D552D1E1BB0EA05C076BC73E2092DF75", "BIGipServerpool_prod_iso_obp=914903434.36895.0000"}
   req.Header["Origin"] = []string{"https://www.iso.org"}
   req.Header["Referer"] = []string{"https://www.iso.org/obp/ui"}
   req.Header["Sec-Fetch-Dest"] = []string{"empty"}
   req.Header["Sec-Fetch-Mode"] = []string{"cors"}
   req.Header["Sec-Fetch-Site"] = []string{"same-origin"}
   req.Header["Te"] = []string{"trailers"}
   req.Header["User-Agent"] = []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/111.0"}
   req.Method = "POST"
   req.ProtoMajor = 1
   req.ProtoMinor = 1
   req.URL = new(url.URL)
   req.URL.Host = "www.iso.org"
   req.URL.Path = "/obp/ui"
   req.URL.RawPath = ""
   val := make(url.Values)
   val["v-1708141620752"] = []string{""}
   req.URL.RawQuery = val.Encode()
   req.URL.Scheme = "https"
   req.Body = io.NopCloser(body)
   res, err := new(http.Transport).RoundTrip(&req)
   if err != nil {
      panic(err)
   }
   defer res.Body.Close()
   res.Write(os.Stdout)
}

var body = strings.NewReader(`v-browserDetails=1&theme=iso-red&v-appId=obpui-105541713&v-sh=864&v-sw=1536&v-cw=1192&v-ch=622&v-curdate=1708141620752&v-tzo=360&v-dstd=60&v-rtzo=360&v-dston=false&v-tzid=America%2FChicago&v-vw=1192&v-vh=0&v-loc=https%3A%2F%2Fwww.iso.org%2Fobp%2Fui%23search%2Fcode&v-wn=obpui-105541713-0.726762453241973`)
