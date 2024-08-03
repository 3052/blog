package encoding

import (
   "fmt"
   "testing"
)

func TestJson3(b *testing.T) {
   var resp json3
   err := resp.New()
   if err != nil {
      b.Fatal(err)
   }
   text, err := resp.marshal()
   if err != nil {
      b.Fatal(err)
   }
   err = resp.unmarshal(text)
   if err != nil {
      b.Fatal(err)
   }
   fmt.Printf("%+v\n", resp)
}

func BenchmarkJson3(b *testing.B) {
   var resp json3
   err := resp.New()
   if err != nil {
      b.Fatal(err)
   }
   for range b.N {
      text, err := resp.marshal()
      if err != nil {
         b.Fatal(err)
      }
      err = resp.unmarshal(text)
      if err != nil {
         b.Fatal(err)
      }
   }
}
