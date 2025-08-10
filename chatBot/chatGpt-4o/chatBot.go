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

const baseMPDURL = "http://test.test/test.mpd"

type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   BaseURL                   string   `xml:"BaseURL"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
   Periods                   []Period `xml:"Period"`
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
   StartNumber     *int             `xml:"startNumber,attr"` // pointer to detect missing vs 0
   EndNumber       int              `xml:"endNumber,attr"`
   Timescale       int              `xml:"timescale,attr"`
   Duration        int              `xml:"duration,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   Segments []S `xml:"S"`
}

type S struct {
   T int `xml:"t,attr"`
   D int `xml:"d,attr"`
   R int `xml:"r,attr"`
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
      fmt.Println("Usage: go run main.go <mpd_file>")
      os.Exit(1)
   }

   data, err := ioutil.ReadFile(os.Args[1])
   if err != nil {
      fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Fprintf(os.Stderr, "XML parse error: %v\n", err)
      os.Exit(1)
   }

   base, _ := url.Parse(baseMPDURL)
   base = base.ResolveReference(&url.URL{Path: "."})
   mpdBase := resolve(base, mpd.BaseURL)

   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      periodBase := resolve(mpdBase, period.BaseURL)

      periodDurationSeconds := parseISODuration(period.Duration)
      if periodDurationSeconds == 0 {
         periodDurationSeconds = parseISODuration(mpd.MediaPresentationDuration)
      }

      for _, aset := range period.AdaptationSets {
         asetBase := resolve(periodBase, aset.BaseURL)

         for _, rep := range aset.Representations {
            repBase := resolve(asetBase, rep.BaseURL)
            segList := []string{}

            // --- SegmentList ---
            list := rep.SegmentList
            if list == nil {
               list = aset.SegmentList
            }
            if list != nil {
               if list.Initialization != nil && list.Initialization.SourceURL != "" {
                  segList = append(segList, resolve(repBase, list.Initialization.SourceURL).String())
               }
               for _, seg := range list.SegmentURLs {
                  segList = append(segList, resolve(repBase, seg.Media).String())
               }
               result[rep.ID] = append(result[rep.ID], segList...)
               continue
            }

            // --- SegmentTemplate ---
            tmpl := rep.SegmentTemplate
            if tmpl == nil {
               tmpl = aset.SegmentTemplate
            }
            if tmpl != nil && tmpl.Media != "" {
               if tmpl.Initialization != "" {
                  init := resolveSegmentURL(tmpl.Initialization, rep.ID, 0, 0)
                  segList = append(segList, resolve(repBase, init).String())
               }

               start := 1
               if tmpl.StartNumber != nil {
                  start = *tmpl.StartNumber
               }

               timescale := tmpl.Timescale
               if timescale == 0 {
                  timescale = 1
               }

               if tmpl.SegmentTimeline != nil {
                  seq := start
                  currentTime := 0
                  for _, s := range tmpl.SegmentTimeline.Segments {
                     if s.T != 0 {
                        currentTime = s.T
                     }
                     repeat := s.R
                     if repeat < 0 {
                        repeat = 0
                     }
                     for i := 0; i <= repeat; i++ {
                        url := resolveSegmentURL(tmpl.Media, rep.ID, seq, currentTime)
                        segList = append(segList, resolve(repBase, url).String())
                        currentTime += s.D
                        seq++
                     }
                  }
               } else if tmpl.EndNumber > 0 {
                  for n := start; n <= tmpl.EndNumber; n++ {
                     url := resolveSegmentURL(tmpl.Media, rep.ID, n, 0)
                     segList = append(segList, resolve(repBase, url).String())
                  }
               } else if tmpl.Duration > 0 && periodDurationSeconds > 0 {
                  count := int(math.Ceil(periodDurationSeconds * float64(timescale) / float64(tmpl.Duration)))
                  for i := 0; i < count; i++ {
                     n := start + i
                     url := resolveSegmentURL(tmpl.Media, rep.ID, n, 0)
                     segList = append(segList, resolve(repBase, url).String())
                  }
               } else {
                  for i := 0; i < 5; i++ {
                     n := start + i
                     url := resolveSegmentURL(tmpl.Media, rep.ID, n, 0)
                     segList = append(segList, resolve(repBase, url).String())
                  }
               }

               result[rep.ID] = append(result[rep.ID], segList...)
               continue
            }

            // --- BaseURL fallback ---
            if rep.BaseURL != "" {
               result[rep.ID] = append(result[rep.ID], repBase.String())
            }
         }
      }
   }

   jsonData, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "JSON encode error: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonData))
}

func resolve(base *url.URL, ref string) *url.URL {
   u, err := url.Parse(ref)
   if err != nil {
      return base
   }
   return base.ResolveReference(u)
}

func resolveSegmentURL(template, repID string, number int, time int) string {
   s := template

   re := regexp.MustCompile(`\$(Number|Time)(?:%0(\d+)d)?\$`)
   s = re.ReplaceAllStringFunc(s, func(match string) string {
      m := re.FindStringSubmatch(match)
      var val int
      if m[1] == "Number" {
         val = number
      } else {
         val = time
      }
      if m[2] != "" {
         width, _ := strconv.Atoi(m[2])
         return fmt.Sprintf("%0*d", width, val)
      }
      return strconv.Itoa(val)
   })

   s = strings.ReplaceAll(s, "$RepresentationID$", repID)

   return s
}

func parseISODuration(dur string) float64 {
   if dur == "" {
      return 0
   }
   re := regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?`)
   m := re.FindStringSubmatch(dur)
   if m == nil {
      return 0
   }
   hours, _ := strconv.Atoi(defaultZero(m[1]))
   minutes, _ := strconv.Atoi(defaultZero(m[2]))
   seconds, _ := strconv.ParseFloat(defaultZero(m[3]), 64)
   return float64(hours*3600+minutes*60) + seconds
}

func defaultZero(s string) string {
   if s == "" {
      return "0"
   }
   return s
}
