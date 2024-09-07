package empty

import "testing"

type byte_slice [999]byte

func (byte_slice) value() bool {
   return false
}

func (*byte_slice) pointer() bool {
   return false
}

func BenchmarkValue(b *testing.B) {
   var slice byte_slice
   for range b.N {
      slice.value()
   }
}

func BenchmarkPointer(b *testing.B) {
   var slice byte_slice
   for range b.N {
      slice.pointer()
   }
}
