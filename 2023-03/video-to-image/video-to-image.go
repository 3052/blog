package main

import (
   "flag"
   "fmt"
   "os"
   "os/exec"
   "strconv"
)

func (e extensions) String() string {
   var b []byte
   b = append(b, "extension"...)
   for k, v := range e {
      b = append(b, '\n')
      b = strconv.AppendInt(b, k, 10)
      b = append(b, ' ')
      b = append(b, v...)
   }
   return string(b)
}

func main() {
   flag.BoolVar(&f.all, "a", false, "output all frames")
   flag.StringVar(&f.duration, "d", "", "duration")
   flag.Int64Var(&f.ext, "e", 0, exts.String())
   flag.StringVar(&f.name, "f", "", "file")
   flag.StringVar(&f.start, "s", "", "start")
   flag.Parse()
   arg := []string{"-hide_banner"}
   if f.start != "" {
      arg = append(arg, "-ss", f.start)
   }
   if f.name != "" {
      arg = append(arg, "-i", f.name)
      if f.duration != "" {
         arg = append(arg, "-t", f.duration)
      }
      arg = append(arg, "-q", "1", "-vsync", "vfr")
      if !f.all {
         arg = append(arg, "-vf", "select='eq(pict_type, I)'")
      }
      arg = append(arg, "%d" + exts[f.ext])
      cmd := exec.Command("ffmpeg", arg...)
      cmd.Stderr = os.Stderr
      fmt.Println("Run", cmd)
      err := cmd.Run()
      if err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}
var f struct {
   all bool
   duration string
   name string
   start string
   ext int64
}

type extensions map[int64]string

var exts = extensions{
   0: ".jpg",
   1: ".png",
}

