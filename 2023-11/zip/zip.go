package zip

import (
   "archive/zip"
   "os"
   "path/filepath"
   "strings"
)

func strip(s string, level int) (string, bool) {
   for level >= 1 {
      var ok bool
      _, s, ok = strings.Cut(s, "/")
      if !ok {
         return "", false
      }
      level--
   }
   return s, true
}

func Zip(in, dir string, level int) error {
   read, err := zip.OpenReader(in)
   if err != nil {
      return err
   }
   defer read.Close()
   for _, head := range read.File {
      if head.Mode().IsDir() {
         continue
      }
      name, ok := strip(head.Name, level)
      if !ok {
         continue
      }
      name = filepath.Join(dir, name)
      err := func() error {
         rc, err := head.Open()
         if err != nil {
            return err
         }
         defer rc.Close()
         os.MkdirAll(filepath.Dir(name), 0666)
         file, err := os.Create(name)
         if err != nil {
            return err
         }
         defer file.Close()
         file.ReadFrom(rc)
         return nil
      }()
      if err != nil {
         return err
      }
   }
   return nil
}
