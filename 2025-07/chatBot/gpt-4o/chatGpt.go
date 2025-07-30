package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "net/url"
   "os"
   "regexp"
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
   Timescale       int              `xml:"timescale,attr"`
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
      return
   }
   filePath := os.Args[1]
   data, err := ioutil.ReadFile(filePath)
   if err != nil {
      fmt.Println("Error reading MPD file:", err)
      return
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Println("Error parsing MPD:", err)
      return
   }

   baseURL := "http://test.test/test.mpd"
   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      periodBase := resolveURL(baseURL, period.BaseURL)
      for _, aset := range period.AdaptationSets {
         asetBase := resolveURL(periodBase, aset.BaseURL)
         for _, rep := range aset.Representations {
            repBase := resolveURL(asetBase, rep.BaseURL)

            // Handle SegmentList if present
            slist := inheritSegmentList(aset.SegmentList, rep.SegmentList)
            if slist != nil {
               var urls []string
               if slist.Initialization != nil {
                  urls = append(urls, resolveURL(repBase, slist.Initialization.SourceURL))
               }
               for _, s := range slist.SegmentURLs {
                  urls = append(urls, resolveURL(repBase, s.Media))
               }
               result[rep.ID] = urls
               continue
            }

            // Handle SegmentTemplate
            tmpl := inheritTemplate(aset.SegmentTemplate, rep.SegmentTemplate)

            if tmpl == nil || tmpl.Media == "" {
               // Fallback to BaseURL segments
               // Representation.BaseURL (segments) overrides AdaptationSet.BaseURL
               if rep.BaseURL != "" {
                  result[rep.ID] = []string{resolveURL(repBase, "")}
               } else {
                  // No specific segment logic, fallback to AdaptationSet BaseURL (if it's a segment list)
                  result[rep.ID] = []string{asetBase}
               }
               continue
            }

            var urls []string
            if tmpl.Initialization != "" {
               initURL := replaceTemplate(tmpl.Initialization, rep.ID, 0, 0)
               urls = append(urls, resolveURL(repBase, initURL))
            }

            if tmpl.SegmentTimeline != nil {
               times := expandSegmentTimeline(tmpl.SegmentTimeline)
               for _, t := range times {
                  media := replaceTemplate(tmpl.Media, rep.ID, 0, t)
                  urls = append(urls, resolveURL(repBase, media))
               }
            } else {
               start := tmpl.StartNumber
               if start == 0 {
                  start = 1
               }
               end := tmpl.EndNumber
               if end == 0 {
                  end = start + 4 // arbitrary 5 segments if endNumber missing
               }
               for i := start; i <= end; i++ {
                  media := replaceTemplate(tmpl.Media, rep.ID, i, 0)
                  urls = append(urls, resolveURL(repBase, media))
               }
            }

            result[rep.ID] = urls
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   enc.Encode(result)
}

func resolveURL(baseStr, refStr string) string {
   base, err := url.Parse(baseStr)
   if err != nil {
      return refStr
   }
   ref, err := url.Parse(refStr)
   if err != nil {
      return refStr
   }
   return base.ResolveReference(ref).String()
}

func inheritTemplate(parent, child *SegmentTemplate) *SegmentTemplate {
   if parent == nil && child == nil {
      return nil
   }
   if parent == nil {
      return child
   }
   if child == nil {
      return parent
   }
   result := *parent
   if child.Initialization != "" {
      result.Initialization = child.Initialization
   }
   if child.Media != "" {
      result.Media = child.Media
   }
   if child.StartNumber != 0 {
      result.StartNumber = child.StartNumber
   }
   if child.EndNumber != 0 {
      result.EndNumber = child.EndNumber
   }
   if child.Timescale != 0 {
      result.Timescale = child.Timescale
   }
   if child.SegmentTimeline != nil {
      result.SegmentTimeline = child.SegmentTimeline
   }
   return &result
}

func inheritSegmentList(parent, child *SegmentList) *SegmentList {
   if child != nil {
      return child
   }
   return parent
}

func replaceTemplate(template, repID string, number int, time int64) string {
   re := regexp.MustCompile(`\$(\w+)(%[^$]+)?\$`)
   return re.ReplaceAllStringFunc(template, func(m string) string {
      parts := re.FindStringSubmatch(m)
      if len(parts) < 2 {
         return m
      }
      switch parts[1] {
      case "RepresentationID":
         return repID
      case "Number":
         format := "%d"
         if parts[2] != "" {
            format = parts[2]
         }
         return fmt.Sprintf(format, number)
      case "Time":
         format := "%d"
         if parts[2] != "" {
            format = parts[2]
         }
         return fmt.Sprintf(format, time)
      default:
         return m
      }
   })
}

func expandSegmentTimeline(tl *SegmentTimeline) []int64 {
   var result []int64
   var t int64 = 0
   for _, s := range tl.Segments {
      count := 1
      if s.R > 0 {
         count += s.R
      }
      if s.T != 0 {
         t = s.T
      }
      for i := 0; i < count; i++ {
         result = append(result, t)
         t += s.D
      }
   }
   return result
}
