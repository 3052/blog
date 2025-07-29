package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "math"
   "net/url"
   "os"
   "path/filepath"
   "regexp"
   "strconv"
   "strings"
   "time"
)

var numberFormatRe = regexp.MustCompile(`\$Number(%[0-9]*d)?\$`)

func substitute(template, representationID string, number int, time int64) string {
   // Replace $RepresentationID$
   s := strings.ReplaceAll(template, "$RepresentationID$", representationID)

   // Replace formatted $Number...$ patterns
   s = numberFormatRe.ReplaceAllStringFunc(s, func(m string) string {
      // m example: "$Number%05d$" or "$Number$"
      if m == "$Number$" {
         return fmt.Sprintf("%d", number)
      }
      // Extract format string: between $Number and $
      format := m[len("$Number") : len(m)-1] // e.g. "%05d"
      return fmt.Sprintf(format, number)
   })

   // Replace $Time$ (no formatting supported)
   s = strings.ReplaceAll(s, "$Time$", strconv.FormatInt(time, 10))

   return s
}

const baseMPDURL = "http://test.test/test.mpd"

type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   BaseURL                   *string  `xml:"BaseURL"`
   Periods                   []Period `xml:"Period"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
}

type Period struct {
   BaseURL        *string         `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
   Duration       string          `xml:"duration,attr"`
}

type AdaptationSet struct {
   ID              string           `xml:"id,attr"`
   BaseURL         *string          `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Initialization  *Initialization  `xml:"Initialization"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         *string          `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Initialization  *Initialization  `xml:"Initialization"`
}

type SegmentTemplate struct {
   Timescale       *int             `xml:"timescale,attr"`
   Duration        *int             `xml:"duration,attr"`
   StartNumber     *int             `xml:"startNumber,attr"`
   EndNumber       *int             `xml:"endNumber,attr"`
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   Segments []S `xml:"S"`
}

type S struct {
   T int64 `xml:"t,attr"`
   D int64 `xml:"d,attr"`
   R int   `xml:"r,attr"`
}

type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

func parseDuration(dur string) time.Duration {
   d, err := time.ParseDuration(strings.Replace(strings.TrimPrefix(dur, "PT"), "S", "s", 1))
   if err != nil {
      return 0
   }
   return d
}

// resolveURL uses net/url.URL.ResolveReference exclusively
func resolveURL(baseStr string, refStr string) string {
   base, err := url.Parse(baseStr)
   if err != nil {
      // fallback to ref if base is invalid
      return refStr
   }
   ref, err := url.Parse(refStr)
   if err != nil {
      // fallback to ref if invalid
      return refStr
   }
   return base.ResolveReference(ref).String()
}

func inheritTemplate(parent, child *SegmentTemplate) *SegmentTemplate {
   if parent == nil && child == nil {
      return nil
   }
   if parent == nil {
      p := *child
      return &p
   }
   if child == nil {
      p := *parent
      return &p
   }
   merged := *parent
   if child.Timescale != nil {
      merged.Timescale = child.Timescale
   }
   if child.Duration != nil {
      merged.Duration = child.Duration
   }
   if child.StartNumber != nil {
      merged.StartNumber = child.StartNumber
   }
   if child.EndNumber != nil {
      merged.EndNumber = child.EndNumber
   }
   if child.Media != "" {
      merged.Media = child.Media
   }
   if child.Initialization != "" {
      merged.Initialization = child.Initialization
   }
   if child.SegmentTimeline != nil {
      merged.SegmentTimeline = child.SegmentTimeline
   }
   return &merged
}

func buildSegments(mpd *MPD) map[string][]string {
   result := make(map[string][]string)

   mpdBase := baseMPDURL
   if mpd.BaseURL != nil {
      mpdBase = resolveURL(baseMPDURL, *mpd.BaseURL)
   }

   for _, period := range mpd.Periods {
      periodBase := mpdBase
      if period.BaseURL != nil {
         periodBase = resolveURL(periodBase, *period.BaseURL)
      }

      periodDuration := 0.0
      if period.Duration != "" {
         periodDuration = parseDuration(period.Duration).Seconds()
      } else if mpd.MediaPresentationDuration != "" {
         periodDuration = parseDuration(mpd.MediaPresentationDuration).Seconds()
      }

      for _, aset := range period.AdaptationSets {
         asetBase := periodBase
         if aset.BaseURL != nil {
            asetBase = resolveURL(asetBase, *aset.BaseURL)
         }

         for _, rep := range aset.Representations {
            repBase := asetBase
            if rep.BaseURL != nil {
               repBase = resolveURL(repBase, *rep.BaseURL)
            }

            id := rep.ID
            if _, ok := result[id]; !ok {
               result[id] = []string{}
            }

            // Initialization segment
            initAdded := false
            if rep.Initialization != nil {
               result[id] = append(result[id], resolveURL(repBase, rep.Initialization.SourceURL))
               initAdded = true
            } else if rep.SegmentList != nil && rep.SegmentList.Initialization != nil {
               result[id] = append(result[id], resolveURL(repBase, rep.SegmentList.Initialization.SourceURL))
               initAdded = true
            } else if aset.Initialization != nil {
               result[id] = append(result[id], resolveURL(repBase, aset.Initialization.SourceURL))
               initAdded = true
            } else if aset.SegmentList != nil && aset.SegmentList.Initialization != nil {
               result[id] = append(result[id], resolveURL(repBase, aset.SegmentList.Initialization.SourceURL))
               initAdded = true
            } else {
               tmpl := inheritTemplate(aset.SegmentTemplate, rep.SegmentTemplate)
               if tmpl != nil && tmpl.Initialization != "" {
                  initURL := substitute(tmpl.Initialization, id, 0, 0)
                  result[id] = append(result[id], resolveURL(repBase, initURL))
                  initAdded = true
               }
            }

            // SegmentList
            list := rep.SegmentList
            if list == nil {
               list = aset.SegmentList
            }
            if list != nil {
               // If initialization not added and present here, add it first
               if !initAdded && list.Initialization != nil {
                  result[id] = append(result[id], resolveURL(repBase, list.Initialization.SourceURL))
                  initAdded = true
               }
               for _, seg := range list.SegmentURLs {
                  result[id] = append(result[id], resolveURL(repBase, seg.Media))
               }
               continue
            }

            // SegmentTemplate
            tmpl := inheritTemplate(aset.SegmentTemplate, rep.SegmentTemplate)
            if tmpl != nil && tmpl.Media != "" {
               timescale := 1
               if tmpl.Timescale != nil {
                  timescale = *tmpl.Timescale
               }
               start := 1
               if tmpl.StartNumber != nil {
                  start = *tmpl.StartNumber
               }
               end := -1
               if tmpl.EndNumber != nil {
                  end = *tmpl.EndNumber
               }

               if tmpl.SegmentTimeline != nil {
                  var timeAcc int64 = 0
                  number := start
                  for _, seg := range tmpl.SegmentTimeline.Segments {
                     repeat := seg.R
                     if repeat < 0 {
                        // Infinite repeat not supported; treat as 0
                        repeat = 0
                     }
                     repeat = repeat + 1 // 1 + r repeats per spec
                     if seg.T != 0 {
                        timeAcc = seg.T
                     }
                     for i := 0; i < repeat; i++ {
                        url := substitute(tmpl.Media, id, number, timeAcc)
                        result[id] = append(result[id], resolveURL(repBase, url))
                        timeAcc += seg.D
                        number++
                     }
                  }
               } else {
                  if end < 0 && tmpl.Duration != nil {
                     count := int(math.Ceil(periodDuration * float64(timescale) / float64(*tmpl.Duration)))
                     end = start + count - 1
                  }
                  for i := start; i <= end; i++ {
                     url := substitute(tmpl.Media, id, i, 0)
                     result[id] = append(result[id], resolveURL(repBase, url))
                  }
               }
               continue
            }

            // Fallback BaseURL segment if exists
            if rep.BaseURL != nil {
               // repBase is already resolved with rep.BaseURL
               result[id] = append(result[id], repBase)
            }
         }
      }
   }
   return result
}

func main() {
   if len(os.Args) != 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path>")
      os.Exit(1)
   }
   path := os.Args[1]
   data, err := ioutil.ReadFile(filepath.Clean(path))
   if err != nil {
      fmt.Println("Error reading MPD file:", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Println("Error parsing MPD XML:", err)
      os.Exit(1)
   }

   segments := buildSegments(&mpd)
   jsonData, _ := json.MarshalIndent(segments, "", "  ")
   fmt.Println(string(jsonData))
}
