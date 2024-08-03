package http

import (
   "fmt"
   "testing"
)

func TestNew(b *testing.T) {
   var resp response_new
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

func BenchmarkNew(b *testing.B) {
   var resp response_new
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
