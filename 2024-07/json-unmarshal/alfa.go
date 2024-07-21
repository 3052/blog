package hello

import "encoding/json"

type date_alfa struct {
   Month int
   Day int
}

type raw_date_alfa []byte

func (r *raw_date_alfa) Make() {
   *r = []byte(`{"month": 12, "day": 31}`)
}

func (r raw_date_alfa) date() (*date_alfa, error) {
   date := new(date_alfa)
   err := json.Unmarshal(r, date)
   if err != nil {
      return nil, err
   }
   return date, nil
}
