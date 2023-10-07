package main

import (
   "154.pages.dev/strconv"
   "fmt"
   "io"
   "net/http"
   "net/http/httputil"
   "unicode/utf8"
)

func write(req *http.Request, dst io.Writer) error {
   res, err := new(http.Transport).RoundTrip(req)
   if err != nil {
      return err
   }
   defer res.Body.Close()
   if dst != nil {
      b, err := httputil.DumpResponse(res, false)
      if err != nil {
         return err
      }
      fmt.Println(string(b))
      if _, err := io.Copy(dst, res.Body); err != nil {
         return err
      }
   } else {
      b, err := httputil.DumpResponse(res, true)
      if err != nil {
         return err
      }
      fmt.Println(strconv.Encode(b))
   }
   return nil
}

// go.dev/ref/spec#String_literals
func can_backquote(s string) bool {
   for _, r := range s {
      if r == '\r' {
         return false
      }
      if r == '`' {
         return false
      }
      if strconv.Binary_Rune(r) {
         return false
      }
   }
   return utf8.ValidString(s)
}
