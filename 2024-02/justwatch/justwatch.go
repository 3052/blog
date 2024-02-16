package justwatch

import (
   "bytes"
   "encoding/json"
   "errors"
   "html"
   "net/http"
   "sort"
   "strings"
)

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
               presentationType
               standardWebURL
            }
         }
      }
   }
}
`

// I am including `presentationType` to differentiate the different options,
// but the data seems to be incorrect in some cases. For example, JustWatch
// reports this as SD:
// fetchtv.com.au/movie/details/19285
// when the site itself reports as HD.
type TitleDetails struct {
   Data struct {
      URL struct {
         Node struct {
            Offers []struct {
               MonetizationType string
               PresentationType string
               StandardWebUrl string
            }
         }
      }
   }
}

///////////////

type details_request struct {
   Query string
   Variables Variables
}

func (v Variables) Details() (*TitleDetails, error) {
   body, err := func() ([]byte, error) {
      var d details_request
      d.Query = graphql_compact(title_details)
      d.Variables = v
      return json.Marshal(d)
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
   detail := new(TitleDetails)
   if err := json.NewDecoder(res.Body).Decode(detail); err != nil {
      return nil, err
   }
   return detail, nil
}

func sort_keys[M ~map[string]V, V any](group M) []string {
   var keys []string
   for key := range group {
      keys = append(keys, key)
   }
   sort.Strings(keys)
   return keys
}

func (t LangTag) CountryCode() string {
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
      for _, web_url := range sort_keys(url_codes) {
         b.WriteString("\n\n")
         b.WriteString(tab)
         b.WriteString(web_url)
         for _, code := range sort_keys(url_codes[web_url]) {
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

func (o Offers) Add(country_code string, detail *TitleDetails) {
   for _, node := range detail.Data.URL.Node.Offers {
      offer := o[node.MonetizationType]
      if offer == nil {
         offer = make(map[string]CountryCodes)
      }
      ref := html.UnescapeString(node.StandardWebUrl)
      codes := offer[ref]
      if codes == nil {
         codes = make(CountryCodes)
      }
      codes[country_code] = struct{}{}
      offer[ref] = codes
      o[node.MonetizationType] = offer
   }
}

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

// this is better than strings.Replace and strings.ReplaceAll
func graphql_compact(s string) string {
   f := strings.Fields(s)
   return strings.Join(f, " ")
}

// cant use encoding.TextMarshaler because we are JSON marshalling this
func (v Variables) Text() (string, error) {
   var b strings.Builder
   country, err := get_country(v.Country)
   if err != nil {
      return "", err
   }
   b.WriteString(country)
   b.WriteByte(' ')
   b.WriteString(v.FullPath)
   return b.String(), nil
}

func (t LangTag) Language() string {
   lang, _, _ := strings.Cut(t.Href_Lang, "-")
   return lang
}

func (t LangTag) Variables() Variables {
   var v Variables
   v.Country = t.CountryCode()
   v.FullPath = t.Href
   return v
}

type LangTag struct {
   Href string
   Href_Lang string
}

type CountryCodes map[string]struct{}

// map[monetizationType]map[standardWebURL]CountryCodes
type Offers map[string]map[string]CountryCodes

type Variables struct {
   Country string
   FullPath string
}
