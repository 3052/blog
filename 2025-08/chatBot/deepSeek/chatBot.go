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
   "time"
)

type MPD struct {
   XMLName                   xml.Name         `xml:"MPD"`
   MediaPresentationDuration string           `xml:"mediaPresentationDuration,attr"`
   BaseURL                   string           `xml:"BaseURL"`
   Periods                   []Period         `xml:"Period"`
   SegmentTemplate           *SegmentTemplate `xml:"SegmentTemplate"`
}

type Period struct {
   ID              string           `xml:"id,attr"`
   Duration        string           `xml:"duration,attr"`
   BaseURL         string           `xml:"BaseURL"`
   AdaptationSets  []AdaptationSet  `xml:"AdaptationSet"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type AdaptationSet struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   Representations []Representation `xml:"Representation"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   Bandwidth       int              `xml:"bandwidth,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   Duration        int              `xml:"duration,attr"`
   Timescale       int              `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media      string `xml:"media,attr"`
   MediaRange string `xml:"mediaRange,attr"`
}

type SegmentTimeline struct {
   Segments []SegmentTimelineSegment `xml:"S"`
}

type SegmentTimelineSegment struct {
   T int `xml:"t,attr"`
   D int `xml:"d,attr"`
   R int `xml:"r,attr"`
}

func main() {
   if len(os.Args) < 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path>")
      os.Exit(1)
   }

   mpdContent, err := ioutil.ReadFile(os.Args[1])
   if err != nil {
      fmt.Printf("Error reading MPD file: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(mpdContent, &mpd); err != nil {
      fmt.Printf("Error parsing MPD XML: %v\n", err)
      os.Exit(1)
   }

   initialMPDURL := "http://test.test/test.mpd"
   result := make(map[string][]string)

   // Start with initial MPD URL as base
   currentBase, err := url.Parse(initialMPDURL)
   if err != nil {
      fmt.Printf("Error parsing initial URL: %v\n", err)
      os.Exit(1)
   }

   // Apply MPD@BaseURL if present
   if mpd.BaseURL != "" {
      currentBase, err = resolveRelative(mpd.BaseURL, currentBase)
      if err != nil {
         fmt.Printf("Error resolving MPD BaseURL: %v\n", err)
      }
   }

   for _, period := range mpd.Periods {
      periodDuration, err := parseDuration(period.Duration)
      if err != nil {
         fmt.Printf("Error parsing period duration: %v\n", err)
         continue
      }
      if periodDuration == 0 {
         periodDuration, err = parseDuration(mpd.MediaPresentationDuration)
         if err != nil {
            fmt.Printf("Error parsing MPD duration: %v\n", err)
            continue
         }
      }

      periodBase := currentBase
      // Apply Period@BaseURL if present
      if period.BaseURL != "" {
         periodBase, err = resolveRelative(period.BaseURL, currentBase)
         if err != nil {
            fmt.Printf("Error resolving Period BaseURL: %v\n", err)
            continue
         }
      }

      for _, adaptationSet := range period.AdaptationSets {
         adaptationBase := periodBase
         // Apply AdaptationSet@BaseURL if present
         if adaptationSet.BaseURL != "" {
            adaptationBase, err = resolveRelative(adaptationSet.BaseURL, periodBase)
            if err != nil {
               fmt.Printf("Error resolving AdaptationSet BaseURL: %v\n", err)
               continue
            }
         }

         for _, representation := range adaptationSet.Representations {
            representationBase := adaptationBase
            // Apply Representation@BaseURL if present
            if representation.BaseURL != "" {
               representationBase, err = resolveRelative(representation.BaseURL, adaptationBase)
               if err != nil {
                  fmt.Printf("Error resolving Representation BaseURL: %v\n", err)
                  continue
               }
            }

            var segmentURLs []string

            if representation.SegmentList != nil {
               segments, err := processSegmentList(representation.SegmentList, representationBase)
               if err != nil {
                  fmt.Printf("Error processing segment list: %v\n", err)
                  continue
               }
               segmentURLs = append(segmentURLs, segments...)
            } else {
               segmentTemplate := getEffectiveSegmentTemplate(
                  representation.SegmentTemplate,
                  adaptationSet.SegmentTemplate,
                  period.SegmentTemplate,
                  mpd.SegmentTemplate,
               )

               if segmentTemplate != nil {
                  if segmentTemplate.Initialization != "" {
                     initURL, err := resolveRelative(segmentTemplate.Initialization, representationBase)
                     if err != nil {
                        fmt.Printf("Error resolving initialization URL: %v\n", err)
                        continue
                     }
                     segmentURLs = append(segmentURLs, initURL.String())
                  }

                  if segmentTemplate.Media != "" {
                     var segments []string
                     var err error

                     if segmentTemplate.SegmentTimeline != nil {
                        segments, err = generateTimelineSegments(segmentTemplate, representation.ID, representationBase)
                     } else if segmentTemplate.EndNumber > 0 {
                        segments, err = generateNumberedSegments(segmentTemplate, representation.ID, representationBase)
                     } else if segmentTemplate.Duration > 0 && periodDuration > 0 {
                        timescale := segmentTemplate.Timescale
                        if timescale == 0 {
                           timescale = 1
                        }
                        segments, err = generateDurationBasedSegments(segmentTemplate, representation.ID, representationBase, periodDuration, timescale)
                     } else {
                        mediaURL, err := resolveRelative(segmentTemplate.Media, representationBase)
                        if err != nil {
                           fmt.Printf("Error resolving media URL: %v\n", err)
                           continue
                        }
                        segments = []string{mediaURL.String()}
                     }

                     if err != nil {
                        fmt.Printf("Error generating segments: %v\n", err)
                        continue
                     }
                     segmentURLs = append(segmentURLs, segments...)
                  }
               } else {
                  // No segment information - use the base URL directly
                  segmentURLs = append(segmentURLs, representationBase.String())
               }
            }

            if len(segmentURLs) > 0 {
               result[representation.ID] = append(result[representation.ID], segmentURLs...)
            }
         }
      }
   }

   jsonResult, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Printf("Error generating JSON: %v\n", err)
      os.Exit(1)
   }
   fmt.Println(string(jsonResult))
}

func resolveRelative(relative string, base *url.URL) (*url.URL, error) {
   // Handle absolute URLs
   if strings.HasPrefix(relative, "http://") || strings.HasPrefix(relative, "https://") {
      return url.Parse(relative)
   }

   // Parse relative URL
   relURL, err := url.Parse(relative)
   if err != nil {
      return nil, fmt.Errorf("error parsing relative URL '%s': %w", relative, err)
   }

   // Resolve against base URL
   return base.ResolveReference(relURL), nil
}

func replaceTemplateVariables(pattern, representationID string, number, time int) string {
   result := pattern

   // Handle RepresentationID
   result = strings.ReplaceAll(result, "$RepresentationID$", representationID)

   // Handle Number with formatting (e.g., $Number%08d$)
   for {
      start := strings.Index(result, "$Number%")
      if start < 0 {
         break
      }

      formatStart := start + len("$Number%")
      end := strings.Index(result[formatStart:], "$")
      if end < 0 {
         break
      }
      end += formatStart

      if formatStart >= end {
         break
      }
      formatSpec := result[formatStart:end]
      formattedNumber := fmt.Sprintf("%"+formatSpec, number)
      result = result[:start] + formattedNumber + result[end+1:]
   }

   // Default number replacement
   result = strings.ReplaceAll(result, "$Number$", strconv.Itoa(number))

   // Handle Time
   result = strings.ReplaceAll(result, "$Time$", strconv.Itoa(time))

   // Handle Bandwidth
   result = strings.ReplaceAll(result, "$Bandwidth$", "0")

   return result
}

func parseDuration(durationStr string) (time.Duration, error) {
   if durationStr == "" {
      return 0, nil
   }

   // ISO 8601 duration format (only time portion after PT)
   if strings.HasPrefix(durationStr, "PT") {
      timePart := strings.TrimPrefix(durationStr, "PT")
      if timePart == "" {
         return 0, nil
      }

      var duration time.Duration
      for len(timePart) > 0 {
         // Find numeric portion (including decimal point)
         i := 0
         decimalFound := false
         for i < len(timePart) &&
            ((timePart[i] >= '0' && timePart[i] <= '9') ||
               (timePart[i] == '.' && !decimalFound)) {
            if timePart[i] == '.' {
               decimalFound = true
            }
            i++
         }
         if i == 0 {
            return 0, fmt.Errorf("invalid duration format, expected number: %s", durationStr)
         }

         valStr := timePart[:i]
         val, err := strconv.ParseFloat(valStr, 64)
         if err != nil {
            return 0, fmt.Errorf("invalid duration value '%s': %w", valStr, err)
         }

         // Get time unit
         if i >= len(timePart) {
            return 0, fmt.Errorf("incomplete duration, missing unit: %s", durationStr)
         }
         unit := timePart[i]
         timePart = timePart[i+1:]

         switch unit {
         case 'H':
            duration += time.Duration(val * float64(time.Hour))
         case 'M':
            duration += time.Duration(val * float64(time.Minute))
         case 'S':
            duration += time.Duration(val * float64(time.Second))
         default:
            return 0, fmt.Errorf("unknown time unit '%c' in duration: %s", unit, durationStr)
         }
      }
      return duration, nil
   }

   // Fallback to simple duration parsing
   duration, err := time.ParseDuration(durationStr)
   if err != nil {
      return 0, fmt.Errorf("invalid duration format: %w", err)
   }
   return duration, nil
}

func processSegmentList(segmentList *SegmentList, base *url.URL) ([]string, error) {
   var segments []string

   // Handle initialization segment
   if segmentList.Initialization != nil && segmentList.Initialization.SourceURL != "" {
      initURL, err := resolveRelative(segmentList.Initialization.SourceURL, base)
      if err != nil {
         return nil, fmt.Errorf("error resolving initialization URL: %w", err)
      }
      segments = append(segments, initURL.String())
   }

   // Handle media segments
   for _, segmentURL := range segmentList.SegmentURLs {
      if segmentURL.Media != "" {
         mediaURL, err := resolveRelative(segmentURL.Media, base)
         if err != nil {
            return nil, fmt.Errorf("error resolving segment URL: %w", err)
         }
         segments = append(segments, mediaURL.String())
      }
   }

   return segments, nil
}

func generateTimelineSegments(template *SegmentTemplate, representationID string, base *url.URL) ([]string, error) {
   var segments []string
   startNumber := template.StartNumber
   if startNumber == 0 {
      startNumber = 1
   }

   currentTime := 0
   if len(template.SegmentTimeline.Segments) > 0 {
      currentTime = template.SegmentTimeline.Segments[0].T
   }

   segmentCount := 0
   for _, s := range template.SegmentTimeline.Segments {
      repeat := max(0, s.R)
      for i := 0; i <= repeat; i++ {
         segmentNumber := startNumber + segmentCount
         mediaURL, err := resolveRelative(template.Media, base)
         if err != nil {
            return nil, fmt.Errorf("error resolving media URL: %w", err)
         }
         urlStr := replaceTemplateVariables(mediaURL.String(), representationID, segmentNumber, currentTime)
         segments = append(segments, urlStr)
         currentTime += s.D
         segmentCount++
      }
   }
   return segments, nil
}

func generateNumberedSegments(template *SegmentTemplate, representationID string, base *url.URL) ([]string, error) {
   var segments []string
   startNumber := template.StartNumber
   if startNumber == 0 {
      startNumber = 1
   }

   for number := startNumber; number <= template.EndNumber; number++ {
      mediaURL, err := resolveRelative(template.Media, base)
      if err != nil {
         return nil, fmt.Errorf("error resolving media URL: %w", err)
      }
      urlStr := replaceTemplateVariables(mediaURL.String(), representationID, number, 0)
      segments = append(segments, urlStr)
   }
   return segments, nil
}

func generateDurationBasedSegments(template *SegmentTemplate, representationID string, base *url.URL, periodDuration time.Duration, timescale int) ([]string, error) {
   var segments []string
   startNumber := template.StartNumber
   if startNumber == 0 {
      startNumber = 1
   }

   durationSeconds := periodDuration.Seconds()
   totalSegments := int(math.Ceil(durationSeconds * float64(timescale) / float64(template.Duration)))

   for i := 0; i < totalSegments; i++ {
      segmentNumber := startNumber + i
      time := i * template.Duration
      mediaURL, err := resolveRelative(template.Media, base)
      if err != nil {
         return nil, fmt.Errorf("error resolving media URL: %w", err)
      }
      urlStr := replaceTemplateVariables(mediaURL.String(), representationID, segmentNumber, time)
      segments = append(segments, urlStr)
   }

   return segments, nil
}

func getEffectiveSegmentTemplate(templates ...*SegmentTemplate) *SegmentTemplate {
   for _, t := range templates {
      if t != nil {
         return t
      }
   }
   return nil
}

func max(a, b int) int {
   if a > b {
      return a
   }
   return b
}
