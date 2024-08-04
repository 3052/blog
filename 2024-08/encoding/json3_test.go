package encoding

import (
   "fmt"
   "testing"
)

func BenchmarkJson3_00(b *testing.B) {
   for range b.N {
      var resp json3
      err := resp.New()
      if err != nil {
         b.Fatal(err)
      }
      text, err := resp.marshal_00()
      if err != nil {
         b.Fatal(err)
      }
      err = resp.unmarshal(text)
      if err != nil {
         b.Fatal(err)
      }
   }
}

func BenchmarkJson3_10(b *testing.B) {
   for range b.N {
      var resp json3
      err := resp.New()
      if err != nil {
         b.Fatal(err)
      }
      text, err := resp.marshal_10()
      if err != nil {
         b.Fatal(err)
      }
      err = resp.unmarshal(text)
      if err != nil {
         b.Fatal(err)
      }
   }
}

func BenchmarkJson3_11(b *testing.B) {
   for range b.N {
      var resp json3
      err := resp.New()
      if err != nil {
         b.Fatal(err)
      }
      text, err := resp.marshal_11()
      if err != nil {
         b.Fatal(err)
      }
      err = resp.unmarshal(text)
      if err != nil {
         b.Fatal(err)
      }
   }
}

func TestJson3_00(b *testing.T) {
   var resp json3
   err := resp.New()
   if err != nil {
      b.Fatal(err)
   }
   text, err := resp.marshal_00()
   if err != nil {
      b.Fatal(err)
   }
   err = resp.unmarshal(text)
   if err != nil {
      b.Fatal(err)
   }
   fmt.Printf("%+v\n", resp)
}

func TestJson3_10(b *testing.T) {
   var resp json3
   err := resp.New()
   if err != nil {
      b.Fatal(err)
   }
   text, err := resp.marshal_10()
   if err != nil {
      b.Fatal(err)
   }
   err = resp.unmarshal(text)
   if err != nil {
      b.Fatal(err)
   }
   fmt.Printf("%+v\n", resp)
}

func TestJson3_11(b *testing.T) {
   var resp json3
   err := resp.New()
   if err != nil {
      b.Fatal(err)
   }
   text, err := resp.marshal_11()
   if err != nil {
      b.Fatal(err)
   }
   err = resp.unmarshal(text)
   if err != nil {
      b.Fatal(err)
   }
   fmt.Printf("%+v\n", resp)
}
