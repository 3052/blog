package encoding

import (
   "fmt"
   "testing"
)

func TestXml(b *testing.T) {
   var resp response_xml
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

func BenchmarkXml(b *testing.B) {
   var resp response_xml
   err := resp.New()
   if err != nil {
      b.Fatal(err)
   }
   for range b.N {
      text, err := resp.marshal()
      if err != nil {
         b.Fatal(err)
      }
      resp = response_xml{}
      err = resp.unmarshal(text)
      if err != nil {
         b.Fatal(err)
      }
   }
}
