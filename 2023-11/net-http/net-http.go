package main

import (
   "bufio"
   "bytes"
   "embed"
   "encoding/json"
   "fmt"
   "io"
   "net/http"
   "net/http/httputil"
   "net/textproto"
   "net/url"
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
         v.Raw_Req_Body = "`" + dst.String() + "`"
      } else if f.form {
         form, err := url.ParseQuery(string(src))
         if err != nil {
            return err
         }
         v.Raw_Req_Body = fmt.Sprintf("\n%#v.Encode(),\n", form)
      } else {
         v.Raw_Req_Body = fmt.Sprintf("%#q", src)
      }
      v.Req_Body = "io.NopCloser(req_body)"
   } else {
      v.Raw_Req_Body = `""`
      v.Req_Body = "nil"
   }
   v.Query = req.URL.Query()
   v.Request = req
   temp, err := template.ParseFS(content, "_template.go")
   if err != nil {
      return err
   }
   return temp.Execute(dst, v)
}

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

//go:embed _template.go
var content embed.FS

type values struct {
   *http.Request
   Query url.Values
   Req_Body string
   Raw_Req_Body string
}

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
      fmt.Println(string(b))
   }
   return nil
}

