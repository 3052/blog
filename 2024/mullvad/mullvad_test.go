package mullvad

import (
   "fmt"
   "testing"
)

func Test_Relay(t *testing.T) {
   var relay app_relay
   err := relay.get()
   if err != nil {
      t.Fatal(err)
   }
   relay.countries(func(country string) {
      fmt.Println(country)
   })
}
