package main

import (
   "flag"
   "fmt"
   "net/http"
)

func main() {
   dir := flag.String("d", "", "dir")
   port := flag.String("p", ":8080", "port")
   flag.Parse()
   fmt.Println("localhost" + *port)
   http.ListenAndServe(*port, http.FileServer(http.Dir(*dir)))
}
