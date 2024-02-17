package iso

import "net/http"

type sessionId struct {
   cookie *http.Cookie
}

func (s *sessionId) New() error {
   res, err := http.Head("https://www.iso.org/obp/ui")
   if err != nil {
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
