package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "math"
   "net/url"
   "os"
   "strconv"
   "strings"
)

type MPD struct {
   XMLName              xml.Name `xml:"MPD"`
   BaseURL              string   `xml:"BaseURL"`
   MediaPresentationDur string   `xml:"mediaPresentationDuration,attr"`
   Periods              []Period `xml:"Period"`
}

type Period struct {
   BaseURL  string          `xml:"BaseURL"`
   Duration string          `xml:"duration,attr"`
   Adapt    []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Representations []Representation `xml:"Representation"`
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
   Timescale       int              `xml:"timescale,attr"`
   Duration        int              `xml:"duration,attr"`
   StartNumber     *int             `xml:"startNumber,attr"`
   EndNumber       *int             `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentList struct {
   Initialization  *Initialization  `xml:"Initialization"`
   Timescale       int              `xml:"timescale,attr"`
   Duration        int              `xml:"duration,attr"`
   StartNumber     *int             `xml:"startNumber,attr"`
   EndNumber       *int             `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
   SegmentURLs     []SegmentURL     `xml:"SegmentURL"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   R int `xml:"r,attr"`
   D int `xml:"d,attr"`
}

var root, _ = url.Parse("http://test.test/test.mpd")

func main() {
   if len(os.Args) != 2 {
      panic("usage: dashmpd <local.mpd>")
   }
   mpdPath := os.Args[1]

   data, err := os.ReadFile(mpdPath)
   if err != nil {
      panic(err)
   }
   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      panic(err)
   }

   out := make(map[string][]string)

   for _, period := range mpd.Periods {
      periodBase := period.BaseURL
      periodDur := period.Duration
      if periodDur == "" {
         periodDur = mpd.MediaPresentationDur
      }
      sec := parseDuration(periodDur)

      for _, adapt := range period.Adapt {
         adaptBase := adapt.BaseURL
         for _, rep := range adapt.Representations {
            repBase := rep.BaseURL
            effectiveBase := resolveChain(root, mpd.BaseURL, periodBase, adaptBase, repBase)

            switch {
            case rep.SegmentTemplate != nil:
               st := rep.SegmentTemplate
               if st.Timescale == 0 {
                  st.Timescale = 1
               }
               if st.Initialization != "" {
                  init := expandTemplate(st.Initialization, rep.ID, 0, 0)
                  out[rep.ID] = append(out[rep.ID], resolveChain(effectiveBase, init).String())
               }
               out[rep.ID] = append(out[rep.ID], templateSegments(effectiveBase, rep.ID, st, sec)...)
            case rep.SegmentList != nil:
               sl := rep.SegmentList
               if sl.Timescale == 0 {
                  sl.Timescale = 1
               }
               if sl.Initialization != nil && sl.Initialization.SourceURL != "" {
                  out[rep.ID] = append(out[rep.ID], resolveChain(effectiveBase, sl.Initialization.SourceURL).String())
               }
               out[rep.ID] = append(out[rep.ID], segmentListSegments(effectiveBase, sl, sec)...)
            case adapt.SegmentTemplate != nil:
               st := adapt.SegmentTemplate
               if st.Timescale == 0 {
                  st.Timescale = 1
               }
               if st.Initialization != "" {
                  init := expandTemplate(st.Initialization, rep.ID, 0, 0)
                  out[rep.ID] = append(out[rep.ID], resolveChain(effectiveBase, init).String())
               }
               out[rep.ID] = append(out[rep.ID], templateSegments(effectiveBase, rep.ID, st, sec)...)
            case adapt.SegmentList != nil:
               sl := adapt.SegmentList
               if sl.Timescale == 0 {
                  sl.Timescale = 1
               }
               if sl.Initialization != nil && sl.Initialization.SourceURL != "" {
                  out[rep.ID] = append(out[rep.ID], resolveChain(effectiveBase, sl.Initialization.SourceURL).String())
               }
               out[rep.ID] = append(out[rep.ID], segmentListSegments(effectiveBase, sl, sec)...)
            default:
               out[rep.ID] = append(out[rep.ID], effectiveBase.String())
            }
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetEscapeHTML(false)
   if err := enc.Encode(out); err != nil {
      panic(err)
   }
}

func resolveChain(base *url.URL, parts ...string) *url.URL {
   res := base
   for _, p := range parts {
      if p == "" {
         continue
      }
      u, err := url.Parse(p)
      if err != nil {
         panic(err)
      }
      res = res.ResolveReference(u)
   }
   return res
}

func expandTemplate(tmpl string, repID string, number int, time int) string {
   tmpl = strings.ReplaceAll(tmpl, "$RepresentationID$", repID)
   tmpl = strings.ReplaceAll(tmpl, "$Time$", strconv.Itoa(time))
   for d := 1; d <= 9; d++ {
      pat := fmt.Sprintf("$Number%%0%dd$", d)
      if strings.Contains(tmpl, pat) {
         format := fmt.Sprintf("%%0%dd", d)
         tmpl = strings.ReplaceAll(tmpl, pat, fmt.Sprintf(format, number))
      }
   }
   tmpl = strings.ReplaceAll(tmpl, "$Number$", strconv.Itoa(number))
   return tmpl
}

func templateSegments(base *url.URL, repID string, st *SegmentTemplate, periodSec float64) []string {
   var urls []string
   start := 1
   if st.StartNumber != nil {
      start = *st.StartNumber
   }

   // 1. Explicit @endNumber
   if st.EndNumber != nil {
      for num := start; num <= *st.EndNumber; num++ {
         t := (num - start) * st.Duration
         media := expandTemplate(st.Media, repID, num, t)
         urls = append(urls, resolveChain(base, media).String())
      }
      return urls
   }

   // 2. SegmentTimeline
   if st.SegmentTimeline != nil {
      time := 0
      for _, s := range st.SegmentTimeline.S {
         repeats := 1 + s.R
         for r := 0; r < repeats; r++ {
            media := expandTemplate(st.Media, repID, start+len(urls), time)
            urls = append(urls, resolveChain(base, media).String())
            time += s.D
         }
      }
      return urls
   }

   // 3. Duration / timescale fallback
   if st.Duration == 0 {
      return urls
   }
   count := int(math.Ceil(periodSec * float64(st.Timescale) / float64(st.Duration)))
   for i := 0; i < count; i++ {
      num := start + i
      t := i * st.Duration
      media := expandTemplate(st.Media, repID, num, t)
      urls = append(urls, resolveChain(base, media).String())
   }
   return urls
}

func segmentListSegments(base *url.URL, sl *SegmentList, periodSec float64) []string {
   var urls []string
   start := 1
   if sl.StartNumber != nil {
      start = *sl.StartNumber
   }

   // 1. Explicit @endNumber
   if sl.EndNumber != nil {
      for num := start; num <= *sl.EndNumber; num++ {
         t := (num - start) * sl.Duration
         media := expandTemplate("$Number$", "", num, t)
         urls = append(urls, resolveChain(base, media).String())
      }
      return urls
   }

   // 2. <SegmentURL> elements
   if len(sl.SegmentURLs) > 0 {
      for _, su := range sl.SegmentURLs {
         urls = append(urls, resolveChain(base, su.Media).String())
      }
      return urls
   }

   // 3. Duration / timescale fallback
   if sl.Duration == 0 {
      return urls
   }
   count := int(math.Ceil(periodSec * float64(sl.Timescale) / float64(sl.Duration)))
   for i := 0; i < count; i++ {
      num := start + i
      t := i * sl.Duration
      media := expandTemplate("$Number$", "", num, t)
      urls = append(urls, resolveChain(base, media).String())
   }
   return urls
}

func parseDuration(dur string) float64 {
   if dur == "" {
      return 0
   }
   if !strings.HasPrefix(dur, "PT") {
      panic("invalid duration format")
   }
   dur = strings.TrimPrefix(dur, "PT")
   var sec float64
   if strings.Contains(dur, "H") {
      parts := strings.Split(dur, "H")
      h, err := strconv.ParseFloat(parts[0], 64)
      if err != nil {
         panic(err)
      }
      sec += h * 3600
      dur = parts[1]
   }
   if strings.Contains(dur, "M") {
      parts := strings.Split(dur, "M")
      m, err := strconv.ParseFloat(parts[0], 64)
      if err != nil {
         panic(err)
      }
      sec += m * 60
      dur = parts[1]
   }
   if strings.Contains(dur, "S") {
      parts := strings.Split(dur, "S")
      s, err := strconv.ParseFloat(parts[0], 64)
      if err != nil {
         panic(err)
      }
      sec += s
   }
   return sec
}
