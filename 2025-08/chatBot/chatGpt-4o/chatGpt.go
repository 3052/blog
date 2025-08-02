package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "math"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
)

type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
   Periods                   []Period `xml:"Period"`
}

type Period struct {
   Duration        string           `xml:"duration,attr"`
   BaseURL         *BaseURL         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   AdaptationSets  []AdaptationSet  `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         *BaseURL         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type BaseURL struct {
   URL string `xml:",chardata"`
}

type SegmentTemplate struct {
   Timescale       int              `xml:"timescale,attr"`
   Duration        int              `xml:"duration,attr"`
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []Segment `xml:"S"`
}

type Segment struct {
   T int64 `xml:"t,attr"`
   D int64 `xml:"d,attr"`
   R int   `xml:"r,attr"`
}

type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

var varPattern = regexp.MustCompile(`\$(Number|Time)(%0\d+d)?\$`)

func formatTemplate(template string, vars map[string]int64) string {
   return varPattern.ReplaceAllStringFunc(template, func(m string) string {
      matches := varPattern.FindStringSubmatch(m)
      key := matches[1]
      format := matches[2]
      value, ok := vars[key]
      if !ok {
         return m
      }
      if format == "" {
         return fmt.Sprintf("%d", value)
      }
      return fmt.Sprintf(format, value)
   })
}

func resolveURL(base, rel string) string {
   baseURL, _ := url.Parse(base)
   relURL, _ := url.Parse(strings.TrimSpace(rel))
   return baseURL.ResolveReference(relURL).String()
}

func parseISODuration(s string) float64 {
   s = strings.ToUpper(strings.TrimPrefix(s, "PT"))
   var h, m int
   var sec float64
   for len(s) > 0 {
      if i := strings.IndexAny(s, "HMS"); i >= 0 {
         val := s[:i]
         unit := s[i]
         s = s[i+1:]
         switch unit {
         case 'H':
            h, _ = strconv.Atoi(val)
         case 'M':
            m, _ = strconv.Atoi(val)
         case 'S':
            sec, _ = strconv.ParseFloat(val, 64)
         }
      } else {
         break
      }
   }
   return float64(h*3600+m*60) + sec
}

func main() {
   if len(os.Args) < 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path>")
      return
   }
   filePath := os.Args[1]
   data, err := ioutil.ReadFile(filePath)
   if err != nil {
      panic(err)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      panic(err)
   }

   baseMPD := "http://test.test/test.mpd"
   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      periodBase := baseMPD
      if period.BaseURL != nil {
         periodBase = resolveURL(baseMPD, period.BaseURL.URL)
      }

      for _, aset := range period.AdaptationSets {
         st := aset.SegmentTemplate
         sl := aset.SegmentList

         for _, rep := range aset.Representations {
            var urls []string
            repBase := periodBase
            if rep.BaseURL != nil {
               repBase = resolveURL(periodBase, rep.BaseURL.URL)
            }

            rst := rep.SegmentTemplate
            if rst == nil {
               rst = st
            }
            rsl := rep.SegmentList
            if rsl == nil {
               rsl = sl
            }

            if rst != nil {
               // Initialization
               if rst.Initialization != "" {
                  initURL := strings.ReplaceAll(rst.Initialization, "$RepresentationID$", rep.ID)
                  urls = append(urls, resolveURL(repBase, initURL))
               }

               // SegmentTimeline
               if rst.SegmentTimeline != nil {
                  number := rst.StartNumber
                  if number == 0 {
                     number = 1
                  }
                  t := int64(0)
                  for _, s := range rst.SegmentTimeline.S {
                     count := 1 + s.R
                     if s.R < 0 {
                        count = 1
                     }
                     for i := 0; i < count; i++ {
                        if s.T > 0 && i == 0 {
                           t = s.T
                        }
                        media := strings.ReplaceAll(rst.Media, "$RepresentationID$", rep.ID)
                        media = formatTemplate(media, map[string]int64{
                           "Number": int64(number),
                           "Time":   t,
                        })
                        urls = append(urls, resolveURL(repBase, media))
                        t += s.D
                        number++
                     }
                  }
               } else {
                  start := rst.StartNumber
                  if start == 0 {
                     start = 1
                  }
                  var end int
                  if rst.EndNumber > 0 {
                     end = rst.EndNumber
                  } else if rst.Duration > 0 {
                     timescale := rst.Timescale
                     if timescale == 0 {
                        timescale = 1
                     }
                     var seconds float64
                     if period.Duration != "" {
                        seconds = parseISODuration(period.Duration)
                     } else if mpd.MediaPresentationDuration != "" {
                        seconds = parseISODuration(mpd.MediaPresentationDuration)
                     }
                     if seconds > 0 {
                        segCount := int(math.Ceil((seconds * float64(timescale)) / float64(rst.Duration)))
                        end = start + segCount - 1
                     } else {
                        end = start + 4
                     }
                  } else {
                     end = start + 4
                  }
                  for i := start; i <= end; i++ {
                     media := strings.ReplaceAll(rst.Media, "$RepresentationID$", rep.ID)
                     media = formatTemplate(media, map[string]int64{
                        "Number": int64(i),
                     })
                     urls = append(urls, resolveURL(repBase, media))
                  }
               }
            } else if rsl != nil {
               if rsl.Initialization != nil {
                  urls = append(urls, resolveURL(repBase, rsl.Initialization.SourceURL))
               }
               for _, su := range rsl.SegmentURLs {
                  urls = append(urls, resolveURL(repBase, su.Media))
               }
            } else if rep.BaseURL != nil {
               urls = append(urls, repBase)
            }

            result[rep.ID] = append(result[rep.ID], urls...)
         }
      }
   }

   out, _ := json.MarshalIndent(result, "", "  ")
   fmt.Println(string(out))
}
