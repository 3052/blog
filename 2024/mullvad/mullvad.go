package mullvad

import (
   "encoding/json"
   "net/http"
   "slices"
)

func (a *app_relay) get() error {
   res, err := http.Get("https://api.mullvad.net/app/v1/relays")
   if err != nil {
      return err
   }
   defer res.Body.Close()
   return json.NewDecoder(res.Body).Decode(a)
}

type app_relay struct {
   Locations map[string]struct {
      Country string
   }
}

func (a app_relay) countries(f func(string)) {
   var countries []string
   for _, location := range a.Locations {
      countries = append(countries, location.Country)
   }
   slices.Sort(countries)
   for _, country := range slices.Compact(countries) {
      f(country)
   }
}
