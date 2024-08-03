package encoding

import (
   "fmt"
   "testing"
)

func TestJson1(b *testing.T) {
   var resp json1
   err := resp.New()
   if err != nil {
      b.Fatal(err)
   }
   err = resp.unmarshal()
   if err != nil {
      b.Fatal(err)
   }
   fmt.Println(resp.date.value)
   fmt.Printf("%+v\n", resp.body.value)
}

func BenchmarkJson1(b *testing.B) {
   var resp json1
   err := resp.New()
   if err != nil {
      b.Fatal(err)
   }
   for range b.N {
      err = resp.unmarshal()
      if err != nil {
         b.Fatal(err)
      }
   }
}
