package encoding

import (
   "encoding/gob"
   "encoding/json"
   "io"
   "net/http"
   "time"
)

func (g *response_gob) decode(r io.Reader) error {
   return gob.NewDecoder(r).Decode(g)
}

func (g *response_gob) encode(w io.Writer) error {
   return gob.NewEncoder(w).Encode(g)
}

type response_gob struct {
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

func (g *response_gob) New() error {
   resp, err := http.Get("http://httpbingo.org/json")
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   g.Date, err = time.Parse(time.RFC1123, resp.Header.Get("date"))
   if err != nil {
      return err
   }
   return json.NewDecoder(resp.Body).Decode(&g.Body)
}
