package palgate

import (
   "encoding/json"
   "net/http"
)

type one_time_token struct {
   K string
   P string
}

// @o("un/k11")
// e<OneTimeTokenResponse> requestOneTimeToken(@a Map<String, Object> map);
func (o *one_time_token) New() error {
   req, err := http.NewRequest(
      "POST", "https://api1.pal-es.com/v1/bt/un/k11", nil,
   )
   if err != nil {
      return err
   }
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   return json.NewDecoder(resp.Body).Decode(o)
}
