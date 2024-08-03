package encoding

import (
   "encoding/json"
   "io"
   "net/http"
   "time"
)

type json1 struct {
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

func (j *json1) New() error {
   resp, err := http.Get("http://httpbingo.org/json")
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   j.date.raw = []byte(resp.Header.Get("date"))
   j.body.raw, err = io.ReadAll(resp.Body)
   if err != nil {
      return err
   }
   return nil
}

func (j *json1) unmarshal() error {
   date, err := time.Parse(time.RFC1123, string(j.date.raw))
   if err != nil {
      return err
   }
   j.date.value = &date
   j.body.New()
   return json.Unmarshal(j.body.raw, j.body.value)
}

type value[T any] struct {
   value *T
   raw []byte
}

func (v *value[T]) New() {
   v.value = new(T)
}
