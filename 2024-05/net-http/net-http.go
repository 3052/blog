package main

import (
   "bufio"
   "bytes"
   "embed"
   "encoding/json"
   "fmt"
   "io"
   "net/http"
   "net/textproto"
   "net/url"
   "os"
   "strings"
   "text/template"
)

func (f flags) write(req *http.Request, dst io.Writer) error {
   var v values
   if req.Body != nil && req.Method != "GET" {
      src, err := io.ReadAll(req.Body)
      if err != nil {
         return err
      }
      if req.Header.Get("content-type") == "application/json" {
         var dst bytes.Buffer
         json.Indent(&dst, src, "", " ")
         v.RawBody = "`" + dst.String() + "`"
      } else if f.form {
         form, err := url.ParseQuery(string(src))
         if err != nil {
            return err
         }
         v.RawBody = fmt.Sprintf("\n%#v.Encode(),\n", form)
      } else {
         v.RawBody = fmt.Sprintf("%#q", src)
      }
      v.RequestBody = "io.NopCloser(body)"
   } else {
      v.RawBody = `""`
      v.RequestBody = "nil"
   }
   v.Query = req.URL.Query()
   v.Request = req
   temp, err := template.ParseFS(content, "_template.go")
   if err != nil {
      return err
   }
   return temp.Execute(dst, v)
}

//go:embed _template.go
var content embed.FS

type values struct {
   *http.Request
   Query url.Values
   RequestBody string
   RawBody string
}

func write(req *http.Request, dst io.Writer) error {
   res, err := http.DefaultClient.Do(req)
   if err != nil {
      return err
   }
   defer res.Body.Close()
   if dst != nil {
      _, err := io.Copy(dst, res.Body)
      if err != nil {
         return err
      }
      res.Write(os.Stdout)
   } else {
      res.Write(os.Stdout)
   }
   return nil
}

// why is this needed?
func read_request(r *bufio.Reader) (*http.Request, error) {
   var req http.Request
   text := textproto.NewReader(r)
   // .Method
   raw_method_path, err := text.ReadLine()
   if err != nil {
      return nil, err
   }
   method_path := strings.Fields(raw_method_path)
   req.Method = method_path[0]
   // .URL
   ref, err := url.Parse(method_path[1])
   if err != nil {
      return nil, err
   }
   req.URL = ref
   // .URL.Host
   head, err := text.ReadMIMEHeader()
   if err != nil {
      return nil, err
   }
   if req.URL.Host == "" {
      req.URL.Host = head.Get("Host")
   }
   // .Header
   req.Header = http.Header(head)
   // .Body
   buf := new(bytes.Buffer)
   length, err := text.R.WriteTo(buf)
   if err != nil {
      return nil, err
   }
   if length >= 1 {
      req.Body = io.NopCloser(buf)
   }
   // .ContentLength
   req.ContentLength = length
   return &req, nil
}

