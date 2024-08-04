package encoding

import (
   "fmt"
   "testing"
)

func BenchmarkAsn1(b *testing.B) {
   var resp response_asn1
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

func TestAsn1(b *testing.T) {
   var resp response_asn1
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
