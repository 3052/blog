package iso

import "testing"

func TestThree(t *testing.T) {
   var key securityKey
   err := key.New()
   if err != nil {
      t.Fatal(err)
   }
   if err := key.three(); err != nil {
      t.Fatal(err)
   }
}
