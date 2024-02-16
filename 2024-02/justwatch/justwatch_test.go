package justwatch

import (
   "fmt"
   "os"
   "testing"
   "text/template"
   "time"
)

// justwatch.com/us/movie/mulholland-drive
const movie = "/us/movie/mulholland-drive"

func TestContent(t *testing.T) {
   var content ContentUrls
   err := content.New(movie)
   if err != nil {
     t.Fatal(err)
   }
   for i, tag := range content.Href_Lang_Tags {
      if i >= 1 {
         fmt.Println("---------------------------------------------------------")
      }
      offers, err := tag.Offers()
      if err != nil {
         t.Fatal(err)
      }
      line, err := new(template.Template).Parse(ModeLine)
      if err != nil {
         t.Fatal(err)
      }
      if err := line.Execute(os.Stdout, offers); err != nil {
         t.Fatal(err)
      }
      time.Sleep(99 * time.Millisecond)
   }
}
