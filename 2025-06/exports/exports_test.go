package exports

import (
   "fmt"
   "os"
   "os/exec"
   "regexp"
   "sort"
   "strings"
   "testing"
)

func Test_Exports(t *testing.T) {
   home, err := home_dir()
   if err != nil {
      t.Fatal(err)
   }
   exps, err := Exports(home)
   if err != nil {
      t.Fatal(err)
   }
   sort.Slice(exps, func(i, j int) bool {
      return fmt.Sprint(exps[i]) < fmt.Sprint(exps[j])
   })
   for _, exp := range exps {
      fmt.Println(exp)
   }
}

func print_line(s string) bool {
   if regexp.MustCompile(`^\t[a-z_][^ .]*( |$)`).MatchString(s) {
      return false
   }
   if regexp.MustCompile("^const [a-z_]").MatchString(s) {
      return false
   }
   if regexp.MustCompile("^func [a-z_]").MatchString(s) {
      return false
   }
   if regexp.MustCompile(`^func \([^)]+\) [a-z_]`).MatchString(s) {
      return false
   }
   if regexp.MustCompile("^type [a-z_]").MatchString(s) {
      return false
   }
   if regexp.MustCompile("^var [a-z_]").MatchString(s) {
      return false
   }
   if regexp.MustCompile(`^[^(]+\)`).MatchString(s) {
      return false
   }
   if s == ")" {
      return false
   }
   if s == "CONSTANTS" {
      return false
   }
   if s == "FUNCTIONS" {
      return false
   }
   if s == "TYPES" {
      return false
   }
   if s == "VARIABLES" {
      return false
   }
   if s == "const (" {
      return false
   }
   if s == "var (" {
      return false
   }
   if s == "}" {
      return false
   }
   if strings.HasPrefix(s, "    ") {
      return false
   }
   if strings.HasPrefix(s, "\t//") {
      return false
   }
   if strings.HasPrefix(s, "package ") {
      return false
   }
   if strings.HasSuffix(s, ",") {
      return false
   }
   return true
}

func Test_Doc(t *testing.T) {
   cmd := exec.Command("go", "doc", "-all", "-u")
   var err error
   cmd.Dir, err = home_dir()
   if err != nil {
      t.Fatal(err)
   }
   out, err := cmd.Output()
   if err != nil {
      t.Fatal(err)
   }
   lines := strings.FieldsFunc(string(out), func(r rune) bool {
      return r == '\n'
   })
   for _, line := range lines {
      if print_line(line) {
         fmt.Println(line)
         if regexp.MustCompile(`^\t\w+, `).MatchString(line) {
            fmt.Println(line)
         }
      }
   }
}

func home_dir() (string, error) {
   home, err := os.UserHomeDir()
   if err != nil {
      return "", err
   }
   return home + "/Documents/utls-2179f28", nil
}
