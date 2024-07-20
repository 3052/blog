package hello

import (
   "fmt"
   "testing"
)

func BenchmarkBravo(b *testing.B) {
   for range b.N {
      var date date_bravo
      date.New()
      err := date.unmarshal()
      if err != nil {
         b.Fatal(err)
      }
   }
}

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

func TestBravo(t *testing.T) {
   var date date_bravo
   date.New()
   err := date.unmarshal()
   if err != nil {
      t.Fatal(err)
   }
   fmt.Printf("%+v\n", date.v)
}
