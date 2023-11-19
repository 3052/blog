package justwatch

import (
   "fmt"
   "html"
   "sort"
   "strings"
)

var countries = map[string]string{
   "AG": "Antigua and Barbuda",
   "AU": "Australia (Mullvad)",
   "BB": "Barbados",
   "BM": "Bermuda",
   "BS": "Bahamas",
   "BZ": "Belize",
   "CA": "Canada (Mullvad)",
   "CM": "Cameroon",
   "DK": "Denmark (Mullvad)",
   "FJ": "Fiji",
   "GB": "United Kingdom (Mullvad)",
   "GG": "Guernsey",
   "GH": "Ghana",
   "GI": "Gibraltar",
   "GY": "Guyana",
   "ID": "Indonesia",
   "IE": "Ireland (Mullvad)",
   "IN": "India",
   "JM": "Jamaica",
   "KE": "Kenya",
   "LC": "Saint Lucia",
   "MW": "Malawi",
   "MY": "Malaysia",
   "NG": "Nigeria",
   "NL": "Netherlands (Mullvad)",
   "NO": "Norway (Mullvad)",
   "NZ": "New Zealand (Mullvad)",
   "PG": "Papua New Guinea",
   "PH": "Philippines",
   "SG": "Singapore (Mullvad)",
   "TC": "Turks and Caicos",
   "TH": "Thailand",
   "TT": "Trinidad and Tobago",
   "UG": "Uganda",
   "US": "United States (Mullvad)",
   "ZA": "South Africa (Mullvad)",
   "ZM": "Zambia",
   "ZW": "Zimbabwe",
}

// lets match Variables type
func (o Offers) Text() (string, error) {
   var b strings.Builder
   for mon_type, url_codes := range o {
      if b.Len() >= 1 {
         b.WriteByte('\n')
      }
      b.WriteString(mon_type)
      for _, web_URL := range sort_keys(url_codes) {
         b.WriteString("\n\n")
         b.WriteString(tab)
         b.WriteString(web_URL)
         for _, code := range sort_keys(url_codes[web_URL]) {
            b.WriteByte('\n')
            b.WriteString(tab)
            b.WriteString("- ")
            country, err := get_country(code)
            if err != nil {
               return "", err
            }
            b.WriteString(country)
         }
      }
   }
   return b.String(), nil
}

func (o Offers) Add(country_code string, detail *Details) {
   for _, node := range detail.Data.URL.Node.Offers {
      offer := o[node.Monetization_Type]
      if offer == nil {
         offer = make(map[string]Country_Codes)
      }
      ref := html.UnescapeString(node.Standard_Web_URL)
      codes := offer[ref]
      if codes == nil {
         codes = make(Country_Codes)
      }
      codes[country_code] = struct{}{}
      offer[ref] = codes
      o[node.Monetization_Type] = offer
   }
}

type Country_Codes map[string]struct{}

// map[monetizationType]map[standardWebURL]Country_Codes
type Offers map[string]map[string]Country_Codes

func sort_keys[M ~map[string]V, V any](group M) []string {
   var keys []string
   for key := range group {
      keys = append(keys, key)
   }
   sort.Strings(keys)
   return keys
}

func get_country(code string) (string, error) {
   country, found := countries[code]
   if !found {
      return "", fmt.Errorf("country code %q not found", code)
   }
   return country, nil
}

func (o Offers) Stream() Offers {
   p := make(Offers)
   for m_type, offer := range o {
      if !buy_rent[m_type] {
         p[m_type] = offer
      }
   }
   return p
}

const tab = "   "

var buy_rent = map[string]bool{
   "BUY": true,
   "RENT": true,
}
