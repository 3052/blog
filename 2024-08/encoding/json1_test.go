package encoding

import (
   "fmt"
   "testing"
)

func TestJson1(b *testing.T) {
   var resp json1
   resp.New()
   err := resp.unmarshal()
   if err != nil {
      b.Fatal(err)
   }
   fmt.Println(resp.date.value)
   fmt.Printf("%+v\n", resp.body.value)
}

func BenchmarkJson1(b *testing.B) {
   for range b.N {
      var resp json1
      resp.New()
      err := resp.unmarshal()
      if err != nil {
         b.Fatal(err)
      }
   }
}
