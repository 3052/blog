package iso

import (
   "fmt"
   "testing"
)

func TestToken(t *testing.T) {
   var key securityKey
   err := key.New()
   if err != nil {
      t.Fatal(err)
   }
   fmt.Printf("%+v\n", key.cookie)
   fmt.Printf("%+v\n", key.key)
}
