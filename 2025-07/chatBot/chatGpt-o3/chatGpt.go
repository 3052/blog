package main

import (
   "encoding/json"
   "encoding/xml"
   "errors"
   "fmt"
   "io"
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
   XMLName                   xml.Name         `xml:"MPD"`
   BaseURL                   []string         `xml:"BaseURL"`
   MediaPresentationDuration string           `xml:"mediaPresentationDuration,attr"`
   SegmentTemplate           *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList               *SegmentList     `xml:"SegmentList"`
   Periods                   []Period         `xml:"Period"`
}

type Period struct {
   BaseURL         []string         `xml:"BaseURL"`
   Duration        string           `xml:"duration,attr"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   AdaptationSets  []AdaptationSet  `xml:"AdaptationSet"`
   Representations []Representation `xml:"Representation"` // uncommon, but supported
}

type AdaptationSet struct {
   BaseURL         []string         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   Bandwidth       *int64           `xml:"bandwidth,attr"`
   BaseURL         []string         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
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
   Initialization         string           `xml:"initialization,attr"`
   Media                  string           `xml:"media,attr"`
   Timescale              *int64           `xml:"timescale,attr"`
   Duration               *int64           `xml:"duration,attr"`
   StartNumber            *int64           `xml:"startNumber,attr"`
   PresentationTimeOffset *int64           `xml:"presentationTimeOffset,attr"`
   SegmentTimeline        *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T *int64 `xml:"t,attr"`
   D *int64 `xml:"d,attr"`
   R *int64 `xml:"r,attr"`
}

func main() {
   log.SetFlags(0) // no timestamps
   if len(os.Args) != 2 {
      log.Printf("usage: go run main.go <mpd_file_path>")
      os.Exit(1)
   }
   path := os.Args[1]
   f, err := os.Open(path)
   if err != nil {
      log.Printf("failed to open MPD %q: %v", path, err)
      os.Exit(1)
   }
   defer f.Close()
   b, err := io.ReadAll(f)
   if err != nil {
      log.Printf("failed to read MPD %q: %v", path, err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(b, &mpd); err != nil {
      log.Printf("failed to parse MPD XML: %v", err)
      os.Exit(1)
   }

   mpdDur, _ := parseISODuration(mpd.MediaPresentationDuration)

   result := make(map[string][]string)
   for pi, p := range mpd.Periods {
      var perDurPtr *time.Duration
      if d, ok := parseISODuration(p.Duration); ok {
         perDurPtr = &d
      }
      parentBase := layeredBaseURL(mpd.BaseURL, p.BaseURL, nil, nil)
      // Process representations directly under period (rare)
      for ri := range p.Representations {
         rep := p.Representations[ri]
         if err := processRepresentation(&mpd, &p, nil, &rep, parentBase, perDurPtr, mpdDur, result, fmt.Sprintf("Period[%d] Representation[%d]", pi, ri)); err != nil {
            log.Printf("%v", err)
            os.Exit(1)
         }
      }
      for ai := range p.AdaptationSets {
         as := p.AdaptationSets[ai]
         asBase := layeredBaseURL(mpd.BaseURL, p.BaseURL, as.BaseURL, nil)
         for ri := range as.Representations {
            rep := as.Representations[ri]
            if err := processRepresentation(&mpd, &p, &as, &rep, asBase, perDurPtr, mpdDur, result, fmt.Sprintf("Period[%d] AdaptationSet[%d] Representation[%d]", pi, ai, ri)); err != nil {
               log.Printf("%v", err)
               os.Exit(1)
            }
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   if err := enc.Encode(result); err != nil {
      log.Printf("failed to encode JSON: %v", err)
      os.Exit(1)
   }
}

func processRepresentation(mpd *MPD, per *Period, as *AdaptationSet, rep *Representation, parentBase *url.URL, periodDur *time.Duration, mpdDur time.Duration, out map[string][]string, ctx string) error {
   if rep.ID == "" {
      return fmt.Errorf("%s: Representation@id is missing", ctx)
   }
   // Compute full base including Representation.BaseURL for segment resolution
   fullBase := cloneURL(parentBase)
   if len(rep.BaseURL) > 0 {
      fullBase = resolveAgainst(fullBase, rep.BaseURL[0])
   }

   // Choose addressing: prefer most specific (Representation > AdaptationSet > Period > MPD)
   segList := chooseSegmentList(rep.SegmentList, asSegmentList(as), per.SegmentList, mpd.SegmentList)
   segTmpl := effectiveTemplate(mpd.SegmentTemplate, per.SegmentTemplate, asSegmentTemplate(as), rep.SegmentTemplate)

   var segments []string
   var err error

   switch {
   case segList != nil:
      segments, err = expandSegmentList(segList, fullBase)
      if err != nil {
         return fmt.Errorf("%s (Representation id=%q): %v", ctx, rep.ID, err)
      }
   case segTmpl != nil:
      segments, err = expandSegmentTemplate(segTmpl, fullBase, rep, periodDur, &mpdDur)
      if err != nil {
         return fmt.Errorf("%s (Representation id=%q): %v", ctx, rep.ID, err)
      }
   default:
      // Fallback: single resource via Representation.BaseURL (must exist and be a single entry)
      if len(rep.BaseURL) == 1 {
         u := resolveAgainst(parentBase, rep.BaseURL[0])
         segments = []string{u.String()}
      } else {
         return fmt.Errorf("%s (Representation id=%q): no SegmentList or SegmentTemplate; and BaseURL is not a single resource", ctx, rep.ID)
      }
   }

   out[rep.ID] = segments
   return nil
}

func chooseSegmentList(candidates ...*SegmentList) *SegmentList {
   for _, c := range candidates {
      if c != nil {
         return c
      }
   }
   return nil
}

func asSegmentList(as *AdaptationSet) *SegmentList {
   if as == nil {
      return nil
   }
   return as.SegmentList
}

func asSegmentTemplate(as *AdaptationSet) *SegmentTemplate {
   if as == nil {
      return nil
   }
   return as.SegmentTemplate
}

func expandSegmentList(sl *SegmentList, base *url.URL) ([]string, error) {
   var out []string
   if sl.Initialization != nil && sl.Initialization.SourceURL != "" {
      out = append(out, resolveAgainst(base, sl.Initialization.SourceURL).String())
   }
   for i, su := range sl.SegmentURLs {
      if su.Media == "" {
         return nil, fmt.Errorf("SegmentList: SegmentURL[%d]@media is empty", i)
      }
      out = append(out, resolveAgainst(base, su.Media).String())
   }
   return out, nil
}

func expandSegmentTemplate(t *SegmentTemplate, base *url.URL, rep *Representation, periodDur *time.Duration, mpdDur *time.Duration) ([]string, error) {
   var out []string
   ts := int64(1)
   if t.Timescale != nil && *t.Timescale > 0 {
      ts = *t.Timescale
   }
   startNumber := int64(1)
   if t.StartNumber != nil && *t.StartNumber > 0 {
      startNumber = *t.StartNumber
   }
   pto := int64(0)
   if t.PresentationTimeOffset != nil && *t.PresentationTimeOffset >= 0 {
      pto = *t.PresentationTimeOffset
   }

   // Initialization (if provided)
   if strings.TrimSpace(t.Initialization) != "" {
      initURL := applyTemplate(t.Initialization, rep, startNumber, 0)
      out = append(out, resolveAgainst(base, initURL).String())
   }

   media := strings.TrimSpace(t.Media)
   if media == "" {
      // It's legal but uncommon; if no media and we had init only, return that; else error
      if len(out) > 0 {
         return out, nil
      }
      return nil, errors.New("SegmentTemplate@media is empty")
   }

   hasTimeToken := strings.Contains(media, "$Time$")
   hasNumberToken := strings.Contains(media, "$Number$")

   // With SegmentTimeline
   if t.SegmentTimeline != nil && len(t.SegmentTimeline.S) > 0 {
      if !hasTimeToken && !hasNumberToken {
         // Strictly speaking allowed, but then no varying token would give identical URLs; treat as error.
         return nil, errors.New("SegmentTemplate with SegmentTimeline must contain $Time$ or $Number$ in media template")
      }
      cur := pto // start from PTO so that first $Time$ becomes 0 when t is absent
      number := startNumber
      for si, s := range t.SegmentTimeline.S {
         if s.T != nil {
            cur = *s.T
         }
         // Determine duration per S or template
         var d int64
         if s.D != nil && *s.D > 0 {
            d = *s.D
         } else if t.Duration != nil && *t.Duration > 0 {
            d = *t.Duration
         } else {
            return nil, fmt.Errorf("SegmentTimeline S[%d]: missing duration (d) and template@duration", si)
         }

         reps := int64(1)
         if s.R != nil {
            reps = *s.R + 1
            if *s.R == -1 {
               // repeat to end of Period/MPD duration
               total, ok := knownDuration(periodDur, mpdDur)
               if !ok {
                  return nil, fmt.Errorf("SegmentTimeline S[%d]: R=-1 but total duration unknown", si)
               }
               endT := pto + int64(total.Seconds()*float64(ts))
               if d <= 0 {
                  return nil, fmt.Errorf("SegmentTimeline S[%d]: invalid duration d=%d", si, d)
               }
               // Compute number of repeats until cur >= endT
               reps = 0
               for cur+reps*d < endT {
                  reps++
               }
            }
         }

         for i := int64(0); i < reps; i++ {
            timeVal := cur - pto
            u := applyTemplate(media, rep, number, timeVal)
            out = append(out, resolveAgainst(base, u).String())
            cur += d
            number++
         }
      }
      return out, nil
   }

   // No SegmentTimeline
   if hasTimeToken {
      return nil, errors.New("SegmentTemplate@media contains $Time$ but no SegmentTimeline is present")
   }
   // Number-based expansion
   if t.Duration == nil || *t.Duration <= 0 {
      return nil, errors.New("number-based SegmentTemplate requires positive @duration")
   }
   total, ok := knownDuration(periodDur, mpdDur)
   if !ok {
      return nil, errors.New("cannot determine segment count: Period or MPD duration unknown")
   }
   totalUnits := total.Seconds() * float64(ts)
   segUnits := float64(*t.Duration)
   count := int64(math.Ceil(totalUnits / segUnits))
   if count <= 0 {
      return nil, fmt.Errorf("computed non-positive segment count: totalUnits=%.3f segUnits=%.3f", totalUnits, segUnits)
   }
   number := startNumber
   for i := int64(0); i < count; i++ {
      u := applyTemplate(media, rep, number, 0)
      out = append(out, resolveAgainst(base, u).String())
      number++
   }
   return out, nil
}

func knownDuration(periodDur *time.Duration, mpdDur *time.Duration) (time.Duration, bool) {
   if periodDur != nil && periodDur.Seconds() > 0 {
      return *periodDur, true
   }
   if mpdDur != nil && mpdDur.Seconds() > 0 {
      return *mpdDur, true
   }
   return 0, false
}

func effectiveTemplate(mpd, per, as, rep *SegmentTemplate) *SegmentTemplate {
   var t *SegmentTemplate
   t = mergeTemplates(nil, mpd)
   t = mergeTemplates(t, per)
   t = mergeTemplates(t, as)
   t = mergeTemplates(t, rep)
   return t
}

func mergeTemplates(base, over *SegmentTemplate) *SegmentTemplate {
   if base == nil && over == nil {
      return nil
   }
   if base == nil {
      cp := *over
      return &cp
   }
   if over == nil {
      cp := *base
      return &cp
   }
   out := *base
   if strings.TrimSpace(over.Initialization) != "" {
      out.Initialization = over.Initialization
   }
   if strings.TrimSpace(over.Media) != "" {
      out.Media = over.Media
   }
   if over.Timescale != nil {
      out.Timescale = over.Timescale
   }
   if over.Duration != nil {
      out.Duration = over.Duration
   }
   if over.StartNumber != nil {
      out.StartNumber = over.StartNumber
   }
   if over.PresentationTimeOffset != nil {
      out.PresentationTimeOffset = over.PresentationTimeOffset
   }
   if over.SegmentTimeline != nil {
      out.SegmentTimeline = over.SegmentTimeline
   }
   return &out
}

func layeredBaseURL(mpdBase, perBase, asBase, repBase []string) *url.URL {
   base, _ := url.Parse(defaultBase)
   for _, lst := range [][]string{mpdBase, perBase, asBase, repBase} {
      if len(lst) > 0 && strings.TrimSpace(lst[0]) != "" {
         base = resolveAgainst(base, lst[0])
      }
   }
   return base
}

func resolveAgainst(base *url.URL, refStr string) *url.URL {
   ref, err := url.Parse(strings.TrimSpace(refStr))
   if err != nil {
      // If parsing fails, fall back to base to avoid panic; caller will likely error elsewhere.
      return cloneURL(base)
   }
   return base.ResolveReference(ref)
}

func cloneURL(u *url.URL) *url.URL {
   if u == nil {
      return nil
   }
   cp := *u
   return &cp
}

func applyTemplate(tmpl string, rep *Representation, number int64, timeVal int64) string {
   s := tmpl
   bw := int64(0)
   if rep.Bandwidth != nil {
      bw = *rep.Bandwidth
   }
   s = strings.ReplaceAll(s, "$RepresentationID$", rep.ID)
   s = strings.ReplaceAll(s, "$Bandwidth$", strconv.FormatInt(bw, 10))
   s = strings.ReplaceAll(s, "$Number$", strconv.FormatInt(number, 10))
   s = strings.ReplaceAll(s, "$Time$", strconv.FormatInt(timeVal, 10))
   // Unescape literal $$
   s = strings.ReplaceAll(s, "$$", "$")
   return s
}

// parseISODuration parses minimal ISO-8601 durations for forms like PT#H#M#S, P#DT#H#M#S, and P#D.
// Seconds may be fractional. Returns (duration, true) on success; on failure logs a warning and returns (0, false).
func parseISODuration(s string) (time.Duration, bool) {
   str := strings.TrimSpace(s)
   if str == "" {
      return 0, false
   }

   // PT#H#M#S
   rePT := regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?$`)
   if m := rePT.FindStringSubmatch(str); m != nil {
      d := hmsToDuration("", m[1], m[2], m[3]) // daysStr must be "" in PT-only path
      return d, true
   }

   // P#DT#H#M#S
   rePDT := regexp.MustCompile(`^P(?:(\d+)D)?T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?$`)
   if m := rePDT.FindStringSubmatch(str); m != nil {
      d := hmsToDuration(m[1], m[2], m[3], m[4])
      return d, true
   }

   // P#D only
   rePD := regexp.MustCompile(`^P(\d+)D$`)
   if m := rePD.FindStringSubmatch(str); m != nil {
      d := hmsToDuration(m[1], "", "", "")
      return d, true
   }

   log.Printf("warning: failed to parse ISO-8601 duration %q; treating as unknown", s)
   return 0, false
}

// hmsToDuration converts provided day/hour/minute/second strings to time.Duration.
// Empty strings are treated as zero. Seconds may be fractional.
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
