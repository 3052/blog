package manifest

import (
   "encoding/xml"
   "fmt"
   "os"
   "testing"
)

var names = []string{
   "com.canal.android.canal.xml",
   "com.canalplus.canalplus.xml",
}

func Test(t *testing.T) {
   for _, name := range names {
      fmt.Println(name)
      data, err := os.ReadFile(name)
      if err != nil {
         t.Fatal(err)
      }
      var manifest1 manifest
      err = xml.Unmarshal(data, &manifest1)
      if err != nil {
         t.Fatal(err)
      }
      for intent := range manifest1.intent_filter() {
         fmt.Print(&intent, "\n\n")
      }
   }
}
