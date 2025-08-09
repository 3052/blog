package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "math"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
)

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   BaseURL         string           `xml:"BaseURL"`
   Duration        string           `xml:"duration,attr"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   AdaptationSets  []AdaptationSet  `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
   Initialization string `xml:"initialization,attr"`
   Media          string `xml:"media,attr"`
   StartNumber    int    `xml:"startNumber,attr"`
   EndNumber      int    `xml:"endNumber,attr"`
   Duration       int    `xml:"duration,attr"`
   Timescale      int    `xml:"timescale,attr"`
   Times          []S    `xml:"SegmentTimeline>S"`
}

type S struct {
   T int `xml:"t,attr"`
   D int `xml:"d,attr"`
   R int `xml:"r,attr"`
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

func main() {
   if len(os.Args) != 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path>")
      return
   }

   data, err := os.ReadFile(os.Args[1])
   if err != nil {
      panic(err)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      panic(err)
   }

   base, _ := url.Parse("http://test.test/test.mpd")

   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      periodBase := resolveURL(base, mpd.BaseURL)
      periodBase = resolveURL(periodBase, period.BaseURL)

      for _, aset := range period.AdaptationSets {
         asetBase := resolveURL(periodBase, aset.BaseURL)

         for _, rep := range aset.Representations {
            repBase := resolveURL(asetBase, rep.BaseURL)

            var urls []string
            // Append to existing if already present (multi-period)
            if existing, ok := result[rep.ID]; ok {
               urls = existing
            }

            // SEGMENT LIST HANDLING
            if segList := selectSegmentList(rep.SegmentList, aset.SegmentList, period.SegmentList); segList != nil {
               if segList.Initialization != nil && segList.Initialization.SourceURL != "" {
                  urls = append(urls, resolveToString(repBase, segList.Initialization.SourceURL))
               }
               for _, su := range segList.SegmentURLs {
                  if su.Media != "" {
                     urls = append(urls, resolveToString(repBase, su.Media))
                  }
               }
               result[rep.ID] = urls
               continue
            }

            // SEGMENT TEMPLATE HANDLING
            if tpl := selectSegmentTemplate(rep.SegmentTemplate, aset.SegmentTemplate, period.SegmentTemplate); tpl != nil {

               // before using tpl.Timescale
               if tpl.Timescale == 0 {
                  tpl.Timescale = 1
               }

               // Initialization
               if tpl.Initialization != "" {
                  initURL := replacePlaceholders(tpl.Initialization, rep.ID, 0, 0)
                  urls = append(urls, resolveToString(repBase, initURL))
               }

               // SegmentTimeline
               if len(tpl.Times) > 0 {
                  num := tpl.StartNumber
                  if num == 0 {
                     num = 1
                  }
                  var currentTime int
                  for i, s := range tpl.Times {
                     if i == 0 {
                        if s.T != 0 {
                           currentTime = s.T
                        } else {
                           currentTime = 0
                        }
                     } else {
                        if s.T != 0 {
                           currentTime = s.T
                        }
                     }
                     repeat := s.R
                     if repeat < 0 {
                        repeat = 0
                     }
                     for r := 0; r <= repeat; r++ {
                        segName := replacePlaceholders(tpl.Media, rep.ID, num, currentTime)
                        urls = append(urls, resolveToString(repBase, segName))
                        currentTime += s.D
                        num++
                     }
                  }
               } else if tpl.EndNumber != 0 {
                  start := tpl.StartNumber
                  if start == 0 {
                     start = 1
                  }
                  for number := start; number <= tpl.EndNumber; number++ {
                     segName := replacePlaceholders(tpl.Media, rep.ID, number, 0)
                     urls = append(urls, resolveToString(repBase, segName))
                  }
               } else if tpl.Duration != 0 && tpl.Timescale != 0 && period.Duration != "" {
                  start := tpl.StartNumber
                  if start == 0 {
                     start = 1
                  }
                  periodSeconds := parseISODuration(period.Duration)
                  totalSegments := int(math.Ceil(periodSeconds * float64(tpl.Timescale) / float64(tpl.Duration)))
                  for number := start; number < start+totalSegments; number++ {
                     segName := replacePlaceholders(tpl.Media, rep.ID, number, 0)
                     urls = append(urls, resolveToString(repBase, segName))
                  }
               }
               result[rep.ID] = urls
               continue
            }

            // FALLBACK: BaseURL only
            if rep.BaseURL != "" {
               urls = append(urls, repBase.String())
               result[rep.ID] = urls
            }
         }
      }
   }

   out, _ := json.MarshalIndent(result, "", "  ")
   fmt.Println(string(out))
}

func selectSegmentTemplate(repTpl, asetTpl, periodTpl *SegmentTemplate) *SegmentTemplate {
   if repTpl != nil {
      return repTpl
   }
   if asetTpl != nil {
      return asetTpl
   }
   return periodTpl
}

func selectSegmentList(repList, asetList, periodList *SegmentList) *SegmentList {
   if repList != nil {
      return repList
   }
   if asetList != nil {
      return asetList
   }
   return periodList
}

func resolveURL(base *url.URL, ref string) *url.URL {
   if ref == "" {
      return base
   }
   refURL, err := url.Parse(ref)
   if err != nil {
      return base
   }
   return base.ResolveReference(refURL)
}

func resolveToString(base *url.URL, ref string) string {
   return resolveURL(base, ref).String()
}

func parseISODuration(dur string) float64 {
   re := regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?$`)
   m := re.FindStringSubmatch(dur)
   if m == nil {
      return 0
   }
   hours := 0.0
   mins := 0.0
   secs := 0.0
   if m[1] != "" {
      h, _ := strconv.ParseFloat(m[1], 64)
      hours = h
   }
   if m[2] != "" {
      mn, _ := strconv.ParseFloat(m[2], 64)
      mins = mn
   }
   if m[3] != "" {
      s, _ := strconv.ParseFloat(m[3], 64)
      secs = s
   }
   return hours*3600 + mins*60 + secs
}

func replacePlaceholders(s, repID string, number, time int) string {
   s = strings.ReplaceAll(s, "$RepresentationID$", repID)

   // Replace $Number$ or $Number%0Xd$
   reNumber := regexp.MustCompile(`\$Number(%0?\d*d)?\$`)
   s = reNumber.ReplaceAllStringFunc(s, func(match string) string {
      sub := reNumber.FindStringSubmatch(match)
      if sub[1] != "" {
         // Has formatting
         return fmt.Sprintf(sub[1], number)
      }
      return fmt.Sprintf("%d", number)
   })

   // Replace $Time$
   reTime := regexp.MustCompile(`\$Time\$`)
   s = reTime.ReplaceAllString(s, fmt.Sprintf("%d", time))

   return s
}
