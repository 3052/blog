package main

import (
   "flag"
   "fmt"
   "github.com/xi2/xz"
   "io"
   "net/http"
   "os"
   "os/exec"
   "path/filepath"
   "strings"
   "time"
)

func (f flags) run(server string) error {
   commands := []command{
      run("adb", "root"),
      run("adb", "wait-for-device"),
      run("adb", "push", server, "/data/app/frida-server"),
      run("adb", "shell", "chmod", "755", "/data/app/frida-server"),
      start("adb", "shell", "/data/app/frida-server"),
   }
   if f.version < "16." {
      commands = append(commands,
         run("frida", "--no-pause", "-U", "-l", f.script, "-f", f.app),
      )
   } else {
      commands = append(commands,
         run("frida", "-U", "-l", f.script, "-f", f.app),
      )
   }
   for _, command := range commands {
      fmt.Println(command.Args)
      err := command.Start()
      if err != nil {
         return err
      }
      if command.wait {
         err := command.Wait()
         if err != nil {
            return err
         }
      } else {
         time.Sleep(time.Second)
      }
   }
   return nil
}

func stem(s string) string {
   base := filepath.Base(s)
   ext := filepath.Ext(base)
   return base[:len(base)-len(ext)]
}

type command struct {
   *exec.Cmd
   wait bool
}

func new_command(wait bool, name string, arg ...string) command {
   var cmd command
   cmd.Cmd = exec.Command(name, arg...)
   cmd.Stdout = os.Stdout
   cmd.Stderr = os.Stderr
   cmd.wait = wait
   return cmd
}

func run(name string, arg ...string) command {
   return new_command(true, name, arg...)
}

func start(name string, arg ...string) command {
   return new_command(false, name, arg...)
}

func download_server(in, out string) error {
   fmt.Println("Stat", out)
   _, err := os.Stat(out)
   if err == nil {
      return nil
   }
   fmt.Println("GET", in)
   res, err := http.Get(in)
   if err != nil {
      return err
   }
   defer res.Body.Close()
   read, err := xz.NewReader(res.Body, 0)
   if err != nil {
      return err
   }
   data, err := io.ReadAll(read)
   if err != nil {
      return err
   }
   return os.WriteFile(out, data, 0777)
}

type flags struct {
   app string
   script string
   version string
}

func main() {
   var f flags
   flag.StringVar(&f.app, "a", "", "app")
   flag.StringVar(&f.script, "s", "", "script")
   flag.StringVar(&f.version, "v", "16.0.0", "version")
   flag.Parse()
   if _, err := exec.LookPath("frida"); err != nil {
      panic("pip install frida-tools")
   }
   if f.app != "" && f.script != "" {
      home, err := os.UserHomeDir()
      if err != nil {
         panic(err)
      }
      server := f.server()
      home += "/Documents/" + stem(server)
      if err := download_server(server, home); err != nil {
         panic(err)
      }
      if err := f.run(home); err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}

func (f flags) server() string {
   var b strings.Builder
   b.WriteString("https://github.com/frida/frida/releases/download/")
   b.WriteString(f.version)
   b.WriteString("/frida-server-")
   b.WriteString(f.version)
   b.WriteString("-android-x86.xz")
   return b.String()
}
