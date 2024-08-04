package encoding

import (
   "encoding/json"
   "net/http"
   "time"
)

func (j json3) marshal_00() ([]byte, error) {
   return json.Marshal(j)
}

func (j *json3) marshal_10() ([]byte, error) {
   return json.Marshal(j)
}

func (j *json3) marshal_11() ([]byte, error) {
   return json.Marshal(*j)
}

func (j *json3) unmarshal(text []byte) error {
   return json.Unmarshal(text, j)
}

type json3 struct {
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

func (j *json3) New() error {
   resp, err := http.Get("http://httpbingo.org/json")
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   j.Date, err = time.Parse(time.RFC1123, resp.Header.Get("date"))
   if err != nil {
      return err
   }
   return json.NewDecoder(resp.Body).Decode(&j.Body)
}
