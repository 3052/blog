package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
)

// ---------- DASH structs (minimal subset) ----------

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL *string  `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   BaseURL        *string          `xml:"BaseURL"`
   AdaptationSets []AdaptationSet  `xml:"AdaptationSet"`
   SegmentList    *SegmentList     `xml:"SegmentList"`
   SegmentTmpl    *SegmentTemplate `xml:"SegmentTemplate"`
}

type AdaptationSet struct {
   BaseURL         *string          `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTmpl     *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID          string           `xml:"id,attr"`
   BaseURL     *string          `xml:"BaseURL"`
   SegmentList *SegmentList     `xml:"SegmentList"`
   SegmentTmpl *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentList struct {
   Timescale      *int         `xml:"timescale,attr"`
   Duration       *int         `xml:"duration,attr"`
   Initialization *Init        `xml:"Initialization"`
   SegmentURLs    []SegmentURL `xml:"SegmentURL"`
}

type Init struct {
   Source string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

type SegmentTemplate struct {
   Timescale *int             `xml:"timescale,attr"`
   Duration  *int             `xml:"duration,attr"`
   StartNum  *int             `xml:"startNumber,attr"`
   EndNum    *int             `xml:"endNumber,attr"`
   Media     string           `xml:"media,attr"`
   Init      string           `xml:"initialization,attr"`
   Timeline  *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T *int64 `xml:"t,attr"`
   D int64  `xml:"d,attr"`
   R *int64 `xml:"r,attr"`
}

// ---------- helpers ----------

func must(err error) {
   if err != nil {
      panic(err)
   }
}

func resolve(base *url.URL, ref string) *url.URL {
   u, err := url.Parse(strings.TrimSpace(ref))
   if err != nil {
      return base
   }
   return base.ResolveReference(u)
}

var reVar = regexp.MustCompile(`\$(RepresentationID|Number|Time)(%0(\d+)d)?\$`)

func expandTmpl(tmpl, repID string, num int, t int64) string {
   return reVar.ReplaceAllStringFunc(tmpl, func(s string) string {
      m := reVar.FindStringSubmatch(s)
      name, pad := m[1], m[3]
      width := 0
      if pad != "" {
         width, _ = strconv.Atoi(pad)
      }
      switch name {
      case "RepresentationID":
         return repID
      case "Number":
         if width > 0 {
            return fmt.Sprintf("%0*d", width, num)
         }
         return strconv.Itoa(num)
      case "Time":
         if width > 0 {
            return fmt.Sprintf("%0*d", width, t)
         }
         return fmt.Sprintf("%d", t)
      }
      return s
   })
}

func coalesce[T any](v ...*T) *T {
   for _, p := range v {
      if p != nil {
         return p
      }
   }
   return nil
}

// ---------- segment URL generation for one Representation ----------

func buildURLs(rep Representation, as AdaptationSet, per Period, mpd MPD, base *url.URL) []string {
   // 1. Hierarchical BaseURL
   b := base
   if mpd.BaseURL != nil {
      b = resolve(b, *mpd.BaseURL)
   }
   if per.BaseURL != nil {
      b = resolve(b, *per.BaseURL)
   }
   if as.BaseURL != nil {
      b = resolve(b, *as.BaseURL)
   }
   if rep.BaseURL != nil {
      b = resolve(b, *rep.BaseURL)
   }

   // 2. Effective SegmentList / SegmentTemplate (closest wins)
   sl := coalesce(rep.SegmentList, as.SegmentList, per.SegmentList)
   st := coalesce(rep.SegmentTmpl, as.SegmentTmpl, per.SegmentTmpl)

   // 3. NEW RULE: Representation has its own BaseURL and *no* Segment*
   if rep.BaseURL != nil && sl == nil && st == nil {
      return []string{b.String()}
   }

   var out []string

   // -------- SegmentList --------
   if sl != nil {
      if sl.Initialization != nil && sl.Initialization.Source != "" {
         out = append(out, resolve(b, sl.Initialization.Source).String())
      }
      for _, s := range sl.SegmentURLs {
         out = append(out, resolve(b, s.Media).String())
      }
      return out
   }

   // -------- SegmentTemplate --------
   if st == nil {
      return out // nothing to emit
   }

   start := 1
   if st.StartNum != nil {
      start = *st.StartNum
   }

   // Build the list of (num, time) pairs
   type seg struct {
      num int
      t   int64
   }
   var segments []seg

   if st.Timeline != nil {
      cur := int64(0)
      num := start
      for _, s := range st.Timeline.S {
         if s.T != nil {
            cur = *s.T
         }
         repCount := int64(0)
         if s.R != nil {
            repCount = *s.R
         }
         for i := int64(0); i <= repCount; i++ {
            segments = append(segments, seg{num, cur})
            cur += s.D
            num++
         }
      }
   } else {
      if st.EndNum == nil {
         return out // unknown length without timeline
      }
      dur := 0
      if st.Duration != nil {
         dur = *st.Duration
      }
      for n := start; n <= *st.EndNum; n++ {
         t := int64(dur * (n - start))
         segments = append(segments, seg{n, t})
      }
   }

   // Initialization segment
   if st.Init != "" {
      out = append(out, resolve(b, expandTmpl(st.Init, rep.ID, segments[0].num, segments[0].t)).String())
   }

   // Media segments
   for _, s := range segments {
      out = append(out, resolve(b, expandTmpl(st.Media, rep.ID, s.num, s.t)).String())
   }
   return out
}

// ---------- main ----------

func main() {
   mpdPath := "-" // default: stdin
   if len(os.Args) > 1 {
      mpdPath = os.Args[1]
   }

   var r io.Reader
   if mpdPath == "-" {
      r = os.Stdin
   } else {
      f, err := os.Open(mpdPath)
      must(err)
      defer f.Close()
      r = f
   }

   data, err := io.ReadAll(r)
   must(err)

   var mpd MPD
   must(xml.Unmarshal(data, &mpd))

   // Starting absolute URL (per requirement)
   baseURL, _ := url.Parse("http://test.test/test.mpd")

   result := make(map[string][]string)
   for _, p := range mpd.Periods {
      for _, as := range p.AdaptationSets {
         for _, rep := range as.Representations {
            result[rep.ID] = buildURLs(rep, as, p, mpd, baseURL)
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   must(enc.Encode(result))
}
