package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "math"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
)

// MPD root
// Represents the top-level MPD element
// with an optional BaseURL and multiple Periods
type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

// Period element
// Inherits BaseURL, has Duration attribute, and contains AdaptationSets
type Period struct {
   BaseURL        string          `xml:"BaseURL"`
   Duration       string          `xml:"duration,attr"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet element
// May define its own BaseURL, a SegmentList or SegmentTemplate,
// and multiple Representations
type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

// Representation element
// Individual bitrate/audio/video stream
// May also define BaseURL, SegmentList, or SegmentTemplate
type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

// SegmentList with Initialization and multiple SegmentURL
type SegmentList struct {
   Initialization Initialization `xml:"Initialization"`
   Segments       []SegmentURL   `xml:"SegmentURL"`
}

// Initialization helper for SegmentList or Template
type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

// Single segment URL inside SegmentList
type SegmentURL struct {
   Media string `xml:"media,attr"`
}

// SegmentTemplate supports macros and optional SegmentTimeline
type SegmentTemplate struct {
   InitializationURL string           `xml:"initialization,attr"`
   Initialization    *Initialization  `xml:"Initialization"`
   Media             string           `xml:"media,attr"`
   StartNumber       int              `xml:"startNumber,attr"`
   EndNumber         int              `xml:"endNumber,attr"`
   Timescale         int              `xml:"timescale,attr"`
   Duration          int              `xml:"duration,attr"`
   SegmentTimeline   *SegmentTimeline `xml:"SegmentTimeline"`
}

// Timeline of segments inside a template
type SegmentTimeline struct {
   S []S `xml:"S"`
}

// Individual timeline entry
type S struct {
   T int64 `xml:"t,attr"`
   D int64 `xml:"d,attr"`
   R int   `xml:"r,attr,omitempty"`
}

// parseDuration parses simple XS duration of form PT#S into seconds
func parseDuration(s string) float64 {
   s = strings.TrimPrefix(s, "PT")
   if strings.HasSuffix(s, "S") {
      val := strings.TrimSuffix(s, "S")
      if f, err := strconv.ParseFloat(val, 64); err == nil {
         return f
      }
   }
   return 0
}

// pick first non-empty BaseURL string
func pickBaseURL(candidates ...string) string {
   for _, u := range candidates {
      if s := strings.TrimSpace(u); s != "" {
         return s
      }
   }
   return ""
}

// resolve relative paths against a base URL
func resolveURLs(base *url.URL, elems ...string) *url.URL {
   curr := base
   for _, e := range elems {
      if strings.TrimSpace(e) == "" {
         continue
      }
      u, err := url.Parse(e)
      if err != nil {
         continue
      }
      curr = curr.ResolveReference(u)
   }
   return curr
}

// buildTimePoints converts a SegmentTimeline into a list of time offsets
func buildTimePoints(tl *SegmentTimeline) []int64 {
   var times []int64
   var lastT int64
   for i, entry := range tl.S {
      if entry.T != 0 || i == 0 {
         lastT = entry.T
      }
      count := entry.R + 1
      for j := 0; j < count; j++ {
         times = append(times, lastT+int64(j)*entry.D)
      }
      lastT += int64(count) * entry.D
   }
   return times
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "usage: go run main.go <mpd_file_path>\n")
      os.Exit(1)
   }

   data, err := ioutil.ReadFile(os.Args[1])
   if err != nil {
      fmt.Fprintf(os.Stderr, "error reading MPD file: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Fprintf(os.Stderr, "error parsing MPD XML: %v\n", err)
      os.Exit(1)
   }

   rootBase, _ := url.Parse("http://test.test/test.mpd")
   out := make(map[string][]string)

   // regex for printf-style replacement
   numRe := regexp.MustCompile(`\$Number%0(\d+)d\$`)

   for _, period := range mpd.Periods {
      periodSecs := parseDuration(period.Duration)
      periodBase := resolveURLs(rootBase, pickBaseURL(mpd.BaseURL, period.BaseURL))

      for _, aset := range period.AdaptationSets {
         asBase := resolveURLs(periodBase, aset.BaseURL)

         for _, rep := range aset.Representations {
            repBase := resolveURLs(asBase, rep.BaseURL)
            var segs []string

            if rep.SegmentList == nil && aset.SegmentList == nil && rep.SegmentTemplate == nil && aset.SegmentTemplate == nil {
               segs = append(segs, repBase.String())
            } else {
               tpl := rep.SegmentTemplate
               if tpl == nil {
                  tpl = aset.SegmentTemplate
               }
               if tpl != nil && tpl.Timescale == 0 {
                  tpl.Timescale = 1
               }
               if rep.SegmentList != nil {
                  if init := rep.SegmentList.Initialization.SourceURL; init != "" {
                     segs = append(segs, resolveURLs(repBase, init).String())
                  }
                  for _, s := range rep.SegmentList.Segments {
                     segs = append(segs, resolveURLs(repBase, s.Media).String())
                  }
               } else if aset.SegmentList != nil {
                  if init := aset.SegmentList.Initialization.SourceURL; init != "" {
                     segs = append(segs, resolveURLs(asBase, init).String())
                  }
                  for _, s := range aset.SegmentList.Segments {
                     segs = append(segs, resolveURLs(asBase, s.Media).String())
                  }
               } else if tpl != nil {
                  if tpl.InitializationURL != "" {
                     segs = append(segs, resolveURLs(repBase, tpl.InitializationURL).String())
                  } else if tpl.Initialization != nil && tpl.Initialization.SourceURL != "" {
                     segs = append(segs, resolveURLs(repBase, tpl.Initialization.SourceURL).String())
                  }
                  if tpl.SegmentTimeline != nil {
                     times := buildTimePoints(tpl.SegmentTimeline)
                     for idx, t := range times {
                        media := tpl.Media
                        media = strings.ReplaceAll(media, "$Time$", strconv.FormatInt(t, 10))
                        // simple $Number$
                        media = strings.ReplaceAll(media, "$Number$", strconv.Itoa(tpl.StartNumber+idx))
                        media = strings.ReplaceAll(media, "$RepresentationID$", rep.ID)
                        segs = append(segs, resolveURLs(repBase, media).String())
                     }
                  } else {
                     start := tpl.StartNumber
                     if start == 0 {
                        start = 1
                     }
                     if tpl.EndNumber > 0 {
                        for n := start; n <= tpl.EndNumber; n++ {
                           media := tpl.Media
                           // printf-style width
                           if numRe.MatchString(media) {
                              media = numRe.ReplaceAllStringFunc(media, func(m string) string {
                                 width := numRe.FindStringSubmatch(m)[1]
                                 format := "%0" + width + "d"
                                 return fmt.Sprintf(format, n)
                              })
                           } else {
                              media = strings.ReplaceAll(media, "$Number$", strconv.Itoa(n))
                           }
                           media = strings.ReplaceAll(media, "$RepresentationID$", rep.ID)
                           segs = append(segs, resolveURLs(repBase, media).String())
                        }
                     } else if tpl.Duration > 0 && periodSecs > 0 {
                        count := int(math.Ceil(periodSecs * float64(tpl.Timescale) / float64(tpl.Duration)))
                        for i := 0; i < count; i++ {
                           num := start + i
                           media := tpl.Media
                           if numRe.MatchString(media) {
                              media = numRe.ReplaceAllStringFunc(media, func(m string) string {
                                 width := numRe.FindStringSubmatch(m)[1]
                                 format := "%0" + width + "d"
                                 return fmt.Sprintf(format, num)
                              })
                           } else {
                              media = strings.ReplaceAll(media, "$Number$", strconv.Itoa(num))
                           }
                           media = strings.ReplaceAll(media, "$RepresentationID$", rep.ID)
                           segs = append(segs, resolveURLs(repBase, media).String())
                        }
                     }
                  }
               }
            }
            out[rep.ID] = append(out[rep.ID], segs...)
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   if err := enc.Encode(out); err != nil {
      fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
      os.Exit(1)
   }
}
