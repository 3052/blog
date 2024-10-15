package main

import (
   "flag"
   "github.com/beevik/etree"
   "os"
)

type flags struct {
   indent int
   name string
}

func main() {
   var f flags
   flag.IntVar(&f.indent, "i", 1, "indent")
   flag.StringVar(&f.name, "n", "", "name")
   flag.Parse()
   if f.name != "" {
      doc := etree.NewDocument()
      err := doc.ReadFromFile(f.name)
      if err != nil {
         panic(err)
      }
      doc.Indent(f.indent)
      _, err = doc.WriteTo(os.Stdout)
      if err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}
