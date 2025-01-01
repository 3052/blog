package strings

import (
   "strings"
   "testing"
)

const input = "zero one two three four five six seven eight nine ten eleven"

func BenchmarkContains(b *testing.B) {
   for range b.N {
      if strings.Contains(input, "eleven") {
         _ = strings.Replace(input, "eleven", "twelve", 1)
      }
   }
}

func BenchmarkReplace(b *testing.B) {
   for range b.N {
      output := strings.Replace(input, "eleven", "twelve", 1)
      _ = output != input
   }
}
