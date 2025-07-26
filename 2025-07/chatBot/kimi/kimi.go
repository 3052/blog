package main

import (
   "encoding/json"
   "encoding/xml"
   "errors"
   "fmt"
   "math"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
   "time"
)

// ---------------------------------------------------------------------------
// CLI / main
// ---------------------------------------------------------------------------

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintln(os.Stderr, "usage: mpdexpand <local.mpd>")
      os.Exit(1)
   }
   mpdPath := os.Args[1]

   f, err := os.Open(mpdPath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "open mpd: %v\n", err)
      os.Exit(1)
   }
   defer f.Close()

   var mpd MPD
   if err := xml.NewDecoder(f).Decode(&mpd); err != nil {
      fmt.Fprintf(os.Stderr, "parse mpd: %v\n", err)
      os.Exit(1)
   }

   // The root BaseURL is *always* the literal URL from the xml url element
   rootBase := "http://test.test/test.mpd" // per requirement #2
   segments, err := expandMPD(&mpd, rootBase)
   if err != nil {
      fmt.Fprintf(os.Stderr, "expand: %v\n", err)
      os.Exit(1)
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetEscapeHTML(false)
   if err := enc.Encode(segments); err != nil {
      fmt.Fprintf(os.Stderr, "encode: %v\n", err)
      os.Exit(1)
   }
}

// ---------------------------------------------------------------------------
// Data model
// ---------------------------------------------------------------------------

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   DurationISO string          `xml:"duration,attr"` // ISO-8601
   BaseURL     string          `xml:"BaseURL"`
   AdaptSets   []AdaptationSet `xml:"AdaptationSet"`
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
   Media     string           `xml:"media,attr"`
   Timescale int              `xml:"timescale,attr"`
   Duration  int              `xml:"duration,attr"`
   StartNum  *int             `xml:"startNumber,attr"` // nil = absent => default 1
   EndNum    *int             `xml:"endNumber,attr"`
   Timeline  *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentList struct {
   Timescale   int              `xml:"timescale,attr"`
   Duration    int              `xml:"duration,attr"`
   StartNum    *int             `xml:"startNumber,attr"`
   EndNum      *int             `xml:"endNumber,attr"`
   Timeline    *SegmentTimeline `xml:"SegmentTimeline"`
   SegmentURLs []SegmentURL     `xml:"SegmentURL"`
}

type SegmentURL struct {
   MediaURL string `xml:"media,attr"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T int `xml:"t,attr"` // d or t
   D int `xml:"d,attr"`
   R int `xml:"r,attr"` // repetitions
}

// ---------------------------------------------------------------------------
// Expansion
// ---------------------------------------------------------------------------

type urlSet map[string]struct{} // used to deduplicate per Representation

func expandMPD(mpd *MPD, rootBase string) (map[string][]string, error) {
   out := make(map[string][]string) // RepresentationID -> []URL

   for _, period := range mpd.Periods {
      periodBase, err := resolveBase(rootBase, mpd.BaseURL, period.BaseURL)
      if err != nil {
         return nil, err
      }
      periodDur, err := parseISODuration(period.DurationISO)
      if err != nil {
         return nil, fmt.Errorf("period duration: %w", err)
      }
      periodDurSec := periodDur.Seconds()

      for _, as := range period.AdaptSets {
         asBase, err := resolveBase(periodBase, as.BaseURL)
         if err != nil {
            return nil, err
         }

         for _, rep := range as.Representations {
            repBase, err := resolveBase(asBase, rep.BaseURL)
            if err != nil {
               return nil, err
            }

            // effective segment template / list
            var st *SegmentTemplate
            var sl *SegmentList
            if rep.SegmentTemplate != nil {
               st = rep.SegmentTemplate
            } else if as.SegmentTemplate != nil {
               st = as.SegmentTemplate
            }
            if rep.SegmentList != nil {
               sl = rep.SegmentList
            } else if as.SegmentList != nil {
               sl = as.SegmentList
            }

            var urls []string
            switch {
            case st != nil:
               urls, err = expandSegmentTemplate(st, rep.ID, repBase, periodDurSec)
            case sl != nil:
               urls, err = expandSegmentList(sl, rep.ID, repBase, periodDurSec)
            default:
               // requirement #5 – single file from effective BaseURL
               u, e := url.Parse(repBase)
               if e != nil {
                  return nil, e
               }
               urls = []string{u.String()}
            }
            if err != nil {
               return nil, fmt.Errorf("representation %s: %w", rep.ID, err)
            }

            // requirement #8 – deduplicate
            seen := make(urlSet)
            for _, u := range urls {
               if _, ok := seen[u]; ok {
                  continue
               }
               seen[u] = struct{}{}
               out[rep.ID] = append(out[rep.ID], u)
            }
         }
      }
   }
   return out, nil
}

// ---------------------------------------------------------------------------
// Template / List expansion
// ---------------------------------------------------------------------------

func expandSegmentTemplate(st *SegmentTemplate, repID string, baseURL string, periodDurSec float64) ([]string, error) {
   timescale := 1
   if st.Timescale != 0 {
      timescale = st.Timescale
   }
   mediaTpl := st.Media
   if mediaTpl == "" {
      return nil, errors.New("SegmentTemplate missing @media")
   }

   startNum := 1
   if st.StartNum != nil {
      startNum = *st.StartNum
   }

   // Build list of (number, time) pairs
   segPairs, err := buildSegmentPairs(st, periodDurSec, timescale, startNum)
   if err != nil {
      return nil, err
   }

   var urls []string
   for _, pair := range segPairs {
      url := substituteTemplate(mediaTpl, repID, pair.num, pair.time)
      u, e := resolveURL(baseURL, url)
      if e != nil {
         return nil, e
      }
      urls = append(urls, u)
   }
   return urls, nil
}

func expandSegmentList(sl *SegmentList, repID string, baseURL string, periodDurSec float64) ([]string, error) {
   // If <SegmentURL> elements exist, use them directly.
   if len(sl.SegmentURLs) > 0 {
      var urls []string
      for _, su := range sl.SegmentURLs {
         u, e := resolveURL(baseURL, su.MediaURL)
         if e != nil {
            return nil, e
         }
         urls = append(urls, u)
      }
      return urls, nil
   }

   // Otherwise fall back to template-style expansion (no @media attr defined).
   timescale := 1
   if sl.Timescale != 0 {
      timescale = sl.Timescale
   }
   startNum := 1
   if sl.StartNum != nil {
      startNum = *sl.StartNum
   }

   segPairs, err := buildSegmentPairsForList(sl, periodDurSec, timescale, startNum)
   if err != nil {
      return nil, err
   }

   // Fabricate template "$Number$"
   media := "$Number$"
   var urls []string
   for _, pair := range segPairs {
      url := substituteTemplate(media, repID, pair.num, pair.time)
      u, e := resolveURL(baseURL, url)
      if e != nil {
         return nil, e
      }
      urls = append(urls, u)
   }
   return urls, nil
}

type segPair struct {
   num  int
   time int64
}

func buildSegmentPairs(st *SegmentTemplate, periodDurSec float64, timescale int, startNum int) ([]segPair, error) {
   // Timeline mode?
   if st.Timeline != nil && len(st.Timeline.S) > 0 {
      var pairs []segPair
      time := int64(0)
      num := startNum
      for _, s := range st.Timeline.S {
         if s.T != 0 {
            time = int64(s.T)
         }
         reps := 1
         if s.R > 0 {
            reps = s.R + 1
         }
         for i := 0; i < reps; i++ {
            pairs = append(pairs, segPair{num: num, time: time})
            time += int64(s.D)
            num++
         }
      }
      return pairs, nil
   }

   // Simple @duration/@endNumber mode
   duration := st.Duration
   if duration == 0 {
      return nil, errors.New("SegmentTemplate missing @duration and no timeline")
   }

   endNum := 0
   if st.EndNum != nil {
      endNum = *st.EndNum
   } else {
      // Compute from period duration
      if periodDurSec <= 0 {
         return nil, errors.New("period duration missing and no @endNumber")
      }
      segDur := float64(duration) / float64(timescale)
      endNum = startNum + int(math.Ceil(periodDurSec/segDur)) - 1
   }

   var pairs []segPair
   time := int64(0)
   for n := startNum; n <= endNum; n++ {
      pairs = append(pairs, segPair{num: n, time: time})
      time += int64(duration)
   }
   return pairs, nil
}

func buildSegmentPairsForList(sl *SegmentList, periodDurSec float64, timescale int, startNum int) ([]segPair, error) {
   if sl.Timeline != nil && len(sl.Timeline.S) > 0 {
      var pairs []segPair
      time := int64(0)
      num := startNum
      for _, s := range sl.Timeline.S {
         if s.T != 0 {
            time = int64(s.T)
         }
         reps := 1
         if s.R > 0 {
            reps = s.R + 1
         }
         for i := 0; i < reps; i++ {
            pairs = append(pairs, segPair{num: num, time: time})
            time += int64(s.D)
            num++
         }
      }
      return pairs, nil
   }

   duration := sl.Duration
   if duration == 0 {
      return nil, errors.New("SegmentList missing @duration and no timeline")
   }

   endNum := 0
   if sl.EndNum != nil {
      endNum = *sl.EndNum
   } else {
      if periodDurSec <= 0 {
         return nil, errors.New("period duration missing and no @endNumber")
      }
      segDur := float64(duration) / float64(timescale)
      endNum = startNum + int(math.Ceil(periodDurSec/segDur)) - 1
   }

   var pairs []segPair
   time := int64(0)
   for n := startNum; n <= endNum; n++ {
      pairs = append(pairs, segPair{num: n, time: time})
      time += int64(duration)
   }
   return pairs, nil
}

// ---------------------------------------------------------------------------
// URL helpers
// ---------------------------------------------------------------------------

func resolveBase(bases ...string) (string, error) {
   base := ""
   for _, b := range bases {
      if b == "" {
         continue
      }
      u, e := url.Parse(strings.TrimSpace(b))
      if e != nil {
         return "", e
      }
      if base == "" {
         base = b
         continue
      }
      baseURL, e := url.Parse(base)
      if e != nil {
         return "", e
      }
      resolved := baseURL.ResolveReference(u)
      base = resolved.String()
   }
   return base, nil
}

func resolveURL(base string, rel string) (string, error) {
   baseURL, e := url.Parse(base)
   if e != nil {
      return "", e
   }
   relURL, e := url.Parse(rel)
   if e != nil {
      return "", e
   }
   return baseURL.ResolveReference(relURL).String(), nil
}

// ---------------------------------------------------------------------------
// Template substitution
// ---------------------------------------------------------------------------

var (
   repIDRe = regexp.MustCompile(`\$RepresentationID\$`)
   numRe   = regexp.MustCompile(`\$Number(?:%0(\d+)d)?\$`)
   timeRe  = regexp.MustCompile(`\$Time(?:%0(\d+)d)?\$`)
)

func substituteTemplate(tpl string, repID string, num int, time int64) string {
   s := repIDRe.ReplaceAllString(tpl, repID)

   s = numRe.ReplaceAllStringFunc(s, func(match string) string {
      sub := numRe.FindStringSubmatch(match)
      if len(sub) < 2 || sub[1] == "" {
         return strconv.Itoa(num)
      }
      width, e := strconv.Atoi(sub[1])
      if e != nil {
         return strconv.Itoa(num)
      }
      return fmt.Sprintf("%0*d", width, num)
   })

   s = timeRe.ReplaceAllStringFunc(s, func(match string) string {
      sub := timeRe.FindStringSubmatch(match)
      if len(sub) < 2 || sub[1] == "" {
         return strconv.FormatInt(time, 10)
      }
      width, e := strconv.Atoi(sub[1])
      if e != nil {
         return strconv.FormatInt(time, 10)
      }
      return fmt.Sprintf("%0*d", width, time)
   })
   return s
}

// ---------------------------------------------------------------------------
// ISO-8601 duration parser
// ---------------------------------------------------------------------------

func parseISODuration(s string) (time.Duration, error) {
   if s == "" {
      return 0, errors.New("empty duration")
   }
   s = strings.TrimPrefix(s, "P")
   if s == "" {
      return 0, errors.New("malformed duration")
   }

   tIndex := strings.Index(s, "T")
   var datePart, timePart string
   if tIndex >= 0 {
      datePart = s[:tIndex]
      timePart = s[tIndex+1:]
   } else {
      datePart = s
   }

   var total float64
   parsePart := func(part string, units map[rune]float64) error {
      i := 0
      for i < len(part) {
         j := i
         for j < len(part) && ((part[j] >= '0' && part[j] <= '9') || part[j] == '.') {
            j++
         }
         if j == i {
            return errors.New("bad duration format")
         }
         val, e := strconv.ParseFloat(part[i:j], 64)
         if e != nil {
            return e
         }
         if j >= len(part) {
            return errors.New("missing unit")
         }
         unit := rune(part[j])
         mul, ok := units[unit]
         if !ok {
            return fmt.Errorf("invalid unit %c", unit)
         }
         total += val * mul
         i = j + 1
      }
      return nil
   }

   dateUnits := map[rune]float64{'D': 24 * 3600}
   timeUnits := map[rune]float64{'H': 3600, 'M': 60, 'S': 1}

   if e := parsePart(datePart, dateUnits); e != nil {
      return 0, e
   }
   if e := parsePart(timePart, timeUnits); e != nil {
      return 0, e
   }
   return time.Duration(total * float64(time.Second)), nil
}
