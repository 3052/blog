package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "net/url"
   "os"
   "strings"
)

const baseURL = "http://test.test/test.mpd"

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   Periods []Period `xml:"Period"`
   BaseURL *string  `xml:"BaseURL"`
}

type Period struct {
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
   BaseURL        *string         `xml:"BaseURL"`
}

type AdaptationSet struct {
   Representations []Representation `xml:"Representation"`
   BaseURL         *string          `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         *string          `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentTemplate struct {
   Timescale       int              `xml:"timescale,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   Segments []S `xml:"S"`
}

type S struct {
   T int64 `xml:"t,attr"`
   D int64 `xml:"d,attr"`
   R int   `xml:"r,attr"`
}

func resolveURL(base, ref string) string {
   u, err := url.Parse(base)
   if err != nil {
      return ""
   }
   refURL, err := url.Parse(ref)
   if err != nil {
      return ""
   }
   return u.ResolveReference(refURL).String()
}

func buildBaseURL(mpdBase, periodBase, asBase, repBase string) string {
   base := baseURL
   if mpdBase != "" {
      base = resolveURL(base, mpdBase)
   }
   if periodBase != "" {
      base = resolveURL(base, periodBase)
   }
   if asBase != "" {
      base = resolveURL(base, asBase)
   }
   if repBase != "" {
      base = resolveURL(base, repBase)
   }
   return base
}

func expandSegmentTimeline(timeline *SegmentTimeline) []int64 {
   var times []int64
   var currentTime int64
   for _, seg := range timeline.Segments {
      repeat := seg.R
      if repeat < 0 {
         repeat = 0
      }
      if seg.T != 0 {
         currentTime = seg.T
      }
      for i := 0; i <= repeat; i++ {
         times = append(times, currentTime)
         currentTime += seg.D
      }
   }
   return times
}

func main() {
   if len(os.Args) != 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path>")
      os.Exit(1)
   }

   mpdPath := os.Args[1]
   data, err := ioutil.ReadFile(mpdPath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading MPD file: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD XML: %v\n", err)
      os.Exit(1)
   }

   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      periodBase := ""
      if period.BaseURL != nil {
         periodBase = *period.BaseURL
      }
      for _, aset := range period.AdaptationSets {
         asBase := ""
         if aset.BaseURL != nil {
            asBase = *aset.BaseURL
         }
         for _, rep := range aset.Representations {
            repBase := ""
            if rep.BaseURL != nil {
               repBase = *rep.BaseURL
            }
            base := buildBaseURL("", periodBase, asBase, repBase)

            var tmpl SegmentTemplate
            if rep.SegmentTemplate != nil {
               tmpl = *rep.SegmentTemplate
            } else if aset.SegmentTemplate != nil {
               tmpl = *aset.SegmentTemplate
            } else {
               continue
            }

            var urls []string

            if tmpl.Initialization != "" {
               init := strings.ReplaceAll(tmpl.Initialization, "$RepresentationID$", rep.ID)
               urls = append(urls, resolveURL(base, init))
            }

            if tmpl.SegmentTimeline != nil {
               times := expandSegmentTimeline(tmpl.SegmentTimeline)
               for _, t := range times {
                  url := strings.ReplaceAll(tmpl.Media, "$RepresentationID$", rep.ID)
                  url = strings.ReplaceAll(url, "$Time$", fmt.Sprintf("%d", t))
                  urls = append(urls, resolveURL(base, url))
               }
            } else if strings.Contains(tmpl.Media, "$Number$") {
               start := tmpl.StartNumber
               if start == 0 {
                  start = 1
               }
               end := tmpl.EndNumber
               count := 5
               if end > 0 && end >= start {
                  count = end - start + 1
               }
               for i := 0; i < count; i++ {
                  number := start + i
                  url := strings.ReplaceAll(tmpl.Media, "$RepresentationID$", rep.ID)
                  url = strings.ReplaceAll(url, "$Number$", fmt.Sprintf("%d", number))
                  urls = append(urls, resolveURL(base, url))
               }
            }

            result[rep.ID] = urls
         }
      }
   }

   output, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
      os.Exit(1)
   }
   fmt.Println(string(output))
}
