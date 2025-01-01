package palgate

import (
   "fmt"
   "testing"
)

func TestToken(t *testing.T) {
   var token one_time_token
   err := token.New()
   if err != nil {
      t.Fatal(err)
   }
   fmt.Printf("%+v\n", token)
}
