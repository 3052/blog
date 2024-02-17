package iso

import (
   "encoding/json"
   "net/http"
   "net/url"
)

type securityKey struct {
   cookie *http.Cookie
   key struct {
      VaadinSecurityKey string `json:"Vaadin-Security-Key"`
   }
}

func (s *securityKey) New() error {
   res, err := http.PostForm("https://www.iso.org/obp/ui", url.Values{
      "v-browserDetails": {"1"},
   })
   if err != nil {
      return err
   }
   defer res.Body.Close()
   var ui struct {
      UIDL string
   }
   if err := json.NewDecoder(res.Body).Decode(&ui); err != nil {
      return err
   }
   if err := json.Unmarshal([]byte(ui.UIDL), &s.key); err != nil {
      return err
   }
   for _, cookie := range res.Cookies() {
      if cookie.Name == "JSESSIONID" {
         s.cookie = cookie
         return nil
      }
   }
   return http.ErrNoCookie
}
