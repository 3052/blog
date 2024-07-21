package hello

import "encoding/json"

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

func (d *date_bravo) New() {
   d.data = []byte(`{"month": 12, "day": 31}`)
}

func (d *date_bravo) unmarshal() error {
   d.v = New(d.v)
   return json.Unmarshal(d.data, d.v)
}
