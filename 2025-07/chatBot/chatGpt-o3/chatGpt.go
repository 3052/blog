package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "log"
   "net/url"
   "os"
   "path"
   "regexp"
   "strconv"
   "strings"
)

const defaultBase = "http://test.test/test.mpd"

// ──────────────────────────────────────────────────────────────────────────────
// Minimal MPEG-DASH data model
// ──────────────────────────────────────────────────────────────────────────────
type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL *string  `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   BaseURL         *string          `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   AdaptationSets  []AdaptationSet  `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         *string          `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         *string          `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentList struct {
   SegmentURLs []SegmentURL `xml:"SegmentURL"`
}

type SegmentURL struct {
   Media     string `xml:"media,attr"`
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentTemplate struct {
   Timescale      int              `xml:"timescale,attr"`
   StartNumber    int64            `xml:"startNumber,attr"`
   EndNumber      int64            `xml:"endNumber,attr"`
   Media          string           `xml:"media,attr"`
   Initialization string           `xml:"initialization,attr"`
   Duration       int64            `xml:"duration,attr"`
   Timeline       *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T int64 `xml:"t,attr"`
   D int64 `xml:"d,attr"`
   R int64 `xml:"r,attr"`
}

// ──────────────────────────────────────────────────────────────────────────────
// main
// ──────────────────────────────────────────────────────────────────────────────
func main() {
   log.SetFlags(0)

   if len(os.Args) != 2 {
      log.Fatalf("usage: go run main.go <mpd_file_path>")
   }
   mpdPath := os.Args[1]

   data, err := os.ReadFile(mpdPath)
   if err != nil {
      log.Fatalf("read mpd: %v", err)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      log.Fatalf("parse mpd: %v", err)
   }

   // ── Layer 1: defaultBase ────────────────────────────────────────────────
   rootBase := defaultBase
   // ── Layer 2: MPD.BaseURL ────────────────────────────────────────────────
   rootBase = combineBase(rootBase, mpd.BaseURL)

   out := map[string][]string{}

   for _, per := range mpd.Periods {
      // ── Layer 3: Period.BaseURL ────────────────────────────────────────
      pBase := combineBase(rootBase, per.BaseURL)

      for _, as := range per.AdaptationSets {
         // ── (optional) AdaptationSet.BaseURL ─────────────────────────
         aBase := combineBase(pBase, as.BaseURL)

         for _, rep := range as.Representations {
            // ── Layer 4: Representation.BaseURL ──────────────────────
            rBase := combineBase(aBase, rep.BaseURL)

            // Try to build segment list
            urls, err := segmentURLs(rep, as, per)
            if err == nil {
               resolved := make([]string, len(urls))
               for i, u := range urls {
                  resolved[i] = resolve(rBase, u)
               }
               out[rep.ID] = resolved
               continue
            }

            // No Segment* anywhere; fall-back to single-resource BaseURL
            if rep.BaseURL != nil && !strings.HasSuffix(*rep.BaseURL, "/") {
               out[rep.ID] = []string{rBase}
               continue
            }

            log.Fatalf("representation %q: %v", rep.ID, err)
         }
      }
   }

   enc, err := json.MarshalIndent(out, "", "  ")
   if err != nil {
      log.Fatalf("json marshal: %v", err)
   }
   fmt.Println(string(enc))
}

// ──────────────────────────────────────────────────────────────────────────────
// BaseURL helpers
// ──────────────────────────────────────────────────────────────────────────────
func combineBase(parent string, child *string) string {
   if child == nil || *child == "" {
      return parent
   }
   ref := *child
   if isAbs(ref) {
      return ref
   }
   pURL, perr := url.Parse(parent)
   rURL, rerr := url.Parse(ref)
   if perr == nil && rerr == nil {
      return pURL.ResolveReference(rURL).String()
   }
   // graceful fallback
   if strings.HasSuffix(parent, "/") {
      return parent + ref
   }
   return path.Dir(parent) + "/" + ref
}

func resolve(base, ref string) string {
   if isAbs(ref) {
      return ref
   }
   bu, berr := url.Parse(base)
   ru, rerr := url.Parse(ref)
   if berr == nil && rerr == nil {
      return bu.ResolveReference(ru).String()
   }
   // fallback
   if strings.HasSuffix(base, "/") {
      return base + ref
   }
   return path.Dir(base) + "/" + ref
}

func isAbs(s string) bool { return strings.Contains(s, "://") }

// ──────────────────────────────────────────────────────────────────────────────
// Pick nearest SegmentList / SegmentTemplate
// ──────────────────────────────────────────────────────────────────────────────
func segmentURLs(rep Representation, as AdaptationSet, per Period) ([]string, error) {
   switch {
   case rep.SegmentList != nil:
      return fromList(rep.SegmentList, rep.ID), nil
   case rep.SegmentTemplate != nil:
      return fromTemplate(rep.SegmentTemplate, rep.ID)
   case as.SegmentList != nil:
      return fromList(as.SegmentList, rep.ID), nil
   case as.SegmentTemplate != nil:
      return fromTemplate(as.SegmentTemplate, rep.ID)
   case per.SegmentList != nil:
      return fromList(per.SegmentList, rep.ID), nil
   case per.SegmentTemplate != nil:
      return fromTemplate(per.SegmentTemplate, rep.ID)
   default:
      return nil, fmt.Errorf("no SegmentList/SegmentTemplate found")
   }
}

// ──────────────────────────────────────────────────────────────────────────────
// SegmentList
// ──────────────────────────────────────────────────────────────────────────────
func fromList(sl *SegmentList, repID string) []string {
   out := make([]string, 0, len(sl.SegmentURLs))
   for _, s := range sl.SegmentURLs {
      u := s.Media
      if u == "" {
         u = s.SourceURL
      }
      out = append(out, strings.ReplaceAll(u, "$RepresentationID$", repID))
   }
   return out
}

// ──────────────────────────────────────────────────────────────────────────────
// SegmentTemplate
// ──────────────────────────────────────────────────────────────────────────────
var tokenRE = regexp.MustCompile(`\$(RepresentationID|Number|Time)(?:%0?(\d+)d)?\$`)

func fromTemplate(tpl *SegmentTemplate, repID string) ([]string, error) {
   if tpl.Media == "" {
      return nil, fmt.Errorf("SegmentTemplate missing @media")
   }

   var urls []string

   // init segment first
   if tpl.Initialization != "" {
      start := tpl.StartNumber
      if start == 0 {
         start = 1
      }
      urls = append(urls, applyTemplate(tpl.Initialization, repID, start, 0))
   }

   if tpl.Timeline != nil {
      ms, err := fromTemplateTimeline(tpl, repID)
      if err != nil {
         return nil, err
      }
      return append(urls, ms...), nil
   }

   if tpl.EndNumber != 0 {
      ms, err := fromTemplateRange(tpl, repID)
      if err != nil {
         return nil, err
      }
      return append(urls, ms...), nil
   }

   return nil, fmt.Errorf("SegmentTemplate requires <SegmentTimeline> or @endNumber")
}

func fromTemplateTimeline(tpl *SegmentTemplate, repID string) ([]string, error) {
   start := tpl.StartNumber
   if start == 0 {
      start = 1
   }
   end := tpl.EndNumber
   var (
      num     = start
      curTime int64
      out     []string
   )
   for _, s := range tpl.Timeline.S {
      reps := int(s.R)
      if reps < 0 {
         reps = 0
      }
      if s.T != 0 {
         curTime = s.T
      }
      for i := 0; i <= reps; i++ {
         if end != 0 && num > end {
            return out, nil
         }
         out = append(out, applyTemplate(tpl.Media, repID, num, curTime))
         if end != 0 && num == end {
            return out, nil
         }
         num++
         curTime += s.D
      }
   }
   return out, nil
}

func fromTemplateRange(tpl *SegmentTemplate, repID string) ([]string, error) {
   start := tpl.StartNumber
   if start == 0 {
      start = 1
   }
   end := tpl.EndNumber
   if end < start {
      return nil, fmt.Errorf("@endNumber (%d) < @startNumber (%d)", end, start)
   }
   dur := tpl.Duration
   var out []string
   var curTime int64
   for n := start; n <= end; n++ {
      out = append(out, applyTemplate(tpl.Media, repID, n, curTime))
      curTime += dur
   }
   return out, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Token expansion
// ──────────────────────────────────────────────────────────────────────────────
func applyTemplate(pattern, repID string, num, ts int64) string {
   return tokenRE.ReplaceAllStringFunc(pattern, func(m string) string {
      mm := tokenRE.FindStringSubmatch(m)
      token, widthStr := mm[1], mm[2]
      width := 0
      if widthStr != "" {
         w, _ := strconv.Atoi(widthStr)
         width = w
      }
      switch token {
      case "RepresentationID":
         return repID
      case "Number":
         return fmtInt(num, width)
      case "Time":
         return fmtInt(ts, width)
      }
      return m
   })
}

func fmtInt(v int64, w int) string {
   if w == 0 {
      return strconv.FormatInt(v, 10)
   }
   return fmt.Sprintf("%0*d", w, v)
}
