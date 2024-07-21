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

func TestBravo(t *testing.T) {
   var date date_bravo
   date.New()
   err := date.unmarshal()
   if err != nil {
      t.Fatal(err)
   }
   fmt.Printf("%+v\n", date.v)
}
