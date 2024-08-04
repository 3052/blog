package encoding

import (
   "encoding/json"
   "time"
)

type json3 struct {
   Date time.Time
   Body body
}

func (j *json3) New() error {
   var err error
   j.Date, err = time.Parse(time.RFC1123, raw_date)
   if err != nil {
      return err
   }
   return json.Unmarshal([]byte(raw_body), &j.Body)
}

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
