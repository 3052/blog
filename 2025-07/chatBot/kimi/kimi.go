package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "math"
   "net/url"
   "os"
   "strconv"
   "strings"
   "time"
)

type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
   BaseURL                   string   `xml:"BaseURL"`
   Periods                   []Period `xml:"Period"`
}

type Period struct {
   BaseURL       string          `xml:"BaseURL"`
   Duration      string          `xml:"duration,attr"`
   AdaptationSet []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Representation  []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   StartNumber     *int             `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   Duration        int              `xml:"duration,attr"`
   Timescale       int              `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentList struct {
   Initialization  *Initialization  `xml:"Initialization"`
   SegmentURL      []SegmentURL     `xml:"SegmentURL"`
   Duration        int              `xml:"duration,attr"`
   Timescale       int              `xml:"timescale,attr"`
   StartNumber     *int             `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T int `xml:"t,attr"`
   D int `xml:"d,attr"`
   R int `xml:"r,attr"`
}

func main() {
   if len(os.Args) < 2 {
      fmt.Fprintf(os.Stderr, "usage: %s <mpd-file>\n", os.Args[0])
      os.Exit(1)
   }
   mpdPath := os.Args[1]

   data, err := os.ReadFile(mpdPath)
   if err != nil {
      panic(err)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      panic(err)
   }

   fixedBase := "http://test.test/test.mpd"
   baseURL, err := url.Parse(fixedBase)
   if err != nil {
      panic(err)
   }

   output := make(map[string][]string)

   for _, period := range mpd.Periods {
      periodBase := baseURL
      if period.BaseURL != "" {
         u, err := url.Parse(period.BaseURL)
         if err != nil {
            panic(err)
         }
         periodBase = baseURL.ResolveReference(u)
      }

      for _, as := range period.AdaptationSet {
         asBase := periodBase
         if as.BaseURL != "" {
            u, err := url.Parse(as.BaseURL)
            if err != nil {
               panic(err)
            }
            asBase = periodBase.ResolveReference(u)
         }

         for _, rep := range as.Representation {
            repBase := asBase
            if rep.BaseURL != "" {
               u, err := url.Parse(rep.BaseURL)
               if err != nil {
                  panic(err)
               }
               repBase = asBase.ResolveReference(u)
            }

            var template *SegmentTemplate
            var list *SegmentList

            if rep.SegmentTemplate != nil {
               template = rep.SegmentTemplate
            } else if as.SegmentTemplate != nil {
               template = as.SegmentTemplate
            } else if rep.SegmentList != nil {
               list = rep.SegmentList
            } else if as.SegmentList != nil {
               list = as.SegmentList
            }

            var segments []string

            if template != nil {
               if template.Initialization != "" {
                  init := expandTemplate(template.Initialization, rep.ID, 1, 0)
                  initURL, err := url.Parse(init)
                  if err != nil {
                     panic(err)
                  }
                  initAbs := repBase.ResolveReference(initURL)
                  segments = append(segments, initAbs.String())
               }

               start := 1
               if template.StartNumber != nil {
                  start = *template.StartNumber
               }

               if template.EndNumber > 0 {
                  for i := start; i <= template.EndNumber; i++ {
                     media := expandTemplate(template.Media, rep.ID, i, 0)
                     mediaURL, err := url.Parse(media)
                     if err != nil {
                        panic(err)
                     }
                     abs := repBase.ResolveReference(mediaURL)
                     segments = append(segments, abs.String())
                  }
               } else if template.SegmentTimeline != nil {
                  seq := start
                  time := 0
                  for _, s := range template.SegmentTimeline.S {
                     if s.T > 0 {
                        time = s.T
                     }
                     for j := 0; j <= s.R; j++ {
                        media := expandTemplate(template.Media, rep.ID, seq, time)
                        mediaURL, err := url.Parse(media)
                        if err != nil {
                           panic(err)
                        }
                        abs := repBase.ResolveReference(mediaURL)
                        segments = append(segments, abs.String())
                        time += s.D
                        seq++
                     }
                  }
               } else {
                  duration := template.Duration
                  timescale := template.Timescale
                  if timescale == 0 {
                     timescale = 1
                  }

                  totalDur := getDuration(mpd.MediaPresentationDuration, period.Duration)
                  segDur := time.Duration(duration) * time.Second / time.Duration(timescale)
                  count := int(math.Ceil(float64(totalDur) / float64(segDur)))

                  for i := 0; i < count; i++ {
                     n := start + i
                     media := expandTemplate(template.Media, rep.ID, n, 0)
                     mediaURL, err := url.Parse(media)
                     if err != nil {
                        panic(err)
                     }
                     abs := repBase.ResolveReference(mediaURL)
                     segments = append(segments, abs.String())
                  }
               }
            } else if list != nil {
               if list.Initialization != nil && list.Initialization.SourceURL != "" {
                  initURL, err := url.Parse(list.Initialization.SourceURL)
                  if err != nil {
                     panic(err)
                  }
                  initAbs := repBase.ResolveReference(initURL)
                  segments = append(segments, initAbs.String())
               }

               start := 1
               if list.StartNumber != nil {
                  start = *list.StartNumber
               }

               if len(list.SegmentURL) > 0 {
                  for _, su := range list.SegmentURL {
                     u, err := url.Parse(su.Media)
                     if err != nil {
                        panic(err)
                     }
                     abs := repBase.ResolveReference(u)
                     segments = append(segments, abs.String())
                  }
               } else if list.SegmentTimeline != nil {
                  seq := start
                  time := 0
                  for _, s := range list.SegmentTimeline.S {
                     if s.T > 0 {
                        time = s.T
                     }
                     for j := 0; j <= s.R; j++ {
                        u, err := url.Parse(fmt.Sprintf("%d", seq))
                        if err != nil {
                           panic(err)
                        }
                        abs := repBase.ResolveReference(u)
                        segments = append(segments, abs.String())
                        time += s.D
                        seq++
                     }
                  }
               } else if list.EndNumber > 0 {
                  for i := start; i <= list.EndNumber; i++ {
                     u, err := url.Parse(fmt.Sprintf("%d", i))
                     if err != nil {
                        panic(err)
                     }
                     abs := repBase.ResolveReference(u)
                     segments = append(segments, abs.String())
                  }
               } else {
                  duration := list.Duration
                  timescale := list.Timescale
                  if timescale == 0 {
                     timescale = 1
                  }

                  totalDur := getDuration(mpd.MediaPresentationDuration, period.Duration)
                  segDur := time.Duration(duration) * time.Second / time.Duration(timescale)
                  count := int(math.Ceil(float64(totalDur) / float64(segDur)))

                  for i := 0; i < count; i++ {
                     u, err := url.Parse(fmt.Sprintf("%d", start+i))
                     if err != nil {
                        panic(err)
                     }
                     abs := repBase.ResolveReference(u)
                     segments = append(segments, abs.String())
                  }
               }
            } else {
               segments = append(segments, repBase.String())
            }

            output[rep.ID] = append(output[rep.ID], segments...)
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetEscapeHTML(false)
   if err := enc.Encode(output); err != nil {
      panic(err)
   }
}

func expandTemplate(tpl, repID string, number int, time int) string {
   s := tpl
   s = strings.ReplaceAll(s, "$RepresentationID$", repID)
   s = strings.ReplaceAll(s, "$Time$", strconv.Itoa(time))

   for d := 1; d <= 9; d++ {
      pat := fmt.Sprintf("$Number%%0%dd$", d)
      s = strings.ReplaceAll(s, pat, fmt.Sprintf("%0*d", d, number))
   }
   s = strings.ReplaceAll(s, "$Number$", strconv.Itoa(number))
   return s
}

func getDuration(mpdDur, periodDur string) time.Duration {
   durStr := periodDur
   if durStr == "" {
      durStr = mpdDur
   }
   dur, err := parseISO8601Duration(durStr)
   if err != nil {
      panic(err)
   }
   return dur
}

func parseISO8601Duration(s string) (time.Duration, error) {
   s = strings.TrimPrefix(s, "P")
   var d time.Duration
   if strings.Contains(s, "T") {
      parts := strings.Split(s, "T")
      if len(parts) != 2 {
         return 0, fmt.Errorf("invalid duration")
      }
      if v, err := strconv.Atoi(strings.TrimSuffix(parts[0], "D")); err == nil {
         d += time.Duration(v) * 24 * time.Hour
      }
      t := parts[1]
      if v, err := strconv.Atoi(strings.TrimSuffix(t, "H")); err == nil {
         d += time.Duration(v) * time.Hour
         return d, nil
      }
      if v, err := strconv.Atoi(strings.TrimSuffix(t, "M")); err == nil {
         d += time.Duration(v) * time.Minute
         return d, nil
      }
      if v, err := strconv.ParseFloat(strings.TrimSuffix(t, "S"), 64); err == nil {
         d += time.Duration(v * float64(time.Second))
         return d, nil
      }
   } else {
      if v, err := strconv.Atoi(strings.TrimSuffix(s, "D")); err == nil {
         d += time.Duration(v) * 24 * time.Hour
         return d, nil
      }
   }
   return 0, fmt.Errorf("unsupported duration format")
}
