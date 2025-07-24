package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "net/url"
   "os"
   "strconv"
   "strings"
)

// ---------- DASH structs (simplified) ----------

type MPD struct {
   BaseURL string    `xml:"BaseURL"`
   Period  Period    `xml:"Period"`
}

type Period struct {
   BaseURL string          `xml:"BaseURL"`
   AS      []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL string           `xml:"BaseURL"`
   ST      *SegmentTemplate `xml:"SegmentTemplate"`
   Reps    []Representation `xml:"Representation"`
}

type Representation struct {
   ID      string           `xml:"id,attr"`
   Band    int64            `xml:"bandwidth,attr"`
   BaseURL string           `xml:"BaseURL"`
   ST      *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentTemplate struct {
   Media   string           `xml:"media,attr"`
   StartN  int64            `xml:"startNumber,attr"`
   Dur     int64            `xml:"duration,attr"`
   TScale  int64            `xml:"timescale,attr"`
   Timeline *SegmentTimeline `xml:"SegmentTimeline"`
   EndNumber int64 `xml:"endNumber,attr"`
}

type SegmentTimeline struct{ S []S }

type S struct {
   T int64 `xml:"t,attr"`
   D int64 `xml:"d,attr"`
   R int64 `xml:"r,attr"`
}

// ---------- helpers ----------

const originalMPD = "http://test.test/test.mpd"

func base(u *url.URL, extra string) *url.URL {
   if extra == "" {
      return u
   }
   next, _ := url.Parse(extra)
   return u.ResolveReference(next)
}

func resolve(base *url.URL, seg string) string {
   if strings.HasPrefix(seg, "http") {
      return seg
   }
   next, _ := url.Parse(seg)
   return base.ResolveReference(next).String()
}

func substitute(tpl, id string, bw, num, t int64) string {
   tpl = strings.ReplaceAll(tpl, "$RepresentationID$", id)
   tpl = strings.ReplaceAll(tpl, "$Bandwidth$", strconv.FormatInt(bw, 10))
   tpl = strings.ReplaceAll(tpl, "$Number$", strconv.FormatInt(num, 10))
   tpl = strings.ReplaceAll(tpl, "$Time$", strconv.FormatInt(t, 10))

   tpl = reSub(`\$Number%0?(\d+)d\$`, tpl, num)
   tpl = reSub(`\$Time%0?(\d+)d\$`, tpl, t)
   tpl = reSub(`\$Bandwidth%0?(\d+)d\$`, tpl, bw)
   return tpl
}

func reSub(pat, tpl string, val int64) string {
   i := strings.Index(tpl, "$Number%")
   if !strings.Contains(pat, "Number") {
      i = strings.Index(tpl, "$Time%")
   }
   if i == -1 && !strings.Contains(pat, "Time") {
      i = strings.Index(tpl, "$Bandwidth%")
   }
   if i == -1 {
      return tpl
   }
   end := strings.Index(tpl[i:], "$") + i
   seg := tpl[i : end+1]
   var w int
   fmt.Sscanf(seg, "$%*[^%]%d", &w)
   return strings.Replace(tpl, seg, fmt.Sprintf("%0*d", w, val), 1)
}

func segments(u *url.URL, tpl *SegmentTemplate, id string, bw int64) []string {
   if tpl == nil || tpl.Media == "" {
      return nil
   }

   start := tpl.StartN
   if start == 0 {
      start = 1
   }
   var urls []string

   if tpl.Timeline != nil {
      time := int64(0)
      num := start
      for _, s := range tpl.Timeline.S {
         if s.T != 0 {
            time = s.T
         }
         repeats := s.R
         if repeats < 0 {
            repeats = 0
         }
         for i := int64(0); i <= repeats; i++ {
            seg := substitute(tpl.Media, id, bw, num, time)
            urls = append(urls, resolve(u, seg))
            num++
            time += s.D
         }
      }
      return urls
   }

// fallback: @duration + optional @endNumber
end := start + 999
if tpl.EndNumber > 0 {
	end = tpl.EndNumber
}
for n := start; n <= end; n++ {
	t := (n - start) * tpl.Dur
	seg := substitute(tpl.Media, id, bw, n, t)
	urls = append(urls, resolve(u, seg))
}

   return urls
}

// ---------- main ----------

func main() {
   if len(os.Args) < 2 {
      fmt.Fprintf(os.Stderr, "usage: %s local.mpd\n", os.Args[0])
      os.Exit(1)
   }

   f, err := os.Open(os.Args[1])
   if err != nil {
      fmt.Fprintln(os.Stderr, err)
      os.Exit(1)
   }
   defer f.Close()

   var mpd MPD
   if err := xml.NewDecoder(f).Decode(&mpd); err != nil {
      fmt.Fprintln(os.Stderr, "xml:", err)
      os.Exit(1)
   }

   start, _ := url.Parse(originalMPD)
   out := map[string][]string{}

   for _, p := range []Period{mpd.Period} {
      u := base(start, mpd.BaseURL)
      u = base(u, p.BaseURL)
      for _, as := range p.AS {
         u = base(u, as.BaseURL)
         for _, rep := range as.Reps {
            u := base(u, rep.BaseURL)
            tpl := as.ST
            if rep.ST != nil {
               tpl = rep.ST
            }
            out[rep.ID] = segments(u, tpl, rep.ID, rep.Band)
         }
      }
   }

   j, _ := json.MarshalIndent(out, "", "  ")
   fmt.Println(string(j))
}
