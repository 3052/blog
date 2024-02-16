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
   Href string
   Locale locale
}

func (t LangTag) Details() (*TitleDetails, error) {
   body, err := func() ([]byte, error) {
      var s struct {
         Variables struct {
            Country string
            FullPath string
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
   title := new(TitleDetails)
   if err := json.NewDecoder(res.Body).Decode(title); err != nil {
      return nil, err
   }
   return title, nil
}

var buy_rent = map[string]bool{
   "BUY": true,
   "RENT": true,
}
