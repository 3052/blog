package main

import (
   "encoding/json"
   "encoding/xml"
   "errors"
   "fmt"
   "log"
   "math"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
   "time"
)

const defaultBase = "http://test.test/test.mpd"

type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
   BaseURL                   []string `xml:"BaseURL"`
   Periods                   []Period `xml:"Period"`
}

type Period struct {
   Duration  string          `xml:"duration,attr"`
   BaseURL   []string        `xml:"BaseURL"`
   AdaptSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         []string         `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Reps            []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   Bandwidth       int64            `xml:"bandwidth,attr"`
   BaseURL         []string         `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   URLs           []SegmentURL    `xml:"SegmentURL"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
   Range     string `xml:"range,attr"`
}

type SegmentURL struct {
   Media      string `xml:"media,attr"`
   MediaRange string `xml:"mediaRange,attr"`
   Index      string `xml:"index,attr"`
}

type SegmentTemplate struct {
   Timescale              int64            `xml:"timescale,attr"`
   Duration               int64            `xml:"duration,attr"`
   StartNumberRaw         string           `xml:"startNumber,attr"` // distinguish missing vs "0"
   EndNumber              int64            `xml:"endNumber,attr"`   // inclusive if present (>0)
   Initialization         string           `xml:"initialization,attr"`
   Media                  string           `xml:"media,attr"`
   PresentationTimeOffset int64            `xml:"presentationTimeOffset,attr"`
   SegmentTimelineCont    *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []SegmentTimelineS `xml:"S"`
}

type SegmentTimelineS struct {
   T string `xml:"t,attr"`
   D string `xml:"d,attr"`
   R string `xml:"r,attr"`
}

func main() {
   log.SetFlags(0)
   if len(os.Args) != 2 {
      log.Printf("usage: go run main.go <mpd_file_path>")
      os.Exit(1)
   }
   path := os.Args[1]
   data, err := os.ReadFile(path)
   if err != nil {
      log.Printf("error: failed to read MPD file %q: %v", path, err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      log.Printf("error: failed to parse MPD XML: %v", err)
      os.Exit(1)
   }

   result := make(map[string][]string)

   for pIdx, period := range mpd.Periods {
      for aIdx, as := range period.AdaptSets {
         for rIdx, rep := range as.Reps {
            repID := rep.ID
            if repID == "" {
               repID = fmt.Sprintf("period%d_adaptation%d_representation%d", pIdx, aIdx, rIdx)
            }

            base, repSingle, repResource, err := composeBase(mpd.BaseURL, period.BaseURL, as.BaseURL, rep.BaseURL)
            if err != nil {
               log.Printf("error: representation %q: failed to compose BaseURL: %v", rep.ID, err)
               os.Exit(1)
            }

            effList := rep.SegmentList
            if effList == nil {
               effList = as.SegmentList
            }
            effTpl := rep.SegmentTemplate
            if effTpl == nil {
               effTpl = as.SegmentTemplate
            }

            var segments []string
            switch {
            case effList != nil:
               segments, err = expandSegmentList(effList, base)
               if err != nil {
                  log.Printf("error: representation %q: %v", rep.ID, err)
                  os.Exit(1)
               }
            case effTpl != nil:
               totalDur, ok := totalDuration(period.Duration, mpd.MediaPresentationDuration)
               segments, err = expandSegmentTemplate(effTpl, base, rep, totalDur, ok)
               if err != nil {
                  log.Printf("error: representation %q: %v", rep.ID, err)
                  os.Exit(1)
               }
            case repSingle && repResource != nil:
               segments = []string{repResource.String()}
            default:
               log.Printf("error: representation %q has neither SegmentList nor SegmentTemplate, and no single-resource BaseURL.", rep.ID)
               os.Exit(1)
            }

            result[repID] = append(result[repID], segments...)
         }
      }
   }

   enc, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      log.Printf("error: failed to encode JSON output: %v", err)
      os.Exit(1)
   }
   fmt.Println(string(enc))
}

// -------- BaseURL handling --------

func firstNonEmpty(ss []string) string {
   for _, s := range ss {
      if strings.TrimSpace(s) != "" {
         return strings.TrimSpace(s)
      }
   }
   return ""
}

func composeBase(mpdBase, periodBase, adaptBase, repBase []string) (*url.URL, bool, *url.URL, error) {
   base, err := url.Parse(defaultBase)
   if err != nil {
      return nil, false, nil, fmt.Errorf("invalid defaultBase: %v", err)
   }
   apply := func(b *url.URL, s string) (*url.URL, error) {
      u, err := url.Parse(s)
      if err != nil {
         return nil, fmt.Errorf("invalid BaseURL %q: %w", s, err)
      }
      return b.ResolveReference(u), nil
   }
   if v := firstNonEmpty(mpdBase); v != "" {
      base, err = apply(base, v)
      if err != nil {
         return nil, false, nil, err
      }
   }
   if v := firstNonEmpty(periodBase); v != "" {
      base, err = apply(base, v)
      if err != nil {
         return nil, false, nil, err
      }
   }
   if v := firstNonEmpty(adaptBase); v != "" {
      base, err = apply(base, v)
      if err != nil {
         return nil, false, nil, err
      }
   }

   var repSingle bool
   var repResource *url.URL
   if v := firstNonEmpty(repBase); v != "" {
      if !strings.HasSuffix(v, "/") {
         repSingle = true
         u, err := url.Parse(v)
         if err != nil {
            return nil, false, nil, fmt.Errorf("invalid Representation BaseURL %q: %w", v, err)
         }
         repResource = base.ResolveReference(u)
      } else {
         base, err = apply(base, v)
         if err != nil {
            return nil, false, nil, err
         }
      }
   }
   return base, repSingle, repResource, nil
}

// -------- SegmentList --------

func expandSegmentList(sl *SegmentList, base *url.URL) ([]string, error) {
   var out []string
   if sl.Initialization != nil && strings.TrimSpace(sl.Initialization.SourceURL) != "" {
      out = append(out, mustResolve(base, sl.Initialization.SourceURL))
   }
   for i, su := range sl.URLs {
      if strings.TrimSpace(su.Media) == "" {
         return nil, fmt.Errorf("SegmentList SegmentURL[%d] missing @media", i)
      }
      out = append(out, mustResolve(base, su.Media))
   }
   return out, nil
}

// -------- SegmentTemplate --------

func expandSegmentTemplate(st *SegmentTemplate, base *url.URL, rep Representation, total time.Duration, hasTotal bool) ([]string, error) {
   var out []string

   timescale := st.Timescale
   if timescale <= 0 {
      timescale = 1
   }

   // startNumber: default 1 when missing; if explicitly "0", use 0
   startNumberVal, present, err := parseOptionalInt64(st.StartNumberRaw)
   if err != nil {
      return nil, fmt.Errorf("invalid SegmentTemplate@startNumber=%q: %v", st.StartNumberRaw, err)
   }
   var startNumber int64
   if present {
      startNumber = startNumberVal
   } else {
      startNumber = 1
   }

   endNumber := st.EndNumber // inclusive if > 0
   pto := st.PresentationTimeOffset

   if strings.TrimSpace(st.Initialization) != "" {
      uri := applyTemplate(st.Initialization, rep, nil, nil)
      out = append(out, mustResolve(base, uri))
   }

   mediaTpl := strings.TrimSpace(st.Media)
   if mediaTpl == "" {
      return out, nil
   }

   hasTimeToken := templateHasName(mediaTpl, "Time")
   if st.SegmentTimelineCont != nil {
      times, err := buildTimeline(st.SegmentTimelineCont, timescale, pto, total, hasTotal)
      if err != nil {
         return nil, err
      }
      for i, t := range times {
         num := startNumber + int64(i)
         if endNumber > 0 && num > endNumber {
            break // endNumber is inclusive
         }
         var tPtr *int64
         if hasTimeToken {
            tt := t
            tPtr = &tt
         }
         uri := applyTemplate(mediaTpl, rep, &num, tPtr)
         out = append(out, mustResolve(base, uri))
      }
   } else {
      if hasTimeToken {
         return nil, errors.New("SegmentTemplate@media uses $Time$ but no SegmentTimeline is present")
      }
      var count int
      if endNumber > 0 {
         if endNumber < startNumber {
            return nil, fmt.Errorf("invalid numbering: endNumber (%d) < startNumber (%d)", endNumber, startNumber)
         }
         count = int(endNumber-startNumber) + 1
      } else {
         segDurUnits := st.Duration
         if segDurUnits <= 0 {
            return nil, errors.New("SegmentTemplate without SegmentTimeline requires positive @duration (or an inclusive @endNumber)")
         }
         if !hasTotal {
            return nil, errors.New("cannot expand number-based template: Period/MPD duration is unknown (and no @endNumber provided)")
         }
         boundUnits := durationToUnits(total, timescale)
         if boundUnits <= 0 {
            return nil, errors.New("invalid total duration for number-based expansion")
         }
         // Ceil division for segment count: ceil(boundUnits / segDurUnits)
         count64 := int64(math.Ceil(boundUnits / float64(segDurUnits)))
         if count64 <= 0 {
            return nil, fmt.Errorf("computed zero segments (bound=%.0f, duration=%d)", boundUnits, segDurUnits)
         }
         count = int(count64)
      }

      for i := 0; i < count; i++ {
         num := startNumber + int64(i)
         uri := applyTemplate(mediaTpl, rep, &num, nil)
         out = append(out, mustResolve(base, uri))
      }
   }

   return out, nil
}

func durationToUnits(d time.Duration, timescale int64) float64 {
   ns := d.Nanoseconds()
   if ns <= 0 {
      return 0
   }
   return (float64(timescale) * float64(ns)) / 1_000_000_000.0
}

// -------- Template token detection/replacement (with formatting) --------

var tokenRe = regexp.MustCompile(`\$(RepresentationID|Bandwidth|Number|Time)(%[^$]+)?\$`)

func templateHasName(tpl, name string) bool {
   t := strings.ReplaceAll(tpl, "$$", "\x00")
   ms := tokenRe.FindAllStringSubmatch(t, -1)
   for _, m := range ms {
      if len(m) >= 2 && m[1] == name {
         return true
      }
   }
   return false
}

func applyTemplate(tpl string, rep Representation, number *int64, timeVal *int64) string {
   // Protect escaped "$$"
   t := strings.ReplaceAll(tpl, "$$", "\x00")

   out := tokenRe.ReplaceAllStringFunc(t, func(m string) string {
      sub := tokenRe.FindStringSubmatch(m)
      if len(sub) < 3 {
         return m
      }
      name := sub[1]
      fmtSpec := sub[2] // includes leading %, e.g., %08d

      switch name {
      case "RepresentationID":
         if fmtSpec == "" {
            return rep.ID
         }
         // Only permit string-like verbs to avoid fmt errors
         last := fmtSpec[len(fmtSpec)-1]
         if last == 's' || last == 'q' || last == 'v' {
            return fmt.Sprintf(fmtSpec, rep.ID)
         }
         return rep.ID

      case "Bandwidth":
         val := rep.Bandwidth
         if fmtSpec == "" {
            return fmt.Sprintf("%d", val)
         }
         return fmt.Sprintf(fmtSpec, val)

      case "Number":
         var val int64
         if number != nil {
            val = *number
         }
         if fmtSpec == "" {
            return fmt.Sprintf("%d", val)
         }
         return fmt.Sprintf(fmtSpec, val)

      case "Time":
         var val int64
         if timeVal != nil {
            val = *timeVal
         }
         if fmtSpec == "" {
            return fmt.Sprintf("%d", val)
         }
         return fmt.Sprintf(fmtSpec, val)
      }
      return m
   })

   // Unescape protected dollars
   return strings.ReplaceAll(out, "\x00", "$")
}

// -------- SegmentTimeline building --------

func buildTimeline(st *SegmentTimeline, timescale int64, pto int64, total time.Duration, hasTotal bool) ([]int64, error) {
   var out []int64
   var cur int64
   var haveCur bool

   var boundUnits float64
   if hasTotal {
      boundUnits = durationToUnits(total, timescale)
   }

   for idx, s := range st.S {
      var (
         tVal int64
         dVal int64
         rVal int64
         err  error
      )
      if strings.TrimSpace(s.D) == "" {
         return nil, fmt.Errorf("SegmentTimeline S[%d] missing @d", idx)
      }
      dVal, err = parseInt64(s.D)
      if err != nil || dVal <= 0 {
         return nil, fmt.Errorf("SegmentTimeline S[%d] has invalid @d=%q", idx, s.D)
      }
      if strings.TrimSpace(s.T) != "" {
         tVal, err = parseInt64(s.T)
         if err != nil {
            return nil, fmt.Errorf("SegmentTimeline S[%d] has invalid @t=%q", idx, s.T)
         }
         cur = tVal - pto
         if cur < 0 {
            cur = 0
         }
         haveCur = true
      } else if !haveCur {
         cur = 0
         haveCur = true
      }
      if strings.TrimSpace(s.R) != "" {
         rVal, err = parseInt64(s.R)
         if err != nil {
            return nil, fmt.Errorf("SegmentTimeline S[%d] has invalid @r=%q", idx, s.R)
         }
      } else {
         rVal = 0
      }

      // First occurrence
      if !hasTotal || float64(cur) < boundUnits {
         out = append(out, cur)
      }

      if rVal == -1 {
         if !hasTotal {
            return nil, errors.New("SegmentTimeline uses R=-1 but total Period/MPD duration is unknown")
         }
         for {
            next := cur + dVal
            if float64(next) >= boundUnits {
               break
            }
            out = append(out, next)
            cur = next
         }
         cur += dVal
         continue
      }

      // Repeat rVal times after the first
      for i := int64(0); i < rVal; i++ {
         next := cur + dVal
         cur = next
         if !hasTotal || float64(next) < boundUnits {
            out = append(out, next)
         }
      }
      cur += dVal
   }
   return out, nil
}

func parseOptionalInt64(s string) (int64, bool, error) {
   s = strings.TrimSpace(s)
   if s == "" {
      return 0, false, nil
   }
   v, err := strconv.ParseInt(s, 10, 64)
   return v, true, err
}

func parseInt64(s string) (int64, error) {
   return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
}

func mustResolve(base *url.URL, rel string) string {
   u, err := url.Parse(rel)
   if err != nil {
      return base.String()
   }
   return base.ResolveReference(u).String()
}

// -------- ISO-8601 duration parsing --------

var rePDTHMS = regexp.MustCompile(`^P(?:(\d+)D)?T?(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?$`)
var rePTHMS = regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?$`)

func parseISODuration(s string) (time.Duration, bool) {
   s = strings.TrimSpace(s)
   if s == "" {
      return 0, false
   }
   if m := rePDTHMS.FindStringSubmatch(s); m != nil {
      return hmsToDuration(m[1], m[2], m[3], m[4]), true
   }
   if m := rePTHMS.FindStringSubmatch(s); m != nil {
      // Ensure we pass "" for days
      return hmsToDuration("", m[1], m[2], m[3]), true
   }
   log.Printf("warning: failed to parse ISO-8601 duration %q; treating as unknown", s)
   return 0, false
}

// hmsToDuration converts days/hours/mins/secs strings to time.Duration.
// secsStr may be fractional. Empty strings are treated as zero.
func hmsToDuration(daysStr, hoursStr, minsStr, secsStr string) time.Duration {
   parseInt := func(s string) int64 {
      if strings.TrimSpace(s) == "" {
         return 0
      }
      v, _ := strconv.ParseInt(s, 10, 64)
      return v
   }
   parseFloat := func(s string) float64 {
      if strings.TrimSpace(s) == "" {
         return 0
      }
      v, _ := strconv.ParseFloat(s, 64)
      return v
   }
   days := parseInt(daysStr)
   hours := parseInt(hoursStr)
   mins := parseInt(minsStr)
   secs := parseFloat(secsStr)

   total := time.Duration(days) * 24 * time.Hour
   total += time.Duration(hours) * time.Hour
   total += time.Duration(mins) * time.Minute
   total += time.Duration(secs * float64(time.Second))
   return total
}

func totalDuration(periodDurStr, mpdDurStr string) (time.Duration, bool) {
   if d, ok := parseISODuration(periodDurStr); ok {
      return d, true
   }
   if d, ok := parseISODuration(mpdDurStr); ok {
      return d, true
   }
   return 0, false
}
