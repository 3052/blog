package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "log"
   "math"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
)

// numberPattern matches $Number$ or $Number%08d$-style specifiers
var numberPattern = regexp.MustCompile(`\$Number(%[^$]+)?\$`)

// replacePlaceholders substitutes Number (with optional fmt spec), Time and RepresentationID
func replacePlaceholders(template string, num, currentTime int64, repID string) string {
   // replace $Number...$
   result := numberPattern.ReplaceAllStringFunc(template, func(m string) string {
      // m is like "$Number%08d$" or "$Number$"
      spec := m[len("$Number") : len(m)-1] // "%08d" or ""
      if spec == "" {
         return strconv.FormatInt(num, 10)
      }
      // ensure spec begins with '%'
      return fmt.Sprintf(spec, num)
   })
   // simple replaces for time and rep ID
   result = strings.ReplaceAll(result, "$Time$", strconv.FormatInt(currentTime, 10))
   result = strings.ReplaceAll(result, "$RepresentationID$", repID)
   return result
}

// MPD represents the root element
type MPD struct {
   XMLName         xml.Name         `xml:"MPD"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Periods         []Period         `xml:"Period"`
}

// Period may include a duration attribute in ISO8601 format
type Period struct {
   Duration        string           `xml:"duration,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   AdaptationSets  []AdaptationSet  `xml:"AdaptationSet"`
}

// AdaptationSet groups Representations
type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

// Representation holds segment info or falls back to Template/BaseURL
type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

// SegmentList with optional Initialization@sourceURL
type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

// Initialization element inside SegmentList
type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

// SegmentURL inside SegmentList
type SegmentURL struct {
   Media string `xml:"media,attr"`
}

// SegmentTemplate for timeline or template-based segments
type SegmentTemplate struct {
   Timescale       int64            `xml:"timescale,attr"`
   Duration        int64            `xml:"duration,attr"`
   StartNumber     int64            `xml:"startNumber,attr"`
   EndNumber       int64            `xml:"endNumber,attr"`
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline with multiple S entries
type SegmentTimeline struct {
   S []SegmentTimelineS `xml:"S"`
}

// SegmentTimelineS describes a run of segments
type SegmentTimelineS struct {
   T *int64 `xml:"t,attr"`
   D int64  `xml:"d,attr"`
   R *int   `xml:"r,attr"`
}

// chooseTemplate picks the most specific SegmentTemplate: rep > adap > period > mpd
func chooseTemplate(mpdT, periodT, adapT, repT *SegmentTemplate) *SegmentTemplate {
   if repT != nil {
      return repT
   }
   if adapT != nil {
      return adapT
   }
   if periodT != nil {
      return periodT
   }
   return mpdT
}

// parsePeriodDuration parses an ISO8601 duration like PT1H2M3.5S
func parsePeriodDuration(s string) (float64, error) {
   re := regexp.MustCompile(`PT(?:(\d+(?:\.\d+)?)H)?(?:(\d+(?:\.\d+)?)M)?(?:(\d+(?:\.\d+)?)S)?`)
   m := re.FindStringSubmatch(s)
   if m == nil {
      return 0, fmt.Errorf("invalid duration format: %s", s)
   }
   var secs float64
   if m[1] != "" {
      h, _ := strconv.ParseFloat(m[1], 64)
      secs += h * 3600
   }
   if m[2] != "" {
      mn, _ := strconv.ParseFloat(m[2], 64)
      secs += mn * 60
   }
   if m[3] != "" {
      sec, _ := strconv.ParseFloat(m[3], 64)
      secs += sec
   }
   return secs, nil
}

func main() {
   if len(os.Args) < 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path>")
      os.Exit(1)
   }
   mpdPath := os.Args[1]

   data, err := ioutil.ReadFile(mpdPath)
   if err != nil {
      log.Fatalf("Failed to read MPD file: %v", err)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      log.Fatalf("Failed to parse MPD XML: %v", err)
   }

   // initial URL for resolving all relative BaseURLs
   initialMPD := "http://test.test/test.mpd"
   base, err := url.Parse(initialMPD)
   if err != nil {
      log.Fatalf("Invalid initial MPD URL: %v", err)
   }
   if mpd.BaseURL != "" {
      if resolved, err := base.Parse(mpd.BaseURL); err == nil {
         base = resolved
      }
   }

   segmentsMap := make(map[string][]string)

   for _, period := range mpd.Periods {
      periodBase := base
      if period.BaseURL != "" {
         if resolved, err := periodBase.Parse(period.BaseURL); err == nil {
            periodBase = resolved
         }
      }

      // parse Period duration once
      pdur := 0.0
      if period.Duration != "" {
         if d, err := parsePeriodDuration(period.Duration); err == nil {
            pdur = d
         }
      }

      for _, adap := range period.AdaptationSets {
         adapBase := periodBase
         if adap.BaseURL != "" {
            if resolved, err := adapBase.Parse(adap.BaseURL); err == nil {
               adapBase = resolved
            }
         }

         for _, rep := range adap.Representations {
            repBase := adapBase
            if rep.BaseURL != "" {
               if resolved, err := repBase.Parse(rep.BaseURL); err == nil {
                  repBase = resolved
               }
            }

            tmpl := chooseTemplate(
               mpd.SegmentTemplate,
               period.SegmentTemplate,
               adap.SegmentTemplate,
               rep.SegmentTemplate,
            )

            var urls []string

            // 1) explicit SegmentList
            if rep.SegmentList != nil {
               if rep.SegmentList.Initialization != nil && rep.SegmentList.Initialization.SourceURL != "" {
                  if full, err := repBase.Parse(rep.SegmentList.Initialization.SourceURL); err == nil {
                     urls = append(urls, full.String())
                  }
               }
               for _, seg := range rep.SegmentList.SegmentURLs {
                  if seg.Media == "" {
                     continue
                  }
                  if full, err := repBase.Parse(seg.Media); err == nil {
                     urls = append(urls, full.String())
                  }
               }

               // 2) template-based
            } else if tmpl != nil {
               // initialization from template
               if tmpl.Initialization != "" {
                  initURL := replacePlaceholders(
                     tmpl.Initialization, 0, 0, rep.ID,
                  )
                  if full, err := repBase.Parse(initURL); err == nil {
                     urls = append(urls, full.String())
                  }
               }

               // media segments
               if tmpl.Media != "" {
                  // default timescale to 1
                  ts := tmpl.Timescale
                  if ts == 0 {
                     ts = 1
                  }

                  start := tmpl.StartNumber
                  if start == 0 {
                     start = 1
                  }

                  switch {
                  // a) timeline-based
                  case tmpl.SegmentTimeline != nil:
                     var idx, curTime int64
                     for _, s := range tmpl.SegmentTimeline.S {
                        reps := 1
                        if s.R != nil {
                           reps = *s.R + 1
                        }
                        if s.T != nil {
                           curTime = *s.T
                        }
                        for r := 0; r < reps; r++ {
                           num := start + idx
                           segment := replacePlaceholders(
                              tmpl.Media, num, curTime, rep.ID,
                           )
                           if full, err := repBase.Parse(segment); err == nil {
                              urls = append(urls, full.String())
                           }
                           curTime += s.D
                           idx++
                        }
                     }

                  // b) endNumber-based
                  case tmpl.EndNumber > 0:
                     for num := start; num <= tmpl.EndNumber; num++ {
                        segment := replacePlaceholders(
                           tmpl.Media, num, 0, rep.ID,
                        )
                        if full, err := repBase.Parse(segment); err == nil {
                           urls = append(urls, full.String())
                        }
                     }

                  // c) duration/timescale + periodDuration
                  case tmpl.Duration > 0 && pdur > 0:
                     count := int(math.Ceil(pdur * float64(ts) / float64(tmpl.Duration)))
                     for i := 0; i < count; i++ {
                        num := start + int64(i)
                        segment := replacePlaceholders(
                           tmpl.Media, num, 0, rep.ID,
                        )
                        if full, err := repBase.Parse(segment); err == nil {
                           urls = append(urls, full.String())
                        }
                     }

                  // d) single-segment fallback
                  default:
                     num := start
                     segment := replacePlaceholders(
                        tmpl.Media, num, 0, rep.ID,
                     )
                     if full, err := repBase.Parse(segment); err == nil {
                        urls = append(urls, full.String())
                     }
                  }
               }
            }

            // 3) no segment info → direct BaseURL
            if len(urls) == 0 {
               urls = append(urls, repBase.String())
            }

            // append segments across periods
            segmentsMap[rep.ID] = append(segmentsMap[rep.ID], urls...)
         }
      }
   }

   out, err := json.MarshalIndent(segmentsMap, "", "  ")
   if err != nil {
      log.Fatalf("Failed to marshal JSON: %v", err)
   }
   fmt.Println(string(out))
}
