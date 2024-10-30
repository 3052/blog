package encoding

import (
   "bytes"
   "encoding/asn1"
   "encoding/gob"
   "encoding/json"
   "testing"
)

type greeting struct {
   Hello string
   World string
}

func BenchmarkJson(b *testing.B) {
   for range b.N {
      data, err := json.Marshal(greeting{"hello", "world"})
      if err != nil {
         b.Fatal(err)
      }
      var greet greeting
      err = json.Unmarshal(data, &greet)
      if err != nil {
         b.Fatal(err)
      }
   }
}

func BenchmarkAsn1(b *testing.B) {
   for range b.N {
      data, err := asn1.Marshal(greeting{"hello", "world"})
      if err != nil {
         b.Fatal(err)
      }
      var greet greeting
      _, err = asn1.Unmarshal(data, &greet)
      if err != nil {
         panic(err)
      }
   }
}

func BenchmarkGob(b *testing.B) {
   for range b.N {
      data := &bytes.Buffer{}
      err := gob.NewEncoder(data).Encode(greeting{"hello", "world"})
      if err != nil {
         b.Fatal(err)
      }
      var greet greeting
      err = gob.NewDecoder(data).Decode(&greet)
      if err != nil {
         b.Fatal(err)
      }
   }
}
