package main

import (
   "encoding/json"
   "encoding/xml"
   "errors"
   "flag"
   "fmt"
   "io"
   "log"
   "math"
   "net/url"
   "os"
   "path/filepath"
   "regexp"
   "strconv"
   "strings"
   "time"
)

const defaultBase = "http://test.test/test.mpd"

func main() {
   log.SetFlags(0) // cleaner stderr

   flag.Usage = func() {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", filepath.Base(os.Args[0]))
   }
   flag.Parse()

   if flag.NArg() != 1 {
      flag.Usage()
      os.Exit(2)
   }

   filePath := flag.Arg(0)
   f, err := os.Open(filePath)
   if err != nil {
      log.Fatalf("error: opening MPD file: %v", err)
   }
   defer f.Close()

   mpd, err := parseMPD(f)
   if err != nil {
      log.Fatalf("error: parsing MPD: %v", err)
   }

   out, err := buildAllSegments(mpd)
   if err != nil {
      log.Fatalf("error: building segments: %v", err)
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   if err := enc.Encode(out); err != nil {
      log.Fatalf("error: encoding JSON: %v", err)
   }
}

// ---------- XML models (subset of DASH schema needed for segment expansion) ----------

type MPD struct {
   XMLName                    xml.Name  `xml:"MPD"`
   BaseURL                    []BaseURL `xml:"BaseURL"`
   Periods                    []Period  `xml:"Period"`
   MediaPresentationDuration  string    `xml:"mediaPresentationDuration,attr"`
   Type                       string    `xml:"type,attr"`
   TimeShiftBufferDepth       string    `xml:"timeShiftBufferDepth,attr"`
   SuggestedPresentationDelay string    `xml:"suggestedPresentationDelay,attr"`
}

type BaseURL struct {
   Value string `xml:",chardata"`
}

type Period struct {
   BaseURL  []BaseURL       `xml:"BaseURL"`
   Duration string          `xml:"duration,attr"`
   AS       []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         []BaseURL        `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Reps            []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   Bandwidth       int64            `xml:"bandwidth,attr"`
   BaseURL         []BaseURL        `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentList struct {
   Timescale      int64           `xml:"timescale,attr"`
   Duration       int64           `xml:"duration,attr"`
   Initialization *Initialization `xml:"Initialization"`
   URLs           []SegmentURL    `xml:"SegmentURL"`
}

type SegmentURL struct {
   Media      string `xml:"media,attr"`
   Index      string `xml:"index,attr"`
   MediaRange string `xml:"mediaRange,attr"`
   IndexRange string `xml:"indexRange,attr"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
   Range     string `xml:"range,attr"`
}

type SegmentTemplate struct {
   Timescale              int64            `xml:"timescale,attr"`
   Duration               int64            `xml:"duration,attr"`
   StartNumber            int64            `xml:"startNumber,attr"`
   Initialization         string           `xml:"initialization,attr"`
   Media                  string           `xml:"media,attr"`
   PresentationTimeOffset int64            `xml:"presentationTimeOffset,attr"`
   SegmentTimeline        *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T int64 `xml:"t,attr"`
   D int64 `xml:"d,attr"`
   R int64 `xml:"r,attr"`
}

// ---------- Parsing ----------

func parseMPD(r io.Reader) (*MPD, error) {
   var mpd MPD
   dec := xml.NewDecoder(r)
   dec.Strict = false
   dec.AutoClose = xml.HTMLAutoClose
   dec.Entity = xml.HTMLEntity
   if err := dec.Decode(&mpd); err != nil {
      return nil, err
   }
   return &mpd, nil
}

// ---------- Core logic ----------

type ctx struct {
   base      string
   mpd       *MPD
   period    *Period
   aset      *AdaptationSet
   rep       *Representation
   periodDur time.Duration // 0 if unknown
}

func buildAllSegments(mpd *MPD) (map[string][]string, error) {
   results := make(map[string][]string)

   for pi := range mpd.Periods {
      p := &mpd.Periods[pi]
      periodDur := firstNonZero(parseISODuration(p.Duration), parseISODuration(mpd.MediaPresentationDuration))
      for ai := range p.AS {
         as := &p.AS[ai]
         for ri := range as.Reps {
            r := &as.Reps[ri]
            if strings.TrimSpace(r.ID) == "" {
               log.Printf("warning: skipping Representation without id in Period %d AdaptationSet %d", pi, ai)
               continue
            }
            base, err := effectiveBase(mpd, p, as, r)
            if err != nil {
               return nil, fmt.Errorf("resolving BaseURL for Representation %q: %w", r.ID, err)
            }

            c := ctx{
               base:      base,
               mpd:       mpd,
               period:    p,
               aset:      as,
               rep:       r,
               periodDur: periodDur,
            }

            segments, err := collectSegmentsForRep(c)
            if err != nil {
               return nil, fmt.Errorf("representation %q: %w", r.ID, err)
            }

            results[r.ID] = append(results[r.ID], segments...)
         }
      }
   }

   return results, nil
}

func collectSegmentsForRep(c ctx) ([]string, error) {
   var out []string

   // Prefer Representation-level SegmentList/Template; otherwise inherit from AdaptationSet.
   sl := c.rep.SegmentList
   if sl == nil {
      sl = c.aset.SegmentList
   }
   st := c.rep.SegmentTemplate
   if st == nil {
      st = c.aset.SegmentTemplate
   }

   switch {
   case sl != nil:
      segs, err := expandSegmentList(c, sl)
      if err != nil {
         return nil, err
      }
      out = append(out, segs...)

   case st != nil:
      segs, err := expandSegmentTemplate(c, st)
      if err != nil {
         return nil, err
      }
      out = append(out, segs...)

   default:
      // No explicit segment info; try to treat Representation BaseURL as a single resource (rare).
      if len(c.rep.BaseURL) > 0 {
         u, err := resolveURL(c.base, strings.TrimSpace(c.rep.BaseURL[0].Value))
         if err != nil {
            return nil, fmt.Errorf("resolving Representation BaseURL: %w", err)
         }
         out = append(out, u)
      } else {
         return nil, errors.New("no SegmentList or SegmentTemplate found (and no usable BaseURL) — cannot determine segments")
      }
   }

   return out, nil
}

// ---------- SegmentList expansion ----------

func expandSegmentList(c ctx, sl *SegmentList) ([]string, error) {
   var out []string
   // Initialization first (if present)
   if sl.Initialization != nil && strings.TrimSpace(sl.Initialization.SourceURL) != "" {
      u, err := resolveURL(c.base, sl.Initialization.SourceURL)
      if err != nil {
         return nil, fmt.Errorf("SegmentList initialization: %w", err)
      }
      out = append(out, u)
   }

   for i, su := range sl.URLs {
      if strings.TrimSpace(su.Media) == "" {
         log.Printf("warning: SegmentList SegmentURL[%d] has empty @media — skipping", i)
         continue
      }
      u, err := resolveURL(c.base, su.Media)
      if err != nil {
         return nil, fmt.Errorf("SegmentList SegmentURL[%d]: %w", i, err)
      }
      out = append(out, u)
   }
   return out, nil
}

// ---------- SegmentTemplate expansion ----------

func expandSegmentTemplate(c ctx, st *SegmentTemplate) ([]string, error) {
   var out []string

   timescale := st.Timescale
   if timescale == 0 {
      timescale = 1
   }
   startNum := st.StartNumber
   if startNum == 0 {
      startNum = 1
   }

   // Initialization URL (if provided)
   if strings.TrimSpace(st.Initialization) != "" {
      initURL, err := expandAndResolveTemplate(c, st.Initialization, startNum, 0)
      if err != nil {
         return nil, fmt.Errorf("SegmentTemplate initialization: %w", err)
      }
      out = append(out, initURL)
   }

   media := strings.TrimSpace(st.Media)
   if media == "" {
      return out, nil // No media template means nothing to expand beyond initialization.
   }

   // With SegmentTimeline
   if st.SegmentTimeline != nil && len(st.SegmentTimeline.S) > 0 {
      curT := int64(0)
      num := startNum

      for idx, s := range st.SegmentTimeline.S {
         if s.D <= 0 {
            return nil, fmt.Errorf("SegmentTemplate SegmentTimeline S[%d]: invalid duration %d", idx, s.D)
         }
         t := s.T
         if t == 0 && idx == 0 {
            // If the first S has no t, it starts at 0 (or PTO). We'll honor PTO by simply starting at 0 here,
            // because media templates usually incorporate PTO on the player side. For URL $Time$, we use t directly.
         }
         if t == 0 {
            t = curT
         }
         repeats := s.R
         if repeats < 0 {
            // "repeat until period end" — estimate if we know the period duration.
            if c.periodDur == 0 {
               return nil, fmt.Errorf("SegmentTemplate SegmentTimeline S[%d]: r=-1 but Period/MPD duration unknown", idx)
            }
            periodUnits := int64(float64(c.periodDur) / (float64(time.Second) / float64(timescale)))
            if periodUnits < t {
               return nil, fmt.Errorf("SegmentTemplate SegmentTimeline S[%d]: r=-1 but t (%d) exceeds period length (%d units)", idx, t, periodUnits)
            }
            left := periodUnits - t
            repeats = left/int64(s.D) - 1
            if repeats < 0 {
               repeats = 0
            }
         }

         occ := repeats + 1
         for i := int64(0); i < occ; i++ {
            tt := t + i*int64(s.D)
            u, err := expandAndResolveTemplate(c, media, num, tt)
            if err != nil {
               return nil, fmt.Errorf("SegmentTemplate media expansion: %w", err)
            }
            out = append(out, u)
            num++
         }
         curT = t + occ*int64(s.D)
      }
      return out, nil
   }

   // Number-based, no SegmentTimeline.
   if strings.Contains(media, "$Time$") {
      return nil, errors.New("SegmentTemplate uses $Time$ but no SegmentTimeline is provided")
   }
   if st.Duration <= 0 {
      return nil, errors.New("SegmentTemplate missing/invalid @duration for number-based expansion")
   }
   if c.periodDur == 0 {
      return nil, errors.New("cannot determine segment count: Period/MPD duration unknown")
   }

   segSeconds := float64(st.Duration) / float64(timescale)
   if segSeconds <= 0 {
      return nil, errors.New("computed non-positive segment duration")
   }
   count := int(math.Ceil(float64(c.periodDur) / (segSeconds * float64(time.Second))))
   if count <= 0 {
      return out, nil
   }

   for i := 0; i < count; i++ {
      num := startNum + int64(i)
      tt := int64(i) * st.Duration
      u, err := expandAndResolveTemplate(c, media, num, tt)
      if err != nil {
         return nil, fmt.Errorf("SegmentTemplate media expansion: %w", err)
      }
      out = append(out, u)
   }
   return out, nil
}

func expandAndResolveTemplate(c ctx, template string, number, timeVal int64) (string, error) {
   // Implement $$ escaping then token replacement.
   const esc = "\x00DOLLAR\x00"
   s := strings.ReplaceAll(template, "$$", esc)

   repls := map[string]string{
      "$RepresentationID$": c.rep.ID,
      "$Bandwidth$":        strconv.FormatInt(c.rep.Bandwidth, 10),
      "$Number$":           strconv.FormatInt(number, 10),
      "$Time$":             strconv.FormatInt(timeVal, 10),
   }

   for k, v := range repls {
      if strings.Contains(s, k) {
         s = strings.ReplaceAll(s, k, v)
      }
   }

   s = strings.ReplaceAll(s, esc, "$")
   return resolveURL(c.base, s)
}

// ---------- BaseURL resolution ----------

func effectiveBase(mpd *MPD, p *Period, as *AdaptationSet, r *Representation) (string, error) {
   base := defaultBase
   var err error
   if len(mpd.BaseURL) > 0 {
      base, err = resolveURL(base, strings.TrimSpace(mpd.BaseURL[0].Value))
      if err != nil {
         return "", fmt.Errorf("MPD BaseURL: %w", err)
      }
   }
   if len(p.BaseURL) > 0 {
      base, err = resolveURL(base, strings.TrimSpace(p.BaseURL[0].Value))
      if err != nil {
         return "", fmt.Errorf("Period BaseURL: %w", err)
      }
   }
   if len(as.BaseURL) > 0 {
      base, err = resolveURL(base, strings.TrimSpace(as.BaseURL[0].Value))
      if err != nil {
         return "", fmt.Errorf("AdaptationSet BaseURL: %w", err)
      }
   }
   if len(r.BaseURL) > 0 {
      base, err = resolveURL(base, strings.TrimSpace(r.BaseURL[0].Value))
      if err != nil {
         return "", fmt.Errorf("Representation BaseURL: %w", err)
      }
   }
   return base, nil
}

func resolveURL(base, ref string) (string, error) {
   if ref == "" {
      return base, nil
   }
   bu, err := url.Parse(base)
   if err != nil {
      return "", fmt.Errorf("parse base %q: %w", base, err)
   }
   ru, err := url.Parse(ref)
   if err != nil {
      return "", fmt.Errorf("parse ref %q: %w", ref, err)
   }
   return bu.ResolveReference(ru).String(), nil
}

// ---------- Helpers ----------

func firstNonZero(durs ...time.Duration) time.Duration {
   for _, d := range durs {
      if d > 0 {
         return d
      }
   }
   return 0
}

// Minimal ISO 8601 duration parser for values like PT#H#M#S and P#DT#H#M#S.
// Supports days, hours, minutes, seconds (seconds may be fractional).
func parseISODuration(s string) time.Duration {
   s = strings.TrimSpace(s)
   if s == "" {
      return 0
   }
   re := regexp.MustCompile(`^P(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?)?$`)
   m := re.FindStringSubmatch(s)
   if m == nil {
      // Some MPDs may specify only PT#S or PT#M etc.; try a more permissive fallback for PT-only.
      re2 := regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?$`)
      m2 := re2.FindStringSubmatch(s)
      if m2 == nil {
         log.Printf("warning: could not parse ISO8601 duration %q", s)
         return 0
      }
      return hmsToDuration("", m2[1], m2[2], m2[3])
   }
   return hmsToDuration(m[1], m[2], m[3], m[4])
}

func hmsToDuration(daysStr, hoursStr, minsStr, secsStr string) time.Duration {
   var days, hours, mins int64
   var secs float64
   if daysStr != "" {
      if v, err := strconv.ParseInt(daysStr, 10, 64); err == nil {
         days = v
      }
   }
   if hoursStr != "" {
      if v, err := strconv.ParseInt(hoursStr, 10, 64); err == nil {
         hours = v
      }
   }
   if minsStr != "" {
      if v, err := strconv.ParseInt(minsStr, 10, 64); err == nil {
         mins = v
      }
   }
   if secsStr != "" {
      if v, err := strconv.ParseFloat(secsStr, 64); err == nil {
         secs = v
      }
   }
   total := (time.Duration(days) * 24 * time.Hour) +
      (time.Duration(hours) * time.Hour) +
      (time.Duration(mins) * time.Minute) +
      time.Duration(secs*float64(time.Second))
   return total
}
