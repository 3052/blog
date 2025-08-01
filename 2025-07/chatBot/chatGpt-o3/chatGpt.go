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

func main() {
   log.SetFlags(0)
   if len(os.Args) != 2 {
      log.Fatalf("usage: go run main.go <mpd_file_path>")
   }

   data, err := os.ReadFile(os.Args[1])
   if err != nil {
      log.Fatalf("read: %v", err)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      log.Fatalf("parse: %v", err)
   }

   startURL, _ := url.Parse("http://test.test/test.mpd")
   out := make(map[string][]string)

   for _, period := range mpd.Periods { // ­– keep natural Period order
      periodDurSec := parseISODuration(period.Duration)
      for _, as := range period.AdaptationSets {
         for _, rep := range as.Representations {
            repBase := resolveBaseChain(
               startURL,
               mpd.BaseURL(),
               period.BaseURL(),
               as.BaseURL(),
               rep.BaseURL(),
            )

            segList := firstNonNil(rep.SegmentList, as.SegmentList, period.SegmentList, mpd.SegmentList)
            segTmpl := firstNonNil(rep.SegmentTemplate, as.SegmentTemplate, period.SegmentTemplate, mpd.SegmentTemplate)

            var segs []string
            switch {
            case segList != nil:
               segs = buildSegmentList(rep.ID, repBase, segList)
            case segTmpl != nil:
               segs = buildSegmentTemplate(rep.ID, repBase, segTmpl, periodDurSec)
            default:
               if rb := rep.BaseURL(); rb != "" && !strings.HasSuffix(rb, "/") {
                  segs = []string{repBase.String()}
               }
            }

            if len(segs) > 0 {
               out[rep.ID] = append(out[rep.ID], segs...)
            }
         }
      }
   }

   j, err := json.MarshalIndent(out, "", "  ")
   if err != nil {
      log.Fatalf("json: %v", err)
   }
   fmt.Println(string(j))
}

/* ---------- XML structures ---------- */

type BaseURL struct {
   Value string `xml:",chardata"`
}

type MPD struct {
   XMLName         xml.Name         `xml:"MPD"`
   BaseURLs        []BaseURL        `xml:"BaseURL"`
   Periods         []Period         `xml:"Period"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

func (m MPD) BaseURL() string { return firstVal(m.BaseURLs) }

type Period struct {
   Duration        string           `xml:"duration,attr"` // ISO-8601
   BaseURLs        []BaseURL        `xml:"BaseURL"`
   AdaptationSets  []AdaptationSet  `xml:"AdaptationSet"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

func (p Period) BaseURL() string { return firstVal(p.BaseURLs) }

type AdaptationSet struct {
   BaseURLs        []BaseURL        `xml:"BaseURL"`
   Representations []Representation `xml:"Representation"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

func (a AdaptationSet) BaseURL() string { return firstVal(a.BaseURLs) }

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURLs        []BaseURL        `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

func (r Representation) BaseURL() string { return firstVal(r.BaseURLs) }

type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   StartNumber     int64            `xml:"startNumber,attr"`
   EndNumber       int64            `xml:"endNumber,attr"`
   Timescale       int64            `xml:"timescale,attr"`
   Duration        int64            `xml:"duration,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T int64 `xml:"t,attr"`
   D int64 `xml:"d,attr"`
   R int64 `xml:"r,attr"`
}

type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media     string `xml:"media,attr"`
   SourceURL string `xml:"sourceURL,attr"`
}

/* ---------- Helpers ---------- */

func firstVal(b []BaseURL) string {
   if len(b) == 0 {
      return ""
   }
   return strings.TrimSpace(b[0].Value)
}

func resolve(base *url.URL, s string) *url.URL {
   u, err := url.Parse(strings.TrimSpace(s))
   if err != nil {
      return base
   }
   return base.ResolveReference(u)
}

func resolveBaseChain(start *url.URL, parts ...string) *url.URL {
   cur := start
   for _, p := range parts {
      if p != "" {
         cur = resolve(cur, p)
      }
   }
   return cur
}

func firstNonNil[T any](candidates ...*T) *T {
   for _, c := range candidates {
      if c != nil {
         return c
      }
   }
   return nil
}

/* ---------- SegmentList processing ---------- */

func buildSegmentList(repID string, base *url.URL, sl *SegmentList) []string {
   var out []string
   num := int64(1)

   if sl.Initialization != nil && sl.Initialization.SourceURL != "" {
      out = append(out, expandAndResolve(repID, num, 0, base, sl.Initialization.SourceURL))
   }

   for _, s := range sl.SegmentURLs {
      src := s.Media
      if src == "" {
         src = s.SourceURL
      }
      out = append(out, expandAndResolve(repID, num, 0, base, src))
      num++
   }
   return out
}

/* ---------- SegmentTemplate processing ---------- */

func buildSegmentTemplate(repID string, base *url.URL, st *SegmentTemplate, periodDurSec float64) []string {
   var out []string

   start := st.StartNumber
   if start == 0 {
      start = 1
   }
   end := st.EndNumber

   // Initialization always first if present
   if st.Initialization != "" {
      out = append(out, expandAndResolve(repID, start, 0, base, st.Initialization))
   }

   num := start
   switch {
   case st.SegmentTimeline != nil:
      var curTime int64
      for _, s := range st.SegmentTimeline.S {
         if s.T != 0 {
            curTime = s.T
         }
         repCnt := s.R
         if repCnt < 0 {
            repCnt = 0
         }
         for i := int64(0); i <= repCnt; i++ {
            if end != 0 && num > end {
               return out
            }
            out = append(out, expandAndResolve(repID, num, curTime, base, st.Media))
            curTime += s.D
            num++
         }
      }

   case end != 0: // simple numbered range
      for n := num; n <= end; n++ {
         out = append(out, expandAndResolve(repID, n, 0, base, st.Media))
      }

   case st.Duration > 0 && st.Timescale > 0 && periodDurSec > 0:
      // compute segment count from Period duration
      segCount := int64(math.Ceil(periodDurSec * float64(st.Timescale) / float64(st.Duration)))
      curTime := int64(0)
      for i := int64(0); i < segCount; i++ {
         out = append(out, expandAndResolve(repID, num, curTime, base, st.Media))
         num++
         curTime += st.Duration
      }
   }

   return out
}

/* ---------- Template expansion ---------- */

var tokenRE = regexp.MustCompile(`\$(RepresentationID|Number|Time)(?:%0(\d+)d)?\$`)

func expandAndResolve(repID string, num, tim int64, base *url.URL, tpl string) string {
   replaced := tokenRE.ReplaceAllStringFunc(tpl, func(m string) string {
      sub := tokenRE.FindStringSubmatch(m)
      name := sub[1]
      width := 0
      if sub[2] != "" {
         w, _ := strconv.Atoi(sub[2])
         width = w
      }
      switch name {
      case "RepresentationID":
         return repID
      case "Number":
         return formatInt(num, width)
      case "Time":
         return formatInt(tim, width)
      default:
         return m
      }
   })
   return resolve(base, replaced).String()
}

func formatInt(v int64, width int) string {
   if width > 0 {
      return fmt.Sprintf("%0*d", width, v)
   }
   return strconv.FormatInt(v, 10)
}

/* ---------- ISO-8601 duration parsing (very small subset) ---------- */

var isoDurRE = regexp.MustCompile(`^P(?:(\d+(?:\.\d+)?)D)?(?:T(?:(\d+(?:\.\d+)?)H)?(?:(\d+(?:\.\d+)?)M)?(?:(\d+(?:\.\d+)?)S)?)?$`)

// parseISODuration returns seconds from a limited ISO-8601 duration subset (PnDTnHnMnS)
func parseISODuration(s string) float64 {
   m := isoDurRE.FindStringSubmatch(strings.TrimSpace(s))
   if m == nil {
      return 0
   }
   toF := func(x string, mul float64) float64 {
      if x == "" {
         return 0
      }
      v, _ := strconv.ParseFloat(x, 64)
      return v * mul
   }
   return toF(m[1], 86400) + toF(m[2], 3600) + toF(m[3], 60) + toF(m[4], 1)
}
