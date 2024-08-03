package http

import (
   "encoding/json"
   "io"
   "net/http"
   "time"
)

type response_new struct {
   date value[time.Time]
   body value[struct {
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
   }]
}

func (r *response_new) New() error {
   resp, err := http.Get("http://httpbingo.org/json")
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   r.date.raw = []byte(resp.Header.Get("date"))
   r.body.raw, err = io.ReadAll(resp.Body)
   if err != nil {
      return err
   }
   return nil
}

func (r *response_new) unmarshal() error {
   date, err := time.Parse(time.RFC1123, string(r.date.raw))
   if err != nil {
      return err
   }
   r.date.value = &date
   r.body.New()
   return json.Unmarshal(r.body.raw, r.body.value)
}

type value[T any] struct {
   value *T
   raw []byte
}

func (v *value[T]) New() {
   v.value = new(T)
}
