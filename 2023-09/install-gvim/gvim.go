package main

import (
   "154.pages.dev/encoding/zip"
   "fmt"
   "io"
   "net/http"
   "os"
   "path/filepath"
)

func download(in, out string) error {
   fmt.Println("GET", in)
   res, err := http.Get(in)
   if err != nil {
      return err
   }
   defer res.Body.Close()
   if err := os.MkdirAll(filepath.Dir(out), 0666); err != nil {
      return err
   }
   data, err := io.ReadAll(res.Body)
   if err != nil {
      return err
   }
   return os.WriteFile(out, data, 0666)
}

const vim_installer =
   "https://github.com/vim/vim-win32-installer/releases/download/" +
   "v8.2.3526/gvim_8.2.3526_x64.zip"

var patches = []struct{
   dir string
   base string
}{
   // github.com/fleiner/vim/issues/2
   // github.com/vim/vim/pull/8023
   {"vim/vim/a942f9ad/runtime/", "syntax/javascript.vim"},
   // github.com/tpope/vim-markdown/pull/175
   {"tpope/vim-markdown/564d7436/", "syntax/markdown.vim"},
   // github.com/NLKNguyen/papercolor-theme/pull/167
   {"NLKNguyen/papercolor-theme/e397d18a/", "colors/PaperColor.vim"},
   // github.com/vim/vim/issues/11996
   {"google/vim-ft-go/master/", "syntax/go.vim"},
}

func do_gvim(home string) error {
   home += "/vim/" + filepath.Base(vim_installer)
   fmt.Println("Stat", home)
   if _, err := os.Stat(home); err != nil {
      err := download(vim_installer, home)
      if err != nil {
         return err
      }
   }
   fmt.Println("Zip", home)
   if err := zip.Zip(home, `D:\vim`, 2); err != nil {
      return err
   }
   for _, pat := range patches {
      err := download(
         "https://raw.githubusercontent.com/" + pat.dir + pat.base,
         filepath.Join(`D:\vim`, pat.base),
      )
      if err != nil {
         return err
      }
   }
   return nil
}
