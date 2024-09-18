package main

import (
   "flag"
   "os"
   "os/exec"
   "time"
)

func main() {
   after := flag.Duration("a", 9*time.Second, "after")
   flag.Parse()
   arg := flag.Args()
   cmd := exec.Command(arg[0], arg[1:]...)
   cmd.Stderr = os.Stderr
   cmd.Stdout = os.Stdout
   cmd.Start()
   errs := make(chan error)
   go func() {
      errs <- cmd.Wait()
   }()
   select {
   case <-time.After(*after):
      cmd.Process.Kill()
   case <-errs:
   }
}
