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
   "strings"
)

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL *BaseURL `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   BaseURL        *BaseURL        `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         *BaseURL         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         *BaseURL         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentTemplate struct {
   Timescale       int              `xml:"timescale,attr"`
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   Segments []Segment `xml:"S"`
}

type Segment struct {
   T int64 `xml:"t,attr"`
   D int64 `xml:"d,attr"`
   R int   `xml:"r,attr"`
}

type BaseURL struct {
   URL string `xml:",chardata"`
}

func resolveURL(base string, parts ...*BaseURL) string {
   for _, part := range parts {
      if part != nil {
         base = joinURL(base, strings.TrimSpace(part.URL))
      }
   }
   return base
}

func joinURL(base, rel string) string {
   u, err := url.Parse(base)
   if err != nil {
      return rel
   }
   r, err := url.Parse(rel)
   if err != nil {
      return rel
   }
   return u.ResolveReference(r).String()
}

func substituteTemplate(template string, vars map[string]interface{}) string {
   re := regexp.MustCompile(`\$(\w+)(%0(\d+)d)?\$`)
   return re.ReplaceAllStringFunc(template, func(m string) string {
      sub := re.FindStringSubmatch(m)
      key := sub[1]
      format := "%v"
      if sub[2] != "" {
         width, _ := strconv.Atoi(sub[3])
         format = "%0" + strconv.Itoa(width) + "d"
      }
      val, ok := vars[key]
      if !ok {
         return m
      }
      switch v := val.(type) {
      case int:
         return fmt.Sprintf(format, v)
      case int64:
         return fmt.Sprintf(format, v)
      case string:
         return v
      default:
         return fmt.Sprintf("%v", v)
      }
   })
}

func main() {
   if len(os.Args) != 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path>")
      os.Exit(1)
   }

   file := os.Args[1]
   data, err := ioutil.ReadFile(file)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Failed to read file: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Fprintf(os.Stderr, "Failed to parse MPD: %v\n", err)
      os.Exit(1)
   }

   const baseURL = "http://test.test/test.mpd"
   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      for _, aset := range period.AdaptationSets {
         for _, rep := range aset.Representations {
            effectiveBase := resolveURL(baseURL, mpd.BaseURL, period.BaseURL, aset.BaseURL, rep.BaseURL)

            // Get SegmentTemplate
            tmpl := rep.SegmentTemplate
            if tmpl == nil {
               tmpl = aset.SegmentTemplate
            }
            if tmpl == nil || tmpl.Media == "" {
               continue
            }

            vars := map[string]interface{}{
               "RepresentationID": rep.ID,
            }
            var urls []string

            // Initialization
            if tmpl.Initialization != "" {
               initURL := substituteTemplate(tmpl.Initialization, vars)
               urls = append(urls, joinURL(effectiveBase, initURL))
            }

            // Defaults
            startNum := tmpl.StartNumber
            if startNum == 0 {
               startNum = 1
            }
            endNum := tmpl.EndNumber

            // SegmentTimeline
            if tmpl.SegmentTimeline != nil {
               t := int64(0)
               num := startNum
               for _, seg := range tmpl.SegmentTimeline.Segments {
                  r := seg.R
                  if r < 0 {
                     r = 0
                  }
                  if seg.T != 0 {
                     t = seg.T
                  }
                  for i := 0; i <= r; i++ {
                     if endNum > 0 && num > endNum {
                        break
                     }
                     vars["Number"] = num
                     vars["Time"] = t
                     segURL := substituteTemplate(tmpl.Media, vars)
                     urls = append(urls, joinURL(effectiveBase, segURL))
                     num++
                     t += seg.D
                  }
               }
            } else {

               num := startNum
               if endNum > 0 {
                  for num <= endNum {
                     vars["Number"] = num
                     segURL := substituteTemplate(tmpl.Media, vars)
                     urls = append(urls, joinURL(effectiveBase, segURL))
                     num++
                  }
               } else {
                  // fallback: generate 5 segments
                  for i := 0; i < 5; i++ {
                     vars["Number"] = num
                     segURL := substituteTemplate(tmpl.Media, vars)
                     urls = append(urls, joinURL(effectiveBase, segURL))
                     num++
                  }
               }

            }

            result[rep.ID] = urls
         }
      }
   }

   jout, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "JSON encode error: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jout))
}
