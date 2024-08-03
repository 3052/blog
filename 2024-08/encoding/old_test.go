package http

import (
   "fmt"
   "testing"
)

func TestOld(b *testing.T) {
   var old response_old
   err := old.New()
   if err != nil {
      b.Fatal(err)
   }
   text, err := old.marshal()
   if err != nil {
      b.Fatal(err)
   }
   err = old.unmarshal(text)
   if err != nil {
      b.Fatal(err)
   }
   fmt.Printf("%+v\n", old)
}

func BenchmarkOld(b *testing.B) {
   var old response_old
   err := old.New()
   if err != nil {
      b.Fatal(err)
   }
   for range b.N {
      text, err := old.marshal()
      if err != nil {
         b.Fatal(err)
      }
      err = old.unmarshal(text)
      if err != nil {
         b.Fatal(err)
      }
   }
}
