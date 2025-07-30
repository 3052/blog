package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "log"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
)

type MPD struct {
   XMLName  xml.Name  `xml:"MPD"`
   BaseURLs []BaseURL `xml:"BaseURL"`
   Periods  []Period  `xml:"Period"`
}

type BaseURL struct {
   Value string `xml:",chardata"`
}

type Period struct {
   BaseURLs       []BaseURL       `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURLs        []BaseURL        `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURLs        []BaseURL        `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   Timescale       int              `xml:"timescale,attr"`
   Duration        int              `xml:"duration,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T *int64 `xml:"t,attr"`
   D int64  `xml:"d,attr"`
   R *int   `xml:"r,attr"`
}

func resolveBaseURL(root string, levels ...[]BaseURL) string {
   resolved := root
   for _, level := range levels {
      for _, bu := range level {
         resolved = resolveURL(resolved, strings.TrimSpace(bu.Value))
      }
   }
   return resolved
}

func resolveURL(base, ref string) string {
   baseURL, err := url.Parse(base)
   if err != nil {
      log.Fatalf("invalid base URL: %v", err)
   }
   refURL, err := url.Parse(ref)
   if err != nil {
      log.Fatalf("invalid relative URL: %v", err)
   }
   return baseURL.ResolveReference(refURL).String()
}

func getEffectiveTemplate(rep Representation, adapt AdaptationSet) *SegmentTemplate {
   if rep.SegmentTemplate != nil {
      return rep.SegmentTemplate
   }
   return adapt.SegmentTemplate
}

func substituteTemplate(template, repID string, number, timeVal int64) string {
   re := regexp.MustCompile(`\$(RepresentationID|Number|Time)(%0\d+d|%x|%X)?\$`)
   return re.ReplaceAllStringFunc(template, func(m string) string {
      parts := re.FindStringSubmatch(m)
      key := parts[1]
      format := parts[2]
      switch key {
      case "RepresentationID":
         return repID
      case "Number":
         return formatInt(number, format)
      case "Time":
         return formatInt(timeVal, format)
      default:
         return m
      }
   })
}

func formatInt(val int64, format string) string {
   if format == "" {
      return strconv.FormatInt(val, 10)
   }
   switch format {
   case "%x":
      return fmt.Sprintf("%x", val)
   case "%X":
      return fmt.Sprintf("%X", val)
   default:
      return fmt.Sprintf(format, val)
   }
}

func expandSegmentTimeline(tl *SegmentTimeline) []int64 {
   var times []int64
   if tl == nil {
      return times
   }
   var cur int64 = 0
   for i, s := range tl.S {
      repeat := 0
      if s.R != nil {
         repeat = *s.R
      }
      if i == 0 && s.T != nil {
         cur = *s.T
      } else if s.T != nil {
         cur = *s.T
      }
      for j := 0; j <= repeat; j++ {
         times = append(times, cur)
         cur += s.D
      }
   }
   return times
}

func main() {
   if len(os.Args) != 2 {
      fmt.Println("Usage: go run main.go <mpd_file>")
      os.Exit(1)
   }

   data, err := ioutil.ReadFile(os.Args[1])
   if err != nil {
      log.Fatalf("Failed to read MPD: %v", err)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      log.Fatalf("Failed to parse MPD XML: %v", err)
   }

   rootBase := "http://test.test/test.mpd"
   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      for _, adapt := range period.AdaptationSets {
         for _, rep := range adapt.Representations {
            tmpl := getEffectiveTemplate(rep, adapt)
            if tmpl == nil {
               continue
            }
            baseURL := resolveBaseURL(rootBase, mpd.BaseURLs, period.BaseURLs, adapt.BaseURLs, rep.BaseURLs)

            var urls []string

            // Initialization
            if tmpl.Initialization != "" {
               init := substituteTemplate(tmpl.Initialization, rep.ID, 0, 0)
               urls = append(urls, resolveURL(baseURL, init))
            }

            start := tmpl.StartNumber
            if start == 0 {
               start = 1
            }

            if tmpl.SegmentTimeline != nil {
               times := expandSegmentTimeline(tmpl.SegmentTimeline)
               for i, t := range times {
                  num := int64(start + i)
                  seg := substituteTemplate(tmpl.Media, rep.ID, num, t)
                  urls = append(urls, resolveURL(baseURL, seg))
               }
            } else {
               // Use startNumber and endNumber
               end := tmpl.EndNumber
               if end == 0 {
                  end = start + 4 // fallback to 5 segments
               }
               for n := start; n <= end; n++ {
                  timeVal := int64((n - start) * tmpl.Duration)
                  seg := substituteTemplate(tmpl.Media, rep.ID, int64(n), timeVal)
                  urls = append(urls, resolveURL(baseURL, seg))
               }
            }

            result[rep.ID] = urls
         }
      }
   }

   jsonOut, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      log.Fatalf("Failed to marshal JSON: %v", err)
   }
   fmt.Println(string(jsonOut))
}
