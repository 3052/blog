package encoding

import (
   "encoding/json"
   "time"
)

type value[T any] struct {
   value *T
   raw []byte
}

func (v *value[T]) New() {
   v.value = new(T)
}

type json1 struct {
   date value[time.Time]
   body value[body]
}

func (j *json1) New() {
   j.date.raw = []byte(raw_date)
   j.body.raw = []byte(raw_body)
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
