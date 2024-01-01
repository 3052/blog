package main

import (
   "fmt"
   "io"
   "net/http"
   "os"
   "path/filepath"
)

func do_git(home string) error {
   home = filepath.Join(home, "git")
   local := filepath.Join(home, filepath.Base(release))
   fmt.Println("Stat", local)
   if _, err := os.Stat(local); err != nil {
      err := download(release, local)
      if err != nil {
         return err
      }
   }
   if err := zip.Zip(local, filepath.Dir(local), 0); err != nil {
      return err
   }
   for _, file := range files {
      in := filepath.Join(home, file)
      out := filepath.Join(`C:\git`, file)
      if err := os.Rename(in, out); err != nil {
         return err
      }
   }
   return nil
}

const release = "https://github.com" +
   "/git-for-windows/git/releases/download/v2.35.1.windows.1/" +
   "MinGit-2.35.1-busybox-64-bit.zip"

var files = []string{
   "mingw64/bin/git-remote-https.exe",
   "mingw64/bin/git.exe",
   "mingw64/bin/libbrotlicommon.dll",
   "mingw64/bin/libbrotlidec.dll",
   "mingw64/bin/libcrypto-1_1-x64.dll",
   "mingw64/bin/libcurl-4.dll",
   "mingw64/bin/libiconv-2.dll",
   "mingw64/bin/libidn2-0.dll",
   "mingw64/bin/libintl-8.dll",
   "mingw64/bin/libnghttp2-14.dll",
   "mingw64/bin/libpcre2-8-0.dll",
   "mingw64/bin/libssh2-1.dll",
   "mingw64/bin/libssl-1_1-x64.dll",
   "mingw64/bin/libssp-0.dll",
   "mingw64/bin/libunistring-2.dll",
   "mingw64/bin/libwinpthread-1.dll",
   "mingw64/bin/libzstd.dll",
   "mingw64/bin/zlib1.dll",
   "mingw64/ssl/certs/ca-bundle.crt",
}

func download(in, out string) error {
   fmt.Println("GET", in)
   res, err := http.Get(in)
   if err != nil {
      return err
   }
   defer res.Body.Close()
   data, err := io.ReadAll(res.Body)
   if err != nil {
      return err
   }
   return os.WriteFile(out, data, 0666)
}
