package justwatch

import (
   "fmt"
   "testing"
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
   offer := make(Offers)
   for _, tag := range content.Href_Lang_Tags {
      if tag.Language() == "en" {
         v := tag.Variables()
         detail, err := v.Details()
         if err != nil {
            t.Fatal(err)
         }
         offer.Add(v.Country, detail)
         time.Sleep(99 * time.Millisecond)
      }
   }
   text, err := offer.Stream().Text()
   if err != nil {
      t.Fatal(err)
   }
   fmt.Println(text)
}
