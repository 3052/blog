package main

import (
   "bytes"
   "encoding/json"
   "flag"
   "os"
)

func main() {
   input := flag.String("i", "", "input")
   output := flag.String("o", "", "output")
   flag.Parse()
   if *input != "" {
      src, err := os.ReadFile(*input)
      if err != nil {
         panic(err)
      }
      var dst bytes.Buffer
      err = json.Indent(&dst, src, "", " ")
      if err != nil {
         panic(err)
      }
      err = os.WriteFile(*output, dst.Bytes(), os.ModePerm)
      if err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}
