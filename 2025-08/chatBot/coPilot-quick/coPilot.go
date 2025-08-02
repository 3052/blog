package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "net/url"
   "os"
   "regexp"
   "strconv"
)

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   BaseURL        string          `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

type SegmentTemplate struct {
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   Timescale       int              `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []SegmentTime `xml:"S"`
}

type SegmentTime struct {
   T int64 `xml:"t,attr"`
   D int64 `xml:"d,attr"`
   R int64 `xml:"r,attr"`
}

func main() {
   if len(os.Args) < 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path>")
      return
   }

   raw, err := os.ReadFile(os.Args[1])
   if err != nil {
      panic(err)
   }

   var mpd MPD
   if err := xml.Unmarshal(raw, &mpd); err != nil {
      panic(err)
   }

   startBase, _ := url.Parse("http://test.test/test.mpd")
   baseMPD := resolve(startBase, mpd.BaseURL)

   output := map[string][]string{}

   for _, period := range mpd.Periods {
      basePeriod := resolve(baseMPD, period.BaseURL)

      for _, as := range period.AdaptationSets {
         baseAS := resolve(basePeriod, as.BaseURL)

         for _, rep := range as.Representations {
            baseRep := resolve(baseAS, rep.BaseURL)
            repID := rep.ID
            var urls []string

            // SegmentTemplate inheritance
            tmpl := rep.SegmentTemplate
            if tmpl == nil {
               tmpl = as.SegmentTemplate
            }

            // SegmentList initialization
            if rep.SegmentList != nil {
               if rep.SegmentList.Initialization != nil && rep.SegmentList.Initialization.SourceURL != "" {
                  initURL := resolve(baseRep, rep.SegmentList.Initialization.SourceURL).String()
                  urls = append(urls, initURL)
               }
            }

            // SegmentTemplate initialization
            if tmpl != nil && tmpl.Initialization != "" {
               initMedia := replaceTokens(tmpl.Initialization, map[string]interface{}{
                  "RepresentationID": repID,
               })
               urls = append(urls, resolve(baseRep, initMedia).String())
            }

            // SegmentList media
            if rep.SegmentList != nil {
               for _, seg := range rep.SegmentList.SegmentURLs {
                  urls = append(urls, resolve(baseRep, seg.Media).String())
               }
            } else if tmpl != nil {
               // SegmentTemplate with SegmentTimeline
               if tmpl.SegmentTimeline != nil {
                  var currentTime int64 = 0
                  for _, s := range tmpl.SegmentTimeline.S {
                     if s.T != 0 {
                        currentTime = s.T
                     }
                     total := s.R + 1
                     for i := int64(0); i < total; i++ {
                        media := replaceTokens(tmpl.Media, map[string]interface{}{
                           "RepresentationID": repID,
                           "Time":             currentTime,
                        })
                        urls = append(urls, resolve(baseRep, media).String())
                        currentTime += s.D
                     }
                  }
               } else {
                  for i := tmpl.StartNumber; i <= tmpl.EndNumber; i++ {
                     media := replaceTokens(tmpl.Media, map[string]interface{}{
                        "RepresentationID": repID,
                        "Number":           i,
                     })
                     urls = append(urls, resolve(baseRep, media).String())
                  }
               }
            }

            // BaseURL-only Representation fallback
            if rep.SegmentList == nil && tmpl == nil && rep.BaseURL != "" {
               urls = append(urls, resolve(baseAS, rep.BaseURL).String())
            }

            output[repID] = urls
         }
      }
   }

   out, _ := json.MarshalIndent(output, "", "  ")
   fmt.Println(string(out))
}

func resolve(base *url.URL, rel string) *url.URL {
   if rel == "" {
      return base
   }
   u, err := url.Parse(rel)
   if err != nil {
      return base
   }
   return base.ResolveReference(u)
}

func replaceTokens(template string, vals map[string]interface{}) string {
   res := template

   if rid, ok := vals["RepresentationID"].(string); ok {
      res = regexp.MustCompile(`\$RepresentationID\$`).ReplaceAllString(res, rid)
   }

   if num, ok := vals["Number"].(int); ok {
      res = regexp.MustCompile(`\$Number\$`).ReplaceAllString(res, fmt.Sprintf("%d", num))
      res = regexp.MustCompile(`\$Number%0(\d+)d\$`).ReplaceAllStringFunc(res, func(m string) string {
         width, _ := strconv.Atoi(regexp.MustCompile(`\d+`).FindString(m))
         return fmt.Sprintf("%0*d", width, num)
      })
   }

   if t, ok := vals["Time"].(int64); ok {
      res = regexp.MustCompile(`\$Time\$`).ReplaceAllString(res, fmt.Sprintf("%d", t))
   }

   return res
}
