package main

import (
   "bufio"
   "flag"
   "io"
   "os"
)

type flags struct {
   golang bool
   https bool
   name string
   output string
}

func main() {
   var f flags
   flag.StringVar(&f.name, "i", "", "input file")
   flag.BoolVar(&f.golang, "g", false, "request as Go code")
   flag.StringVar(&f.output, "o", "", "output file")
   flag.BoolVar(&f.https, "s", false, "HTTPS")
   flag.Parse()
   if f.name == "" {
      flag.Usage()
   } else {
      var create io.WriteCloser
      if f.output != "" {
         var err error
         create, err = os.Create(f.output)
         if err != nil {
            panic(err)
         }
         defer create.Close()
      }
      open, err := os.Open(f.name)
      if err != nil {
         panic(err)
      }
      defer open.Close()
      req, err := read_request(bufio.NewReader(open))
      if err != nil {
         panic(err)
      }
      if req.URL.Scheme == "" {
         if f.https {
            req.URL.Scheme = "https"
         } else {
            req.URL.Scheme = "http"
         }
      }
      if f.golang {
         err := f.write(req, create)
         if err != nil {
            panic(err)
         }
      } else {
         err := write(req, create)
         if err != nil {
            panic(err)
         }
      }
   }
}
