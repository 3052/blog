package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "math"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
)

// MPD represents the root MPD element
type MPD struct {
   XMLName  xml.Name `xml:"MPD"`
   BaseURL  string   `xml:"BaseURL"`
   Duration string   `xml:"mediaPresentationDuration,attr"`
   Periods  []Period `xml:"Period"`
}

// Period represents a Period element
type Period struct {
   XMLName        xml.Name        `xml:"Period"`
   BaseURL        string          `xml:"BaseURL"`
   Duration       string          `xml:"duration,attr"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet represents an AdaptationSet element
type AdaptationSet struct {
   XMLName         xml.Name         `xml:"AdaptationSet"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

// Representation represents a Representation element
type Representation struct {
   XMLName         xml.Name         `xml:"Representation"`
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

// SegmentTemplate represents SegmentTemplate element
type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   StartNumber     string           `xml:"startNumber,attr"`
   EndNumber       string           `xml:"endNumber,attr"`
   Timescale       string           `xml:"timescale,attr"`
   Duration        string           `xml:"duration,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline represents SegmentTimeline element
type SegmentTimeline struct {
   Segments []SegmentTimelineSegment `xml:"S"`
}

// SegmentTimelineSegment represents S element in SegmentTimeline
type SegmentTimelineSegment struct {
   T string `xml:"t,attr"` // Time
   D string `xml:"d,attr"` // Duration
   R string `xml:"r,attr"` // Repeat count
}

// SegmentBase represents SegmentBase element
type SegmentBase struct {
   Initialization *Initialization `xml:"Initialization"`
}

// Initialization represents Initialization element
type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

// SegmentList represents SegmentList element
type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

// SegmentURL represents SegmentURL element
type SegmentURL struct {
   Media string `xml:"media,attr"`
}

func main() {
   if len(os.Args) < 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path>")
      os.Exit(1)
   }

   mpdFilePath := os.Args[1]

   // Read the MPD file
   data, err := os.ReadFile(mpdFilePath)
   if err != nil {
      fmt.Printf("Error reading MPD file: %v\n", err)
      os.Exit(1)
   }

   // Parse the XML
   var mpd MPD
   err = xml.Unmarshal(data, &mpd)
   if err != nil {
      fmt.Printf("Error parsing MPD XML: %v\n", err)
      os.Exit(1)
   }

   // Create the base URL resolver
   baseMPDURL := "http://test.test/test.mpd"

   // Generate the mapping - using map[string][]string to accumulate segments
   result := make(map[string][]string)

   // Process each Period
   for periodIndex, period := range mpd.Periods {
      // Calculate period duration for this period
      periodDuration := getPeriodDuration(&mpd, &period, periodIndex)

      for _, adaptationSet := range period.AdaptationSets {
         // Get SegmentTemplate from AdaptationSet level
         adaptationSetTemplate := adaptationSet.SegmentTemplate

         for _, representation := range adaptationSet.Representations {
            if representation.ID == "" {
               continue
            }

            // Determine which SegmentTemplate to use (Representation > AdaptationSet > nil)
            var effectiveTemplate *SegmentTemplate
            if representation.SegmentTemplate != nil {
               effectiveTemplate = representation.SegmentTemplate
            } else if adaptationSetTemplate != nil {
               effectiveTemplate = adaptationSetTemplate
            }

            // Resolve the base URL for this representation
            repBaseURL := resolveBaseURL(baseMPDURL, mpd.BaseURL, period.BaseURL, adaptationSet.BaseURL, representation.BaseURL)

            // Collect segment URLs
            var segmentURLs []string

            // Handle SegmentList segments
            if representation.SegmentList != nil {
               // Add initialization segment if it exists as child of SegmentList
               if representation.SegmentList.Initialization != nil && representation.SegmentList.Initialization.SourceURL != "" {
                  initURL := representation.SegmentList.Initialization.SourceURL
                  fullInitURL := resolveURL(repBaseURL, initURL)
                  segmentURLs = append(segmentURLs, fullInitURL)
               }

               // Add segment URLs
               for _, segmentURL := range representation.SegmentList.SegmentURLs {
                  if segmentURL.Media != "" {
                     fullURL := resolveURL(repBaseURL, segmentURL.Media)
                     segmentURLs = append(segmentURLs, fullURL)
                  }
               }
            } else if effectiveTemplate != nil {
               // Handle SegmentTemplate with proper segment count calculation
               segmentURLs = generateSegmentURLsFromTemplate(repBaseURL, effectiveTemplate, representation.ID, periodDuration)
            } else if representation.BaseURL != "" {
               // Fallback to BaseURL
               fullURL := resolveURL(repBaseURL, "")
               if fullURL != "" {
                  segmentURLs = append(segmentURLs, fullURL)
               }
            }

            // Append segments to existing segments for this Representation ID
            if existingSegments, exists := result[representation.ID]; exists {
               result[representation.ID] = append(existingSegments, segmentURLs...)
            } else {
               result[representation.ID] = segmentURLs
            }
         }
      }
   }

   // Output as JSON
   jsonData, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Printf("Error generating JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonData))
}

// getPeriodDuration calculates the duration of a period in seconds
func getPeriodDuration(mpd *MPD, period *Period, periodIndex int) float64 {
   // First check if period has explicit duration
   if period.Duration != "" {
      if duration := parseDuration(period.Duration); duration > 0 {
         return duration
      }
   }

   // If no explicit period duration, try to calculate from MPD duration
   if mpd.Duration != "" {
      mpdDuration := parseDuration(mpd.Duration)
      if mpdDuration > 0 && len(mpd.Periods) > 0 {
         // For simplicity, divide MPD duration equally among periods
         // In a real implementation, this would be more complex
         return mpdDuration / float64(len(mpd.Periods))
      }
   }

   // Default fallback
   return 10.0 // 10 seconds
}

// parseDuration parses ISO 8601 duration format (e.g., "PT30S", "PT1M30S")
func parseDuration(durationStr string) float64 {
   // Remove "P" prefix and "T" separator
   durationStr = strings.TrimPrefix(durationStr, "P")
   if strings.Contains(durationStr, "T") {
      parts := strings.Split(durationStr, "T")
      durationStr = parts[len(parts)-1] // Get time part
   }

   // Parse duration (simplified - handles common cases)
   var totalSeconds float64

   // Handle hours (H)
   if strings.Contains(durationStr, "H") {
      parts := strings.Split(durationStr, "H")
      if len(parts) > 1 {
         if hours, err := strconv.ParseFloat(parts[0], 64); err == nil {
            totalSeconds += hours * 3600
         }
         durationStr = parts[1]
      }
   }

   // Handle minutes (M)
   if strings.Contains(durationStr, "M") {
      parts := strings.Split(durationStr, "M")
      if len(parts) > 1 {
         if minutes, err := strconv.ParseFloat(parts[0], 64); err == nil {
            totalSeconds += minutes * 60
         }
         durationStr = parts[1]
      }
   }

   // Handle seconds (S)
   if strings.Contains(durationStr, "S") {
      secondsStr := strings.TrimSuffix(durationStr, "S")
      if seconds, err := strconv.ParseFloat(secondsStr, 64); err == nil {
         totalSeconds += seconds
      }
   }

   return totalSeconds
}

// resolveBaseURL resolves the base URL by combining all base URLs from parent elements
func resolveBaseURL(baseMPDURL string, urls ...string) string {
   currentURL := baseMPDURL

   for _, baseURL := range urls {
      if baseURL != "" {
         currentURL = resolveURL(currentURL, baseURL)
      }
   }

   return currentURL
}

// resolveURL resolves a relative URL against a base URL using only net/url.URL.ResolveReference
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

   resolved := base.ResolveReference(rel)
   return resolved.String()
}

// generateSegmentURLsFromTemplate generates segment URLs from SegmentTemplate
func generateSegmentURLsFromTemplate(baseURL string, template *SegmentTemplate, representationID string, periodDuration float64) []string {
   var urls []string

   // Parse template parameters
   startNumber := 1
   if template.StartNumber != "" {
      if n, err := strconv.Atoi(template.StartNumber); err == nil {
         startNumber = n
      }
   }

   // Add initialization segment if specified as attribute of SegmentTemplate
   if template.Initialization != "" {
      initURL := substituteTemplate(template.Initialization, representationID, 0, startNumber, 0)
      fullInitURL := resolveURL(baseURL, initURL)
      urls = append(urls, fullInitURL)
   }

   // Handle SegmentTimeline if present (highest priority)
   if template.SegmentTimeline != nil && len(template.SegmentTimeline.Segments) > 0 {
      // Generate segments from timeline
      segmentURLs := generateSegmentsFromTimeline(baseURL, template, representationID, startNumber)
      urls = append(urls, segmentURLs...)
   } else {
      // Calculate segment count using duration, timescale, and period duration
      segmentCount := 10 // Default fallback

      duration := 0
      if template.Duration != "" {
         if d, err := strconv.Atoi(template.Duration); err == nil {
            duration = d
         }
      }

      timescale := 1
      if template.Timescale != "" {
         if t, err := strconv.Atoi(template.Timescale); err == nil {
            timescale = t
         }
      }

      endNumber := 0
      if template.EndNumber != "" {
         if n, err := strconv.Atoi(template.EndNumber); err == nil {
            endNumber = n
         }
      }

      // Determine number of segments to generate
      if endNumber > 0 {
         // Use endNumber to determine segment count
         segmentCount = endNumber - startNumber + 1
         if segmentCount <= 0 {
            segmentCount = 10 // Fallback if invalid
         }
      } else if duration > 0 && timescale > 0 {
         // Calculate using: ceil(PeriodDurationInSeconds * timescale / duration)
         segmentCount = int(math.Ceil(periodDuration * float64(timescale) / float64(duration)))
         if segmentCount <= 0 {
            segmentCount = 10 // Fallback if calculation fails
         }
      } else if duration > 0 {
         // Generate segments based on duration only (estimate)
         segmentCount = 50 // Default count for duration-based templates
      }

      // Generate media segments
      for i := 0; i < segmentCount; i++ {
         segmentNumber := startNumber + i
         time := i * 1000 // Simplified time calculation
         if template.Media != "" {
            mediaURL := substituteTemplate(template.Media, representationID, i, segmentNumber, time)
            fullMediaURL := resolveURL(baseURL, mediaURL)
            urls = append(urls, fullMediaURL)
         }
      }
   }

   return urls
}

// generateSegmentsFromTimeline generates segments based on SegmentTimeline
func generateSegmentsFromTimeline(baseURL string, template *SegmentTemplate, representationID string, startNumber int) []string {
   var urls []string

   // Process SegmentTimeline segments
   segmentIndex := 0
   currentTime := int64(0)
   segmentNumber := startNumber

   for _, s := range template.SegmentTimeline.Segments {
      // Parse duration
      duration, err := strconv.ParseInt(s.D, 10, 64)
      if err != nil {
         continue
      }

      // Parse time (if present)
      if s.T != "" {
         if t, err := strconv.ParseInt(s.T, 10, 64); err == nil {
            currentTime = t
         }
      }

      // Parse repeat count (default is 0)
      repeat := int64(0)
      if s.R != "" {
         if r, err := strconv.ParseInt(s.R, 10, 64); err == nil {
            repeat = r
         }
      }

      // Generate the first segment
      if template.Media != "" {
         mediaURL := substituteTemplate(template.Media, representationID, int(currentTime), segmentNumber, int(currentTime))
         fullMediaURL := resolveURL(baseURL, mediaURL)
         urls = append(urls, fullMediaURL)
      }

      segmentIndex++
      segmentNumber++
      currentTime += duration

      // Generate repeated segments
      for r := int64(0); r < repeat; r++ {
         if template.Media != "" {
            mediaURL := substituteTemplate(template.Media, representationID, int(currentTime), segmentNumber, int(currentTime))
            fullMediaURL := resolveURL(baseURL, mediaURL)
            urls = append(urls, fullMediaURL)
         }
         segmentIndex++
         segmentNumber++
         currentTime += duration
      }
   }

   return urls
}

// substituteTemplate substitutes placeholders in template strings
func substituteTemplate(template, representationID string, number, segmentNumber, time int) string {
   result := template

   // Replace $RepresentationID$
   result = strings.ReplaceAll(result, "$RepresentationID$", representationID)

   // Replace $Number$ with segment number
   result = strings.ReplaceAll(result, "$Number$", strconv.Itoa(segmentNumber))

   // Replace $Time$ with time value
   result = strings.ReplaceAll(result, "$Time$", strconv.Itoa(time))

   // Handle $Number%0[width]d$ format using regex
   numberRegex := regexp.MustCompile(`\$Number%0(\d+)d\$`)
   result = numberRegex.ReplaceAllStringFunc(result, func(match string) string {
      // Extract width from the pattern
      widthStr := numberRegex.FindStringSubmatch(match)[1]
      if width, err := strconv.Atoi(widthStr); err == nil {
         format := fmt.Sprintf("%%0%dd", width)
         return fmt.Sprintf(format, segmentNumber)
      }
      return match // Return unchanged if parsing fails
   })

   // Handle $Time%0[width]d$ format using regex
   timeRegex := regexp.MustCompile(`\$Time%0(\d+)d\$`)
   result = timeRegex.ReplaceAllStringFunc(result, func(match string) string {
      // Extract width from the pattern
      widthStr := timeRegex.FindStringSubmatch(match)[1]
      if width, err := strconv.Atoi(widthStr); err == nil {
         format := fmt.Sprintf("%%0%dd", width)
         return fmt.Sprintf(format, time)
      }
      return match // Return unchanged if parsing fails
   })

   return result
}
