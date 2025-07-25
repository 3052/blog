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

/* ---------- DASH structs ---------- */
type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   Media   string   `xml:"mediaPresentationDuration,attr"`
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   Duration string          `xml:"duration,attr"`
   BaseURL  string          `xml:"BaseURL"`
   Adapt    []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Reps            []Representation `xml:"Representation"`
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
   StartNum  *int             `xml:"startNumber,attr"`
   EndNum    *int             `xml:"endNumber,attr"`
   Timeline  *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentList struct {
   Duration  int              `xml:"duration,attr"`
   Timescale int              `xml:"timescale,attr"`
   StartNum  *int             `xml:"startNumber,attr"`
   EndNum    *int             `xml:"endNumber,attr"`
   Timeline  *SegmentTimeline `xml:"SegmentTimeline"`
   URLs      []SegmentURL     `xml:"SegmentURL"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T int `xml:"t,attr"`
   D int `xml:"d,attr"`
   R int `xml:"r,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

/* ---------- main ---------- */
func main() {
   if len(os.Args) < 2 {
      fmt.Fprintf(os.Stderr, "usage: %s <local.mpd>\n", os.Args[0])
      os.Exit(1)
   }

   f, err := os.Open(os.Args[1])
   if err != nil {
      fmt.Fprintf(os.Stderr, "open: %v\n", err)
      os.Exit(1)
   }
   defer f.Close()

   var mpd MPD
   if err := xml.NewDecoder(f).Decode(&mpd); err != nil {
      fmt.Fprintf(os.Stderr, "decode: %v\n", err)
      os.Exit(1)
   }

   const origin = "http://test.test/test.mpd"
   out := make(map[string][]string)

   for _, p := range mpd.Periods {
      pbase := resolve(origin, mpd.BaseURL, p.BaseURL)
      for _, a := range p.Adapt {
         abase := resolve(pbase, a.BaseURL)
         for _, r := range a.Reps {
            rbase := resolve(abase, r.BaseURL)

            var tmpl *SegmentTemplate
            var sl *SegmentList
            switch {
            case r.SegmentTemplate != nil:
               tmpl = r.SegmentTemplate
            case a.SegmentTemplate != nil:
               tmpl = a.SegmentTemplate
            case r.SegmentList != nil:
               sl = r.SegmentList
            case a.SegmentList != nil:
               sl = a.SegmentList
            }

            var urls []string
            switch {
            case tmpl != nil:
               urls, err = expandTemplate(rbase, r.ID, tmpl, parseDuration(p.Duration))
            case sl != nil:
               urls, err = expandSegmentList(rbase, sl, parseDuration(p.Duration))
            default:
               urls = []string{rbase}
            }
            if err != nil {
               fmt.Fprintf(os.Stderr, "expand: %v\n", err)
               os.Exit(1)
            }

            seen := map[string]bool{}
            for _, u := range out[r.ID] {
               seen[u] = true
            }
            for _, u := range urls {
               if !seen[u] {
                  out[r.ID] = append(out[r.ID], u)
                  seen[u] = true
               }
            }
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetEscapeHTML(false)
   if err := enc.Encode(out); err != nil {
      fmt.Fprintf(os.Stderr, "json: %v\n", err)
      os.Exit(1)
   }
}

/* ---------- helpers ---------- */
func resolve(base string, rels ...string) string {
   u, err := url.Parse(base)
   if err != nil {
      panic(err)
   }
   for _, rel := range rels {
      if rel == "" {
         continue
      }
      r, err := url.Parse(rel)
      if err != nil {
         panic(err)
      }
      u = u.ResolveReference(r)
   }
   return u.String()
}

func expandSegmentList(base string, sl *SegmentList, dur time.Duration) ([]string, error) {
   if len(sl.URLs) > 0 {
      var urls []string
      for _, u := range sl.URLs {
         urls = append(urls, resolve(base, u.Media))
      }
      return urls, nil
   }

   ts := 1
   if sl.Timescale > 0 {
      ts = sl.Timescale
   }

   start := 1
   if sl.StartNum != nil {
      start = *sl.StartNum
   }

   var (
      numbers []int
      times   []int
   )

   if sl.Timeline != nil {
      var t int64
      for _, s := range sl.Timeline.S {
         if s.T != 0 {
            t = int64(s.T)
         }
         repeat := s.R + 1
         for i := 0; i < repeat; i++ {
            numbers = append(numbers, start+len(numbers))
            times = append(times, int(t))
            t += int64(s.D)
         }
      }
   } else {
      end := sl.EndNum
      if end == nil {
         if dur == 0 || sl.Duration == 0 {
            end = intPtr(start)
         } else {
            total := int(math.Ceil(dur.Seconds() * float64(ts) / float64(sl.Duration)))
            end = intPtr(start + total - 1)
         }
      }
      for n := start; n <= *end; n++ {
         numbers = append(numbers, n)
         times = append(times, (n-start)*sl.Duration)
      }
   }

   var urls []string
   for i := range numbers {
      u, err := sub(base, "", numbers[i], times[i], "$Number$")
      if err != nil {
         return nil, err
      }
      urls = append(urls, resolve(base, u))
   }
   return urls, nil
}

func sub(base, id string, number, timeVal int, media string) (string, error) {
   // 1. $RepresentationID$
   media = strings.ReplaceAll(media, "$RepresentationID$", id)

   // 2. $Number%0Nd$ → zero-padded number
   reN := regexp.MustCompile(`\$Number%0(\d+)d\$`)
   media = reN.ReplaceAllStringFunc(media, func(m string) string {
      w, err := strconv.Atoi(reN.FindStringSubmatch(m)[1])
      if err != nil {
         panic(fmt.Sprintf("invalid width %q: %v", reN.FindStringSubmatch(m)[1], err))
      }
      return fmt.Sprintf("%0*d", w, number)
   })

   // 3. $Time%0Nd$ → zero-padded time
   reT := regexp.MustCompile(`\$Time%0(\d+)d\$`)
   media = reT.ReplaceAllStringFunc(media, func(m string) string {
      w, err := strconv.Atoi(reT.FindStringSubmatch(m)[1])
      if err != nil {
         panic(fmt.Sprintf("invalid width %q: %v", reT.FindStringSubmatch(m)[1], err))
      }
      return fmt.Sprintf("%0*d", w, timeVal)
   })

   // 4. plain tokens
   media = strings.ReplaceAll(media, "$Number$", strconv.Itoa(number))
   media = strings.ReplaceAll(media, "$Time$", strconv.Itoa(timeVal))

   // 5. resolve absolute URL
   u, err := url.Parse(base)
   if err != nil {
      return "", err
   }
   rel, err := url.Parse(media)
   if err != nil {
      return "", err
   }
   return u.ResolveReference(rel).String(), nil
}

func intPtr(i int) *int { return &i }

func parseDuration(d string) time.Duration {
   if d == "" {
      return 0
   }
   d = strings.ToUpper(strings.TrimSpace(d))
   if !strings.HasPrefix(d, "PT") {
      return 0
   }
   d = d[2:]

   total := 0.0
   for d != "" {
      var val float64
      var unit string
      n, err := fmt.Sscanf(d, "%f%s", &val, &unit)
      if err != nil || n != 2 {
         break
      }
      switch {
      case strings.HasPrefix(unit, "H"):
         total += val * 3600
      case strings.HasPrefix(unit, "M"):
         total += val * 60
      case strings.HasPrefix(unit, "S"):
         total += val
      }
      i := 0
      for i < len(d) && (d[i] == '.' || (d[i] >= '0' && d[i] <= '9')) {
         i++
      }
      if i < len(d) {
         i += len(unit)
      }
      d = d[i:]
   }
   return time.Duration(total * float64(time.Second))
}

// expandTemplate snippet (corrected)
func expandTemplate(base, id string, st *SegmentTemplate, dur time.Duration) ([]string, error) {
   media := st.Media
   if media == "" {
      return []string{base}, nil
   }

   ts := 1
   if st.Timescale > 0 {
      ts = st.Timescale
   }
   start := 1
   if st.StartNum != nil {
      start = *st.StartNum
   }

   var numbers, times []int
   if st.Timeline != nil {
      t := int64(0)
      for _, s := range st.Timeline.S {
         if s.T != 0 {
            t = int64(s.T)
         }
         repeat := s.R + 1
         for i := 0; i < repeat; i++ {
            numbers = append(numbers, start+len(numbers))
            times = append(times, int(t))
            t += int64(s.D)
         }
      }
   } else {
      end := st.EndNum
      if end == nil {
         if dur == 0 || st.Duration == 0 {
            end = intPtr(start)
         } else {
            total := int(math.Ceil(dur.Seconds() * float64(ts) / float64(st.Duration)))
            end = intPtr(start + total - 1)
         }
      }
      for n := start; n <= *end; n++ {
         numbers = append(numbers, n)
         times = append(times, (n-start)*st.Duration)
      }
   }

   var urls []string
   for i := range numbers {
      u, err := sub(base, id, numbers[i], times[i], media)
      if err != nil {
         return nil, err
      }
      urls = append(urls, u)
   }
   return urls, nil
}
