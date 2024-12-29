package main

import (
   "fmt"
   "iter"
)

func seq(ss ...string) iter.Seq[string] {
   return func(yield func(string) bool) {
      for _, s := range ss {
         if !yield(s) {
            return
         }
      }
   }
}

func main() {
   seq("zero", "one", "two", "three")(func(s string) bool {
      fmt.Printf("%q\n", s)
      return true
   })
}
