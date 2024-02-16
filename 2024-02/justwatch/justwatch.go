package justwatch

import (
   "bytes"
   "encoding/json"
   "errors"
   "net/http"
   "strings"
)

// this is better than strings.Replace and strings.ReplaceAll
func graphql_compact(s string) string {
   field := strings.Fields(s)
   return strings.Join(field, " ")
}

type ContentUrls struct {
   Href_Lang_Tags []LangTag
}

func (c *ContentUrls) New(path string) error {
   path = "https://apis.justwatch.com/content/urls?path=" + path
   res, err := http.Get(path)
   if err != nil {
      return err
   }
   defer res.Body.Close()
   if res.StatusCode != http.StatusOK {
      return errors.New(res.Status)
   }
   return json.NewDecoder(res.Body).Decode(c)
}

const title_details = `
query GetUrlTitleDetails(
   $fullPath: String!
   $country: Country!
   $platform: Platform! = WEB
) {
   url(fullPath: $fullPath) {
      node {
         ... on MovieOrShowOrSeason {
            offers(country: $country, platform: $platform) {
               monetizationType
               standardWebURL
            }
         }
      }
   }
}
`

type locale struct {
   country string
   language string
}

func (l *locale) UnmarshalText(b []byte) error {
   var ok bool
   l.language, l.country, ok = strings.Cut(string(b), "_")
   if !ok {
      return errors.New("locale.UnmarshalText")
   }
   return nil
}

type LangTag struct {
   Href string // /ar/pelicula/mulholland-drive
   Locale locale // es_AR
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

func (t LangTag) Offers() ([]Offer, error) {
   body, err := func() ([]byte, error) {
      var s struct {
         Variables struct {
            Country string `json:"country"`
            FullPath string `json:"fullPath"`
         }
         Query string
      }
      s.Query = graphql_compact(title_details)
      s.Variables.Country = t.Locale.country
      s.Variables.FullPath = t.Href
      return json.Marshal(s)
   }()
   if err != nil {
      return nil, err
   }
   res, err := http.Post(
      "https://apis.justwatch.com/graphql", "application/json",
      bytes.NewReader(body),
   )
   if err != nil {
      return nil, err
   }
   defer res.Body.Close()
   if res.StatusCode != http.StatusOK {
      return nil, errors.New(res.Status)
   }
   var s struct {
      Data struct {
         URL struct {
            Node struct {
               Offers []Offer
            }
         }
      }
   }
   if err := json.NewDecoder(res.Body).Decode(&s); err != nil {
      return nil, err
   }
   return s.Data.URL.Node.Offers, nil
}

// `presentationType` data seems to be incorrect in some cases. For example,
// JustWatch reports this as SD: fetchtv.com.au/movie/details/19285
// when the site itself reports as HD
type Offer struct {
   MonetizationType string
   StandardWebUrl string
}

func (o Offer) Stream() bool {
   switch o.MonetizationType {
   case "BUY", "RENT":
      return false
   }
   return true
}

const ModeLine =
"{{ range $index, $_ := . }}" +
   "{{ if .Stream }}" +
      "{{ if $index }}" +
"===========================================================================\n" +
      "{{ end }}" +
"monetization = {{ .MonetizationType }}\n" +
"URL = {{ .StandardWebUrl }}\n" +
   "{{ end }}" +
"{{ end }}"
