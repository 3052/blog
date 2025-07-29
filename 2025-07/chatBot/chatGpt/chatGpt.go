package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io"
   "math"
   "net/url"
   "os"
   "path/filepath"
   "regexp"
   "strconv"
   "strings"
   "time"
)

func fatal(err error) {
   fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
   os.Exit(1)
}

type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   BaseURL                   string   `xml:"BaseURL"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
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
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   Timescale       uint64           `xml:"timescale,attr"`
   StartNumber     uint64           `xml:"startNumber,attr"`
   EndNumber       *uint64          `xml:"endNumber,attr"`
   Duration        uint64           `xml:"duration,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T *uint64 `xml:"t,attr"`
   D uint64  `xml:"d,attr"`
   R *int64  `xml:"r,attr"`
}

type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURL     []SegmentURL    `xml:"SegmentURL"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

func joinBaseURLs(baseURLs ...string) string {
   u, err := url.Parse(baseURLs[0])
   if err != nil {
      return baseURLs[0]
   }
   for _, bu := range baseURLs[1:] {
      if bu == "" {
         continue
      }
      ref, err := url.Parse(bu)
      if err != nil {
         continue
      }
      u = u.ResolveReference(ref)
   }
   return u.String()
}

func resolveURL(base, ref string) (string, error) {
   baseURL, err := url.Parse(base)
   if err != nil {
      return "", err
   }
   refURL, err := url.Parse(ref)
   if err != nil {
      return "", err
   }
   return baseURL.ResolveReference(refURL).String(), nil
}

var reNumber = regexp.MustCompile(`\$Number(%0(\d+)d)?\$`)
var reTime = regexp.MustCompile(`\$Time(%0(\d+)d)?\$`)

func substituteTemplate(template string, number uint64, timestamp uint64, repID string) string {
   s := reNumber.ReplaceAllStringFunc(template, func(m string) string {
      sub := reNumber.FindStringSubmatch(m)
      if sub[2] != "" {
         width, _ := strconv.Atoi(sub[2])
         return fmt.Sprintf("%0*d", width, number)
      }
      return fmt.Sprintf("%d", number)
   })
   s = reTime.ReplaceAllStringFunc(s, func(m string) string {
      sub := reTime.FindStringSubmatch(m)
      if sub[2] != "" {
         width, _ := strconv.Atoi(sub[2])
         return fmt.Sprintf("%0*d", width, timestamp)
      }
      return fmt.Sprintf("%d", timestamp)
   })
   s = strings.ReplaceAll(s, "$RepresentationID$", repID)
   return s
}

func segmentTemplateInitializationURL(baseURL string, st *SegmentTemplate, repID string) (string, error) {
   if st.Initialization == "" {
      return "", nil
   }
   initURL := substituteTemplate(st.Initialization, st.StartNumber, 0, repID)
   return resolveURL(baseURL, initURL)
}

func segmentTemplateURLs(baseURL string, st *SegmentTemplate, repID string, periodDurationSec float64) ([]string, error) {
   media := st.Media
   if media == "" {
      return nil, fmt.Errorf("SegmentTemplate missing media")
   }
   startNumber := st.StartNumber
   if startNumber == 0 {
      startNumber = 1
   }
   endNumber := uint64(0)
   if st.EndNumber != nil {
      endNumber = *st.EndNumber
   }

   var urls []string

   if st.SegmentTimeline != nil && len(st.SegmentTimeline.S) > 0 {
      number := startNumber
      var timestamps []uint64
      for i, s := range st.SegmentTimeline.S {
         var t uint64
         if s.T != nil {
            t = *s.T
         } else if i == 0 {
            t = 0
         } else {
            t = timestamps[len(timestamps)-1] + st.SegmentTimeline.S[i-1].D
         }
         repeat := int64(0)
         if s.R != nil {
            repeat = *s.R
         }
         for j := int64(0); j <= repeat; j++ {
            ts := t + uint64(j)*s.D
            url := substituteTemplate(media, number, ts, repID)
            full, err := resolveURL(baseURL, url)
            if err != nil {
               return nil, err
            }
            timestamps = append(timestamps, ts)
            urls = append(urls, full)
            number++
         }
      }
      return urls, nil
   }

   if strings.Contains(media, "$Number") {

      timescale := st.Timescale
      if timescale == 0 {
         timescale = 1
      }
      if st.Duration != 0 && periodDurationSec > 0 {
         segCount := uint64(math.Ceil(periodDurationSec * float64(timescale) / float64(st.Duration)))
         endNumber = startNumber + segCount - 1
      }

      for i := startNumber; i <= endNumber; i++ {
         url := substituteTemplate(media, i, 0, repID)
         full, err := resolveURL(baseURL, url)
         if err != nil {
            return nil, err
         }
         urls = append(urls, full)
      }
      return urls, nil
   }

   full, err := resolveURL(baseURL, media)
   if err != nil {
      return nil, err
   }
   return []string{full}, nil
}

func segmentListInitializationURL(baseURL string, sl *SegmentList) (string, error) {
   if sl == nil || sl.Initialization == nil {
      return "", nil
   }
   return resolveURL(baseURL, sl.Initialization.SourceURL)
}

func parseISODuration(s string) (time.Duration, error) {
   if !strings.HasPrefix(s, "PT") {
      return 0, fmt.Errorf("unsupported duration format: %s", s)
   }
   s = strings.TrimPrefix(s, "PT")
   var dur time.Duration
   var num float64
   for len(s) > 0 {
      for i, r := range s {
         if r < '0' || r > '9' {
            if r == '.' {
               continue
            }
            numPart := s[:i+1]
            var unit rune
            fmt.Sscanf(numPart, "%f%c", &num, &unit)
            switch unit {
            case 'H':
               dur += time.Duration(num * float64(time.Hour))
            case 'M':
               dur += time.Duration(num * float64(time.Minute))
            case 'S':
               dur += time.Duration(num * float64(time.Second))
            default:
               return 0, fmt.Errorf("unsupported time unit: %c", unit)
            }
            s = s[i+1:]
            break
         }
      }
   }
   return dur, nil
}

func segmentsForRepresentation(mpdBase, periodBase, asetBase, repBase string, aset *AdaptationSet, rep *Representation, periodDurationSec float64) ([]string, error) {
   baseURL := joinBaseURLs(mpdBase, periodBase, asetBase, repBase)
   repID := rep.ID

   var st *SegmentTemplate
   if rep.SegmentTemplate != nil {
      st = rep.SegmentTemplate
   } else {
      st = aset.SegmentTemplate
   }

   var sl *SegmentList
   if rep.SegmentList != nil {
      sl = rep.SegmentList
   } else {
      sl = aset.SegmentList
   }

   var urls []string

   if st != nil {
      initURL, err := segmentTemplateInitializationURL(baseURL, st, repID)
      if err != nil {
         return nil, err
      }
      if initURL != "" {
         urls = append(urls, initURL)
      }
      segURLs, err := segmentTemplateURLs(baseURL, st, repID, periodDurationSec)
      if err != nil {
         return nil, err
      }
      urls = append(urls, segURLs...)
   } else if sl != nil {
      initURL, err := segmentListInitializationURL(baseURL, sl)
      if err != nil {
         return nil, err
      }
      if initURL != "" {
         urls = append(urls, initURL)
      }
      for _, seg := range sl.SegmentURL {
         full, err := resolveURL(baseURL, seg.Media)
         if err != nil {
            return nil, err
         }
         urls = append(urls, full)
      }
   } else if rep.BaseURL != "" {
      full, err := resolveURL(joinBaseURLs(mpdBase, periodBase, asetBase), rep.BaseURL)
      if err != nil {
         return nil, err
      }
      urls = append(urls, full)
   }

   return urls, nil
}

func main() {
   if len(os.Args) < 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", filepath.Base(os.Args[0]))
      os.Exit(1)
   }

   f, err := os.Open(os.Args[1])
   if err != nil {
      fatal(err)
   }
   defer f.Close()

   data, err := io.ReadAll(f)
   if err != nil {
      fatal(err)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fatal(err)
   }

   const mpdBase = "http://test.test/test.mpd"
   out := map[string][]string{}

   for _, period := range mpd.Periods {
      periodDur := 0.0
      if d, err := parseISODuration(period.Duration); err == nil {
         periodDur = d.Seconds()
      } else if d, err := parseISODuration(mpd.MediaPresentationDuration); err == nil {
         periodDur = d.Seconds()
      }
      for _, aset := range period.AdaptationSet {
         for _, rep := range aset.Representations {
            if rep.ID == "" {
               continue
            }
            segments, err := segmentsForRepresentation(
               mpdBase, period.BaseURL, aset.BaseURL, rep.BaseURL,
               &aset, &rep, periodDur,
            )
            if err != nil {
               fatal(fmt.Errorf("representation %s: %w", rep.ID, err))
            }
            out[rep.ID] = append(out[rep.ID], segments...)
         }
      }
   }

   b, err := json.MarshalIndent(out, "", "  ")
   if err != nil {
      fatal(err)
   }
   fmt.Println(string(b))
}
