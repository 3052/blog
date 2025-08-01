package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "net/url"
   "os"
   "regexp"
   "strconv"
)

const baseMPDURL = "http://test.test/test.mpd"
const fallbackSegmentCount = 5

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL *string  `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   BaseURL        *string         `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         *string          `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         *string          `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Initialization  *Initialization  `xml:"Initialization"`
}

type SegmentTemplate struct {
   Timescale       int64            `xml:"timescale,attr"`
   StartNumber     int64            `xml:"startNumber,attr"`
   EndNumber       int64            `xml:"endNumber,attr"`
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
   R int64 `xml:"r,attr"`
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

func main() {
   if len(os.Args) != 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path>")
      os.Exit(1)
   }

   mpdFile := os.Args[1]
   data, err := ioutil.ReadFile(mpdFile)
   if err != nil {
      fmt.Printf("Failed to read file: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Printf("Failed to parse MPD: %v\n", err)
      os.Exit(1)
   }

   base, _ := url.Parse(baseMPDURL)
   if mpd.BaseURL != nil {
      base = resolve(base, *mpd.BaseURL)
   }

   output := make(map[string][]string)

   for _, period := range mpd.Periods {
      basePeriod := base
      if period.BaseURL != nil {
         basePeriod = resolve(base, *period.BaseURL)
      }

      for _, aset := range period.AdaptationSets {
         baseASet := basePeriod
         if aset.BaseURL != nil {
            baseASet = resolve(basePeriod, *aset.BaseURL)
         }

         for _, rep := range aset.Representations {
            baseRep := baseASet
            if rep.BaseURL != nil {
               baseRep = resolve(baseASet, *rep.BaseURL)
            }

            var urls []string
            handled := false

            // SegmentList
            sl := firstNonNilSegmentList(rep.SegmentList, aset.SegmentList)
            if sl != nil {
               if sl.Initialization != nil {
                  urls = append(urls, resolve(baseRep, sl.Initialization.SourceURL).String())
               }
               for _, s := range sl.SegmentURLs {
                  urls = append(urls, resolve(baseRep, s.Media).String())
               }
               handled = true
            }

            // SegmentTemplate
            if !handled {
               st := resolveSegmentTemplate(rep.SegmentTemplate, aset.SegmentTemplate)
               if st != nil && st.Media != "" {
                  if st.Initialization != "" {
                     urls = append(urls, resolve(baseRep, replacePlaceholders(st.Initialization, rep.ID, 0, 0)).String())
                  }
                  if st.SegmentTimeline != nil {
                     urls = append(urls, generateTimelineSegments(st, baseRep, rep.ID)...)
                  } else {
                     urls = append(urls, generateNumberedSegments(st, baseRep, rep.ID)...)
                  }
                  handled = true
               }
            }

            // Representation Initialization only
            if !handled && rep.Initialization != nil {
               urls = append(urls, resolve(baseRep, rep.Initialization.SourceURL).String())
               handled = true
            }

            // Final fallback to BaseURL as single segment
            if !handled {
               urls = append(urls, baseRep.String())
            }

            output[rep.ID] = urls
         }
      }
   }

   j, _ := json.MarshalIndent(output, "", "  ")
   fmt.Println(string(j))
}

func resolve(base *url.URL, ref string) *url.URL {
   u, err := url.Parse(ref)
   if err != nil {
      return base
   }
   return base.ResolveReference(u)
}

func resolveSegmentTemplate(rep, aset *SegmentTemplate) *SegmentTemplate {
   if rep != nil {
      return rep
   }
   return aset
}

func firstNonNilSegmentList(rep, aset *SegmentList) *SegmentList {
   if rep != nil {
      return rep
   }
   return aset
}

func replacePlaceholders(template, repID string, number, time int64) string {
   re := regexp.MustCompile(`\$(\w+)(%0(\d+)d)?\$`)
   return re.ReplaceAllStringFunc(template, func(m string) string {
      parts := re.FindStringSubmatch(m)
      key := parts[1]
      width := parts[3]
      switch key {
      case "RepresentationID":
         return repID
      case "Number":
         return formatNumber(number, width)
      case "Time":
         return formatNumber(time, width)
      default:
         return m
      }
   })
}

func formatNumber(n int64, width string) string {
   if width == "" {
      return strconv.FormatInt(n, 10)
   }
   w, _ := strconv.Atoi(width)
   return fmt.Sprintf("%0*d", w, n)
}

func generateNumberedSegments(st *SegmentTemplate, base *url.URL, repID string) []string {
   var urls []string
   start := st.StartNumber
   if start == 0 {
      start = 1
   }
   end := st.EndNumber
   count := fallbackSegmentCount
   if end >= start && end > 0 {
      count = int(end - start + 1)
   }

   for i := int64(0); i < int64(count); i++ {
      num := start + i
      urlStr := replacePlaceholders(st.Media, repID, num, 0)
      urls = append(urls, resolve(base, urlStr).String())
   }
   return urls
}

func generateTimelineSegments(st *SegmentTemplate, base *url.URL, repID string) []string {
   var urls []string
   var currentTime int64
   number := st.StartNumber
   if number == 0 {
      number = 1
   }
   for _, s := range st.SegmentTimeline.Segments {
      repeat := s.R
      if repeat < 0 {
         repeat = 0
      }
      if s.T != 0 {
         currentTime = s.T
      }
      for i := int64(0); i <= repeat; i++ {
         urlStr := replacePlaceholders(st.Media, repID, number, currentTime)
         urls = append(urls, resolve(base, urlStr).String())
         currentTime += s.D
         number++
      }
   }
   return urls
}
