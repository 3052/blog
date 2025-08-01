package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "net/url"
   "os"
   "path"
   "regexp"
)

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Period  Period   `xml:"Period"`
}

type Period struct {
   BaseURL        string          `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   StartNumber     int              `xml:"startNumber,attr,omitempty"`
   EndNumber       int              `xml:"endNumber,attr,omitempty"` // NEW
   Timescale       int              `xml:"timescale,attr,omitempty"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T *int `xml:"t,attr,omitempty"` // start time
   D int  `xml:"d,attr"`           // duration (required)
   R *int `xml:"r,attr,omitempty"` // repeat count (optional)
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: go run main.go <mpd_file_path>\n")
      os.Exit(1)
   }

   mpdPath := os.Args[1]

   data, err := os.ReadFile(mpdPath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading MPD file: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD XML: %v\n", err)
      os.Exit(1)
   }

   baseURL, err := url.Parse("http://test.test/test.mpd")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing base URL: %v\n", err)
      os.Exit(1)
   }

   result := map[string][]string{}

   for _, adapt := range mpd.Period.AdaptationSets {
      for _, rep := range adapt.Representations {
         repBaseURL := baseURL

         if mpd.BaseURL != "" {
            if u, err := url.Parse(mpd.BaseURL); err == nil {
               repBaseURL = repBaseURL.ResolveReference(u)
            }
         }

         if mpd.Period.BaseURL != "" {
            if u, err := url.Parse(mpd.Period.BaseURL); err == nil {
               repBaseURL = repBaseURL.ResolveReference(u)
            }
         }

         if adapt.BaseURL != "" {
            if u, err := url.Parse(adapt.BaseURL); err == nil {
               repBaseURL = repBaseURL.ResolveReference(u)
            }
         }

         if rep.BaseURL != "" {
            if u, err := url.Parse(rep.BaseURL); err == nil {
               repBaseURL = repBaseURL.ResolveReference(u)
            }
         }

         segTemplate := mergeSegmentTemplate(adapt.SegmentTemplate, rep.SegmentTemplate)

         var segmentURLs []string

         if segTemplate != nil {
            startNum := segTemplate.StartNumber
            if startNum == 0 {
               startNum = 1
            }

            endNum := segTemplate.EndNumber
            hasEndNum := endNum != 0

            timescale := segTemplate.Timescale
            if timescale == 0 {
               timescale = 1
            }

            // Initialization segment
            if segTemplate.Initialization != "" {
               initURL := substituteTemplate(segTemplate.Initialization, rep.ID, startNum, 0)
               initURL = resolveURL(repBaseURL, initURL)
               segmentURLs = append(segmentURLs, initURL)
            }

            if segTemplate.SegmentTimeline != nil && len(segTemplate.SegmentTimeline.S) > 0 {
               segments := expandSegmentTimeline(segTemplate.SegmentTimeline.S)
               // segments are start times, numbered from startNum, so indices + startNum = segment numbers

               for i, t := range segments {
                  number := startNum + i
                  if hasEndNum && number > endNum {
                     break
                  }
                  mediaURL := substituteTemplate(segTemplate.Media, rep.ID, number, t)
                  mediaURL = resolveURL(repBaseURL, mediaURL)
                  segmentURLs = append(segmentURLs, mediaURL)
               }
            } else {
               // No timeline, use startNum..endNum or default 3 segments
               limit := 3
               if hasEndNum {
                  limit = endNum - startNum + 1
                  if limit < 0 {
                     limit = 0
                  }
               }

               for i := 0; i < limit; i++ {
                  number := startNum + i
                  t := i * timescale
                  mediaURL := substituteTemplate(segTemplate.Media, rep.ID, number, t)
                  mediaURL = resolveURL(repBaseURL, mediaURL)
                  segmentURLs = append(segmentURLs, mediaURL)
               }
            }

         } else {
            segmentURLs = append(segmentURLs, repBaseURL.String())
         }

         result[rep.ID] = segmentURLs
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   if err := enc.Encode(result); err != nil {
      fmt.Fprintf(os.Stderr, "Error encoding JSON output: %v\n", err)
      os.Exit(1)
   }
}

func expandSegmentTimeline(s []S) []int {
   var result []int
   var lastT int
   for i, entry := range s {
      count := 1
      if entry.R != nil {
         count = *entry.R + 1
      }

      var startT int
      if entry.T != nil {
         startT = *entry.T
      } else {
         if i == 0 {
            startT = 0
         } else {
            startT = lastT + s[i-1].D
         }
      }

      for j := 0; j < count; j++ {
         result = append(result, startT)
         startT += entry.D
      }
      lastT = result[len(result)-1]
   }
   return result
}

func mergeSegmentTemplate(parent, child *SegmentTemplate) *SegmentTemplate {
   if parent == nil && child == nil {
      return nil
   }
   merged := &SegmentTemplate{}
   if parent != nil {
      *merged = *parent
   }
   if child != nil {
      if child.Initialization != "" {
         merged.Initialization = child.Initialization
      }
      if child.Media != "" {
         merged.Media = child.Media
      }
      if child.StartNumber != 0 {
         merged.StartNumber = child.StartNumber
      }
      if child.EndNumber != 0 {
         merged.EndNumber = child.EndNumber
      }
      if child.Timescale != 0 {
         merged.Timescale = child.Timescale
      }
      if child.SegmentTimeline != nil {
         merged.SegmentTimeline = child.SegmentTimeline
      }
   }
   return merged
}

var templateVarRegexp = regexp.MustCompile(`\$(RepresentationID|Number|Time)(%0?(\d+)d)?\$`)

func substituteTemplate(template, repID string, number, time int) string {
   return templateVarRegexp.ReplaceAllStringFunc(template, func(m string) string {
      matches := templateVarRegexp.FindStringSubmatch(m)
      if len(matches) < 2 {
         return m
      }
      varName := matches[1]
      format := "%d"
      if matches[2] != "" {
         format = matches[2]
      }

      switch varName {
      case "RepresentationID":
         return repID
      case "Number":
         return fmt.Sprintf(format, number)
      case "Time":
         return fmt.Sprintf(format, time)
      default:
         return m
      }
   })
}

func resolveURL(base *url.URL, ref string) string {
   refURL, err := url.Parse(ref)
   if err != nil {
      return base.ResolveReference(&url.URL{Path: path.Join(base.Path, ref)}).String()
   }
   return base.ResolveReference(refURL).String()
}
