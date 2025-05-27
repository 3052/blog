package main

import (
   "bufio"
   "bytes"
   "embed"
   "flag"
   "fmt"
   "io"
   "net/http"
   "net/textproto"
   "net/url"
   "os"
   "strings"
   "text/template"
)

func (f *flags) write(req *http.Request, dst io.Writer) error {
   var v values
   if req.Body != nil && req.Method != "GET" {
      data, err := io.ReadAll(req.Body)
      if err != nil {
         return err
      }
      if f.form {
         form, err := url.ParseQuery(string(data))
         if err != nil {
            return err
         }
         v.RawBody = fmt.Sprintf("\n%#v.Encode(),\n", form)
      } else {
         v.RawBody = fmt.Sprintf("%#q", data)
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
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return err
   }
   if dst != nil {
      _, err = io.Copy(dst, resp.Body)
      if err != nil {
         return err
      }
   }
   return resp.Write(os.Stdout)
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
   ref, err := url.ParseRequestURI(method_path[1])
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
   data := &bytes.Buffer{}
   length, err := text.R.WriteTo(data)
   if err != nil {
      return nil, err
   }
   if length >= 1 {
      req.Body = io.NopCloser(data)
   }
   // .ContentLength
   req.ContentLength = length
   return &req, nil
}

type flags struct {
   golang bool
   https bool
   name string
   output string
   form bool
}

func main() {
   var f flags
   flag.BoolVar(&f.form, "f", false, "form")
   flag.BoolVar(&f.golang, "g", false, "request as Go code")
   flag.StringVar(&f.name, "i", "", "input file")
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
