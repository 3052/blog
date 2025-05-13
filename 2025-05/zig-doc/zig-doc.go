package main

import (
   "flag"
   "log"
   "os"
   "os/exec"
)

const build_zig_zon = `
.{
   .name = .name,
   .paths = .{ "" },
   .version = "0.0.0",
}
`

func write_file(name string, data []byte) error {
   log.Println("WriteFile", name)
   return os.WriteFile(name, data, os.ModePerm)
}

func run(name string, arg ...string) error {
   c := exec.Command(name, arg...)
   c.Stderr = os.Stderr
   c.Stdout = os.Stdout
   log.Println("Run", c.Args)
   return c.Run()
}

func doc(zip string) error {
   err := write_file("build.zig", nil)
   if err != nil {
      return err
   }
   err = write_file("build.zig.zon", []byte(build_zig_zon))
   if err != nil {
      return err
   }
   return run("zig", "fetch", "--save", zip)
}

func main() {
   log.SetFlags(log.Ltime)
   zip := flag.String("z", "", "zip")
   flag.Parse()
   if *zip != "" {
      err := doc(*zip)
      if err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}
