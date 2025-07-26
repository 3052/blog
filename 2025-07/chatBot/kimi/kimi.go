package main

import (
   "encoding/json"
   "encoding/xml"
   "errors"
   "fmt"
   "math"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
   "time"
)

/* ---------- CLI entry ---------- */

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintln(os.Stderr, "usage: dash-expand <local.mpd>")
      os.Exit(1)
   }
   mpdPath := os.Args[1]

   f, err := os.Open(mpdPath)
   if err != nil {
      exitErr(err)
   }
   defer f.Close()

   var mpd MPD
   if err := xml.NewDecoder(f).Decode(&mpd); err != nil {
      exitErr(err)
   }

   segments, err := expand(&mpd)
   if err != nil {
      exitErr(err)
   }

   json.NewEncoder(os.Stdout).Encode(segments)
}

func exitErr(err error) {
   fmt.Fprintln(os.Stderr, "error:", err)
   os.Exit(1)
}

/* ---------- XML types ---------- */

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   Duration string `xml:"duration,attr"`
   BaseURL  string `xml:"BaseURL"`
   AS       []AdaptationSet
}

func (p *Period) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
   type Alias Period
   var aux struct {
      Alias
      AS []AdaptationSet `xml:"AdaptationSet"`
   }
   if err := d.DecodeElement(&aux, &start); err != nil {
      return err
   }
   *p = Period(aux.Alias)
   p.AS = aux.AS
   return nil
}

type AdaptationSet struct {
   BaseURL         string
   SegmentTemplate *SegmentTemplate
   SegmentList     *SegmentList
   Representations []Representation `xml:"Representation"`
}

func (as *AdaptationSet) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
   type Alias AdaptationSet
   var aux struct {
      Alias
      BaseURL         string           `xml:"BaseURL"`
      SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
      SegmentList     *SegmentList     `xml:"SegmentList"`
   }
   if err := d.DecodeElement(&aux, &start); err != nil {
      return err
   }
   *as = AdaptationSet(aux.Alias)
   as.BaseURL = aux.BaseURL
   as.SegmentTemplate = aux.SegmentTemplate
   as.SegmentList = aux.SegmentList
   return nil
}

type Representation struct {
   ID              string `xml:"id,attr"`
   BaseURL         string
   SegmentTemplate *SegmentTemplate
   SegmentList     *SegmentList
}

func (r *Representation) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
   type Alias Representation
   var aux struct {
      Alias
      BaseURL         string           `xml:"BaseURL"`
      SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
      SegmentList     *SegmentList     `xml:"SegmentList"`
   }
   if err := d.DecodeElement(&aux, &start); err != nil {
      return err
   }
   *r = Representation(aux.Alias)
   r.BaseURL = aux.BaseURL
   r.SegmentTemplate = aux.SegmentTemplate
   r.SegmentList = aux.SegmentList
   return nil
}

type SegmentTemplate struct {
   Media           string `xml:"media,attr"`
   Timescale       int    `xml:"timescale,attr"`
   Duration        int    `xml:"duration,attr"`
   StartNumber     *int   `xml:"startNumber,attr"`
   Initialization  string `xml:"initialization,attr"`
   SegmentTimeline *SegmentTimeline
}

func (st *SegmentTemplate) getStartNumber() int {
   if st.StartNumber == nil {
      return 1
   }
   return *st.StartNumber
}

type SegmentList struct {
   Duration        int              `xml:"duration,attr"`
   StartNumber     *int             `xml:"startNumber,attr"`
   Timescale       int              `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
   Initialization  *URL             `xml:"Initialization"`
   SegmentURLs     []SegmentURL     `xml:"SegmentURL"`
}

func (sl *SegmentList) getStartNumber() int {
   if sl.StartNumber == nil {
      return 1
   }
   return *sl.StartNumber
}

type URL struct {
   URL string `xml:"sourceURL,attr"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T int `xml:"t,attr"`
   D int `xml:"d,attr"`
   R int `xml:"r,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

/* ---------- expansion ---------- */

func dedup(in []string) []string {
   seen := make(map[string]struct{})
   out := make([]string, 0, len(in))
   for _, u := range in {
      if _, ok := seen[u]; !ok {
         seen[u] = struct{}{}
         out = append(out, u)
      }
   }
   return out
}

func expandTemplate(base *url.URL, repID string, st *SegmentTemplate, periodDur time.Duration) ([]string, error) {
   if st.Media == "" {
      return nil, errors.New("SegmentTemplate missing @media")
   }
   timescale := st.Timescale
   if timescale == 0 {
      timescale = 1
   }

   var timeline []segmentInfo
   if st.SegmentTimeline != nil {
      timeline = expandTimeline(st.SegmentTimeline, st.getStartNumber())
   } else {
      start := st.getStartNumber()
      if st.Duration == 0 {
         return nil, errors.New("SegmentTemplate missing @duration and SegmentTimeline")
      }
      periodSec := periodDur.Seconds()
      count := int(math.Ceil(periodSec * float64(timescale) / float64(st.Duration)))
      for i := 0; i < count; i++ {
         timeline = append(timeline, segmentInfo{
            number: start + i,
            time:   int64(i) * int64(st.Duration),
         })
      }
   }

   var urls []string
   for _, seg := range timeline {
      media := templateReplace(st.Media, repID, seg.number, seg.time)
      u, err := base.Parse(media)
      if err != nil {
         return nil, err
      }
      urls = append(urls, u.String())
   }
   return urls, nil
}

func expandSegmentList(base *url.URL, repID string, sl *SegmentList, periodDur time.Duration) ([]string, error) {
   var timeline []segmentInfo
   if sl.SegmentTimeline != nil {
      timeline = expandTimeline(sl.SegmentTimeline, sl.getStartNumber())
   } else {
      start := sl.getStartNumber()
      timescale := sl.Timescale
      if timescale == 0 {
         timescale = 1
      }
      periodSec := periodDur.Seconds()
      count := int(math.Ceil(periodSec * float64(timescale) / float64(sl.Duration)))
      for i := 0; i < count; i++ {
         timeline = append(timeline, segmentInfo{
            number: start + i,
            time:   int64(i) * int64(sl.Duration),
         })
      }
   }

   var urls []string
   if len(sl.SegmentURLs) > 0 {
      for i, su := range sl.SegmentURLs {
         if i >= len(timeline) {
            break
         }
         media := templateReplace(su.Media, repID, timeline[i].number, timeline[i].time)
         u, err := base.Parse(media)
         if err != nil {
            return nil, err
         }
         urls = append(urls, u.String())
      }
   } else {
      for _, seg := range timeline {
         media := templateReplace("$Number%09d$", repID, seg.number, seg.time)
         u, err := base.Parse(media)
         if err != nil {
            return nil, err
         }
         urls = append(urls, u.String())
      }
   }
   return urls, nil
}

type segmentInfo struct {
   number int
   time   int64
}

func expandTimeline(stl *SegmentTimeline, startNumber int) []segmentInfo {
   var out []segmentInfo
   number := startNumber
   time := int64(0)

   for _, s := range stl.S {
      if s.T != 0 {
         time = int64(s.T)
      }
      repeat := 1 + s.R // <--- corrected
      for i := 0; i < repeat; i++ {
         out = append(out, segmentInfo{
            number: number,
            time:   time,
         })
         number++
         time += int64(s.D)
      }
   }
   return out
}

/* ---------- helpers ---------- */

func resolveBase(parent *url.URL, bases ...string) *url.URL {
   u := parent
   for _, b := range bases {
      if b == "" {
         continue
      }
      rel, err := url.Parse(strings.TrimSpace(b))
      if err != nil {
         panic(err)
      }
      u = u.ResolveReference(rel)
   }
   return u
}

func parsePeriodDuration(mpd *MPD) (time.Duration, error) {
   if len(mpd.Periods) > 0 && mpd.Periods[0].Duration != "" {
      return parseISO8601Duration(mpd.Periods[0].Duration)
   }
   return 0, nil
}

func parseISO8601Duration(s string) (time.Duration, error) {
   s = strings.TrimSpace(s)
   if !strings.HasPrefix(s, "PT") {
      return 0, errors.New("invalid ISO-8601 duration: " + s)
   }
   s = s[2:]

   var h, m, sec float64
   for len(s) > 0 {
      switch {
      case strings.Contains(s, "H"):
         i := strings.Index(s, "H")
         v, err := strconv.ParseFloat(s[:i], 64)
         if err != nil {
            return 0, err
         }
         h = v
         s = s[i+1:]
      case strings.Contains(s, "M"):
         i := strings.Index(s, "M")
         v, err := strconv.ParseFloat(s[:i], 64)
         if err != nil {
            return 0, err
         }
         m = v
         s = s[i+1:]
      case strings.Contains(s, "S"):
         i := strings.Index(s, "S")
         v, err := strconv.ParseFloat(s[:i], 64)
         if err != nil {
            return 0, err
         }
         sec = v
         s = s[i+1:]
      default:
         return 0, errors.New("invalid ISO-8601 duration: " + s)
      }
   }
   return time.Duration(h*3600+m*60+sec) * time.Second, nil
}

// expand returns map[RepresentationID][]absoluteURL
func expand(mpd *MPD) (map[string][]string, error) {
   rootURL, err := url.Parse("http://test.test/test.mpd")
   if err != nil {
      return nil, err
   }

   // collect all URLs first, then deduplicate per representation
   raw := make(map[string][]string)

   periodDur, err := parsePeriodDuration(mpd)
   if err != nil {
      return nil, err
   }

   for pi, period := range mpd.Periods {
      periodDurCurrent := periodDur
      if period.Duration != "" {
         dur, err := parseISO8601Duration(period.Duration)
         if err != nil {
            return nil, err
         }
         periodDurCurrent = dur
      }

      periodBase := resolveBase(rootURL, mpd.BaseURL, period.BaseURL)
      for _, as := range period.AS {
         asBase := resolveBase(periodBase, as.BaseURL)
         for _, rep := range as.Representations {
            repBase := resolveBase(asBase, rep.BaseURL)
            id := rep.ID
            if id == "" {
               id = strconv.Itoa(pi)
            }

            var stmpl *SegmentTemplate
            switch {
            case rep.SegmentTemplate != nil:
               stmpl = rep.SegmentTemplate
            case as.SegmentTemplate != nil:
               stmpl = as.SegmentTemplate
            }

            var slist *SegmentList
            switch {
            case rep.SegmentList != nil:
               slist = rep.SegmentList
            case as.SegmentList != nil:
               slist = as.SegmentList
            }

            var urls []string
            switch {
            case stmpl != nil:
               u, err := expandTemplate(repBase, id, stmpl, periodDurCurrent)
               if err != nil {
                  return nil, err
               }
               urls = u
            case slist != nil:
               u, err := expandSegmentList(repBase, id, slist, periodDurCurrent)
               if err != nil {
                  return nil, err
               }
               urls = u
            default:
               u, err := repBase.Parse("")
               if err != nil {
                  return nil, err
               }
               urls = []string{u.String()}
            }

            // accumulate across periods
            raw[id] = append(raw[id], urls...)
         }
      }
   }

   // final deduplication per representation
   out := make(map[string][]string, len(raw))
   for id, list := range raw {
      out[id] = dedup(list)
   }
   return out, nil
}

var rePadding = regexp.MustCompile(`%(\d+)d`)

// replacePadding replaces "$Xyz%0xd$" with the zero-padded value of 'val'.
func replacePadding(s, token string, val int) string {
   re := regexp.MustCompile(regexp.QuoteMeta(token) + `%(\d+)d\$`)
   return re.ReplaceAllStringFunc(s, func(m string) string {
      // m is e.g. "$Number%08d$"
      n, _ := strconv.Atoi(m[len(token)+1 : len(m)-2]) // extract width
      return fmt.Sprintf("%0*d", n, val)
   })
}

func templateReplace(tmpl, repID string, number int, time int64) string {
	// 1. RepresentationID
	out := strings.ReplaceAll(tmpl, "$RepresentationID$", repID)

	// 2. $Number%0xd$  →  zero-padded number
	reN := regexp.MustCompile(`\$Number%(\d+)d\$`)
	out = reN.ReplaceAllStringFunc(out, func(m string) string {
		w, _ := strconv.Atoi(m[9 : len(m)-2])
		return fmt.Sprintf("%0*d", w, number)
	})

	// 3. $Time%0xd$  →  zero-padded time
	reT := regexp.MustCompile(`\$Time%(\d+)d\$`)
	out = reT.ReplaceAllStringFunc(out, func(m string) string {
		w, _ := strconv.Atoi(m[6 : len(m)-2])
		return fmt.Sprintf("%0*d", w, time)
	})

	// 4. plain $Number$ / $Time$
	out = strings.ReplaceAll(out, "$Number$", strconv.Itoa(number))
	out = strings.ReplaceAll(out, "$Time$", strconv.FormatInt(time, 10))

	return out
}
