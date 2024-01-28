package justwatch

import (
   "errors"
   "html"
   "sort"
   "strings"
)

func sort_keys[M ~map[string]V, V any](group M) []string {
   var keys []string
   for key := range group {
      keys = append(keys, key)
   }
   sort.Strings(keys)
   return keys
}

type Lang_Tag struct {
   Href string // fullPath
   Href_Lang string // country
}

func (t Lang_Tag) Country_Code() string {
   _, code, _ := strings.Cut(t.Href_Lang, "-")
   return code
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

func get_country(code string) (string, error) {
   country, found := countries[code]
   if !found {
      return "", errors.New("country code " + code)
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

// iban.com/country-codes
var countries = map[string]string{
   "AD": "Andorra",
   "AE": "United Arab Emirates (the)",
   "AG": "Antigua and Barbuda",
   "AL": "Albania",
   "AO": "Angola",
   "AR": "Argentina",
   "AT": "Austria",
   "AU": "Australia",
   "BB": "Barbados",
   "BE": "Belgium",
   "BG": "Bulgaria",
   "BM": "Bermuda",
   "BO": "Bolivia (Plurinational State of)",
   "BR": "Brazil",
   "BS": "Bahamas",
   "BZ": "Belize",
   "CA": "Canada",
   "CH": "Switzerland",
   "CL": "Chile",
   "CM": "Cameroon",
   "CO": "Colombia",
   "CR": "Costa Rica",
   "CZ": "Czechia",
   "DE": "Germany",
   "DK": "Denmark",
   "DO": "Dominican Republic (the)",
   "DZ": "Algeria",
   "EC": "Ecuador",
   "EE": "Estonia",
   "EG": "Egypt",
   "ES": "Spain",
   "FI": "Finland",
   "FJ": "Fiji",
   "FR": "France",
   "GB": "United Kingdom of Great Britain",
   "GG": "Guernsey",
   "GH": "Ghana",
   "GI": "Gibraltar",
   "GR": "Greece",
   "GT": "Guatemala",
   "GY": "Guyana",
   "HK": "Hong Kong",
   "HN": "Honduras",
   "HR": "Croatia",
   "HU": "Hungary",
   "ID": "Indonesia",
   "IE": "Ireland",
   "IL": "Israel",
   "IN": "India",
   "IQ": "Iraq",
   "IS": "Iceland",
   "IT": "Italy",
   "JM": "Jamaica",
   "JP": "Japan",
   "KE": "Kenya",
   "KR": "Korea (the Republic of)",
   "LC": "Saint Lucia",
   "LT": "Lithuania",
   "MA": "Morocco",
   "MW": "Malawi",
   "MX": "Mexico",
   "MY": "Malaysia",
   "NG": "Nigeria",
   "NL": "Netherlands (the)",
   "NO": "Norway",
   "NZ": "New Zealand",
   "PA": "Panama",
   "PE": "Peru",
   "PG": "Papua New Guinea",
   "PH": "Philippines (the)",
   "PK": "Pakistan",
   "PL": "Poland",
   "PT": "Portugal",
   "PY": "Paraguay",
   "RO": "Romania",
   "RS": "Serbia",
   "RU": "Russian Federation (the)",
   "RW": "Rwanda",
   "SA": "Saudi Arabia",
   "SE": "Sweden",
   "SG": "Singapore",
   "SI": "Slovenia",
   "SK": "Slovakia",
   "SV": "El Salvador",
   "TC": "Turks and Caicos",
   "TH": "Thailand",
   "TR": "Turkey",
   "TT": "Trinidad and Tobago",
   "TW": "Taiwan (Province of China)",
   "UA": "Ukraine",
   "UG": "Uganda",
   "US": "United States of America (the)",
   "UY": "Uruguay",
   "VE": "Venezuela (Bolivarian Republic of)",
   "ZA": "South Africa",
   "ZM": "Zambia",
   "ZW": "Zimbabwe",
}

var Blacklist = map[string]bool{
   "ja-JP": true,
   "ru-RU": true,
}
