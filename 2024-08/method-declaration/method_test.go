package method

import "testing"

func BenchmarkValue(b *testing.B) {
   for range b.N {
      _ = response_body{}.value()
   }
}

func BenchmarkPointer(b *testing.B) {
   for range b.N {
      _ = new(response_body).pointer()
   }
}
