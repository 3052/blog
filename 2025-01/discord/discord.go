package main

import (
   "bytes"
   "encoding/json"
   "fmt"
   "io"
   "net/http"
   "net/url"
   "os"
   "time"
)

// being nice (one second) doesnt matter
// 429 Too Many Requests 2139
// so fuck them
const sleep = 99 * time.Millisecond

func (p *pomelo_attempt) New(req *http.Request) error {
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   if resp.StatusCode != http.StatusOK {
      return fmt.Errorf("%v %v", resp.Status, resp.Header.Get("retry-after"))
   }
   return json.NewDecoder(resp.Body).Decode(p)
}

type pomelo_attempt struct {
   Taken bool
}

func main() {
   var req http.Request
   req.Header = http.Header{}
   req.Method = "POST"
   req.URL = &url.URL{}
   req.URL.Host = "discord.com"
   req.URL.Scheme = "https"
   req.URL.Path = "/api/v9/unique-username/username-attempt-unauthed"
   req.Header["Content-Type"] = []string{"application/json"}
   for {
      // read
      data, err := os.ReadFile("username.json")
      if err != nil {
         panic(err)
      }
      fmt.Printf("%s ", data)
      req.Body = io.NopCloser(bytes.NewReader(data))
      var attempt pomelo_attempt
      err = attempt.New(&req)
      if err != nil {
         panic(err)
      }
      fmt.Printf("%+v\n", attempt)
      // write
      var value struct {
         Username int `json:"username,string"`
      }
      json.Unmarshal(data, &value)
      value.Username++
      data, err = json.Marshal(value)
      if err != nil {
         panic(err)
      }
      os.WriteFile("username.json", data, os.ModePerm)
      if !attempt.Taken {
         break
      }
      time.Sleep(sleep)
   }
}
