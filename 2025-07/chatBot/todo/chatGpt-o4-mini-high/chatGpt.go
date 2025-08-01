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
   "time"
)

const defaultBase = "http://test.test/test.mpd"

// MPD, Period, AdaptationSet, Representation and segment structs
type MPD struct {
   XMLName  xml.Name `xml:"MPD"`
   BaseURLs []string `xml:"BaseURL"`
   Periods  []Period `xml:"Period"`
}

type Period struct {
   BaseURLs       []string        `xml:"BaseURL"`
   Duration       string          `xml:"duration,attr"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
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
   Initialization Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL   `xml:"SegmentURL"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

type SegmentTemplate struct {
   TimescaleStr    string           `xml:"timescale,attr"`
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   DurationStr     string           `xml:"duration,attr"`
   StartNumberStr  string           `xml:"startNumber,attr"`
   EndNumberStr    string           `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   TStr string `xml:"t,attr"`
   DStr string `xml:"d,attr"`
   RStr string `xml:"r,attr"`
}

func main() {
   if len(os.Args) != 2 {
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

   defaultBaseURL, err := url.Parse(defaultBase)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing default base URL '%s': %v\n", defaultBase, err)
      os.Exit(1)
   }
   mpdBases := []*url.URL{defaultBaseURL}
   if len(mpd.BaseURLs) > 0 {
      mpdBases, err = applyBaseURLs(mpdBases, mpd.BaseURLs)
      if err != nil {
         fmt.Fprintf(os.Stderr, "Error processing MPD BaseURLs: %v\n", err)
         os.Exit(1)
      }
   }

   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      periodBases := mpdBases
      if len(period.BaseURLs) > 0 {
         periodBases, err = applyBaseURLs(periodBases, period.BaseURLs)
         if err != nil {
            fmt.Fprintf(os.Stderr, "Error processing Period BaseURLs: %v\n", err)
            os.Exit(1)
         }
      }
      periodDur := time.Duration(0)
      if period.Duration != "" {
         periodDur, err = parseISO8601Duration(period.Duration)
         if err != nil {
            fmt.Fprintf(os.Stderr, "Error parsing Period duration '%s': %v\n", period.Duration, err)
            os.Exit(1)
         }
      }

      for _, as := range period.AdaptationSets {
         asBases := periodBases
         if len(as.BaseURLs) > 0 {
            asBases, err = applyBaseURLs(asBases, as.BaseURLs)
            if err != nil {
               fmt.Fprintf(os.Stderr, "Error processing AdaptationSet BaseURLs: %v\n", err)
               os.Exit(1)
            }
         }

         for _, rep := range as.Representations {
            repBases := asBases
            if len(rep.BaseURLs) > 0 {
               repBases, err = applyBaseURLs(repBases, rep.BaseURLs)
               if err != nil {
                  fmt.Fprintf(os.Stderr, "Error processing Representation BaseURLs for rep '%s': %v\n", rep.ID, err)
                  os.Exit(1)
               }
            }

            var urls []string
            switch {
            case rep.SegmentList != nil:
               urls, err = generateURLsFromSegmentList(repBases, rep.SegmentList)
            case rep.SegmentTemplate != nil:
               urls, err = generateURLsFromSegmentTemplate(repBases, rep.SegmentTemplate, rep.ID, periodDur)
            case as.SegmentList != nil:
               urls, err = generateURLsFromSegmentList(repBases, as.SegmentList)
            case as.SegmentTemplate != nil:
               urls, err = generateURLsFromSegmentTemplate(repBases, as.SegmentTemplate, rep.ID, periodDur)
            default:
               for _, b := range repBases {
                  urls = append(urls, b.String())
               }
            }
            if err != nil {
               fmt.Fprintf(os.Stderr, "Error generating segment URLs for rep '%s': %v\n", rep.ID, err)
               os.Exit(1)
            }
            result[rep.ID] = append(result[rep.ID], urls...)
         }
      }
   }

   out, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error marshaling result to JSON: %v\n", err)
      os.Exit(1)
   }
   fmt.Println(string(out))
}

func applyBaseURLs(bases []*url.URL, refs []string) ([]*url.URL, error) {
   var out []*url.URL
   for _, ref := range refs {
      rel, err := url.Parse(strings.TrimSpace(ref))
      if err != nil {
         return nil, fmt.Errorf("invalid BaseURL '%s': %v", ref, err)
      }
      for _, b := range bases {
         out = append(out, b.ResolveReference(rel))
      }
   }
   return out, nil
}

func generateURLsFromSegmentList(bases []*url.URL, sl *SegmentList) ([]string, error) {
   var urls []string
   if sl.Initialization.SourceURL != "" {
      relInit, err := url.Parse(sl.Initialization.SourceURL)
      if err != nil {
         return nil, fmt.Errorf("invalid initialization URL '%s': %v", sl.Initialization.SourceURL, err)
      }
      for _, b := range bases {
         urls = append(urls, b.ResolveReference(relInit).String())
      }
   }
   for _, su := range sl.SegmentURLs {
      relSeg, err := url.Parse(su.Media)
      if err != nil {
         return nil, fmt.Errorf("invalid segment URL '%s': %v", su.Media, err)
      }
      for _, b := range bases {
         urls = append(urls, b.ResolveReference(relSeg).String())
      }
   }
   return urls, nil
}

// generateURLsFromSegmentTemplate handles templates, timelines, formatting, and numbering
func generateURLsFromSegmentTemplate(bases []*url.URL, tmpl *SegmentTemplate, repID string, periodDur time.Duration) ([]string, error) {
   var err error
   // regex for formatting placeholders
   numFmtRe := regexp.MustCompile(`\$Number(%0\d+d)?\$`)
   timeFmtRe := regexp.MustCompile(`\$Time(%0\d+d)?\$`)

   // parse timescale
   timescale := int64(1)
   if tmpl.TimescaleStr != "" {
      timescale, err = strconv.ParseInt(tmpl.TimescaleStr, 10, 64)
      if err != nil {
         return nil, fmt.Errorf("invalid timescale '%s': %v", tmpl.TimescaleStr, err)
      }
   }
   // parse duration units
   var durUnits int64
   if tmpl.DurationStr != "" {
      durUnits, err = strconv.ParseInt(tmpl.DurationStr, 10, 64)
      if err != nil {
         return nil, fmt.Errorf("invalid duration '%s': %v", tmpl.DurationStr, err)
      }
   }
   // parse startNumber
   startNum := int64(1)
   if tmpl.StartNumberStr != "" {
      startNum, err = strconv.ParseInt(tmpl.StartNumberStr, 10, 64)
      if err != nil {
         return nil, fmt.Errorf("invalid startNumber '%s': %v", tmpl.StartNumberStr, err)
      }
   }
   // parse endNumber
   hasEnd := false
   var endNum int64
   if tmpl.EndNumberStr != "" {
      endNum, err = strconv.ParseInt(tmpl.EndNumberStr, 10, 64)
      if err != nil {
         return nil, fmt.Errorf("invalid endNumber '%s': %v", tmpl.EndNumberStr, err)
      }
      hasEnd = true
   }
   // build segment list
   type seg struct{ Number, Time int64 }
   var segments []seg
   if tmpl.SegmentTimeline != nil {
      prevTime := int64(0)
      curNum := startNum
      for _, e := range tmpl.SegmentTimeline.S {
         d, err := strconv.ParseInt(e.DStr, 10, 64)
         if err != nil {
            return nil, fmt.Errorf("invalid timeline duration '%s': %v", e.DStr, err)
         }
         r := int64(0)
         if e.RStr != "" {
            r, err = strconv.ParseInt(e.RStr, 10, 64)
            if err != nil {
               return nil, fmt.Errorf("invalid repeat '%s': %v", e.RStr, err)
            }
         }
         count := r + 1
         // corrected t0 logic
         var t0 int64
         if e.TStr != "" {
            t0, err = strconv.ParseInt(e.TStr, 10, 64)
            if err != nil {
               return nil, fmt.Errorf("invalid timeline time '%s': %v", e.TStr, err)
            }
         } else {
            t0 = prevTime
         }
         for i := int64(0); i < count; i++ {
            segments = append(segments, seg{Number: curNum, Time: t0 + i*d})
            curNum++
         }
         prevTime = t0 + count*d
      }
   } else if hasEnd {
      for n := startNum; n <= endNum; n++ {
         segments = append(segments, seg{Number: n, Time: (n - startNum) * durUnits})
      }
   } else if durUnits > 0 && periodDur > 0 {
      total := int64(math.Ceil(periodDur.Seconds() * float64(timescale) / float64(durUnits)))
      for i := int64(0); i < total; i++ {
         segments = append(segments, seg{Number: startNum + i, Time: i * durUnits})
      }
   } else {
      return nil, fmt.Errorf("cannot determine segments for representation %s", repID)
   }
   var urls []string
   // initialization
   if tmpl.Initialization != "" {
      initT := strings.ReplaceAll(tmpl.Initialization, "$RepresentationID$", repID)
      relInit, err := url.Parse(initT)
      if err != nil {
         return nil, fmt.Errorf("invalid initialization template '%s': %v", initT, err)
      }
      for _, b := range bases {
         urls = append(urls, b.ResolveReference(relInit).String())
      }
   }
   // media segments with formatting
   for _, s := range segments {
      m := tmpl.Media
      // replace RepresentationID
      m = strings.ReplaceAll(m, "$RepresentationID$", repID)
      // replace Number with formatting
      m = numFmtRe.ReplaceAllStringFunc(m, func(match string) string {
         subs := numFmtRe.FindStringSubmatch(match)
         fmtSpec := subs[1] // e.g. %08d or empty
         if fmtSpec == "" {
            return strconv.FormatInt(s.Number, 10)
         }
         return fmt.Sprintf(fmtSpec, s.Number)
      })
      // replace Time with formatting
      m = timeFmtRe.ReplaceAllStringFunc(m, func(match string) string {
         subs := timeFmtRe.FindStringSubmatch(match)
         fmtSpec := subs[1]
         if fmtSpec == "" {
            return strconv.FormatInt(s.Time, 10)
         }
         return fmt.Sprintf(fmtSpec, s.Time)
      })
      relMedia, err := url.Parse(m)
      if err != nil {
         return nil, fmt.Errorf("invalid media template '%s': %v", m, err)
      }
      for _, b := range bases {
         urls = append(urls, b.ResolveReference(relMedia).String())
      }
   }
   return urls, nil
}

// parseISO8601Duration handles a subset of ISO 8601 durations (days, hours, minutes, seconds)
func parseISO8601Duration(s string) (time.Duration, error) {
   if s == "" || s[0] != 'P' {
      return 0, fmt.Errorf("invalid duration: %s", s)
   }
   s = s[1:]
   var dur time.Duration
   var datePart, timePart string
   if idx := strings.IndexByte(s, 'T'); idx != -1 {
      datePart = s[:idx]
      timePart = s[idx+1:]
   } else {
      datePart = s
   }
   // days
   for len(datePart) > 0 {
      pos := strings.IndexAny(datePart, "YMD")
      if pos == -1 {
         break
      }
      num, err := strconv.ParseFloat(datePart[:pos], 64)
      if err != nil {
         return 0, err
      }
      switch datePart[pos] {
      case 'D':
         dur += time.Duration(num) * 24 * time.Hour
      }
      datePart = datePart[pos+1:]
   }
   // time part
   for len(timePart) > 0 {
      pos := strings.IndexAny(timePart, "HMS")
      if pos == -1 {
         break
      }
      num, err := strconv.ParseFloat(timePart[:pos], 64)
      if err != nil {
         return 0, err
      }
      switch timePart[pos] {
      case 'H':
         dur += time.Duration(num) * time.Hour
      case 'M':
         dur += time.Duration(num) * time.Minute
      case 'S':
         dur += time.Duration(num * float64(time.Second))
      }
      timePart = timePart[pos+1:]
   }
   return dur, nil
}
