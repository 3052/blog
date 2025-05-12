package main

import (
   "flag"
   "log"
   "net/http"
   "path"
   "os"
   "os/exec"
)

func get(address string) error {
   resp, err := http.Get(address)
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   file, err := os.Create(path.Base(address))
   if err != nil {
      return err
   }
   defer file.Close()
   _, err = file.ReadFrom(resp.Body)
   if err != nil {
      return err
   }
   return nil
}

func serve(from, to string) error {
   err := get(from)
   if err != nil {
      return err
   }
   command := exec.Command(
      "zig", "test", "--test-no-exec", "-femit-docs", path.Base(from),
   )
   command.Stderr = os.Stderr
   command.Stdout = os.Stdout
   err = command.Run()
   if err != nil {
      return err
   }
   log.Print(to)
   return http.ListenAndServe(
      to, http.FileServer(http.Dir("docs")),
   )
}

func main() {
   from := flag.String("f", "", "from address")
   // need localhost to avoid security alert
   to := flag.String("t", "localhost:8080", "to address")
   flag.Parse()
   if *from != "" {
      err := serve(*from, *to)
      if err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}
