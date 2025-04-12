/*
1. go run .
2. mitmproxy --mode upstream:http://127.0.0.1:8081
3. kodi 127.0.0.1:8080
*/
package main

import (
   "encoding/base64"
   "github.com/elazarl/goproxy"
   "log"
   "net/http"
   "os/exec"
)

const (
   address = "localhost:8081" // need localhost to avoid security alert
   address1 = "res.proxy-seller.com:10000"
)

func main() {
   basic, err := exec.Command("password", "proxy-seller.com#ES").Output()
   if err != nil {
      panic(err)
   }
   handler1 := goproxy.NewProxyHttpServer()
   handler1.Verbose = true
   log.Println("address1", address1)
   go http.ListenAndServe(address1, handler1)
   handler := goproxy.NewProxyHttpServer()
   handler.Verbose = true
   handler.ConnectDial = handler.NewConnectDialToProxyWithHandler(
      "http://" + address1, func(req *http.Request) {
         req.Header.Set(
            "proxy-authorization",
            "Basic "+base64.StdEncoding.EncodeToString(basic),
         )
      },
   )
   log.Println("address", address)
   err = http.ListenAndServe(address, handler)
   if err != nil {
      panic(err)
   }
}
