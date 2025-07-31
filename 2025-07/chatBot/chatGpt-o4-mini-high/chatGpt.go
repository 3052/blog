package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "log"
   "math"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
)

const defaultBase = "http://test.test/test.mpd"

// MPD root
type MPD struct {
   XMLName  xml.Name `xml:"MPD"`
   BaseURLs []string `xml:"BaseURL"`
   Periods  []Period `xml:"Period"`
}

// Period, including its duration for SegmentTemplate fallback
type Period struct {
   XMLName         xml.Name         `xml:"Period"`
   Duration        string           `xml:"duration,attr"` // ISO8601, e.g. "PT60S"
   BaseURLs        []string         `xml:"BaseURL"`
   AdaptationSets  []AdaptationSet  `xml:"AdaptationSet"`
   Representations []Representation `xml:"Representation"`
}

// AdaptationSet may have its own BaseURLs and SegmentTemplate
type AdaptationSet struct {
   XMLName         xml.Name         `xml:"AdaptationSet"`
   BaseURLs        []string         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

// Representation may have a SegmentList or its own SegmentTemplate
type Representation struct {
   XMLName         xml.Name         `xml:"Representation"`
   ID              string           `xml:"id,attr"`
   BaseURLs        []string         `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

// For <SegmentList>
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

// For <SegmentTemplate>
type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   Timescale       int64            `xml:"timescale,attr"` // default 1 if zero
   Duration        int64            `xml:"duration,attr"`
   StartNumber     int64            `xml:"startNumber,attr"` // default 1 if zero
   EndNumber       int64            `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T int64 `xml:"t,attr"` // start time (optional)
   D int64 `xml:"d,attr"` // duration (required)
   R int64 `xml:"r,attr"` // repeat count (optional)
}

// parseISO8601Duration parses PT#H#M#S durations
func parseISO8601Duration(dur string) (float64, error) {
   if !strings.HasPrefix(dur, "P") {
      return 0, fmt.Errorf("invalid duration: %s", dur)
   }
   // extract time part after 'T'
   timePart := dur
   if i := strings.Index(dur, "T"); i >= 0 {
      timePart = dur[i+1:]
   }
   re := regexp.MustCompile(`([\d\.]+)([HMS])`)
   var total float64
   for _, m := range re.FindAllStringSubmatch(timePart, -1) {
      v, err := strconv.ParseFloat(m[1], 64)
      if err != nil {
         return 0, err
      }
      switch m[2] {
      case "H":
         total += v * 3600
      case "M":
         total += v * 60
      case "S":
         total += v
      }
   }
   return total, nil
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }
   mpdPath := os.Args[1]

   data, err := os.ReadFile(mpdPath)
   if err != nil {
      log.Fatalf("Error reading MPD: %v", err)
   }
   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      log.Fatalf("Error parsing MPD XML: %v", err)
   }

   // Start from defaultBase and chain MPD-level BaseURLs
   baseURL, err := url.Parse(defaultBase)
   if err != nil {
      log.Fatalf("Invalid default base URL: %v", err)
   }
   for _, bu := range mpd.BaseURLs {
      rel, err := url.Parse(bu)
      if err != nil {
         log.Fatalf("Invalid MPD BaseURL %q: %v", bu, err)
      }
      baseURL = baseURL.ResolveReference(rel)
   }

   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      // Chain Period-level BaseURLs
      periodBase := baseURL
      for _, bu := range period.BaseURLs {
         rel, err := url.Parse(bu)
         if err != nil {
            log.Fatalf("Invalid Period BaseURL %q: %v", bu, err)
         }
         periodBase = periodBase.ResolveReference(rel)
      }

      // Parse period duration for fallback
      var periodSeconds float64
      if period.Duration != "" {
         sec, err := parseISO8601Duration(period.Duration)
         if err != nil {
            log.Fatalf("Error parsing Period duration %q: %v", period.Duration, err)
         }
         periodSeconds = sec
      }

      // 1) Handle top-level Representations (outside AdaptationSets)
      for _, rep := range period.Representations {
         processRepresentation(periodBase, nil, rep, periodSeconds, result)
      }

      // 2) Handle AdaptationSets (with their own BaseURLs & SegmentTemplate)
      for _, aset := range period.AdaptationSets {
         // Chain AdaptationSet-level BaseURLs
         asetBase := periodBase
         for _, bu := range aset.BaseURLs {
            rel, err := url.Parse(bu)
            if err != nil {
               log.Fatalf("Invalid AdaptationSet BaseURL %q: %v", bu, err)
            }
            asetBase = asetBase.ResolveReference(rel)
         }
         for _, rep := range aset.Representations {
            processRepresentation(asetBase, aset.SegmentTemplate, rep, periodSeconds, result)
         }
      }
   }

   out, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      log.Fatalf("JSON marshal error: %v", err)
   }
   fmt.Println(string(out))
}

// processRepresentation handles one Representation, given its parent base URL,
// an optional inherited SegmentTemplate from its AdaptationSet, the period duration,
// and writes into result map.
func processRepresentation(parentBase *url.URL, inheritedTmpl *SegmentTemplate, rep Representation, periodSec float64, result map[string][]string) {
   // Chain Representation-level BaseURLs
   repBase := parentBase
   for _, bu := range rep.BaseURLs {
      rel, err := url.Parse(bu)
      if err != nil {
         log.Fatalf("Invalid Representation BaseURL %q: %v", bu, err)
      }
      repBase = repBase.ResolveReference(rel)
   }

   var segs []string

   // 1) <SegmentList> if present
   if rep.SegmentList != nil {
      if rep.SegmentList.Initialization != nil {
         rel, err := url.Parse(rep.SegmentList.Initialization.SourceURL)
         if err != nil {
            log.Fatalf("Invalid init URL %q: %v", rep.SegmentList.Initialization.SourceURL, err)
         }
         segs = append(segs, repBase.ResolveReference(rel).String())
      }
      for _, su := range rep.SegmentList.SegmentURLs {
         rel, err := url.Parse(su.Media)
         if err != nil {
            log.Fatalf("Invalid segment URL %q: %v", su.Media, err)
         }
         segs = append(segs, repBase.ResolveReference(rel).String())
      }

   } else {
      // 2) <SegmentTemplate>: rep-level overrides inherited
      tmpl := inheritedTmpl
      if rep.SegmentTemplate != nil {
         tmpl = rep.SegmentTemplate
      }
      if tmpl != nil {
         // Replace helper
         repReplacer := strings.NewReplacer("$RepresentationID$", rep.ID)
         // Initialization
         if tmpl.Initialization != "" {
            initURL := repReplacer.Replace(tmpl.Initialization)
            rel, err := url.Parse(initURL)
            if err != nil {
               log.Fatalf("Invalid init template %q: %v", initURL, err)
            }
            segs = append(segs, repBase.ResolveReference(rel).String())
         }
         startNum := tmpl.StartNumber
         if startNum == 0 {
            startNum = 1
         }
         timescale := tmpl.Timescale
         if timescale == 0 {
            timescale = 1
         }
         // 2a) SegmentTimeline
         if tmpl.SegmentTimeline != nil && len(tmpl.SegmentTimeline.S) > 0 {
            num := startNum
            curTime := int64(0)
            if first := tmpl.SegmentTimeline.S[0]; first.T != 0 {
               curTime = first.T
            }
            for _, e := range tmpl.SegmentTimeline.S {
               repeat := e.R
               if repeat < 0 {
                  repeat = 0
               }
               for i := int64(0); i <= repeat; i++ {
                  r := strings.NewReplacer(
                     "$RepresentationID$", rep.ID,
                     "$Number$", strconv.FormatInt(num, 10),
                     "$Time$", strconv.FormatInt(curTime, 10),
                  )
                  mediaURL := r.Replace(tmpl.Media)
                  rel, err := url.Parse(mediaURL)
                  if err != nil {
                     log.Fatalf("Invalid media template %q: %v", mediaURL, err)
                  }
                  segs = append(segs, repBase.ResolveReference(rel).String())
                  num++
                  curTime += e.D
               }
            }
            // 2b) Numeric startNumber → endNumber
         } else if tmpl.EndNumber > 0 {
            for n := startNum; n <= tmpl.EndNumber; n++ {
               r := strings.NewReplacer(
                  "$RepresentationID$", rep.ID,
                  "$Number$", strconv.FormatInt(n, 10),
               )
               mediaURL := r.Replace(tmpl.Media)
               rel, err := url.Parse(mediaURL)
               if err != nil {
                  log.Fatalf("Invalid media template %q: %v", mediaURL, err)
               }
               segs = append(segs, repBase.ResolveReference(rel).String())
            }
            // 2c) Compute count by ceil(periodSec * timescale / duration)
         } else if tmpl.Duration > 0 && timescale > 0 && periodSec > 0 {
            count := int64(math.Ceil(periodSec * float64(timescale) / float64(tmpl.Duration)))
            for i := int64(0); i < count; i++ {
               num := startNum + i
               timeVal := i * tmpl.Duration
               r := strings.NewReplacer(
                  "$RepresentationID$", rep.ID,
                  "$Number$", strconv.FormatInt(num, 10),
                  "$Time$", strconv.FormatInt(timeVal, 10),
               )
               mediaURL := r.Replace(tmpl.Media)
               rel, err := url.Parse(mediaURL)
               if err != nil {
                  log.Fatalf("Invalid media template %q: %v", mediaURL, err)
               }
               segs = append(segs, repBase.ResolveReference(rel).String())
            }
         }
      }
   }

   if len(segs) > 0 {
      result[rep.ID] = segs
   }
}
