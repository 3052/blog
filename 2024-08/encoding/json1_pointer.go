package encoding

import (
   "encoding/json"
   "time"
)

func (v *value_pointer[T]) New() {
   v.value = new(T)
}

type json1_pointer struct {
   date value_pointer[time.Time]
   body value_pointer[body]
}

func (j *json1_pointer) New() {
   j.date.raw = []byte(raw_date)
   j.body.raw = []byte(raw_body)
}

func (j *json1_pointer) unmarshal() error {
   date, err := time.Parse(time.RFC1123, string(j.date.raw))
   if err != nil {
      return err
   }
   j.date.value = &date
   j.body.New()
   return json.Unmarshal(j.body.raw, j.body.value)
}

type value_pointer[T any] struct {
   value *T
   raw []byte
}
