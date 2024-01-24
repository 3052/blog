package main

import (
   "flag"
   "fmt"
   "os"
   "os/exec"
)

func run(name string, arg ...string) error {
   cmd := exec.Command(name, arg...)
   cmd.Stderr, cmd.Stdout = os.Stderr, os.Stdout
   fmt.Println(cmd.Args)
   return cmd.Run()
}

func main() {
   version := flag.String("v", "", "version")
   module := flag.String("m", "", "module")
   flag.Parse()
   if *version != "" {
      err := run("git", "checkout", *version)
      if err != nil {
         panic(err)
      }
      if err := os.RemoveAll("docs"); err != nil {
         panic(err)
      }
      arg := []string{"-out", "docs", "-pkg-version", *version}
      if *module != "" {
         arg = append(arg, "-home", *module, "./...")
      } else {
         arg = append(arg, "std")
      }
      if err := run("doc2go", arg...); err != nil {
         panic(err)
      }
      if err := os.WriteFile("docs/.nojekyll", nil, 0666); err != nil {
         panic(err)
      }
      if err := run("git", "checkout", "main"); err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}
