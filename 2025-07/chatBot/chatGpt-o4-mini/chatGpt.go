package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io"
   "net/url"
   "os"
   "regexp"
   "strconv"
)

const defaultBase = "http://test.test/test.mpd"

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL *string  `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   BaseURL        *string         `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         *string          `xml:"BaseURL"`
   ID              string           `xml:"id,attr"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   Bandwidth       int              `xml:"bandwidth,attr"`
   BaseURL         *string          `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentList struct {
   Initialization *Init        `xml:"Initialization"`
   SegmentURLs    []SegmentURL `xml:"SegmentURL"`
}

type Init struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

type SegmentTemplate struct {
   Media             string           `xml:"media,attr"`
   InitializationURL string           `xml:"initialization,attr"`
   StartNumber       int64            `xml:"startNumber,attr"`
   EndNumber         int64            `xml:"endNumber,attr"`
   Timescale         int64            `xml:"timescale,attr"`
   SegmentTimeline   *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T int64 `xml:"t,attr"` // start time
   D int64 `xml:"d,attr"` // duration
   R int64 `xml:"r,attr"` // repeat count
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }

   // Open MPD file
   f, err := os.Open(os.Args[1])
   if err != nil {
      panic(err)
   }
   defer f.Close()

   // Parse XML
   mpd := MPD{}
   dec := xml.NewDecoder(f)
   if err := dec.Decode(&mpd); err != nil && err != io.EOF {
      panic(err)
   }

   // Initialize base URL chain starting from default
   baseURL, err := url.Parse(defaultBase)
   if err != nil {
      panic(err)
   }
   // Chain MPD-level BaseURL
   if mpd.BaseURL != nil {
      rel, _ := url.Parse(*mpd.BaseURL)
      baseURL = baseURL.ResolveReference(rel)
   }

   re := regexp.MustCompile(`\$(RepresentationID|Number|Time)(%[^$]+)?\$`)
   results := make(map[string][]string)

   // Walk hierarchy
   for _, period := range mpd.Periods {
      // Chain Period-level BaseURL
      periodURL := baseURL
      if period.BaseURL != nil {
         rel, _ := url.Parse(*period.BaseURL)
         periodURL = periodURL.ResolveReference(rel)
      }

      for _, as := range period.AdaptationSets {
         // Chain AdaptationSet-level BaseURL
         asetURL := periodURL
         if as.BaseURL != nil {
            rel, _ := url.Parse(*as.BaseURL)
            asetURL = asetURL.ResolveReference(rel)
         }

         for _, rep := range as.Representations {
            // Chain Representation-level BaseURL
            repURL := asetURL
            if rep.BaseURL != nil {
               rel, _ := url.Parse(*rep.BaseURL)
               repURL = repURL.ResolveReference(rel)
            }

            // Inherit segment list/template
            sl := rep.SegmentList
            if sl == nil {
               sl = as.SegmentList
            }
            st := rep.SegmentTemplate
            if st == nil {
               st = as.SegmentTemplate
            }

            var segs []string

            // Initialization segments
            if sl != nil && sl.Initialization != nil {
               rel, _ := url.Parse(sl.Initialization.SourceURL)
               u := repURL.ResolveReference(rel)
               segs = append(segs, u.String())
            }
            if st != nil && st.InitializationURL != "" {
               rel, _ := url.Parse(st.InitializationURL)
               u := repURL.ResolveReference(rel)
               segs = append(segs, u.String())
            }

            // SegmentList URLs
            if sl != nil {
               for _, s := range sl.SegmentURLs {
                  rel, _ := url.Parse(s.Media)
                  u := repURL.ResolveReference(rel)
                  segs = append(segs, u.String())
               }
            }

            // SegmentTemplate URLs
            if st != nil {
               // timeline-based
               if st.SegmentTimeline != nil {
                  time := int64(0)
                  count := st.StartNumber
                  if count == 0 {
                     count = 1
                  }
                  for _, s := range st.SegmentTimeline.S {
                     if s.T > 0 {
                        time = s.T
                     }
                     r := s.R
                     if r < 0 {
                        r = 0
                     }
                     for i := int64(0); i <= r; i++ {
                        urlStr := applyTemplate(re, st.Media, rep.ID, count, time)
                        rel, _ := url.Parse(urlStr)
                        u := repURL.ResolveReference(rel)
                        segs = append(segs, u.String())
                        count++
                        time += s.D
                     }
                  }
               } else {
                  // numeric-based
                  start := st.StartNumber
                  if start == 0 {
                     start = 1
                  }
                  end := st.EndNumber
                  if end == 0 {
                     end = start
                  }
                  for n := start; n <= end; n++ {
                     urlStr := applyTemplate(re, st.Media, rep.ID, n, 0)
                     rel, _ := url.Parse(urlStr)
                     u := repURL.ResolveReference(rel)
                     segs = append(segs, u.String())
                  }
               }
            }

            // Fallback: if no segments, use repURL itself
            if len(segs) == 0 {
               segs = append(segs, repURL.String())
            }

            results[rep.ID] = segs
         }
      }
   }

   // Output JSON
   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   if err := enc.Encode(results); err != nil {
      panic(err)
   }
}

// applyTemplate replaces placeholders
func applyTemplate(re *regexp.Regexp, tmpl, repID string, number, tm int64) string {
   return re.ReplaceAllStringFunc(tmpl, func(m string) string {
      parts := re.FindStringSubmatch(m)
      key := parts[1]
      fmtStr := parts[2]
      if fmtStr == "" {
         switch key {
         case "RepresentationID":
            return repID
         case "Number":
            return strconv.FormatInt(number, 10)
         case "Time":
            return strconv.FormatInt(tm, 10)
         }
      }
      format := fmtStr[1:]
      switch key {
      case "RepresentationID":
         return fmt.Sprintf("%"+format, repID)
      case "Number":
         return fmt.Sprintf("%"+format, number)
      case "Time":
         return fmt.Sprintf("%"+format, tm)
      }
      return m
   })
}
