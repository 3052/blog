package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "math"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
   "time"
)

/* ───────────────── MPD data model ───────────────── */

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Period  []Period `xml:"Period"`
}

type Period struct {
   BaseURL  string          `xml:"BaseURL"`
   Duration string          `xml:"duration,attr"` // ISO-8601 (e.g. PT30S)
   AS       []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Rep             []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   Timescale       int64            `xml:"timescale,attr"`
   Duration        int64            `xml:"duration,attr"`
   StartNumber     int64            `xml:"startNumber,attr"`
   EndNumber       int64            `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []SegmentTime `xml:"S"`
}
type SegmentTime struct {
   T int64 `xml:"t,attr"` // earliest presentation time (timescale units)
   D int64 `xml:"d,attr"` // duration             (timescale units) – required
   R int64 `xml:"r,attr"` // repeat count         (–1 = to Period end)
}

type SegmentList struct {
   Initialization *InitElem    `xml:"Initialization"`
   SegmentURL     []SegmentURL `xml:"SegmentURL"`
}
type InitElem struct {
   SourceURL string `xml:"sourceURL,attr"`
}
type SegmentURL struct {
   Media     string `xml:"media,attr"`
   SourceURL string `xml:"sourceURL,attr"`
}

/* ───────────────── helper functions ───────────────── */

// ISO-8601 “PT…” → time.Duration  (only H/M/S forms needed here)
func parseISODuration(s string) (time.Duration, error) {
   re := regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?$`)
   m := re.FindStringSubmatch(strings.ToUpper(strings.TrimSpace(s)))
   if m == nil {
      return 0, fmt.Errorf("unsupported duration %q", s)
   }
   h, _ := strconv.Atoi(z0(m[1]))
   min, _ := strconv.Atoi(z0(m[2]))
   sec, _ := strconv.ParseFloat(z0(m[3]), 64)
   return time.Duration(h)*time.Hour +
      time.Duration(min)*time.Minute +
      time.Duration(sec*1e9)*time.Nanosecond, nil
}
func z0(s string) string {
   if s == "" {
      return "0"
   }
   return s
}

// ─── template replacement with optional “%0Nd” width spec ───

func replaceVar(tpl, name string, val int64) string {
   // Width-specifier form  e.g. $Number%08d$
   re := regexp.MustCompile(fmt.Sprintf(`\$(%s)%%0(\d+)d\$`, name))
   tpl = re.ReplaceAllStringFunc(tpl, func(s string) string {
      m := re.FindStringSubmatch(s)
      w, _ := strconv.Atoi(m[2])
      return fmt.Sprintf("%0*d", w, val)
   })
   // Plain form  e.g. $Number$
   return strings.ReplaceAll(tpl, "$"+name+"$", strconv.FormatInt(val, 10))
}

func applyTplNum(tpl, repID string, num int64) string {
   tpl = strings.ReplaceAll(tpl, "$RepresentationID$", repID)
   return replaceVar(tpl, "Number", num)
}

func applyTplTime(tpl, repID string, num, tVal int64) string {
   tpl = strings.ReplaceAll(tpl, "$RepresentationID$", repID)
   tpl = replaceVar(tpl, "Number", num)
   return replaceVar(tpl, "Time", tVal)
}

// ─── URL helpers ───

func res(base *url.URL, refStr string) string {
   ref, err := url.Parse(strings.TrimSpace(refStr))
   if err != nil {
      return ""
   }
   return base.ResolveReference(ref).String()
}
func firstNonEmpty(ss ...string) string {
   for _, s := range ss {
      if strings.TrimSpace(s) != "" {
         return s
      }
   }
   return ""
}

/* ───────────────── segment generation ───────────────── */

func fromTemplate(base *url.URL, st *SegmentTemplate, repID string, periodDur time.Duration) []string {
   if st == nil || st.Media == "" {
      return nil
   }

   var urls []string
   if st.Initialization != "" {
      urls = append(urls, res(base, applyTplTime(st.Initialization, repID, 0, 0)))
   }

   // ── SegmentTimeline branch ──
   if st.SegmentTimeline != nil {
      urls = append(urls, fromTimeline(base, st, repID, periodDur)...)
      return urls
   }

   // ── $Number$ branch ──
   start := st.StartNumber
   if start == 0 {
      start = 1
   }
   scale := st.Timescale
   if scale == 0 {
      scale = 1
   }
   if st.Duration == 0 {
      return urls
   }

   var cnt int64
   if st.EndNumber != 0 {
      cnt = st.EndNumber - start + 1
   } else {
      segSec := float64(st.Duration) / float64(scale)
      cnt = int64(math.Ceil(periodDur.Seconds() / segSec))
   }

   for n := start; n < start+cnt; n++ {
      urls = append(urls, res(base, applyTplNum(st.Media, repID, n)))
   }
   return urls
}

func fromTimeline(base *url.URL, st *SegmentTemplate, repID string, periodDur time.Duration) []string {
   tl := st.SegmentTimeline
   scale := st.Timescale
   if scale == 0 {
      scale = 1
   }
   num := st.StartNumber
   if num == 0 {
      num = 1
   }
   var urls []string
   cur := int64(0)

   for i, s := range tl.S {
      if s.D == 0 {
         continue
      }
      if s.T != 0 {
         cur = s.T
      } else if i == 0 {
         cur = 0
      }

      repCnt := s.R
      if repCnt < 0 { // repeat to Period end
         if periodDur > 0 {
            remain := float64(periodDur)/float64(time.Second) - float64(cur)/float64(scale)
            segDur := float64(s.D) / float64(scale)
            repCnt = int64(math.Ceil(remain/segDur)) - 1
            if repCnt < 0 {
               repCnt = 0
            }
         } else {
            repCnt = 0
         }
      }

      for r := int64(0); r <= repCnt; r++ {
         urls = append(urls,
            res(base, applyTplTime(st.Media, repID, num, cur)),
         )
         num++
         cur += s.D
      }
   }
   return urls
}

func fromList(base *url.URL, sl *SegmentList) []string {
   if sl == nil {
      return nil
   }
   var urls []string
   if sl.Initialization != nil && sl.Initialization.SourceURL != "" {
      urls = append(urls, res(base, sl.Initialization.SourceURL))
   }
   for _, su := range sl.SegmentURL {
      if p := firstNonEmpty(su.SourceURL, su.Media); p != "" {
         urls = append(urls, res(base, p))
      }
   }
   return urls
}

/* ───────────────── main ───────────────── */

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }

   b, err := ioutil.ReadFile(os.Args[1])
   if err != nil {
      die("reading MPD: %v", err)
   }

   var mpd MPD
   if err := xml.Unmarshal(b, &mpd); err != nil {
      die("parsing XML: %v", err)
   }

   root, _ := url.Parse("http://test.test/test.mpd")
   out := make(map[string][]string)

   for _, p := range mpd.Period {
      pDur, _ := parseISODuration(p.Duration)

      for _, as := range p.AS {
         asST := as.SegmentTemplate

         for _, rep := range as.Rep {
            // hierarchical BaseURL
            base := root
            for _, u := range []string{mpd.BaseURL, p.BaseURL, as.BaseURL, rep.BaseURL} {
               if strings.TrimSpace(u) == "" {
                  continue
               }
               if ref, err := url.Parse(u); err == nil {
                  base = base.ResolveReference(ref)
               }
            }

            // choose segment definition
            st := rep.SegmentTemplate
            if st == nil {
               st = asST
            }

            var segs []string
            switch {
            case st != nil:
               segs = fromTemplate(base, st, rep.ID, pDur)
            case rep.SegmentList != nil:
               segs = fromList(base, rep.SegmentList)
            case as.SegmentList != nil:
               segs = fromList(base, as.SegmentList)
            default:
               // no segment info → emit resolved BaseURL itself
               if strings.TrimSpace(rep.BaseURL) != "" {
                  segs = []string{base.String()}
               }
            }

            if len(segs) > 0 {
               out[rep.ID] = append(out[rep.ID], segs...)
            }
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   if err := enc.Encode(out); err != nil {
      die("encoding JSON: %v", err)
   }
}
func die(f string, a ...interface{}) {
   fmt.Fprintf(os.Stderr, f+"\n", a...)
   os.Exit(1)
}
