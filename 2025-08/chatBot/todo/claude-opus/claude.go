package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "math"
   "net/url"
   "os"
   "path"
   "strconv"
   "strings"
   "time"
)

// MPD structure
type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

// Period structure
type Period struct {
   Duration        string           `xml:"duration,attr"`
   BaseURL         string           `xml:"BaseURL"`
   AdaptationSets  []AdaptationSet  `xml:"AdaptationSet"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

// AdaptationSet structure
type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   Representations []Representation `xml:"Representation"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

// Representation structure
type Representation struct {
   ID              string           `xml:"id,attr"`
   Bandwidth       int              `xml:"bandwidth,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
}

// SegmentTemplate structure
type SegmentTemplate struct {
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   Duration        int              `xml:"duration,attr"`
   Timescale       int              `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline structure
type SegmentTimeline struct {
   S []S `xml:"S"`
}

// S structure for timeline segments
type S struct {
   T int `xml:"t,attr"`
   D int `xml:"d,attr"`
   R int `xml:"r,attr"`
}

// SegmentList structure
type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

// Initialization structure
type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
   Range     string `xml:"range,attr"`
}

// SegmentURL structure
type SegmentURL struct {
   Media string `xml:"media,attr"`
   Range string `xml:"range,attr"`
}

// SegmentBase structure
type SegmentBase struct {
   Initialization *Initialization `xml:"Initialization"`
}

// resolveURL resolves a relative URL against a base URL
func resolveURL(base, relative string) string {
   if relative == "" {
      return base
   }

   // If relative is already absolute, return it
   if strings.HasPrefix(relative, "http://") || strings.HasPrefix(relative, "https://") {
      return relative
   }

   // Parse base URL
   baseURL, err := url.Parse(base)
   if err != nil {
      return path.Join(base, relative)
   }

   // Parse relative URL
   relURL, err := url.Parse(relative)
   if err != nil {
      return path.Join(base, relative)
   }

   // Resolve relative URL against base URL
   resolved := baseURL.ResolveReference(relURL)
   return resolved.String()
}

// replaceTemplateVariables replaces template variables in a URL template
func replaceTemplateVariables(template string, repID string, number int, time int, bandwidth int) string {
   result := template

   // Replace $RepresentationID$
   result = strings.ReplaceAll(result, "$RepresentationID$", repID)

   // Replace $Bandwidth$
   result = strings.ReplaceAll(result, "$Bandwidth$", strconv.Itoa(bandwidth))

   // Replace $Time$
   result = strings.ReplaceAll(result, "$Time$", strconv.Itoa(time))

   // Replace $Number$ with padding support
   if strings.Contains(result, "$Number") {
      // Check for padded format like $Number%05d$
      start := strings.Index(result, "$Number")
      end := strings.Index(result[start:], "$") + start
      end = strings.Index(result[end+1:], "$") + end + 1

      numberFormat := result[start : end+1]
      if strings.Contains(numberFormat, "%") {
         // Extract padding format
         formatStart := strings.Index(numberFormat, "%")
         formatEnd := strings.Index(numberFormat[formatStart:], "d")
         if formatEnd > 0 {
            paddingStr := numberFormat[formatStart+1 : formatStart+formatEnd]
            padding, err := strconv.Atoi(paddingStr)
            if err == nil {
               // Apply padding
               paddedNumber := fmt.Sprintf("%0*d", padding, number)
               result = strings.Replace(result, numberFormat, paddedNumber, 1)
            }
         }
      } else {
         // Simple replacement
         result = strings.ReplaceAll(result, "$Number$", strconv.Itoa(number))
      }
   }

   return result
}

// parseDuration parses ISO 8601 duration format (e.g., PT30S, PT1H30M)
func parseDuration(duration string) (float64, error) {
   if duration == "" {
      return 0, fmt.Errorf("empty duration")
   }

   // Parse ISO 8601 duration
   d, err := time.ParseDuration(strings.ReplaceAll(strings.ReplaceAll(strings.TrimPrefix(duration, "PT"), "H", "h"), "M", "m"))
   if err != nil {
      // Try parsing with S suffix
      if strings.HasPrefix(duration, "PT") && strings.HasSuffix(duration, "S") {
         secondsStr := strings.TrimSuffix(strings.TrimPrefix(duration, "PT"), "S")
         seconds, err := strconv.ParseFloat(secondsStr, 64)
         if err != nil {
            return 0, err
         }
         return seconds, nil
      }
      return 0, err
   }

   return d.Seconds(), nil
}

// extractSegmentURLs extracts all segment URLs for a representation
func extractSegmentURLs(rep Representation, periodTemplate *SegmentTemplate, periodList *SegmentList,
   adaptationTemplate *SegmentTemplate, adaptationList *SegmentList, baseURL string, periodDuration string) []string {
   var urls []string

   // If representation has only BaseURL and no segment information, return the resolved baseURL
   if rep.SegmentTemplate == nil && rep.SegmentList == nil && rep.SegmentBase == nil &&
      periodTemplate == nil && periodList == nil && adaptationTemplate == nil && adaptationList == nil {
      if rep.BaseURL != "" {
         urls = append(urls, baseURL)
      }
      return urls
   }

   // Determine which segment information to use (representation > adaptation > period)
   var segmentTemplate *SegmentTemplate
   var segmentList *SegmentList
   var segmentBase *SegmentBase

   if rep.SegmentTemplate != nil {
      segmentTemplate = rep.SegmentTemplate
   } else if adaptationTemplate != nil {
      segmentTemplate = adaptationTemplate
   } else if periodTemplate != nil {
      segmentTemplate = periodTemplate
   }

   if rep.SegmentList != nil {
      segmentList = rep.SegmentList
   } else if adaptationList != nil {
      segmentList = adaptationList
   } else if periodList != nil {
      segmentList = periodList
   }

   if rep.SegmentBase != nil {
      segmentBase = rep.SegmentBase
   }

   // Handle SegmentTemplate
   if segmentTemplate != nil {
      // Add initialization segment if present
      if segmentTemplate.Initialization != "" {
         initURL := replaceTemplateVariables(segmentTemplate.Initialization, rep.ID, 0, 0, rep.Bandwidth)
         urls = append(urls, resolveURL(baseURL, initURL))
      }

      // Handle timeline-based segments
      if segmentTemplate.SegmentTimeline != nil {
         currentTime := 0
         segmentNumber := 1
         if segmentTemplate.StartNumber > 0 {
            segmentNumber = segmentTemplate.StartNumber
         }

         for _, s := range segmentTemplate.SegmentTimeline.S {
            if s.T > 0 {
               currentTime = s.T
            }

            // Handle repeat count
            repeatCount := 0
            if s.R > 0 {
               repeatCount = s.R
            }

            for i := 0; i <= repeatCount; i++ {
               segmentURL := replaceTemplateVariables(segmentTemplate.Media, rep.ID, segmentNumber, currentTime, rep.Bandwidth)
               urls = append(urls, resolveURL(baseURL, segmentURL))

               currentTime += s.D
               segmentNumber++

               // Check if we've reached endNumber
               if segmentTemplate.EndNumber > 0 && segmentNumber > segmentTemplate.EndNumber {
                  goto done
               }
            }
         }
      } else if segmentTemplate.Duration > 0 {
         // Handle duration-based segments
         startNumber := 1
         if segmentTemplate.StartNumber > 0 {
            startNumber = segmentTemplate.StartNumber
         }

         endNumber := startNumber + 10 // Default to 10 segments if no endNumber specified
         if segmentTemplate.EndNumber > 0 {
            endNumber = segmentTemplate.EndNumber
         } else if periodDuration != "" {
            // Calculate number of segments from period duration
            periodSeconds, err := parseDuration(periodDuration)
            if err == nil {
               // Default timescale to 1 if not specified
               timescale := segmentTemplate.Timescale
               if timescale == 0 {
                  timescale = 1
               }
               // Calculate: ceil(PeriodDurationInSeconds * timescale / duration)
               numSegments := int(math.Ceil(periodSeconds * float64(timescale) / float64(segmentTemplate.Duration)))
               endNumber = startNumber + numSegments - 1
            }
         }

         for i := startNumber; i <= endNumber; i++ {
            segmentURL := replaceTemplateVariables(segmentTemplate.Media, rep.ID, i, 0, rep.Bandwidth)
            urls = append(urls, resolveURL(baseURL, segmentURL))
         }
      }
   }

   // Handle SegmentList
   if segmentList != nil {
      // Add initialization segment if present
      if segmentList.Initialization != nil && segmentList.Initialization.SourceURL != "" {
         urls = append(urls, resolveURL(baseURL, segmentList.Initialization.SourceURL))
      }

      // Add all segment URLs
      for _, segURL := range segmentList.SegmentURLs {
         if segURL.Media != "" {
            urls = append(urls, resolveURL(baseURL, segURL.Media))
         }
      }
   }

   // Handle SegmentBase (single segment)
   if segmentBase != nil {
      if segmentBase.Initialization != nil && segmentBase.Initialization.SourceURL != "" {
         urls = append(urls, resolveURL(baseURL, segmentBase.Initialization.SourceURL))
      }
      // For SegmentBase, the media is typically the BaseURL itself
      urls = append(urls, baseURL)
   }

done:
   return urls
}

func main() {
   if len(os.Args) < 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }

   mpdFilePath := os.Args[1]

   // Read MPD file
   xmlData, err := os.ReadFile(mpdFilePath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
      os.Exit(1)
   }

   // Parse XML
   var mpd MPD
   err = xml.Unmarshal(xmlData, &mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing XML: %v\n", err)
      os.Exit(1)
   }

   // Starting base URL
   baseURL := "http://test.test/test.mpd"

   // Resolve MPD-level BaseURL
   if mpd.BaseURL != "" {
      baseURL = resolveURL(baseURL, mpd.BaseURL)
   }

   // Result map
   result := make(map[string][]string)

   // Process each period
   for _, period := range mpd.Periods {
      // Resolve Period-level BaseURL
      periodBaseURL := baseURL
      if period.BaseURL != "" {
         periodBaseURL = resolveURL(baseURL, period.BaseURL)
      }

      // Process each adaptation set
      for _, adaptationSet := range period.AdaptationSets {
         // Resolve AdaptationSet-level BaseURL
         adaptationBaseURL := periodBaseURL
         if adaptationSet.BaseURL != "" {
            adaptationBaseURL = resolveURL(periodBaseURL, adaptationSet.BaseURL)
         }

         // Process each representation
         for _, rep := range adaptationSet.Representations {
            // Resolve Representation-level BaseURL
            repBaseURL := adaptationBaseURL
            if rep.BaseURL != "" {
               repBaseURL = resolveURL(adaptationBaseURL, rep.BaseURL)
            }

            // Extract segment URLs
            segmentURLs := extractSegmentURLs(rep, period.SegmentTemplate, period.SegmentList,
               adaptationSet.SegmentTemplate, adaptationSet.SegmentList, repBaseURL, period.Duration)

            if len(segmentURLs) > 0 {
               // Append to existing segments if this representation ID already exists
               if existingURLs, exists := result[rep.ID]; exists {
                  result[rep.ID] = append(existingURLs, segmentURLs...)
               } else {
                  result[rep.ID] = segmentURLs
               }
            }
         }
      }
   }

   // Output as JSON
   jsonOutput, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error creating JSON output: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonOutput))
}
