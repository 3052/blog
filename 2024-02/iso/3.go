package iso

import (
   "bytes"
   "encoding/json"
   "io"
   "net/http"
   "net/url"
   "strings"
   "fmt"
)

func (s securityKey) three() error {
   body, err := func() ([]byte, error) {
      m := map[string]any{
         "clientId": 6,
         "csrfToken": s.key.VaadinSecurityKey,
         "syncId": 6,
      }
      return json.Marshal(m)
   }()
   if err != nil {
      return err
   }
   req := new(http.Request)
   req.Header = make(http.Header)
   req.Method = "POST"
   req.ProtoMajor = 1
   req.ProtoMinor = 1
   req.URL = new(url.URL)
   req.URL.Host = "www.iso.org"
   req.URL.Path = "/obp/ui/UIDL/"
   req.URL.Scheme = "https"
   val := make(url.Values)
   val["v-uiId"] = []string{"12"}
   req.URL.RawQuery = val.Encode()
   req.AddCookie(s.cookie)
   req.Body = io.NopCloser(bytes.NewReader(body))
   res, err := http.DefaultClient.Do(req)
   if err != nil {
      return err
   }
   defer res.Body.Close()
   body, err = io.ReadAll(res.Body)
   if err != nil {
      return err
   }
   fmt.Println(string(body))
   if strings.Contains(string(body), "Bhutan") {
      fmt.Println("pass")
   } else {
      fmt.Println("fail")
   }
   return nil
}
