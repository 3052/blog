package hello

import (
   "fmt"
   "testing"
)

func BenchmarkAlfa(b *testing.B) {
   for range b.N {
      var raw raw_date_alfa
      raw.Make()
      _, err := raw.date()
      if err != nil {
         b.Fatal(err)
      }
   }
}

func TestAlfa(t *testing.T) {
   var raw raw_date_alfa
   raw.Make()
   date, err := raw.date()
   if err != nil {
      t.Fatal(err)
   }
   fmt.Printf("%+v\n", date)
}
