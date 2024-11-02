package empty

import "testing"

type hello_world [99_999]byte

func (hello_world) value() bool {
   return false
}

func (*hello_world) pointer() bool {
   return false
}

func BenchmarkPointerNil(b *testing.B) {
   for range b.N {
      (*hello_world).pointer(nil)
   }
}

func BenchmarkValue(b *testing.B) {
   for range b.N {
      hello_world{}.value()
   }
}

func BenchmarkPointer(b *testing.B) {
   for range b.N {
      new(hello_world).pointer()
   }
}
