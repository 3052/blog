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

const defaultBase = "http://test.test/test.mpd"

// MPD represents the top-level DASH manifest.
type MPD struct {
   XMLName  xml.Name `xml:"MPD"`
   BaseURLs []string `xml:"BaseURL"`
   Periods  []Period `xml:"Period"`
}

// Period represents a Period element in the MPD.
type Period struct {
   XMLName        xml.Name        `xml:"Period"`
   BaseURLs       []string        `xml:"BaseURL"`
   Duration       string          `xml:"duration,attr"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet represents an AdaptationSet element.
type AdaptationSet struct {
   XMLName         xml.Name         `xml:"AdaptationSet"`
   BaseURLs        []string         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

// Representation represents a Representation element.
type Representation struct {
   XMLName         xml.Name         `xml:"Representation"`
   ID              string           `xml:"id,attr"`
   BaseURLs        []string         `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

// SegmentList variant.
type SegmentList struct {
   XMLName        xml.Name       `xml:"SegmentList"`
   Initialization Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL   `xml:"SegmentURL"`
}

// Initialization holds the initialization segment reference.
type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

// SegmentURL holds each media segment reference.
type SegmentURL struct {
   Media string `xml:"media,attr"`
}

// SegmentTemplate variant.
type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   Timescale       int              `xml:"timescale,attr"`
   Duration        int              `xml:"duration,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline holds a series of <S> entries.
type SegmentTimeline struct {
   S []S `xml:"S"`
}

// S represents one <S> element.
type S struct {
   T *int64 `xml:"t,attr"`
   D int64  `xml:"d,attr"`
   R int    `xml:"r,attr"`
}

// parseDurationISO8601 parses a simple ISO8601 duration (H/M/S only).
func parseDurationISO8601(d string) (float64, error) {
   var h, m, s float64
   if m1 := regexp.MustCompile(`(\d+\.?\d*)H`).FindStringSubmatch(d); len(m1) == 2 {
      h, _ = strconv.ParseFloat(m1[1], 64)
   }
   if m2 := regexp.MustCompile(`(\d+\.?\d*)M`).FindStringSubmatch(d); len(m2) == 2 {
      m, _ = strconv.ParseFloat(m2[1], 64)
   }
   if m3 := regexp.MustCompile(`(\d+\.?\d*)S`).FindStringSubmatch(d); len(m3) == 2 {
      s, _ = strconv.ParseFloat(m3[1], 64)
   }
   return h*3600 + m*60 + s, nil
}

func main() {
   if len(os.Args) < 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }
   mpdPath := os.Args[1]

   data, err := os.ReadFile(mpdPath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading MPD file: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD XML: %v\n", err)
      os.Exit(1)
   }

   base, err := url.Parse(defaultBase)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Invalid defaultBase: %v\n", err)
      os.Exit(1)
   }
   // Chain MPD-level BaseURLs
   for _, b := range mpd.BaseURLs {
      u, err := url.Parse(b)
      if err != nil {
         fmt.Fprintf(os.Stderr, "Invalid MPD BaseURL %q: %v\n", b, err)
         continue
      }
      base = base.ResolveReference(u)
   }

   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      // Chain Period-level BaseURLs
      pbase := base
      for _, b := range period.BaseURLs {
         u, err := url.Parse(b)
         if err != nil {
            fmt.Fprintf(os.Stderr, "Invalid Period BaseURL %q: %v\n", b, err)
            continue
         }
         pbase = pbase.ResolveReference(u)
      }
      periodSec := 0.0
      if period.Duration != "" {
         periodSec, _ = parseDurationISO8601(period.Duration)
      }

      for _, aset := range period.AdaptationSets {
         // Chain AdaptationSet-level BaseURLs
         abase := pbase
         for _, b := range aset.BaseURLs {
            u, err := url.Parse(b)
            if err != nil {
               fmt.Fprintf(os.Stderr, "Invalid AdaptationSet BaseURL %q: %v\n", b, err)
               continue
            }
            abase = abase.ResolveReference(u)
         }
         adaptTmpl := aset.SegmentTemplate

         for _, rep := range aset.Representations {
            // Chain Representation-level BaseURLs
            rbase := abase
            for _, b := range rep.BaseURLs {
               u, err := url.Parse(b)
               if err != nil {
                  fmt.Fprintf(os.Stderr, "Invalid Representation BaseURL %q: %v\n", b, err)
                  continue
               }
               rbase = rbase.ResolveReference(u)
            }

            // Determine templates
            tmpl := adaptTmpl
            if rep.SegmentTemplate != nil {
               tmpl = rep.SegmentTemplate
            }

            // If no SegmentList and no template, return sole BaseURL
            if rep.SegmentList == nil && tmpl == nil {
               result[rep.ID] = []string{rbase.String()}
               continue
            }

            var segments []string

            // SegmentList handling
            if rep.SegmentList != nil {
               sl := rep.SegmentList
               if sl.Initialization.SourceURL != "" {
                  u, err := url.Parse(sl.Initialization.SourceURL)
                  if err == nil {
                     segments = append(segments, rbase.ResolveReference(u).String())
                  }
               }
               for _, s := range sl.SegmentURLs {
                  if s.Media == "" {
                     continue
                  }
                  u, err := url.Parse(s.Media)
                  if err != nil {
                     continue
                  }
                  segments = append(segments, rbase.ResolveReference(u).String())
               }
            } else if tmpl != nil {
               // SegmentTemplate handling
               if tmpl.Initialization != "" {
                  initURL := strings.ReplaceAll(tmpl.Initialization, "$RepresentationID$", rep.ID)
                  u, err := url.Parse(initURL)
                  if err == nil {
                     segments = append(segments, rbase.ResolveReference(u).String())
                  }
               }

               startNum := tmpl.StartNumber
               if startNum == 0 {
                  startNum = 1
               }
               timescale := tmpl.Timescale
               if timescale == 0 {
                  timescale = 1
               }
               mediaTpl := tmpl.Media

               // Timeline
               if tmpl.SegmentTimeline != nil && len(tmpl.SegmentTimeline.S) > 0 {
                  ct := int64(0)
                  seq := startNum
                  for _, e := range tmpl.SegmentTimeline.S {
                     if e.T != nil {
                        ct = *e.T
                     }
                     repCnt := e.R
                     if repCnt < 0 {
                        repCnt = 0
                     }
                     for i := 0; i <= repCnt; i++ {
                        uStr := strings.ReplaceAll(mediaTpl, "$RepresentationID$", rep.ID)
                        uStr = strings.ReplaceAll(uStr, "$Number$", strconv.Itoa(seq))
                        uStr = strings.ReplaceAll(uStr, "$Time$", strconv.FormatInt(ct, 10))
                        u2, err := url.Parse(uStr)
                        if err == nil {
                           segments = append(segments, rbase.ResolveReference(u2).String())
                        }
                        seq++
                        ct += e.D
                     }
                  }
               } else if tmpl.EndNumber > 0 {
                  for n := startNum; n <= tmpl.EndNumber; n++ {
                     uStr := strings.ReplaceAll(mediaTpl, "$RepresentationID$", rep.ID)
                     uStr = strings.ReplaceAll(uStr, "$Number$", strconv.Itoa(n))
                     u2, err := url.Parse(uStr)
                     if err == nil {
                        segments = append(segments, rbase.ResolveReference(u2).String())
                     }
                  }
               } else if tmpl.Duration > 0 && periodSec > 0 {
                  count := int(math.Ceil(periodSec * float64(timescale) / float64(tmpl.Duration)))
                  for i := 0; i < count; i++ {
                     num := startNum + i
                     uStr := strings.ReplaceAll(mediaTpl, "$RepresentationID$", rep.ID)
                     uStr = strings.ReplaceAll(uStr, "$Number$", strconv.Itoa(num))
                     u2, err := url.Parse(uStr)
                     if err == nil {
                        segments = append(segments, rbase.ResolveReference(u2).String())
                     }
                  }
               } else {
                  fmt.Fprintf(os.Stderr, "No segment info for representation %s\n", rep.ID)
               }
            }

            result[rep.ID] = segments
         }
      }
   }

   out, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
      os.Exit(1)
   }
   fmt.Println(string(out))
}
