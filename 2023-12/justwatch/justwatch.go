package justwatch

import (
   "bytes"
   "encoding/json"
   "fmt"
   "net/http"
   "strings"
)

func New_URLs(ref string) (*URLs, error) {
   ref = "https://apis.justwatch.com/content/urls?path=" + ref
   fmt.Println("GET", ref)
   res, err := http.Get(ref)
   if err != nil {
      return nil, err
   }
   defer res.Body.Close()
   content := new(URLs)
   if err := json.NewDecoder(res.Body).Decode(content); err != nil {
      return nil, err
   }
   return content, nil
}

func (v Variables) Details() (*Details, error) {
   body, err := func() ([]byte, error) {
      var d details_request
      d.Query = graphQL_compact(query)
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
   detail := new(Details)
   if err := json.NewDecoder(res.Body).Decode(detail); err != nil {
      return nil, err
   }
   return detail, nil
}

// this is better than strings.Replace and strings.ReplaceAll
func graphQL_compact(s string) string {
   f := strings.Fields(s)
   return strings.Join(f, " ")
}

// cant use encoding.TextMarshaler because we are JSON marshalling this
func (v Variables) Text() (string, error) {
   var b strings.Builder
   country, err := get_country(v.Country_Code)
   if err != nil {
      return "", err
   }
   b.WriteString(country)
   b.WriteByte(' ')
   b.WriteString(v.Full_Path)
   return b.String(), nil
}

type details_request struct {
   Query string
   Variables Variables
}

const query = `
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

type URLs struct {
   Href_Lang_Tags []Lang_Tag
}

// I am including `presentationType` to differentiate the different options,
// but the data seems to be incorrect in some cases. For example, JustWatch
// reports this as SD:
// fetchtv.com.au/movie/details/19285
// when the site itself reports as HD.
type Details struct {
   Data struct {
      URL struct {
         Node struct {
            Offers []struct {
               Monetization_Type string `json:"monetizationType"`
               Presentation_Type string `json:"presentationType"`
               Standard_Web_URL string `json:"standardWebURL"`
            }
         }
      }
   }
}

type Lang_Tag struct {
   Href string // fullPath
   Href_Lang string // country
}

func (t Lang_Tag) Country_Code() string {
   _, code, _ := strings.Cut(t.Href_Lang, "-")
   return code
}

func (t Lang_Tag) Language() string {
   lang, _, _ := strings.Cut(t.Href_Lang, "-")
   return lang
}

func (t Lang_Tag) Variables() Variables {
   var v Variables
   v.Country_Code = t.Country_Code()
   v.Full_Path = t.Href
   return v
}

type Variables struct {
   Country_Code string `json:"country"`
   Full_Path string `json:"fullPath"`
}
