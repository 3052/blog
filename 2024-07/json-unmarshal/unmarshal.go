package hello

import "encoding/json"

const text = `{"month": 12, "day": 31}` 

type date_alfa struct {
   Month int
   Day int
}

type raw_date_alfa []byte

func (r raw_date_alfa) date() (*date_alfa, error) {
   date := new(date_alfa)
   err := json.Unmarshal(r, date)
   if err != nil {
      return nil, err
   }
   return date, nil
}

func (r *raw_date_alfa) Make() {
   *r = []byte(text)
}

func (d *date_bravo) New() {
   d.data = []byte(text)
}

func New[T any](value *T) *T {
   return new(T)
}

type date_bravo struct {
   data []byte
   v *struct {
      Month int
      Day int
   }
}

func (d *date_bravo) unmarshal() error {
   d.v = New(d.v)
   err := json.Unmarshal(d.data, d.v)
   if err != nil {
      return err
   }
   d.data = nil
   return nil
}
