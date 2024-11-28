package main

import (
   "crypto/md5"
   "crypto/x509"
   "encoding/hex"
   "encoding/pem"
   "flag"
   "fmt"
   "os"
   "os/exec"
   "path/filepath"
)

// outputs the MD5 "hash" of the certificate subject name
func subject_hash(raw []byte) ([]byte, error) {
   block, _ := pem.Decode(raw)
   cert, err := x509.ParseCertificate(block.Bytes)
   if err != nil {
      return nil, err
   }
   sum := md5.Sum(cert.RawSubject)
   return []byte{sum[3], sum[2], sum[1], sum[0]}, nil
}

func main() {
   var f flags
   f.cert = func() string {
      s, err := os.UserHomeDir()
      if err != nil {
         panic(err)
      }
      return filepath.ToSlash(s) + "/.mitmproxy/mitmproxy-ca-cert.pem"
   }()
   flag.StringVar(&f.cert, "c", f.cert, "certificate")
   flag.BoolVar(&f.info, "i", false, "information")
   flag.Parse()
   push := func() string {
      b, err := os.ReadFile(f.cert)
      if err != nil {
         panic(err)
      }
      b, err = subject_hash(b)
      if err != nil {
         panic(err)
      }
      return hex.EncodeToString(b) + ".0"
   }()
   commands := [][]string{
      {"adb", "shell", "mkdir", "-p", data},
      {"adb", "shell", "cp", system + "/*", data},
      {"adb", "push", f.cert, data + "/" + push},
      {"adb", "root"},
      {"adb", "wait-for-device"},
      {"adb", "shell", "mount", "-t", "tmpfs", "tmpfs", system},
      // mv fails with Android API 18
      {"adb", "shell", "cp", data + "/*", system},
      {"adb", "shell", "chcon", "u:object_r:system_file:s0", system + "/*"},
   }
   for _, command := range commands {
      cmd := exec.Command(command[0], command[1:]...)
      cmd.Stderr = os.Stderr
      cmd.Stdout = os.Stdout
      fmt.Println(cmd.Args)
      if !f.info {
         err := cmd.Run()
         if err != nil {
            panic(err)
         }
      }
   }
}

const (
   data = "/data/local/tmp/cacerts"
   system = "/system/etc/security/cacerts"
)

type flags struct {
   cert string
   info bool
}
