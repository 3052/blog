package encoding

import (
   "fmt"
   "testing"
)

func BenchmarkJson1Pointer(b *testing.B) {
   for range b.N {
      var resp json1_pointer
      resp.New()
      err := resp.unmarshal()
      if err != nil {
         b.Fatal(err)
      }
   }
}

func TestJson1Pointer(b *testing.T) {
   var resp json1_pointer
   resp.New()
   err := resp.unmarshal()
   if err != nil {
      b.Fatal(err)
   }
   fmt.Println(resp.date.value)
   fmt.Printf("%+v\n\n", resp.body.value)
}
