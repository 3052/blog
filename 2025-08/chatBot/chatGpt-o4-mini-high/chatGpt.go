package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io"
   "log"
   "net/url"
   "os"
   "strconv"
   "strings"
)

type MPD struct {
   XMLName  xml.Name `xml:"MPD"`
   BaseURLs []string `xml:"BaseURL"`
   Periods  []Period `xml:"Period"`
}

type Period struct {
   BaseURLs       []string        `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   ID              string           `xml:"id,attr,omitempty"`
   BaseURLs        []string         `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURLs        []string         `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
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
   Initialization  string           `xml:"initialization,attr,omitempty"`
   StartNumber     uint64           `xml:"startNumber,attr,omitempty"`
   EndNumber       uint64           `xml:"endNumber,attr,omitempty"`
   Timescale       uint64           `xml:"timescale,attr,omitempty"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []STEntry `xml:"S"`
}

type STEntry struct {
   T *uint64 `xml:"t,attr,omitempty"`
   D uint64  `xml:"d,attr"`
   R *int64  `xml:"r,attr,omitempty"`
}

func main() {
   log.SetFlags(0)

   if len(os.Args) != 2 {
      fmt.Fprintln(os.Stderr, "Usage: go run main.go <mpd_file_path>")
      os.Exit(1)
   }
   mpdPath := os.Args[1]

   f, err := os.Open(mpdPath)
   if err != nil {
      log.Fatalf("Failed to open MPD file: %v", err)
   }
   defer f.Close()

   data, err := io.ReadAll(f)
   if err != nil {
      log.Fatalf("Failed to read MPD file: %v", err)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      log.Fatalf("Failed to parse MPD XML: %v", err)
   }

   // Initial BaseURL
   initialBase := "http://test.test/test.mpd"
   mpdBases := []string{initialBase}
   if len(mpd.BaseURLs) > 0 {
      mpdBases = resolveAll(initialBase, mpd.BaseURLs)
   }

   // Map rep ID to segments
   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      periodBases := mpdBases
      if len(period.BaseURLs) > 0 {
         periodBases = resolveAllSet(mpdBases, period.BaseURLs)
      }
      for _, adap := range period.AdaptationSets {
         adapBases := periodBases
         if len(adap.BaseURLs) > 0 {
            adapBases = resolveAllSet(periodBases, adap.BaseURLs)
         }
         for _, rep := range adap.Representations {
            repBases := adapBases
            if len(rep.BaseURLs) > 0 {
               repBases = resolveAllSet(adapBases, rep.BaseURLs)
            }

            var segments []string
            appendURLs := func(paths []string) {
               for _, base := range repBases {
                  bu, err := url.Parse(base)
                  if err != nil {
                     log.Fatalf("Invalid BaseURL %q: %v", base, err)
                  }
                  for _, p := range paths {
                     rel, err := url.Parse(p)
                     if err != nil {
                        log.Fatalf("Invalid segment URL %q: %v", p, err)
                     }
                     segments = append(segments, bu.ResolveReference(rel).String())
                  }
               }
            }

            // SegmentList fallback
            if rep.SegmentList != nil || adap.SegmentList != nil {
               sl := rep.SegmentList
               if sl == nil {
                  sl = adap.SegmentList
               }
               // init
               if sl.Initialization != nil {
                  initURL := strings.ReplaceAll(sl.Initialization.SourceURL, "$RepresentationID$", rep.ID)
                  appendURLs([]string{initURL})
               }
               // media
               media := make([]string, len(sl.SegmentURLs))
               for i, su := range sl.SegmentURLs {
                  media[i] = strings.ReplaceAll(su.Media, "$RepresentationID$", rep.ID)
               }
               appendURLs(media)

               // SegmentTemplate fallback
            } else if rep.SegmentTemplate != nil || adap.SegmentTemplate != nil {
               st := rep.SegmentTemplate
               if st == nil {
                  st = adap.SegmentTemplate
               }
               start := st.StartNumber
               if start == 0 {
                  start = 1
               }
               // init
               if st.Initialization != "" {
                  initURL := substitute(st.Initialization, rep.ID, start, 0)
                  appendURLs([]string{initURL})
               }
               // timeline
               if st.SegmentTimeline != nil {
                  seq := start
                  var prevT uint64
                  for _, e := range st.SegmentTimeline.S {
                     t0 := prevT
                     if e.T != nil {
                        t0 = *e.T
                     }
                     repCount := int64(0)
                     if e.R != nil {
                        repCount = *e.R
                     }
                     for i := int64(0); i <= repCount; i++ {
                        curT := t0 + uint64(i)*e.D
                        url := substitute(st.Media, rep.ID, seq, curT)
                        appendURLs([]string{url})
                        seq++
                     }
                     prevT = t0 + uint64(repCount+1)*e.D
                  }
                  // no timeline: use number range
               } else {
                  if st.EndNumber == 0 {
                     log.Fatalf("Missing SegmentTimeline or endNumber for %s", rep.ID)
                  }
                  for n := start; n <= st.EndNumber; n++ {
                     url := substitute(st.Media, rep.ID, n, 0)
                     appendURLs([]string{url})
                  }
               }

               // Fallback to BaseURL if no segments
            } else {
               segments = append(segments, repBases...)
            }

            result[rep.ID] = segments
         }
      }
   }

   // Output JSON
   out, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      log.Fatalf("JSON marshal error: %v", err)
   }
   fmt.Println(string(out))
}

func resolveAll(base string, rels []string) []string {
   parent, err := url.Parse(base)
   if err != nil {
      log.Fatalf("Invalid base URL %q: %v", base, err)
   }
   var res []string
   for _, r := range rels {
      u, err := url.Parse(r)
      if err != nil {
         log.Fatalf("Invalid BaseURL %q: %v", r, err)
      }
      res = append(res, parent.ResolveReference(u).String())
   }
   return res
}

func resolveAllSet(bases, rels []string) []string {
   var res []string
   for _, b := range bases {
      res = append(res, resolveAll(b, rels)...)
   }
   return res
}

func substitute(tmpl, repID string, number, tm uint64) string {
   s := strings.ReplaceAll(tmpl, "$RepresentationID$", repID)
   if strings.Contains(s, "$Number$") {
      s = strings.ReplaceAll(s, "$Number$", strconv.FormatUint(number, 10))
   }
   if strings.Contains(s, "$Time$") {
      s = strings.ReplaceAll(s, "$Time$", strconv.FormatUint(tm, 10))
   }
   return s
}
