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

type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
   BaseURL                   string   `xml:"BaseURL"`
   Period                    Period   `xml:"Period"`
}

type Period struct {
   Duration       string          `xml:"duration,attr"`
   BaseURL        string          `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
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
   Duration        string           `xml:"duration,attr"`
   Timescale       string           `xml:"timescale,attr"`
   StartNumber     string           `xml:"startNumber,attr"`
   EndNumber       string           `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentList struct {
   InitializationURL *InitializationURL `xml:"Initialization"`
   SegmentURLs       []SegmentURL       `xml:"SegmentURL"`
}

type InitializationURL struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T string `xml:"t,attr"`
   D string `xml:"d,attr"`
   R string `xml:"r,attr"`
}

func main() {
   if len(os.Args) < 2 {
      fmt.Fprintln(os.Stderr, "usage: dash-expander <mpd-file>")
      os.Exit(1)
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
   root, err := url.Parse("http://test.test/test.mpd")
   if err != nil {
      panic(err)
   }
   result := map[string][]string{}
   for _, as := range mpd.Period.AdaptationSets {
      for _, rep := range as.Representations {
         base := root
         if mpd.BaseURL != "" {
            base = base.ResolveReference(mustParseURL(mpd.BaseURL))
         }
         if mpd.Period.BaseURL != "" {
            base = base.ResolveReference(mustParseURL(mpd.Period.BaseURL))
         }
         if as.BaseURL != "" {
            base = base.ResolveReference(mustParseURL(as.BaseURL))
         }
         if rep.BaseURL != "" {
            base = base.ResolveReference(mustParseURL(rep.BaseURL))
         }

         switch {
         case rep.SegmentList != nil:
            result[rep.ID] = handleSegmentList(base, rep.SegmentList)
         case as.SegmentList != nil:
            result[rep.ID] = handleSegmentList(base, as.SegmentList)
         case rep.SegmentTemplate != nil, as.SegmentTemplate != nil:
            result[rep.ID] = handleTemplate(rep, as, mpd, base)
         default:
            result[rep.ID] = []string{base.String()}
         }
      }
   }
   enc := json.NewEncoder(os.Stdout)
   enc.SetEscapeHTML(false)
   if err := enc.Encode(result); err != nil {
      panic(err)
   }
}

func handleSegmentList(base *url.URL, sl *SegmentList) []string {
   var urls []string
   if sl.InitializationURL != nil && sl.InitializationURL.SourceURL != "" {
      u := base.ResolveReference(mustParseURL(sl.InitializationURL.SourceURL))
      urls = append(urls, u.String())
   }
   for _, su := range sl.SegmentURLs {
      u := base.ResolveReference(mustParseURL(su.Media))
      urls = append(urls, u.String())
   }
   return urls
}

func handleTemplate(rep Representation, as AdaptationSet, mpd MPD, base *url.URL) []string {
   var template *SegmentTemplate
   if rep.SegmentTemplate != nil {
      template = rep.SegmentTemplate
   } else {
      template = as.SegmentTemplate
   }
   if template == nil {
      return nil
   }
   var segURLs []string
   if template.Initialization != "" {
      initURL := expandURL(template.Initialization, rep.ID, 0, 0)
      u := base.ResolveReference(mustParseURL(initURL))
      segURLs = append(segURLs, u.String())
   }
   media := template.Media
   if template.SegmentTimeline != nil {
      number := int64(1)
      if template.StartNumber != "" {
         n, err := strconv.ParseInt(template.StartNumber, 10, 64)
         if err != nil {
            panic(err)
         }
         number = n
      }
      time := int64(0)
      for _, s := range template.SegmentTimeline.S {
         if s.T != "" {
            var err error
            time, err = strconv.ParseInt(s.T, 10, 64)
            if err != nil {
               panic(err)
            }
         }
         d, err := strconv.ParseInt(s.D, 10, 64)
         if err != nil {
            panic(err)
         }
         r := 0
         if s.R != "" {
            r, err = strconv.Atoi(s.R)
            if err != nil {
               panic(err)
            }
         }
         for i := 0; i <= r; i++ {
            seg := expandURL(media, rep.ID, time, int(number))
            u := base.ResolveReference(mustParseURL(seg))
            segURLs = append(segURLs, u.String())
            time += d
            number++
         }
      }
   } else {
      start := int64(1)
      if template.StartNumber != "" {
         n, err := strconv.ParseInt(template.StartNumber, 10, 64)
         if err != nil {
            panic(err)
         }
         start = n
      }
      end := start
      if template.EndNumber != "" {
         e, err := strconv.ParseInt(template.EndNumber, 10, 64)
         if err != nil {
            panic(err)
         }
         end = e
      } else {
         if template.Duration == "" {
            panic("missing @duration")
         }
         dur, err := strconv.ParseInt(template.Duration, 10, 64)
         if err != nil {
            panic(err)
         }
         ts := int64(1)
         if template.Timescale != "" {
            ts, err = strconv.ParseInt(template.Timescale, 10, 64)
            if err != nil {
               panic(err)
            }
         }
         totalDur := parseISO8601Duration(mpd.MediaPresentationDuration)
         if totalDur == 0 {
            totalDur = parseISO8601Duration(mpd.Period.Duration)
         }
         if totalDur == 0 {
            panic("no duration")
         }
         segDur := float64(dur) / float64(ts)
         end = start + int64(totalDur/segDur) - 1
      }
      for n := start; n <= end; n++ {
         seg := expandURL(media, rep.ID, 0, int(n))
         u := base.ResolveReference(mustParseURL(seg))
         segURLs = append(segURLs, u.String())
      }
   }
   return segURLs
}

func mustParseURL(s string) *url.URL {
   u, err := url.Parse(s)
   if err != nil {
      panic(err)
   }
   return u
}

func expandURL(tmpl, repID string, time int64, number int) string {
   s := tmpl
   s = strings.ReplaceAll(s, "$RepresentationID$", repID)
   s = strings.ReplaceAll(s, "$Number$", strconv.Itoa(number))
   s = strings.ReplaceAll(s, "$Time$", strconv.FormatInt(time, 10))
   for {
      start := strings.Index(s, "%0")
      if start == -1 {
         break
      }
      end := start + 2
      for end < len(s) && s[end] >= '0' && s[end] <= '9' {
         end++
      }
      if end == start+2 {
         break
      }
      width, err := strconv.Atoi(s[start+2 : end])
      if err != nil {
         panic(err)
      }
      prefix := ""
      if start > 0 {
         prefix = s[:start]
      }
      suffix := ""
      if end < len(s) {
         suffix = s[end:]
      }
      var val int
      if strings.Contains(prefix, "$Number$") {
         val = number
      } else {
         val = int(time)
      }
      s = fmt.Sprintf("%s%0*d%s", prefix, width, val, suffix)
   }
   return s
}

func parseISO8601Duration(d string) float64 {
   if d == "" {
      return 0
   }
   d = strings.TrimPrefix(d, "P")
   var days, hours, minutes, seconds int
   var err error
   if idx := strings.Index(d, "D"); idx != -1 {
      days, err = strconv.Atoi(d[:idx])
      if err != nil {
         panic(err)
      }
      d = d[idx+1:]
   }
   if len(d) > 0 && d[0] == 'T' {
      d = d[1:]
   }
   if idx := strings.Index(d, "H"); idx != -1 {
      hours, err = strconv.Atoi(d[:idx])
      if err != nil {
         panic(err)
      }
      d = d[idx+1:]
   }
   if idx := strings.Index(d, "M"); idx != -1 {
      minutes, err = strconv.Atoi(d[:idx])
      if err != nil {
         panic(err)
      }
      d = d[idx+1:]
   }
   if idx := strings.Index(d, "S"); idx != -1 {
      secs := d[:idx]
      if strings.Contains(secs, ".") {
         secFloat, err := strconv.ParseFloat(secs, 64)
         if err != nil {
            panic(err)
         }
         seconds = int(secFloat)
      } else {
         seconds, err = strconv.Atoi(secs)
         if err != nil {
            panic(err)
         }
      }
   }
   return float64(((days*24+hours)*60+minutes)*60 + seconds)
}
