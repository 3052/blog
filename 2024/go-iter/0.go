package main

import (
   "fmt"
   "iter"
)

func seq(values ...string) iter.Seq[string] {
   return func(yield func(string) bool) {
      for _, value := range values {
         if !yield(value) {
            return
         }
      }
   }
}

func main() {
   for value := range seq("zero", "one", "two", "three") {
      fmt.Println(value)
   }
}
