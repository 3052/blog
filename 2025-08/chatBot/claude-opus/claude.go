package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "net/url"
   "os"
   "path"
   "strconv"
   "strings"
)

// MPD represents the root element of an MPD file
type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
   BaseURL                   string   `xml:"BaseURL"`
   Period                    []Period `xml:"Period"`
}

// Period represents a Period element in the MPD
type Period struct {
   XMLName       xml.Name        `xml:"Period"`
   ID            string          `xml:"id,attr"`
   Duration      string          `xml:"duration,attr"`
   BaseURL       string          `xml:"BaseURL"`
   AdaptationSet []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet represents an AdaptationSet element
type AdaptationSet struct {
   XMLName         xml.Name         `xml:"AdaptationSet"`
   ID              string           `xml:"id,attr"`
   MimeType        string           `xml:"mimeType,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representation  []Representation `xml:"Representation"`
}

// Representation represents a Representation element
type Representation struct {
   XMLName         xml.Name         `xml:"Representation"`
   ID              string           `xml:"id,attr"`
   Bandwidth       string           `xml:"bandwidth,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
}

// SegmentTemplate represents a SegmentTemplate element
type SegmentTemplate struct {
   XMLName         xml.Name         `xml:"SegmentTemplate"`
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   StartNumber     string           `xml:"startNumber,attr"`
   EndNumber       string           `xml:"endNumber,attr"`
   Duration        string           `xml:"duration,attr"`
   Timescale       string           `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline represents a SegmentTimeline element
type SegmentTimeline struct {
   XMLName xml.Name `xml:"SegmentTimeline"`
   S       []S      `xml:"S"`
}

// S represents a segment in the timeline
type S struct {
   T string `xml:"t,attr"`
   D string `xml:"d,attr"`
   R string `xml:"r,attr"`
}

// SegmentList represents a SegmentList element
type SegmentList struct {
   XMLName        xml.Name        `xml:"SegmentList"`
   Duration       string          `xml:"duration,attr"`
   Timescale      string          `xml:"timescale,attr"`
   Initialization *Initialization `xml:"Initialization"`
   SegmentURL     []SegmentURL    `xml:"SegmentURL"`
}

// SegmentBase represents a SegmentBase element
type SegmentBase struct {
   XMLName        xml.Name        `xml:"SegmentBase"`
   Initialization *Initialization `xml:"Initialization"`
}

// Initialization represents an Initialization element
type Initialization struct {
   XMLName   xml.Name `xml:"Initialization"`
   SourceURL string   `xml:"sourceURL,attr"`
   Range     string   `xml:"range,attr"`
}

// SegmentURL represents a SegmentURL element
type SegmentURL struct {
   XMLName    xml.Name `xml:"SegmentURL"`
   Media      string   `xml:"media,attr"`
   MediaRange string   `xml:"mediaRange,attr"`
}

// resolveURL resolves a relative URL against a base URL
func resolveURL(baseURL, relativeURL string) string {
   if relativeURL == "" {
      return baseURL
   }

   base, err := url.Parse(baseURL)
   if err != nil {
      return relativeURL
   }

   rel, err := url.Parse(relativeURL)
   if err != nil {
      return relativeURL
   }

   // If relative URL is absolute, return it
   if rel.IsAbs() {
      return relativeURL
   }

   resolved := base.ResolveReference(rel)
   return resolved.String()
}

// combineBaseURLs combines multiple BaseURL elements hierarchically
func combineBaseURLs(urls []string) string {
   result := "http://test.test/test.mpd"

   for _, url := range urls {
      if url != "" {
         result = resolveURL(result, url)
      }
   }

   // Ensure the base URL ends with / if it's a directory
   if !strings.HasSuffix(result, "/") && !strings.Contains(path.Base(result), ".") {
      result += "/"
   }

   return result
}

// replaceTemplateVariables replaces template variables in URL templates
func replaceTemplateVariables(template string, repID string, number int, time int64, bandwidth string) string {
   result := template
   result = strings.ReplaceAll(result, "$RepresentationID$", repID)
   result = strings.ReplaceAll(result, "$Number$", strconv.Itoa(number))
   result = strings.ReplaceAll(result, "$Time$", strconv.FormatInt(time, 10))
   result = strings.ReplaceAll(result, "$Bandwidth$", bandwidth)

   // Handle padded number format $Number%0Xd$
   if strings.Contains(result, "$Number%") {
      start := strings.Index(result, "$Number%")
      end := strings.Index(result[start+8:], "$")
      if end != -1 {
         format := result[start+8 : start+8+end]
         padded := fmt.Sprintf("%"+format, number)
         result = strings.Replace(result, "$Number%"+format+"$", padded, -1)
      }
   }

   return result
}

// extractSegmentURLs extracts all segment URLs for a representation
func extractSegmentURLs(mpd *MPD, period *Period, adaptationSet *AdaptationSet, representation *Representation) []string {
   var urls []string

   // Build base URL hierarchy
   baseURL := combineBaseURLs([]string{
      mpd.BaseURL,
      period.BaseURL,
      representation.BaseURL,
   })

   // Handle SegmentTemplate
   segmentTemplate := representation.SegmentTemplate
   if segmentTemplate == nil && adaptationSet.SegmentTemplate != nil {
      segmentTemplate = adaptationSet.SegmentTemplate
   }

   if segmentTemplate != nil {
      // Add initialization segment
      if segmentTemplate.Initialization != "" {
         initURL := replaceTemplateVariables(segmentTemplate.Initialization, representation.ID, 0, 0, representation.Bandwidth)
         urls = append(urls, resolveURL(baseURL, initURL))
      }

      // Generate media segments
      if segmentTemplate.Media != "" {
         if segmentTemplate.SegmentTimeline != nil {
            // Timeline-based segments
            time := int64(0)
            segmentNumber := 1
            if segmentTemplate.StartNumber != "" {
               segmentNumber, _ = strconv.Atoi(segmentTemplate.StartNumber)
            }

            for _, s := range segmentTemplate.SegmentTimeline.S {
               if s.T != "" {
                  time, _ = strconv.ParseInt(s.T, 10, 64)
               }

               duration, _ := strconv.ParseInt(s.D, 10, 64)
               repeat := 0
               if s.R != "" {
                  repeat, _ = strconv.Atoi(s.R)
               }

               for r := 0; r <= repeat; r++ {
                  mediaURL := replaceTemplateVariables(segmentTemplate.Media, representation.ID, segmentNumber, time, representation.Bandwidth)
                  urls = append(urls, resolveURL(baseURL, mediaURL))
                  time += duration
                  segmentNumber++
               }
            }
         } else {
            // Duration-based segments
            startNumber := 1
            if segmentTemplate.StartNumber != "" {
               startNumber, _ = strconv.Atoi(segmentTemplate.StartNumber)
            }

            // Calculate end number
            endNumber := startNumber + 9 // Default to 10 segments
            if segmentTemplate.EndNumber != "" {
               endNumber, _ = strconv.Atoi(segmentTemplate.EndNumber)
            } else if segmentTemplate.Duration != "" && segmentTemplate.Timescale != "" {
               // If endNumber not specified but duration info available,
               // you could calculate it from MPD duration here
               // For now, we'll use the explicit endNumber or default
            }

            // Generate segments from startNumber to endNumber (inclusive)
            for i := startNumber; i <= endNumber; i++ {
               mediaURL := replaceTemplateVariables(segmentTemplate.Media, representation.ID, i, 0, representation.Bandwidth)
               urls = append(urls, resolveURL(baseURL, mediaURL))
            }
         }
      }
   } else if representation.SegmentList != nil {
      // Handle SegmentList
      if representation.SegmentList.Initialization != nil && representation.SegmentList.Initialization.SourceURL != "" {
         urls = append(urls, resolveURL(baseURL, representation.SegmentList.Initialization.SourceURL))
      }

      for _, segment := range representation.SegmentList.SegmentURL {
         if segment.Media != "" {
            urls = append(urls, resolveURL(baseURL, segment.Media))
         }
      }
   } else if representation.SegmentBase != nil {
      // Handle SegmentBase (single segment)
      if representation.SegmentBase.Initialization != nil && representation.SegmentBase.Initialization.SourceURL != "" {
         urls = append(urls, resolveURL(baseURL, representation.SegmentBase.Initialization.SourceURL))
      }
      // For SegmentBase, the media is typically the BaseURL itself
      if representation.BaseURL != "" {
         urls = append(urls, baseURL)
      }
   }

   return urls
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }

   mpdPath := os.Args[1]

   // Read MPD file
   data, err := ioutil.ReadFile(mpdPath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
      os.Exit(1)
   }

   // Parse XML
   var mpd MPD
   err = xml.Unmarshal(data, &mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing XML: %v\n", err)
      os.Exit(1)
   }

   // Extract segment URLs for each representation
   result := make(map[string][]string)

   for _, period := range mpd.Period {
      for _, adaptationSet := range period.AdaptationSet {
         for _, representation := range adaptationSet.Representation {
            urls := extractSegmentURLs(&mpd, &period, &adaptationSet, &representation)
            result[representation.ID] = urls
         }
      }
   }

   // Output as JSON
   jsonOutput, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error creating JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonOutput))
}
