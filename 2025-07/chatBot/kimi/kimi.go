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
   Type                      string   `xml:"type,attr"`
   Period                    []Period `xml:"Period"`
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
   Timescale       int              `xml:"timescale,attr"`
   Duration        int              `xml:"duration,attr"`
   EndNumber       *int             `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentList struct {
   Initialization  *Initialization  `xml:"Initialization"`
   SegmentURL      []SegmentURL     `xml:"SegmentURL"`
   Duration        int              `xml:"duration,attr"`
   Timescale       int              `xml:"timescale,attr"`
   StartNumber     *int             `xml:"startNumber,attr"`
   EndNumber       *int             `xml:"endNumber,attr"`
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

   base, err := url.Parse("http://test.test/test.mpd")
   if err != nil {
      panic(err)
   }

   out := make(map[string][]string)

   for _, period := range mpd.Period {
      periodBase := base
      if period.BaseURL != "" {
         u, err := url.Parse(period.BaseURL)
         if err != nil {
            panic(err)
         }
         periodBase = base.ResolveReference(u)
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

            repID := rep.ID
            var init string
            var segments []string

            st := rep.SegmentTemplate
            if st == nil {
               st = as.SegmentTemplate
            }
            sl := rep.SegmentList
            if sl == nil {
               sl = as.SegmentList
            }

            switch {
            case st != nil:
               if st.Initialization != "" {
                  initURL := expandTemplate(st.Initialization, repID, 0, 0)
                  u, err := url.Parse(initURL)
                  if err != nil {
                     panic(err)
                  }
                  init = repBase.ResolveReference(u).String()
               }

               var startNum int
               if st.StartNumber != nil {
                  startNum = *st.StartNumber
               } else {
                  startNum = 1
               }

               mediaTpl := st.Media

               if st.SegmentTimeline != nil {
                  time := int64(0)
                  for _, s := range st.SegmentTimeline.S {
                     if s.T != 0 {
                        time = int64(s.T)
                     }
                     count := 1 + s.R
                     for i := 0; i < count; i++ {
                        segURL := expandTemplate(mediaTpl, repID, startNum, time)
                        u, err := url.Parse(segURL)
                        if err != nil {
                           panic(err)
                        }
                        segments = append(segments, repBase.ResolveReference(u).String())
                        startNum++
                        time += int64(s.D)
                     }
                  }
               } else if st.EndNumber != nil {
                  for n := startNum; n <= *st.EndNumber; n++ {
                     segURL := expandTemplate(mediaTpl, repID, n, 0)
                     u, err := url.Parse(segURL)
                     if err != nil {
                        panic(err)
                     }
                     segments = append(segments, repBase.ResolveReference(u).String())
                  }
               } else {
                  duration := parseDuration(period.Duration)
                  if duration == 0 {
                     duration = parseDuration(mpd.MediaPresentationDuration)
                  }
                  if st.Timescale == 0 {
                     st.Timescale = 1
                  }
                  count := int(math.Ceil(duration.Seconds() * float64(st.Timescale) / float64(st.Duration)))
                  for n := 0; n < count; n++ {
                     segURL := expandTemplate(mediaTpl, repID, startNum+n, 0)
                     u, err := url.Parse(segURL)
                     if err != nil {
                        panic(err)
                     }
                     segments = append(segments, repBase.ResolveReference(u).String())
                  }
               }

            case sl != nil:
               if sl.Initialization != nil && sl.Initialization.SourceURL != "" {
                  u, err := url.Parse(sl.Initialization.SourceURL)
                  if err != nil {
                     panic(err)
                  }
                  init = repBase.ResolveReference(u).String()
               }

               var startNum int
               if sl.StartNumber != nil {
                  startNum = *sl.StartNumber
               } else {
                  startNum = 1
               }

               if sl.SegmentTimeline != nil {
                  time := int64(0)
                  for _, s := range sl.SegmentTimeline.S {
                     if s.T != 0 {
                        time = int64(s.T)
                     }
                     count := 1 + s.R
                     for i := 0; i < count; i++ {
                        u, err := url.Parse(sl.SegmentURL[startNum-1].Media)
                        if err != nil {
                           panic(err)
                        }
                        segments = append(segments, repBase.ResolveReference(u).String())
                        startNum++
                        time += int64(s.D)
                     }
                  }
               } else if sl.EndNumber != nil {
                  for n := startNum; n <= *sl.EndNumber; n++ {
                     u, err := url.Parse(sl.SegmentURL[n-1].Media)
                     if err != nil {
                        panic(err)
                     }
                     segments = append(segments, repBase.ResolveReference(u).String())
                  }
               } else {
                  duration := parseDuration(period.Duration)
                  if duration == 0 {
                     duration = parseDuration(mpd.MediaPresentationDuration)
                  }
                  if sl.Timescale == 0 {
                     sl.Timescale = 1
                  }
                  count := int(math.Ceil(duration.Seconds() * float64(sl.Timescale) / float64(sl.Duration)))
                  for n := 0; n < count && n < len(sl.SegmentURL); n++ {
                     u, err := url.Parse(sl.SegmentURL[n].Media)
                     if err != nil {
                        panic(err)
                     }
                     segments = append(segments, repBase.ResolveReference(u).String())
                  }
               }

            default:
               segments = append(segments, repBase.String())
            }

            if init != "" {
               out[repID] = append(out[repID], init)
            }
            out[repID] = append(out[repID], segments...)
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetEscapeHTML(false)
   if err := enc.Encode(out); err != nil {
      panic(err)
   }
}

func expandTemplate(tpl string, repID string, number int, time int64) string {
   tpl = strings.ReplaceAll(tpl, "$RepresentationID$", repID)
   tpl = strings.ReplaceAll(tpl, "$Number$", strconv.Itoa(number))
   tpl = strings.ReplaceAll(tpl, "$Time$", strconv.FormatInt(time, 10))
   tpl = strings.ReplaceAll(tpl, "$Number%01d$", fmt.Sprintf("%01d", number))
   tpl = strings.ReplaceAll(tpl, "$Number%02d$", fmt.Sprintf("%02d", number))
   tpl = strings.ReplaceAll(tpl, "$Number%03d$", fmt.Sprintf("%03d", number))
   tpl = strings.ReplaceAll(tpl, "$Number%04d$", fmt.Sprintf("%04d", number))
   tpl = strings.ReplaceAll(tpl, "$Number%05d$", fmt.Sprintf("%05d", number))
   tpl = strings.ReplaceAll(tpl, "$Number%06d$", fmt.Sprintf("%06d", number))
   tpl = strings.ReplaceAll(tpl, "$Number%07d$", fmt.Sprintf("%07d", number))
   tpl = strings.ReplaceAll(tpl, "$Number%08d$", fmt.Sprintf("%08d", number))
   tpl = strings.ReplaceAll(tpl, "$Number%09d$", fmt.Sprintf("%09d", number))
   return tpl
}

func parseDuration(d string) time.Duration {
   if d == "" {
      return 0
   }
   d = strings.TrimPrefix(d, "P")
   d = strings.TrimPrefix(d, "T")
   d = strings.ReplaceAll(d, "H", "h")
   d = strings.ReplaceAll(d, "M", "m")
   d = strings.ReplaceAll(d, "S", "s")
   dur, err := time.ParseDuration(d)
   if err != nil {
      panic(err)
   }
   return dur
}
