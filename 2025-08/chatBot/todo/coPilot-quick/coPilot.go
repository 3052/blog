package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io"
   "math"
   "net/url"
   "os"
   "strconv"
   "strings"
)

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   BaseURL        string          `xml:"BaseURL"`
   Duration       string          `xml:"duration,attr"`
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
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   Duration        int              `xml:"duration,attr"`
   Timescale       int              `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   Segments []S `xml:"S"`
}

type S struct {
   T int64 `xml:"t,attr"`
   D int64 `xml:"d,attr"`
   R int64 `xml:"r,attr"`
}

type SegmentList struct {
   Initialization Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL   `xml:"SegmentURL"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

func main() {
   if len(os.Args) < 2 {
      fmt.Println("Usage: go run main.go <path/to/mpd.xml>")
      return
   }

   base := "http://test.test/test.mpd"
   mpd := parseMPD(os.Args[1])
   output := make(map[string][]string)

   for _, period := range mpd.Periods {
      periodDuration := parseISODuration(period.Duration)
      for _, as := range period.AdaptationSets {
         for _, rep := range as.Representations {
            baseURL := resolveBaseURL(base, mpd.BaseURL, period.BaseURL, as.BaseURL, rep.BaseURL)
            var segments []string
            tmpl := rep.SegmentTemplate
            if tmpl == nil {
               tmpl = as.SegmentTemplate
            }

            if tmpl != nil {
               segments = generateSegmentTemplateURLs(baseURL, rep.ID, tmpl, periodDuration)
            } else if rep.SegmentList != nil {
               segments = generateSegmentListURLs(baseURL, rep.SegmentList)
            } else if rep.BaseURL != "" {
               segments = []string{baseURL}
            }

            output[rep.ID] = append(output[rep.ID], segments...)
         }
      }
   }

   b, _ := json.MarshalIndent(output, "", "  ")
   fmt.Println(string(b))
}

func parseMPD(path string) MPD {
   f, err := os.Open(path)
   if err != nil {
      panic(err)
   }
   defer f.Close()
   data, err := io.ReadAll(f)
   if err != nil {
      panic(err)
   }
   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      panic(err)
   }
   return mpd
}

func resolveBaseURL(base string, urls ...string) string {
   u := mustParseURL(base)
   for _, b := range urls {
      if b != "" {
         ref := mustParseURL(b)
         u = u.ResolveReference(ref)
      }
   }
   return u.String()
}

func mustParseURL(s string) *url.URL {
   u, err := url.Parse(s)
   if err != nil {
      panic(err)
   }
   return u
}

func generateSegmentTemplateURLs(base, repID string, tmpl *SegmentTemplate, periodSeconds float64) []string {
   var urls []string

   if tmpl.Initialization != "" {
      init := replaceTokens(tmpl.Initialization, repID, tmpl.StartNumber, 0)
      url := mustParseURL(base).ResolveReference(mustParseURL(init)).String()
      urls = append(urls, url)
   }

   if tmpl.SegmentTimeline != nil {
      var currentTime int64 = 0
      segmentNumber := tmpl.StartNumber
      if segmentNumber == 0 {
         segmentNumber = 1
      }
      for _, s := range tmpl.SegmentTimeline.Segments {
         repeat := s.R
         if repeat < 0 {
            repeat = 0
         }
         if s.T != 0 {
            currentTime = s.T
         }
         for i := int64(0); i <= repeat; i++ {
            media := replaceTokens(tmpl.Media, repID, segmentNumber, currentTime)
            url := mustParseURL(base).ResolveReference(mustParseURL(media)).String()
            urls = append(urls, url)
            currentTime += s.D
            segmentNumber++
         }
      }
   } else if tmpl.EndNumber > 0 {
      start := tmpl.StartNumber
      if start == 0 {
         start = 1
      }
      for i := start; i <= tmpl.EndNumber; i++ {
         media := replaceTokens(tmpl.Media, repID, i, 0)
         url := mustParseURL(base).ResolveReference(mustParseURL(media)).String()
         urls = append(urls, url)
      }
   } else if tmpl.Duration > 0 && periodSeconds > 0 {
      timescale := tmpl.Timescale
      if timescale == 0 {
         timescale = 1
      }
      start := tmpl.StartNumber
      if start == 0 {
         start = 1
      }
      count := int(math.Ceil(periodSeconds * float64(timescale) / float64(tmpl.Duration)))
      for i := 0; i < count; i++ {
         num := start + i
         media := replaceTokens(tmpl.Media, repID, num, 0)
         url := mustParseURL(base).ResolveReference(mustParseURL(media)).String()
         urls = append(urls, url)
      }
   }

   return urls
}

func generateSegmentListURLs(base string, list *SegmentList) []string {
   var urls []string
   if list.Initialization.SourceURL != "" {
      init := mustParseURL(base).ResolveReference(mustParseURL(list.Initialization.SourceURL)).String()
      urls = append(urls, init)
   }
   for _, s := range list.SegmentURLs {
      url := mustParseURL(base).ResolveReference(mustParseURL(s.Media)).String()
      urls = append(urls, url)
   }
   return urls
}

func replaceTokens(template, repID string, number int, time int64) string {
   s := template
   s = strings.ReplaceAll(s, "$RepresentationID$", repID)
   s = strings.ReplaceAll(s, "$Number$", strconv.Itoa(number))
   s = strings.ReplaceAll(s, "$Time$", strconv.FormatInt(time, 10))

   for {
      start := strings.Index(s, "$Number%0")
      if start == -1 {
         break
      }
      end := strings.Index(s[start:], "d$")
      if end == -1 {
         break
      }
      end += start
      widthStr := s[start+9 : end]
      width, err := strconv.Atoi(widthStr)
      if err != nil {
         break
      }
      s = s[:start] + fmt.Sprintf("%0*d", width, number) + s[end+2:]
   }
   return s
}

func parseISODuration(iso string) float64 {
   iso = strings.TrimPrefix(iso, "PT")
   var seconds float64
   var buf string

   for _, c := range iso {
      switch c {
      case 'H':
         n, _ := strconv.ParseFloat(buf, 64)
         seconds += n * 3600
         buf = ""
      case 'M':
         n, _ := strconv.ParseFloat(buf, 64)
         seconds += n * 60
         buf = ""
      case 'S':
         n, _ := strconv.ParseFloat(buf, 64)
         seconds += n
         buf = ""
      default:
         buf += string(c)
      }
   }
   return seconds
}
