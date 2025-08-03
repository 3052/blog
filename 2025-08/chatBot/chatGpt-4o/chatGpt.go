package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "math"
   "net/url"
   "os"
   "strconv"
   "strings"
)

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   Periods []Period `xml:"Period"`
   BaseURL string   `xml:"BaseURL"`
}

type Period struct {
   BaseURL        string          `xml:"BaseURL"`
   Duration       string          `xml:"duration,attr"` // ISO 8601 duration (e.g., PT30S)
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
   Timescale       int              `xml:"timescale,attr"`
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
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
   Timescale      int             `xml:"timescale,attr"`
   Duration       int             `xml:"duration,attr"`
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

func mustParseURL(raw string) *url.URL {
   u, err := url.Parse(raw)
   if err != nil {
      fmt.Println("Invalid URL:", err)
      os.Exit(1)
   }
   return u
}

func resolve(base *url.URL, ref string) *url.URL {
   u, err := url.Parse(ref)
   if err != nil {
      fmt.Println("Invalid relative URL:", ref)
      os.Exit(1)
   }
   return base.ResolveReference(u)
}

func parseDurationSeconds(iso string) float64 {
   if !strings.HasPrefix(iso, "PT") {
      return 0
   }
   iso = strings.TrimPrefix(iso, "PT")
   var total float64
   var value string
   for _, r := range iso {
      switch r {
      case 'H':
         if h, err := strconv.ParseFloat(value, 64); err == nil {
            total += h * 3600
         }
         value = ""
      case 'M':
         if m, err := strconv.ParseFloat(value, 64); err == nil {
            total += m * 60
         }
         value = ""
      case 'S':
         if s, err := strconv.ParseFloat(value, 64); err == nil {
            total += s
         }
         value = ""
      default:
         value += string(r)
      }
   }
   return total
}

func main() {
   if len(os.Args) != 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path>")
      os.Exit(1)
   }

   data, err := ioutil.ReadFile(os.Args[1])
   if err != nil {
      fmt.Println("Failed to read MPD:", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Println("Failed to parse MPD:", err)
      os.Exit(1)
   }

   rootBase := mustParseURL("http://test.test/test.mpd")

   // Apply <MPD><BaseURL> if present
   if mpd.BaseURL != "" {
      rootBase = resolve(rootBase, mpd.BaseURL)
   }

   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      periodBase := rootBase
      if period.BaseURL != "" {
         periodBase = resolve(rootBase, period.BaseURL)
      }
      periodDurSec := parseDurationSeconds(period.Duration)

      for _, aset := range period.AdaptationSets {
         asetBase := periodBase
         if aset.BaseURL != "" {
            asetBase = resolve(periodBase, aset.BaseURL)
         }

         tmpl := aset.SegmentTemplate
         slist := aset.SegmentList

         for _, rep := range aset.Representations {
            repBase := asetBase
            if rep.BaseURL != "" {
               repBase = resolve(asetBase, rep.BaseURL)
            }

            if rep.SegmentTemplate != nil {
               tmpl = rep.SegmentTemplate
            }
            if rep.SegmentList != nil {
               slist = rep.SegmentList
            }

            var urls []string

            // --- SegmentList ---
            if slist != nil {
               if slist.Initialization != nil && slist.Initialization.SourceURL != "" {
                  initURL := resolve(repBase, slist.Initialization.SourceURL)
                  urls = append(urls, initURL.String())
               }
               for _, seg := range slist.SegmentURLs {
                  full := resolve(repBase, seg.Media)
                  urls = append(urls, full.String())
               }
               result[rep.ID] = append(result[rep.ID], urls...)
               continue
            }

            // --- SegmentTemplate ---
            if tmpl != nil && tmpl.Media != "" {
               start := tmpl.StartNumber
               if start == 0 {
                  start = 1
               }
               end := tmpl.EndNumber

               if tmpl.Initialization != "" {
                  initURL := replacePlaceholders(tmpl.Initialization, rep.ID, 0, 0)
                  urls = append(urls, resolve(repBase, initURL).String())
               }

               if tmpl.SegmentTimeline != nil {
                  number := start
                  var currentTime int
                  for _, s := range tmpl.SegmentTimeline.Segments {
                     repeat := s.R
                     if repeat == 0 {
                        repeat = 1
                     } else {
                        repeat += 1
                     }
                     if s.T > 0 {
                        currentTime = s.T
                     }
                     for i := 0; i < repeat; i++ {
                        seg := replacePlaceholders(tmpl.Media, rep.ID, number, currentTime)
                        full := resolve(repBase, seg)
                        urls = append(urls, full.String())
                        currentTime += s.D
                        number++
                     }
                  }
               } else if tmpl.Duration > 0 {
                  count := 5
                  if end > 0 && end >= start {
                     count = end - start + 1
                  } else if periodDurSec > 0 {
                     timescale := tmpl.Timescale
                     if timescale == 0 {
                        timescale = 1
                     }
                     ratio := periodDurSec * float64(timescale) / float64(tmpl.Duration)
                     count = int(math.Ceil(ratio))
                  }
                  for i := 0; i < count; i++ {
                     num := start + i
                     t := i * tmpl.Duration
                     seg := replacePlaceholders(tmpl.Media, rep.ID, num, t)
                     full := resolve(repBase, seg)
                     urls = append(urls, full.String())
                  }
               }

               result[rep.ID] = append(result[rep.ID], urls...)
               continue
            }

            // --- Only BaseURL ---
            if rep.BaseURL != "" {
               result[rep.ID] = append(result[rep.ID], repBase.String())
            }
         }
      }
   }

   out, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Println("Failed to encode JSON:", err)
      os.Exit(1)
   }
   fmt.Println(string(out))
}

func replacePlaceholders(tmpl, repID string, number, time int) string {
   tmpl = strings.ReplaceAll(tmpl, "$RepresentationID$", repID)

   tmpl = replaceFormatted(tmpl, "Number", number)
   tmpl = replaceFormatted(tmpl, "Time", time)

   return tmpl
}

func replaceFormatted(s, name string, val int) string {
   for {
      prefix := "$" + name + "%"
      start := strings.Index(s, prefix)
      if start == -1 {
         break
      }
      end := strings.Index(s[start+1:], "$")
      if end == -1 {
         break // No closing "$"
      }
      end = start + 1 + end // absolute index of closing "$"

      format := s[start+len(name)+2 : end] // extract e.g., 08d
      goFmt := "%" + format

      replacement := fmt.Sprintf(goFmt, val)
      s = s[:start] + replacement + s[end+1:]
   }
   // fallback for plain $Name$
   return strings.ReplaceAll(s, "$"+name+"$", fmt.Sprintf("%d", val))
}
