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
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   BaseURL        string          `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentList struct {
   SegmentURLs []SegmentURL `xml:"SegmentURL"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

type SegmentTemplate struct {
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   Duration        int              `xml:"duration,attr"`
   Timescale       int              `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   Segments []Segment `xml:"S"`
}

type Segment struct {
   T int64 `xml:"t,attr"`
   D int64 `xml:"d,attr"`
   R int64 `xml:"r,attr"`
}

func resolveURL(base, relative string) string {
   baseURL, _ := url.Parse(base)
   relURL, _ := url.Parse(relative)
   return baseURL.ResolveReference(relURL).String()
}

func replaceTokens(template, repID string, number int, time int64) string {
   s := template
   s = strings.ReplaceAll(s, "$RepresentationID$", repID)
   s = strings.ReplaceAll(s, "$Number$", strconv.Itoa(number))
   s = replaceNumberFormatted(s, number)
   s = strings.ReplaceAll(s, "$Time$", strconv.FormatInt(time, 10))
   return s
}

func replaceNumberFormatted(s string, number int) string {
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
      width, _ := strconv.Atoi(widthStr)
      padded := fmt.Sprintf("%0*d", width, number)
      s = s[:start] + padded + s[end+2:]
   }
   return s
}

func generateSegmentTimelineURLs(t *SegmentTemplate, repID, base string) []string {
   var urls []string
   if t.Initialization != "" {
      urls = append(urls, resolveURL(base, replaceTokens(t.Initialization, repID, 0, 0)))
   }

   startNum := t.StartNumber
   if startNum == 0 {
      startNum = 1
   }
   timescale := t.Timescale
   if timescale == 0 {
      timescale = 1
   }

   number := startNum
   var time int64
   for _, seg := range t.SegmentTimeline.Segments {
      count := seg.R
      if count == 0 {
         count = 1
      } else {
         count++
      }
      if seg.T != 0 {
         time = seg.T
      }
      for i := int64(0); i < count; i++ {
         url := replaceTokens(t.Media, repID, number, time)
         urls = append(urls, resolveURL(base, url))
         time += seg.D
         number++
         if t.EndNumber > 0 && number > t.EndNumber {
            return urls
         }
      }
   }
   return urls
}

func generateSegmentTemplateURLs(t *SegmentTemplate, repID, base string) []string {
   var urls []string
   start := t.StartNumber
   if start == 0 {
      start = 1
   }
   end := start + 4
   if t.EndNumber >= start {
      end = t.EndNumber
   }
   timescale := t.Timescale
   if timescale == 0 {
      timescale = 1
   }
   duration := t.Duration

   if t.Initialization != "" {
      urls = append(urls, resolveURL(base, replaceTokens(t.Initialization, repID, start, 0)))
   }

   for number := start; number <= end; number++ {
      time := int64(duration*(number-start)) / int64(timescale)
      url := replaceTokens(t.Media, repID, number, time)
      urls = append(urls, resolveURL(base, url))
   }
   return urls
}

func main() {
   if len(os.Args) < 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path>")
      os.Exit(1)
   }
   filePath := os.Args[1]
   data, err := os.ReadFile(filePath)
   if err != nil {
      fmt.Println("Failed to read MPD:", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Println("Failed to parse XML:", err)
      os.Exit(1)
   }

   base := "http://test.test/test.mpd"
   if mpd.BaseURL != "" {
      base = resolveURL(base, mpd.BaseURL)
   }

   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      pBase := base
      if period.BaseURL != "" {
         pBase = resolveURL(pBase, period.BaseURL)
      }
      for _, aset := range period.AdaptationSets {
         aBase := pBase
         if aset.BaseURL != "" {
            aBase = resolveURL(aBase, aset.BaseURL)
         }
         for _, rep := range aset.Representations {
            rBase := aBase
            if rep.BaseURL != "" {
               rBase = resolveURL(rBase, rep.BaseURL)
            }

            var urls []string
            switch {
            case rep.SegmentList != nil:
               for _, seg := range rep.SegmentList.SegmentURLs {
                  urls = append(urls, resolveURL(rBase, seg.Media))
               }
            case rep.SegmentTemplate != nil:
               if rep.SegmentTemplate.SegmentTimeline != nil {
                  urls = generateSegmentTimelineURLs(rep.SegmentTemplate, rep.ID, rBase)
               } else {
                  urls = generateSegmentTemplateURLs(rep.SegmentTemplate, rep.ID, rBase)
               }
            case aset.SegmentList != nil:
               for _, seg := range aset.SegmentList.SegmentURLs {
                  urls = append(urls, resolveURL(rBase, seg.Media))
               }
            case aset.SegmentTemplate != nil:
               if aset.SegmentTemplate.SegmentTimeline != nil {
                  urls = generateSegmentTimelineURLs(aset.SegmentTemplate, rep.ID, rBase)
               } else {
                  urls = generateSegmentTemplateURLs(aset.SegmentTemplate, rep.ID, rBase)
               }
            }

            result[rep.ID] = urls
         }
      }
   }

   output, _ := json.MarshalIndent(result, "", "  ")
   fmt.Println(string(output))
}
