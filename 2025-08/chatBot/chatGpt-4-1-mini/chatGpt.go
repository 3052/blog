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
   BaseURL *BaseURL `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type BaseURL struct {
   URI string `xml:",chardata"`
}

type Period struct {
   BaseURL         *BaseURL `xml:"BaseURL"`
   Duration        string   `xml:"duration,attr"`
   durationSeconds float64
   AdaptationSets  []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         *BaseURL         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         *BaseURL         `xml:"BaseURL"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentBase struct {
   Initialization *Initialization `xml:"Initialization"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   Timescale       int              `xml:"timescale,attr"`
   Duration        int              `xml:"duration,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T *int64 `xml:"t,attr,omitempty"`
   D int64  `xml:"d,attr"`
   R *int64 `xml:"r,attr,omitempty"`
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: go run main.go <mpd_file_path>\n")
      os.Exit(1)
   }
   mpdFile := os.Args[1]

   data, err := os.ReadFile(mpdFile)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading MPD file: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD XML: %v\n", err)
      os.Exit(1)
   }

   if len(mpd.Periods) == 0 {
      fmt.Fprintf(os.Stderr, "No Period found in MPD\n")
      os.Exit(1)
   }

   // Parse Period durations to seconds
   for i := range mpd.Periods {
      seconds, err := parseISODuration(mpd.Periods[i].Duration)
      if err != nil {
         fmt.Fprintf(os.Stderr, "Warning: could not parse Period duration '%s': %v\n", mpd.Periods[i].Duration, err)
         mpd.Periods[i].durationSeconds = 0
      } else {
         mpd.Periods[i].durationSeconds = seconds
      }
   }

   baseMPDURL, err := url.Parse("http://test.test/test.mpd")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing base MPD URL: %v\n", err)
      os.Exit(1)
   }

   result := make(map[string][]string)

   for pi, period := range mpd.Periods {
      periodBaseURL := resolveBaseURL(baseMPDURL, mpd.BaseURL, period.BaseURL)

      for _, aset := range period.AdaptationSets {
         adaptBaseURL := resolveBaseURL(periodBaseURL, aset.BaseURL)

         for _, rep := range aset.Representations {
            repBaseURL := resolveBaseURL(adaptBaseURL, rep.BaseURL)

            segments := []string{}

            segmentTemplate := rep.SegmentTemplate
            if segmentTemplate == nil {
               segmentTemplate = aset.SegmentTemplate
            }

            // Representation BaseURL-only case (no SegmentList/SegmentTemplate)
            if rep.SegmentList == nil && segmentTemplate == nil && rep.BaseURL != nil {
               segments = append(segments, repBaseURL.String())
               result[rep.ID] = append(result[rep.ID], segments...)
               continue
            }

            // Initialization segment
            initURL := ""
            if rep.SegmentBase != nil && rep.SegmentBase.Initialization != nil {
               initURL = resolveURL(repBaseURL, rep.SegmentBase.Initialization.SourceURL)
            } else if rep.SegmentList != nil && rep.SegmentList.Initialization != nil {
               initURL = resolveURL(repBaseURL, rep.SegmentList.Initialization.SourceURL)
            } else if segmentTemplate != nil && segmentTemplate.Initialization != "" {
               initURL = resolveURL(repBaseURL, segmentTemplate.Initialization)
            }
            if initURL != "" {
               segments = append(segments, initURL)
            }

            // Media segments
            if rep.SegmentList != nil {
               for _, segURL := range rep.SegmentList.SegmentURLs {
                  u := resolveURL(repBaseURL, segURL.Media)
                  segments = append(segments, u)
               }
            } else if segmentTemplate != nil && segmentTemplate.Media != "" {
               startNum := segmentTemplate.StartNumber
               if startNum == 0 {
                  startNum = 1
               }
               timescale := int64(1)
               if segmentTemplate.Timescale > 0 {
                  timescale = int64(segmentTemplate.Timescale)
               }

               if segmentTemplate.SegmentTimeline != nil && len(segmentTemplate.SegmentTimeline.S) > 0 {
                  segments = append(segments, generateSegmentTimelineURLs(
                     segmentTemplate.Media, repBaseURL, segmentTemplate.SegmentTimeline,
                     startNum, segmentTemplate.EndNumber, rep.ID)...)
               } else {
                  var segmentCount int
                  if segmentTemplate.EndNumber > 0 {
                     segmentCount = segmentTemplate.EndNumber - startNum + 1
                     if segmentCount < 0 {
                        segmentCount = 0
                     }
                  } else if segmentTemplate.Duration > 0 && mpd.Periods[pi].durationSeconds > 0 {
                     count := mpd.Periods[pi].durationSeconds * float64(timescale) / float64(segmentTemplate.Duration)
                     segmentCount = int(math.Ceil(count))
                  } else {
                     segmentCount = 5
                  }

                  for i := 0; i < segmentCount; i++ {
                     num := startNum + i
                     segmentPath := replaceNumberWithFormat(segmentTemplate.Media, num)
                     segmentPath = strings.ReplaceAll(segmentPath, "$RepresentationID$", rep.ID)
                     u := resolveURL(repBaseURL, segmentPath)
                     segments = append(segments, u)
                  }
               }
            }

            result[rep.ID] = append(result[rep.ID], segments...)
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   if err := enc.Encode(result); err != nil {
      fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
      os.Exit(1)
   }
}

// generateSegmentTimelineURLs generates segment URLs honoring EndNumber limit
func generateSegmentTimelineURLs(mediaTmpl string, base *url.URL, timeline *SegmentTimeline, startNumber int, endNumber int, repID string) []string {
   var segments []string
   segmentNum := startNumber
   var t int64

   for i, s := range timeline.S {
      repeat := int64(0)
      if s.R != nil {
         repeat = *s.R
      }

      if s.T != nil {
         t = *s.T
      } else if i == 0 {
         t = 0
      }

      for r := int64(0); r <= repeat; r++ {
         if endNumber > 0 && segmentNum > endNumber {
            return segments
         }

         segmentPath := replaceNumberWithFormat(mediaTmpl, segmentNum)
         segmentPath = strings.ReplaceAll(segmentPath, "$Time$", fmt.Sprintf("%d", t))
         segmentPath = strings.ReplaceAll(segmentPath, "$RepresentationID$", repID)

         u := resolveURL(base, segmentPath)
         segments = append(segments, u)

         segmentNum++
         t += s.D
      }
   }

   return segments
}

// replaceNumberWithFormat replaces $Number or $Number%0Nd$ with formatted number.
func replaceNumberWithFormat(s string, num int) string {
   re := regexp.MustCompile(`\$Number(%0(\d+)d)?\$`)
   return re.ReplaceAllStringFunc(s, func(m string) string {
      matches := re.FindStringSubmatch(m)
      if matches[1] != "" {
         width, _ := strconv.Atoi(matches[2])
         return fmt.Sprintf("%0*d", width, num)
      }
      return fmt.Sprintf("%d", num)
   })
}

func resolveBaseURL(base *url.URL, baseURLs ...*BaseURL) *url.URL {
   current := base
   for _, b := range baseURLs {
      if b != nil {
         u, err := url.Parse(strings.TrimSpace(b.URI))
         if err == nil {
            current = current.ResolveReference(u)
         }
      }
   }
   return current
}

func resolveURL(base *url.URL, ref string) string {
   ref = strings.TrimSpace(ref)
   if ref == "" {
      return ""
   }
   u, err := url.Parse(ref)
   if err != nil {
      return base.String()
   }
   return base.ResolveReference(u).String()
}

// parseISODuration parses simple ISO 8601 durations of form PT#H#M#S to seconds.
func parseISODuration(dur string) (float64, error) {
   re := regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?$`)
   matches := re.FindStringSubmatch(dur)
   if matches == nil {
      return 0, fmt.Errorf("invalid ISO8601 duration: %s", dur)
   }

   var h, m, s float64
   var err error

   if matches[1] != "" {
      h, err = strconv.ParseFloat(matches[1], 64)
      if err != nil {
         return 0, err
      }
   }
   if matches[2] != "" {
      m, err = strconv.ParseFloat(matches[2], 64)
      if err != nil {
         return 0, err
      }
   }
   if matches[3] != "" {
      s, err = strconv.ParseFloat(matches[3], 64)
      if err != nil {
         return 0, err
      }
   }

   return h*3600 + m*60 + s, nil
}
