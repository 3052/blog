package encoding

import (
   "bytes"
   "fmt"
   "testing"
)

func TestGob(b *testing.T) {
   var resp response_gob
   err := resp.New()
   if err != nil {
      b.Fatal(err)
   }
   network := new(bytes.Buffer)
   err = resp.encode(network)
   if err != nil {
      b.Fatal(err)
   }
   err = resp.decode(network)
   if err != nil {
      b.Fatal(err)
   }
   fmt.Printf("%+v\n", resp)
}

func BenchmarkGob(b *testing.B) {
   var resp response_gob
   err := resp.New()
   if err != nil {
      b.Fatal(err)
   }
   network := new(bytes.Buffer)
   for range b.N {
      err = resp.encode(network)
      if err != nil {
         b.Fatal(err)
      }
      err = resp.decode(network)
      if err != nil {
         b.Fatal(err)
      }
   }
}
