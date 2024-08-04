package encoding

import (
   "encoding/json"
   "time"
)

type json1 struct {
   date value[time.Time]
   body value[body]
}

func (j *json1) New() {
   j.date.raw = []byte(raw_date)
   j.body.raw = []byte(raw_body)
}

func (j *json1) unmarshal() error {
   var err error
   j.date.value, err = time.Parse(time.RFC1123, string(j.date.raw))
   if err != nil {
      return err
   }
   return json.Unmarshal(j.body.raw, &j.body.value)
}

type value[T any] struct {
   value T
   raw []byte
}
