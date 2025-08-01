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
   "time"
)

const rootMPDURL = "http://test.test/test.mpd"

// ---------- XML structures ----------

type baseURL struct {
   Value string `xml:",chardata"`
}

type initURL struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type segmentURL struct {
   Media     string `xml:"media,attr"`
   SourceURL string `xml:"sourceURL,attr"`
}

type segmentList struct {
   Initialization *initURL     `xml:"Initialization"`
   SegmentURLs    []segmentURL `xml:"SegmentURL"`
}

type sElem struct {
   T *int64 `xml:"t,attr"`
   D int64  `xml:"d,attr"` // required
   R *int   `xml:"r,attr"`
}

type segmentTimeline struct {
   S []sElem `xml:"S"`
}

type segmentTemplate struct {
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   StartNumber     *int             `xml:"startNumber,attr"`
   EndNumber       *int             `xml:"endNumber,attr"`
   Timescale       *int             `xml:"timescale,attr"`
   Duration        *int             `xml:"duration,attr"`
   SegmentTimeline *segmentTimeline `xml:"SegmentTimeline"`
}

type representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         []baseURL        `xml:"BaseURL"`
   SegmentList     *segmentList     `xml:"SegmentList"`
   SegmentTemplate *segmentTemplate `xml:"SegmentTemplate"`
}

type adaptationSet struct {
   BaseURL         []baseURL        `xml:"BaseURL"`
   SegmentList     *segmentList     `xml:"SegmentList"`
   SegmentTemplate *segmentTemplate `xml:"SegmentTemplate"`
   Representations []representation `xml:"Representation"`
}

type period struct {
   BaseURL         []baseURL        `xml:"BaseURL"`
   Duration        string           `xml:"duration,attr"`
   SegmentList     *segmentList     `xml:"SegmentList"`
   SegmentTemplate *segmentTemplate `xml:"SegmentTemplate"`
   AdaptationSets  []adaptationSet  `xml:"AdaptationSet"`
}

type mpd struct {
   XMLName         xml.Name         `xml:"MPD"`
   BaseURL         []baseURL        `xml:"BaseURL"`
   SegmentList     *segmentList     `xml:"SegmentList"`
   SegmentTemplate *segmentTemplate `xml:"SegmentTemplate"`
   Periods         []period         `xml:"Period"`
}

// ---------- helpers ----------

func firstBaseURL(b []baseURL) string {
   if len(b) > 0 {
      return strings.TrimSpace(b[0].Value)
   }
   return ""
}

func mustParse(raw string) *url.URL {
   u, _ := url.Parse(strings.TrimSpace(raw))
   return u
}

func pickSegmentList(c ...*segmentList) *segmentList {
   for _, x := range c {
      if x != nil {
         return x
      }
   }
   return nil
}

func pickSegmentTemplate(c ...*segmentTemplate) *segmentTemplate {
   for _, x := range c {
      if x != nil {
         return x
      }
   }
   return nil
}

// ISO-8601 “PT##H##M##S” (seconds may be fractional)
var durRe = regexp.MustCompile(`^PT(?:(\d+(?:\.\d+)?)H)?(?:(\d+(?:\.\d+)?)M)?(?:(\d+(?:\.\d+)?)S)?$`)

func parseISODuration(s string) (time.Duration, error) {
   m := durRe.FindStringSubmatch(strings.TrimSpace(s))
   if m == nil {
      return 0, fmt.Errorf("unsupported duration %q", s)
   }
   var sec float64
   if m[1] != "" {
      v, _ := strconv.ParseFloat(m[1], 64)
      sec += v * 3600
   }
   if m[2] != "" {
      v, _ := strconv.ParseFloat(m[2], 64)
      sec += v * 60
   }
   if m[3] != "" {
      v, _ := strconv.ParseFloat(m[3], 64)
      sec += v
   }
   return time.Duration(sec * float64(time.Second)), nil
}

// ---------- template token expansion ----------

var tokRe = regexp.MustCompile(`\$(RepresentationID|Number|Time)(?:%0(\d+)d)?\$`)

func expand(tmpl, repID string, num int, t int64) string {
   return tokRe.ReplaceAllStringFunc(tmpl, func(s string) string {
      m := tokRe.FindStringSubmatch(s)
      name, pad := m[1], m[2]
      width := 0
      if pad != "" {
         width, _ = strconv.Atoi(pad)
      }
      format := func(v interface{}) string {
         if width > 0 {
            return fmt.Sprintf("%0*d", width, v)
         }
         return fmt.Sprint(v)
      }
      switch name {
      case "RepresentationID":
         return repID
      case "Number":
         return format(num)
      case "Time":
         return format(t)
      default:
         return s
      }
   })
}

// ---------- generators ----------

func genSegmentList(sl *segmentList, base *url.URL, repID string) []string {
   var out []string
   if sl.Initialization != nil && sl.Initialization.SourceURL != "" {
      initURL := expand(sl.Initialization.SourceURL, repID, 0, 0)
      out = append(out, base.ResolveReference(mustParse(initURL)).String())
   }
   for i, s := range sl.SegmentURLs {
      path := s.Media
      if path == "" {
         path = s.SourceURL
      }
      full := expand(path, repID, i+1, 0)
      out = append(out, base.ResolveReference(mustParse(full)).String())
   }
   return out
}

func genSegmentTemplate(st *segmentTemplate, base *url.URL, repID string, periodDurSec float64) []string {
   var out []string

   // defaults
   n := 1
   if st.StartNumber != nil {
      n = *st.StartNumber
   }
   stop := -1
   if st.EndNumber != nil {
      stop = *st.EndNumber
   }

   // initialization first
   if st.Initialization != "" {
      initURL := expand(st.Initialization, repID, n, 0)
      out = append(out, base.ResolveReference(mustParse(initURL)).String())
   }

   timescale := 1
   if st.Timescale != nil {
      timescale = *st.Timescale
   }

   push := func(num int, t int64) {
      urlStr := expand(st.Media, repID, num, t)
      out = append(out, base.ResolveReference(mustParse(urlStr)).String())
   }

   // A) SegmentTimeline present
   if tl := st.SegmentTimeline; tl != nil {
      curTime := int64(0)
      first := true
      for _, s := range tl.S {
         if s.T != nil {
            curTime = *s.T
         } else if first {
            // first S without @t starts at 0
            curTime = 0
         }
         first = false

         repeat := 0
         if s.R != nil {
            repeat = *s.R
         }
         for i := 0; i <= repeat; i++ {
            if stop != -1 && n > stop {
               return out
            }
            push(n, curTime)
            n++
            curTime += s.D
         }
      }
      return out
   }

   // B) Sequential start–end
   if stop != -1 {
      for num := n; num <= stop; num++ {
         push(num, 0)
      }
      return out
   }

   // C) Fallback using duration/timescale
   if st.Duration != nil && periodDurSec > 0 {
      cnt := int(math.Ceil(periodDurSec * float64(timescale) / float64(*st.Duration)))
      for i := 0; i < cnt; i++ {
         push(n+i, 0)
      }
   }

   return out
}

// ---------- main ----------

func main() {
   log.SetFlags(0)

   if len(os.Args) != 2 {
      log.Fatalf("usage: go run main.go <mpd_file_path>")
   }

   raw, err := os.ReadFile(os.Args[1])
   if err != nil {
      log.Fatalf("read: %v", err)
   }

   var doc mpd
   if err := xml.Unmarshal(raw, &doc); err != nil {
      log.Fatalf("parse xml: %v", err)
   }

   root := mustParse(rootMPDURL)
   if b := firstBaseURL(doc.BaseURL); b != "" {
      root = root.ResolveReference(mustParse(b))
   }

   res := make(map[string][]string)

   for _, per := range doc.Periods {
      periodBase := root
      if b := firstBaseURL(per.BaseURL); b != "" {
         periodBase = periodBase.ResolveReference(mustParse(b))
      }

      periodDur := 0.0
      if per.Duration != "" {
         if d, err := parseISODuration(per.Duration); err == nil {
            periodDur = d.Seconds()
         }
      }

      for _, aset := range per.AdaptationSets {
         adaptBase := periodBase
         if b := firstBaseURL(aset.BaseURL); b != "" {
            adaptBase = adaptBase.ResolveReference(mustParse(b))
         }

         for _, rep := range aset.Representations {
            repBase := adaptBase
            repBaseRaw := firstBaseURL(rep.BaseURL)
            if repBaseRaw != "" {
               repBase = repBase.ResolveReference(mustParse(repBaseRaw))
            }

            sl := pickSegmentList(rep.SegmentList, aset.SegmentList, per.SegmentList, doc.SegmentList)
            st := pickSegmentTemplate(rep.SegmentTemplate, aset.SegmentTemplate, per.SegmentTemplate, doc.SegmentTemplate)

            // single-URL representation
            if repBaseRaw != "" &&
               !strings.HasSuffix(strings.TrimSpace(repBaseRaw), "/") &&
               sl == nil && st == nil {
               res[rep.ID] = append(res[rep.ID], repBase.String())
               continue
            }

            var segs []string
            if sl != nil {
               segs = genSegmentList(sl, repBase, rep.ID)
            } else if st != nil {
               segs = genSegmentTemplate(st, repBase, rep.ID, periodDur)
            } else {
               log.Fatalf("no segment info for Representation id=%s", rep.ID)
            }
            res[rep.ID] = append(res[rep.ID], segs...)
         }
      }
   }

   out, err := json.MarshalIndent(res, "", "  ")
   if err != nil {
      log.Fatalf("json marshal: %v", err)
   }
   fmt.Println(string(out))
}
