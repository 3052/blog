package encoding

import (
   "encoding/json"
   "encoding/xml"
   "net/http"
   "time"
)

func (r *response_xml) marshal() ([]byte, error) {
   return xml.Marshal(r)
}

func (r *response_xml) unmarshal(text []byte) error {
   return xml.Unmarshal(text, r)
}

type response_xml struct {
   Date time.Time
   Body struct {
      Slideshow struct {
         Author string `json:"author"`
         Date   string `json:"date"`
         Slides []struct {
            Title string   `json:"title"`
            Type  string   `json:"type"`
            Items []string `json:"items,omitempty"`
         } `json:"slides"`
         Title string `json:"title"`
      } `json:"slideshow"`
   }
}

func (r *response_xml) New() error {
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
