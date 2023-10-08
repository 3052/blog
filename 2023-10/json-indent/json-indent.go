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
   if len(os.Args) == 1 {
      flag.Usage()
      return
   }
   flag.Parse()
   src, err := os.ReadFile(*input)
   if err != nil {
      panic(err)
   }
   var dst bytes.Buffer
   json.Indent(&dst, src, "", " ")
   os.WriteFile(*output, dst.Bytes(), 0666)
}
