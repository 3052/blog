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
   for _, tag := range content.Href_Lang_Tags {
      offers, err := tag.Offers()
      if err != nil {
         t.Fatal(err)
      }
      for _, offer := range offers {
         fmt.Println(tag.Locale.String(offer))
      }
      time.Sleep(99 * time.Millisecond)
   }
}
