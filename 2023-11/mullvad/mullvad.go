package mullvad

import (
   "encoding/json"
   "net/http"
   "sort"
)

func (r relays) countries() []string {
   m := make(map[string]struct{})
   for _, location := range r.Locations {
      m[location.Country] = struct{}{}
   }
   var s []string
   for country := range m {
      s = append(s, country)
   }
   sort.Strings(s)
   return s
}

type relays struct {
   Locations map[string]struct {
      Country string
   }
}

func new_relays() (*relays, error) {
   res, err := http.Get("https://api.mullvad.net/app/v1/relays")
   if err != nil {
      return nil, err
   }
   defer res.Body.Close()
   relay := new(relays)
   if err := json.NewDecoder(res.Body).Decode(relay); err != nil {
      return nil, err
   }
   return relay, nil
}
