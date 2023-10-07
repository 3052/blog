package main

import (
   "encoding/json"
   "flag"
   "os"
)

func (f flags) indent_json() error {
   file := os.Stdout
   if f.output != "" {
      var err error
      file, err = os.Create(f.output)
      if err != nil {
         return err
      }
   }
   defer file.Close()
   var value any
   {
      b, err := os.ReadFile(f.input)
      if err != nil {
         return err
      }
      if err := json.Unmarshal(b, &value); err != nil {
         return err
      }
   }
   enc := json.NewEncoder(file)
   enc.SetEscapeHTML(false)
   enc.SetIndent("", " ")
   return enc.Encode(value)
}

type flags struct {
   input string
   output string
}

func main() {
   var f flags
   flag.StringVar(&f.input, "f", "", "input file")
   flag.StringVar(&f.output, "o", "", "output file")
   flag.Parse()
   if f.input != "" {
      err := f.indent_json()
      if err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}
