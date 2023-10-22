package main

import (
   "154.pages.dev/protobuf"
   "bufio"
   "bytes"
   "embed"
   "flag"
   "fmt"
   "io"
   "net/http"
   "net/http/httputil"
   "net/textproto"
   "net/url"
   "os"
   "strings"
   "text/template"
)

func (f flags) write(req *http.Request, dst io.Writer) error {
   var v values
   if req.Body != nil && req.Method != "GET" {
      data, err := io.ReadAll(req.Body)
      if err != nil {
         return err
      }
      if f.protobuf {
         m, err := protobuf.Consume(data)
         if err != nil {
            return err
         }
         v.Raw_Req_Body = fmt.Sprintf("%#v", m)
      } else {
         if f.form {
            form, err := url.ParseQuery(string(data))
            if err != nil {
               return err
            }
            v.Raw_Req_Body = fmt.Sprintf("\n%#v.Encode(),\n", form)
         } else {
            v.Raw_Req_Body = fmt.Sprintf("%#q", data)
         }
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

func main() {
   var f flags
   flag.StringVar(&f.name, "f", "", "input file")
   flag.BoolVar(&f.form, "form", false, "POST form")
   flag.BoolVar(&f.golang, "g", false, "request as Go code")
   flag.StringVar(&f.output, "o", "", "output file")
   flag.BoolVar(&f.protobuf, "p", false, "ProtoBuf")
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

type flags struct {
   form bool
   golang bool
   https bool
   name string
   output string
   protobuf bool
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

