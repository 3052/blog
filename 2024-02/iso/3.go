package iso

import (
   "io"
   "net/http"
   "net/url"
   "strings"
   "fmt"
)

/*
> curl -s -O -w '%{size_download}' https://www.iso.org/home.html
90122

> curl -w '%{size_download}' -s -O https://www.iso.org/obp/ui
1735
*/
func main() {
   var req http.Request
   req.Header = make(http.Header)
   req.Method = "POST"
   req.ProtoMajor = 1
   req.ProtoMinor = 1
   req.URL = new(url.URL)
   req.URL.Host = "www.iso.org"
   req.URL.Path = "/obp/ui/UIDL/"
   req.URL.Scheme = "https"
   req.Header["Cookie"] = []string{"JSESSIONID=4EB57C67B49DCAFC47FE60380D4D278A"}
   val := make(url.Values)
   val["v-uiId"] = []string{"12"}
   req.URL.RawQuery = val.Encode()
   req.Body = io.NopCloser(body)
   res, err := new(http.Transport).RoundTrip(&req)
   if err != nil {
      panic(err)
   }
   defer res.Body.Close()
   body, err := io.ReadAll(res.Body)
   if err != nil {
      panic(err)
   }
   fmt.Println(string(body))
   if strings.Contains(string(body), "Bhutan") {
      fmt.Println("pass")
   } else {
      fmt.Println("fail")
   }
}

var body = strings.NewReader(`
{
   "csrfToken": "71d8e7fa-a794-45dc-85a0-0b9284028640",
   "clientId": 6,
   "syncId": 6
}
`)
