package encoding

import (
   "encoding/asn1"
   "encoding/json"
   "net/http"
   "time"
)

type response_asn1 struct {
   Date time.Time
   Body struct {
      Slideshow struct {
         Author string
         Date   string
         Slides []struct {
            Title string
            Type  string
            Items []string
         }
         Title string
      }
   }
}

func (r *response_asn1) New() error {
   resp, err := http.Get("http://httpbingo.org/json")
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   r.Date, err = time.Parse(time.RFC1123, resp.Header.Get("date"))
   if err != nil {
      return err
   }
   return json.NewDecoder(resp.Body).Decode(&r.Body)
}

func (r *response_asn1) marshal() ([]byte, error) {
   return asn1.Marshal(*r)
}

func (r *response_asn1) unmarshal(text []byte) error {
   _, err := asn1.Unmarshal(text, r)
   if err != nil {
      return err
   }
   return nil
}
