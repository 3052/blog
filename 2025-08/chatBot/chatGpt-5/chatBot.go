package main

import (
   "encoding/json"
   "encoding/xml"
   "errors"
   "fmt"
   "io"
   "math"
   "net/url"
   "os"
   "strconv"
   "strings"
   "time"
)

import "regexp"

var placeholderRe = regexp.MustCompile(`\$(RepresentationID|Bandwidth|Number|Time)(%0?\d*d)?\$`)

func applyTemplate(tmpl string, v templateVars) string {
   return placeholderRe.ReplaceAllStringFunc(tmpl, func(m string) string {
      matches := placeholderRe.FindStringSubmatch(m)
      if len(matches) < 2 {
         return m
      }
      name := matches[1]
      format := matches[2] // e.g., "%08d"

      var val interface{}
      switch name {
      case "RepresentationID":
         // Ignore formatting for string
         return v.RepresentationID
      case "Bandwidth":
         val = v.Bandwidth
      case "Number":
         val = v.Number
      case "Time":
         val = v.Time
      }

      if format != "" {
         // Use Sprintf to respect width/padding
         return fmt.Sprintf(format, val)
      }
      // No formatting specifier — default integer/string conversion
      switch x := val.(type) {
      case int:
         return strconv.Itoa(x)
      case int64:
         return strconv.FormatInt(x, 10)
      default:
         return fmt.Sprint(x)
      }
   })
}

const fixedBase = "http://test.test/test.mpd"

type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   BaseURL                   []string `xml:"BaseURL"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
   Periods                   []Period `xml:"Period"`
}

type Period struct {
   BaseURL  []string        `xml:"BaseURL"`
   Duration string          `xml:"duration,attr"`
   AS       []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         []string         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Reps            []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   Bandwidth       int              `xml:"bandwidth,attr"`
   BaseURL         []string         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
   Initialization string           `xml:"initialization,attr"`
   Media          string           `xml:"media,attr"`
   Duration       int64            `xml:"duration,attr"`
   Timescale      int64            `xml:"timescale,attr"`
   StartNumber    int64            `xml:"startNumber,attr"`
   EndNumber      int64            `xml:"endNumber,attr"` // non-standard but requested preference
   Timeline       *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T *int64 `xml:"t,attr"`
   D int64  `xml:"d,attr"`
   R *int64 `xml:"r,attr"`
}

type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
   Duration       int64           `xml:"duration,attr"`
   Timescale      int64           `xml:"timescale,attr"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintln(os.Stderr, "usage: go run main.go <mpd_file_path>")
      os.Exit(2)
   }
   path := os.Args[1]
   f, err := os.Open(path)
   if err != nil {
      fail(err)
   }
   defer f.Close()
   mpdBytes, err := io.ReadAll(f)
   if err != nil {
      fail(err)
   }

   var mpd MPD
   if err := xml.Unmarshal(mpdBytes, &mpd); err != nil {
      fail(fmt.Errorf("failed to parse MPD XML: %w", err))
   }

   base, err := url.Parse(fixedBase)
   if err != nil {
      fail(err)
   }

   result := make(map[string][]string)

   mpdBase := resolveBase(base, firstOrEmpty(mpd.BaseURL))
   mpdOrTotalDurSec := pickFirstDurationSec(mpd.MediaPresentationDuration, "")

   for _, period := range mpd.Periods {
      periodBase := resolveBase(mpdBase, firstOrEmpty(period.BaseURL))
      periodDurSec := pickFirstDurationSec(period.Duration, mpd.MediaPresentationDuration)
      if periodDurSec == 0 && mpdOrTotalDurSec > 0 {
         periodDurSec = mpdOrTotalDurSec // fallback to MPD duration if Period lacks one
      }

      for _, as := range period.AS {
         asBase := resolveBase(periodBase, firstOrEmpty(as.BaseURL))
         for _, rep := range as.Reps {
            repBase := resolveBase(asBase, firstOrEmpty(rep.BaseURL))
            segList := chooseSegList(rep.SegmentList, as.SegmentList)
            segTmpl := chooseSegTmpl(rep.SegmentTemplate, as.SegmentTemplate)

            urls, err := buildRepURLs(rep, repBase, segList, segTmpl, periodDurSec)
            if err != nil {
               // Skip problematic representations but continue others.
               continue
            }
            if existing, ok := result[rep.ID]; ok {
               result[rep.ID] = append(existing, urls...)
            } else {
               result[rep.ID] = urls
            }
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   if err := enc.Encode(result); err != nil {
      fail(err)
   }
}

func fail(err error) {
   fmt.Fprintln(os.Stderr, err)
   os.Exit(1)
}

func firstOrEmpty(ss []string) string {
   if len(ss) == 0 {
      return ""
   }
   return strings.TrimSpace(ss[0])
}

func resolveBase(parent *url.URL, child string) *url.URL {
   if child == "" {
      return parent
   }
   u, err := url.Parse(child)
   if err != nil {
      return parent
   }
   return parent.ResolveReference(u)
}

func chooseSegList(rep, as *SegmentList) *SegmentList {
   if rep != nil {
      return rep
   }
   return as
}

func chooseSegTmpl(rep, as *SegmentTemplate) *SegmentTemplate {
   if rep != nil {
      return rep
   }
   return as
}

func resolveToString(base *url.URL, ref string) string {
   r, err := url.Parse(strings.TrimSpace(ref))
   if err != nil {
      return base.String()
   }
   return base.ResolveReference(r).String()
}

// ----- Template handling -----

type templateVars struct {
   RepresentationID string
   Bandwidth        int
   Number           int64
   Time             int64
}

func itoa(i int) string     { return strconv.Itoa(i) }
func itoa64(i int64) string { return strconv.FormatInt(i, 10) }

// ----- Duration parsing -----

// pickFirstDurationSec returns seconds from the first non-empty ISO8601 duration string.
func pickFirstDurationSec(a, b string) float64 {
   if strings.TrimSpace(a) != "" {
      if sec, ok := parseISODurationToSeconds(a); ok {
         return sec
      }
   }
   if strings.TrimSpace(b) != "" {
      if sec, ok := parseISODurationToSeconds(b); ok {
         return sec
      }
   }
   return 0
}

// Supports patterns like PT#S, PT#M#S, PT#H#M#S, P#DT#H#M#S (days too).
func parseISODurationToSeconds(s string) (float64, bool) {
   // Very small, permissive parser
   if s == "" || s[0] != 'P' {
      return 0, false
   }
   // Split into date and time parts (P ... T ...)
   p := strings.TrimPrefix(s, "P")
   var days, hours, minutes float64
   var seconds float64

   // If there's a 'T', split date/time
   datePart := p
   timePart := ""
   if tIdx := strings.Index(p, "T"); tIdx >= 0 {
      datePart = p[:tIdx]
      timePart = p[tIdx+1:]
   }

   // Parse datePart for days (D)
   if datePart != "" {
      if idx := strings.Index(datePart, "D"); idx >= 0 {
         val := datePart[:idx]
         if f, err := strconv.ParseFloat(val, 64); err == nil {
            days = f
         }
         datePart = datePart[idx+1:]
      }
      // We ignore years/months because mapping to seconds is calendar-dependent; uncommon in MPDs.
   }

   // Parse timePart for H, M, S in order
   rest := timePart
   consume := func(unit byte) (float64, string) {
      if rest == "" {
         return 0, rest
      }
      idx := strings.IndexByte(rest, unit)
      if idx < 0 {
         return 0, rest
      }
      val := rest[:idx]
      rest2 := ""
      if idx+1 < len(rest) {
         rest2 = rest[idx+1:]
      }
      if f, err := strconv.ParseFloat(val, 64); err == nil {
         return f, rest2
      }
      return 0, rest2
   }
   if strings.ContainsAny(rest, "H") {
      hours, rest = consume('H')
   }
   if strings.ContainsAny(rest, "M") {
      minutes, rest = consume('M')
   }
   if strings.ContainsAny(rest, "S") {
      seconds, rest = consume('S')
      _ = rest
   }

   total := days*24*3600 + hours*3600 + minutes*60 + seconds
   return total, true
}

// (Optional) utility to parse wall-clock durations like "PT30S" into time.Duration if needed.
func parseISODurationToDuration(s string) (time.Duration, bool) {
   sec, ok := parseISODurationToSeconds(s)
   if !ok {
      return 0, false
   }
   return time.Duration(sec * float64(time.Second)), true
}

func buildRepURLs(rep Representation, base *url.URL, sl *SegmentList, st *SegmentTemplate, periodDurSec float64) ([]string, error) {
   var out []string

   // SegmentList wins over SegmentTemplate if present
   if sl != nil && len(sl.SegmentURLs) > 0 {
      // Initialization (optional)
      if sl.Initialization != nil && strings.TrimSpace(sl.Initialization.SourceURL) != "" {
         out = append(out, resolveToString(base, sl.Initialization.SourceURL))
      }
      for _, su := range sl.SegmentURLs {
         if strings.TrimSpace(su.Media) == "" {
            continue
         }
         out = append(out, resolveToString(base, su.Media))
      }
      return out, nil
   }

   // NEW FALLBACK: Only BaseURL, no segment info
   if (sl == nil || len(sl.SegmentURLs) == 0) && (st == nil || (strings.TrimSpace(st.Media) == "" && strings.TrimSpace(st.Initialization) == "")) {
      return []string{base.String()}, nil
   }

   if st == nil || strings.TrimSpace(st.Media) == "" {
      return nil, errors.New("no SegmentList or usable SegmentTemplate")
   }

   // Defaults per DASH
   timescale := st.Timescale
   if timescale == 0 {
      timescale = 1
   }
   startNumber := st.StartNumber
   if startNumber == 0 {
      startNumber = 1
   }

   // Initialization via template (optional)
   if strings.TrimSpace(st.Initialization) != "" {
      initURL := applyTemplate(st.Initialization, templateVars{
         RepresentationID: rep.ID,
         Bandwidth:        rep.Bandwidth,
         Number:           startNumber,
         Time:             0,
      })
      out = append(out, resolveToString(base, initURL))
   }

   // SegmentTimeline vs. duration-based sequence
   if st.Timeline != nil && len(st.Timeline.S) > 0 {
      var number = startNumber
      var currentTime int64 = 0
      for i, s := range st.Timeline.S {
         repeats := int64(0)
         if s.R != nil {
            repeats = *s.R
         }
         if s.T != nil {
            currentTime = *s.T
         } else if i == 0 && s.T == nil && currentTime == 0 {
            // leave as 0
         }
         count := repeats + 1
         if repeats < 0 {
            if i+1 < len(st.Timeline.S) && st.Timeline.S[i+1].T != nil {
               nextStart := *st.Timeline.S[i+1].T
               if s.D > 0 {
                  diff := nextStart - currentTime
                  if diff > 0 {
                     count = diff / s.D
                     if diff%s.D != 0 {
                        count++
                     }
                  } else {
                     count = 0
                  }
               }
            } else {
               count = 1
            }
         }
         for k := int64(0); k < count; k++ {
            segURL := applyTemplate(st.Media, templateVars{
               RepresentationID: rep.ID,
               Bandwidth:        rep.Bandwidth,
               Number:           number,
               Time:             currentTime,
            })
            out = append(out, resolveToString(base, segURL))
            number++
            currentTime += s.D
         }
      }
      return out, nil
   }

   // No SegmentTimeline — use duration/timescale and period duration
   segDurSec := float64(st.Duration) / float64(timescale)
   if segDurSec <= 0 {
      segDurSec = 0
   }
   var endNumber int64 = 0
   if st.EndNumber > 0 {
      endNumber = st.EndNumber
   }

   var count int64
   switch {
   case endNumber > 0:
      count = (endNumber - startNumber + 1)
   case periodDurSec > 0 && segDurSec > 0:
      count = int64(math.Ceil(periodDurSec / segDurSec))
   default:
      count = 0
   }

   for i := int64(0); i < count; i++ {
      number := startNumber + i
      segURL := applyTemplate(st.Media, templateVars{
         RepresentationID: rep.ID,
         Bandwidth:        rep.Bandwidth,
         Number:           number,
         Time:             0,
      })
      out = append(out, resolveToString(base, segURL))
   }

   return out, nil
}
