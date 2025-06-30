package main

import (
   "bytes"
   "flag"
   "fmt"
   "log"
   "os"
   "os/exec"
)

func main() {
   log.SetFlags(log.Ltime)
   spec := flag.String("i", "", "import spec")
   flag.Parse()
   if *spec != "" {
      err := do_import(*spec)
      if err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}

func create(name string) (*os.File, error) {
   log.Println("Create", name)
   return os.Create(name)
}

func run(name string, arg ...string) error {
   command := exec.Command(name, arg...)
   command.Stderr = os.Stderr
   command.Stdout = os.Stdout
   log.Println("Run", command.Args)
   return command.Run()
}

func create_hello(spec string) error {
   file, err := create("hello.go")
   if err != nil {
      return err
   }
   defer file.Close()
   _, err = fmt.Fprintln(file, "package hello")
   if err != nil {
      return err
   }
   _, err = fmt.Fprintf(file, "import _ %q\n", spec)
   if err != nil {
      return err
   }
   return nil
}

func do_import(spec string) error {
   err := create_hello(spec)
   if err != nil {
      return err
   }
   defer os.Remove("hello.go")
   err = run("go", "mod", "init", "hello")
   if err != nil {
      return err
   }
   defer os.Remove("go.mod")
   err = run("go", "mod", "tidy")
   if err != nil {
      return err
   }
   defer os.Remove("go.sum")
   data, err := os.ReadFile("go.sum")
   if err != nil {
      return err
   }
   count := bytes.Count(data, []byte{'\n'})
   log.Println("Count", count)
   return nil
}
