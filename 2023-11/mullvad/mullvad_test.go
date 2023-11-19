package mullvad

import (
   "fmt"
   "testing"
)

func Test_Relay(t *testing.T) {
   relay, err := new_relays()
   if err != nil {
      t.Fatal(err)
   }
   for _, country := range relay.countries() {
      fmt.Println(country)
   }
}
