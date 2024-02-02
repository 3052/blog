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
   checkout := flag.Bool("c", false, "checkout")
   module := flag.String("m", "", "module")
   version := flag.String("v", "", "version")
   flag.Parse()
   if *version != "" {
      temp, err := os.MkdirTemp(".", "")
      if err != nil {
         panic(err)
      }
      args := []string{
         "-home", *module,
         "-out", temp,
         "-pkg-version", *version,
         "./...",
      }
      if *checkout {
         if err := run("git", "checkout", *version); err != nil {
            panic(err)
         }
      }
      if err := run("doc2go", args...); err != nil {
         panic(err)
      }
      if *checkout {
         if err := run("git", "checkout", "main"); err != nil {
            panic(err)
         }
      }
      if err := os.RemoveAll("docs"); err != nil {
         panic(err)
      }
      if err := os.Rename(temp, "docs"); err != nil {
         panic(err)
      }
      err = os.WriteFile("docs/.ignore", []byte("*.html"), 0666)
      if err != nil {
         panic(err)
      }
      if err := os.WriteFile("docs/.nojekyll", nil, 0666); err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}
