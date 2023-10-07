package main

import (
   "blog/exports"
   "fmt"
   "os"
   "sort"
)

func main() {
   if len(os.Args) != 2 {
      fmt.Println("exports [directory]")
      return
   }
   exps, err := exports.Exports(os.Args[1])
   if err != nil {
      panic(err)
   }
   sort.Slice(exps, func(i, j int) bool {
      return fmt.Sprintf("%p", exps[i]) < fmt.Sprintf("%p", exps[j])
   })
   for _, exp := range exps {
      fmt.Printf("gorename -from '\".\".%v' -to %u\n", exp, exp)
   }
}
