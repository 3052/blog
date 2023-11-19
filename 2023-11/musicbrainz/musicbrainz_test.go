package musicbrainz

import (
   "fmt"
   "testing"
)

func Test_Release(t *testing.T) {
   rel, err := New_Release("94f940c4-b1f0-4ccc-8886-424319eb39c8")
   if err != nil {
      t.Fatal(err)
   }
   fmt.Printf("%+v\n", rel)
}

func Test_Artist(t *testing.T) {
   _, err := From_Artist("678d88b2-87b0-403b-b63d-5da7465aecc3", 0)
   if err != nil {
      t.Fatal(err)
   }
}

func Test_Group(t *testing.T) {
   group, err := New_Release_Group("9b5006e5-b276-3a05-bcdd-8d986842320b")
   if err != nil {
      t.Fatal(err)
   }
   group.Sort()
   date := group.Releases[0].Date
   if date != "1973-03-28" {
      t.Fatal(date)
   }
}
