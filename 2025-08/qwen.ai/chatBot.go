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
)

// MPD represents the root MPD element
type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   Duration string   `xml:"mediaPresentationDuration,attr"`
   Periods []Period `xml:"Period"`
}

// Period represents a Period element
type Period struct {
   XMLName        xml.Name         `xml:"Period"`
   Duration       string           `xml:"duration,attr"`
   BaseURL        string           `xml:"BaseURL"`
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
   XMLName   xml.Name `xml:"Representation"`
   ID        string   `xml:"id,attr"`
   BaseURL   string   `xml:"BaseURL"`
   SegmentBase *SegmentBase `xml:"SegmentBase"`
   SegmentList *SegmentList `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

// SegmentBase represents SegmentBase element
type SegmentBase struct {
   XMLName xml.Name `xml:"SegmentBase"`
   Initialization *URL `xml:"Initialization"`
}

// SegmentList represents SegmentList element
type SegmentList struct {
   XMLName xml.Name `xml:"SegmentList"`
   Initialization *URL `xml:"Initialization"`
   SegmentURLs    []SegmentURL `xml:"SegmentURL"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentURL represents SegmentURL element
type SegmentURL struct {
   XMLName xml.Name `xml:"SegmentURL"`
   Media   string   `xml:"media,attr"`
}

// SegmentTemplate represents SegmentTemplate element
type SegmentTemplate struct {
   XMLName      xml.Name `xml:"SegmentTemplate"`
   Initialization string   `xml:"initialization,attr"`
   Media        string   `xml:"media,attr"`
   StartNumber  string   `xml:"startNumber,attr"`
   EndNumber    string   `xml:"endNumber,attr"`
   Timescale    string   `xml:"timescale,attr"`
   Duration     string   `xml:"duration,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline represents SegmentTimeline element
type SegmentTimeline struct {
   XMLName xml.Name `xml:"SegmentTimeline"`
   Segments []SegmentTimelineSegment `xml:"S"`
}

// SegmentTimelineSegment represents S elements in SegmentTimeline
type SegmentTimelineSegment struct {
   XMLName xml.Name `xml:"S"`
   T       string   `xml:"t,attr"` // presentation time
   D       string   `xml:"d,attr"` // duration
   R       string   `xml:"r,attr"` // repeat count
}

// URL represents elements with @sourceURL or @url attributes
type URL struct {
   SourceURL string `xml:"sourceURL,attr"`
   URL       string `xml:"url,attr"`
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintln(os.Stderr, "Usage: go run main.go <mpd_file_path>")
      os.Exit(1)
   }

   mpdFilePath := os.Args[1]
   
   // Read the MPD file
   data, err := os.ReadFile(mpdFilePath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading MPD file: %v\n", err)
      os.Exit(1)
   }

   // Parse the MPD XML
   var mpd MPD
   err = xml.Unmarshal(data, &mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD XML: %v\n", err)
      os.Exit(1)
   }

   // Base URL for resolving relative URLs
   baseMPDURL := "http://test.test/test.mpd"
   baseURL, err := url.Parse(baseMPDURL)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing base URL: %v\n", err)
      os.Exit(1)
   }

   // Map to store representation ID to segment URLs (using slice to preserve order and allow appending)
   result := make(map[string][]string)

   // Process each period
   for _, period := range mpd.Periods {
      periodBaseURL := resolveURL(baseURL, period.BaseURL)
      if periodBaseURL == nil {
         fmt.Fprintf(os.Stderr, "Error resolving period base URL: %s\n", period.BaseURL)
         continue
      }
      
      // Calculate period duration in seconds
      periodDuration := parseDuration(period.Duration)
      if periodDuration <= 0 {
         // Fall back to MPD duration if period duration not specified
         periodDuration = parseDuration(mpd.Duration)
      }
      
      // Process each adaptation set
      for _, adaptationSet := range period.AdaptationSets {
         adaptationSetBaseURL := resolveURL(periodBaseURL, adaptationSet.BaseURL)
         if adaptationSetBaseURL == nil {
            fmt.Fprintf(os.Stderr, "Error resolving adaptation set base URL: %s\n", adaptationSet.BaseURL)
            continue
         }
         
         // Process each representation
         for _, representation := range adaptationSet.Representations {
            representationBaseURL := resolveURL(adaptationSetBaseURL, representation.BaseURL)
            if representationBaseURL == nil {
               fmt.Fprintf(os.Stderr, "Error resolving representation base URL: %s\n", representation.BaseURL)
               continue
            }
            
            var segmentURLs []string
            
            // Handle SegmentList with SegmentTimeline
            if representation.SegmentList != nil {
               // Add initialization segment if it exists
               if representation.SegmentList.Initialization != nil {
                  initURL := representation.SegmentList.Initialization.SourceURL
                  if initURL == "" {
                     initURL = representation.SegmentList.Initialization.URL
                  }
                  if initURL != "" {
                     initResolvedURL := resolveURL(representationBaseURL, initURL)
                     if initResolvedURL != nil {
                        segmentURLs = append(segmentURLs, initResolvedURL.String())
                     } else {
                        fmt.Fprintf(os.Stderr, "Error resolving initialization URL: %s\n", initURL)
                     }
                  }
               }
               
               if representation.SegmentList.SegmentTimeline != nil {
                  // Generate URLs based on SegmentTimeline
                  timelineURLs := generateSegmentTimelineURLs(
                     representation.SegmentList.SegmentTimeline,
                     representationBaseURL,
                     representation.SegmentList.SegmentURLs,
                  )
                  segmentURLs = append(segmentURLs, timelineURLs...)
               } else {
                  // Handle regular SegmentList
                  for _, segmentURL := range representation.SegmentList.SegmentURLs {
                     if segmentURL.Media != "" {
                        mediaResolvedURL := resolveURL(representationBaseURL, segmentURL.Media)
                        if mediaResolvedURL != nil {
                           segmentURLs = append(segmentURLs, mediaResolvedURL.String())
                        } else {
                           fmt.Fprintf(os.Stderr, "Error resolving segment URL: %s\n", segmentURL.Media)
                        }
                     }
                  }
               }
            }
            
            // Handle SegmentTemplate inheritance
            var effectiveSegmentTemplate *SegmentTemplate
            
            // Check Representation level first
            if representation.SegmentTemplate != nil {
               effectiveSegmentTemplate = representation.SegmentTemplate
            } else if adaptationSet.SegmentTemplate != nil {
               // Fall back to AdaptationSet level
               effectiveSegmentTemplate = adaptationSet.SegmentTemplate
            }
            
            // Handle SegmentTemplate with SegmentTimeline
            if effectiveSegmentTemplate != nil {
               // Add initialization segment if it exists
               if effectiveSegmentTemplate.Initialization != "" {
                  initURL := replaceTemplateVariables(effectiveSegmentTemplate.Initialization, 0, 0, representation.ID)
                  initResolvedURL := resolveURL(representationBaseURL, initURL)
                  if initResolvedURL != nil {
                     segmentURLs = append(segmentURLs, initResolvedURL.String())
                  } else {
                     fmt.Fprintf(os.Stderr, "Error resolving initialization URL: %s\n", initURL)
                  }
               }
               
               if effectiveSegmentTemplate.Media != "" {
                  if effectiveSegmentTemplate.SegmentTimeline != nil {
                     // Generate URLs based on SegmentTimeline in SegmentTemplate
                     timelineURLs := generateSegmentTemplateTimelineURLs(
                        effectiveSegmentTemplate.SegmentTimeline,
                        effectiveSegmentTemplate.Media,
                        representationBaseURL,
                        effectiveSegmentTemplate.StartNumber,
                        representation.ID,
                     )
                     segmentURLs = append(segmentURLs, timelineURLs...)
                  } else if effectiveSegmentTemplate.Duration != "" {
                     // Generate URLs based on duration with proper segment count calculation
                     durationURLs := generateSegmentTemplateDurationURLs(
                        effectiveSegmentTemplate.Media,
                        representationBaseURL,
                        effectiveSegmentTemplate.StartNumber,
                        effectiveSegmentTemplate.EndNumber,
                        effectiveSegmentTemplate.Duration,
                        effectiveSegmentTemplate.Timescale,
                        representation.ID,
                        periodDuration,
                     )
                     segmentURLs = append(segmentURLs, durationURLs...)
                  } else {
                     // Handle simple template patterns
                     mediaPattern := effectiveSegmentTemplate.Media
                     if !strings.Contains(mediaPattern, "$") {
                        mediaResolvedURL := resolveURL(representationBaseURL, mediaPattern)
                        if mediaResolvedURL != nil {
                           segmentURLs = append(segmentURLs, mediaResolvedURL.String())
                        } else {
                           fmt.Fprintf(os.Stderr, "Error resolving media URL: %s\n", mediaPattern)
                        }
                     } else {
                        if strings.Contains(mediaPattern, "$Number") {
                           // Generate URLs with start/end number support
                           startNum := 1
                           if effectiveSegmentTemplate.StartNumber != "" {
                              if num, err := strconv.Atoi(effectiveSegmentTemplate.StartNumber); err == nil {
                                 startNum = num
                              } else {
                                 fmt.Fprintf(os.Stderr, "Error parsing startNumber: %s\n", effectiveSegmentTemplate.StartNumber)
                              }
                           }
                           
                           endNum := startNum + 4 // Default 5 segments
                           if effectiveSegmentTemplate.EndNumber != "" {
                              if num, err := strconv.Atoi(effectiveSegmentTemplate.EndNumber); err == nil {
                                 endNum = num
                              } else {
                                 fmt.Fprintf(os.Stderr, "Error parsing endNumber: %s\n", effectiveSegmentTemplate.EndNumber)
                              }
                           }
                           
                           for i := startNum; i <= endNum; i++ {
                              urlStr := replaceTemplateVariables(mediaPattern, i, (i-startNum)*1000, representation.ID)
                              mediaResolvedURL := resolveURL(representationBaseURL, urlStr)
                              if mediaResolvedURL != nil {
                                 segmentURLs = append(segmentURLs, mediaResolvedURL.String())
                              } else {
                                 fmt.Fprintf(os.Stderr, "Error resolving media URL: %s\n", urlStr)
                              }
                           }
                        } else if strings.Contains(mediaPattern, "$Time") {
                           times := []int{0, 1000, 2000, 3000, 4000}
                           for _, time := range times {
                              urlStr := replaceTemplateVariables(mediaPattern, time/1000+1, time, representation.ID)
                              mediaResolvedURL := resolveURL(representationBaseURL, urlStr)
                              if mediaResolvedURL != nil {
                                 segmentURLs = append(segmentURLs, mediaResolvedURL.String())
                              } else {
                                 fmt.Fprintf(os.Stderr, "Error resolving media URL: %s\n", urlStr)
                              }
                           }
                        } else {
                           urlStr := replaceTemplateVariables(mediaPattern, 1, 0, representation.ID)
                           mediaResolvedURL := resolveURL(representationBaseURL, urlStr)
                           if mediaResolvedURL != nil {
                              segmentURLs = append(segmentURLs, mediaResolvedURL.String())
                           } else {
                              fmt.Fprintf(os.Stderr, "Error resolving media URL: %s\n", urlStr)
                           }
                        }
                     }
                  }
               }
            }
            
            // If no segments found, try to use the representation BaseURL as a segment
            if len(segmentURLs) == 0 && representation.BaseURL != "" {
               if representationBaseURL != nil {
                  segmentURLs = append(segmentURLs, representationBaseURL.String())
               }
            }
            
            // Append segments to existing representation ID or create new entry
            if existingSegments, exists := result[representation.ID]; exists {
               // Append new segments to existing ones
               result[representation.ID] = append(existingSegments, segmentURLs...)
            } else {
               // Create new entry
               result[representation.ID] = segmentURLs
            }
         }
      }
   }

   // Output as JSON
   jsonData, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonData))
}

// parseDuration parses ISO 8601 duration format (e.g., "PT30S", "PT1M30S", "PT2H")
func parseDuration(durationStr string) float64 {
   if durationStr == "" {
      return 0
   }
   
   // Remove "P" prefix
   if !strings.HasPrefix(durationStr, "P") {
      return 0
   }
   
   // Handle time part (after T)
   timePart := ""
   if strings.Contains(durationStr, "T") {
      parts := strings.Split(durationStr, "T")
      if len(parts) == 2 {
         timePart = parts[1]
      }
   } else {
      // If no T, the duration might be in the date part, but we're interested in time
      return 0
   }
   
   totalSeconds := 0.0
   
   // Parse hours (H)
   if strings.Contains(timePart, "H") {
      parts := strings.Split(timePart, "H")
      if len(parts) >= 1 && parts[0] != "" {
         if hours, err := strconv.ParseFloat(parts[0], 64); err == nil {
            totalSeconds += hours * 3600
         } else {
            fmt.Fprintf(os.Stderr, "Error parsing hours in duration: %s\n", parts[0])
         }
         if len(parts) > 1 {
            timePart = parts[1]
         }
      }
   }
   
   // Parse minutes (M)
   if strings.Contains(timePart, "M") {
      parts := strings.Split(timePart, "M")
      if len(parts) >= 1 && parts[0] != "" {
         if minutes, err := strconv.ParseFloat(parts[0], 64); err == nil {
            totalSeconds += minutes * 60
         } else {
            fmt.Fprintf(os.Stderr, "Error parsing minutes in duration: %s\n", parts[0])
         }
         if len(parts) > 1 {
            timePart = parts[1]
         }
      }
   }
   
   // Parse seconds (S)
   if strings.Contains(timePart, "S") {
      parts := strings.Split(timePart, "S")
      if len(parts) >= 1 && parts[0] != "" {
         if seconds, err := strconv.ParseFloat(parts[0], 64); err == nil {
            totalSeconds += seconds
         } else {
            fmt.Fprintf(os.Stderr, "Error parsing seconds in duration: %s\n", parts[0])
         }
      }
   }
   
   return totalSeconds
}

// replaceTemplateVariables replaces all template variables in a URL template
func replaceTemplateVariables(template string, number int, time int, representationID string) string {
   result := template
   
   // Replace $Number%format$ and $Number$ patterns
   result = replaceNumberTemplate(result, number)
   
   // Replace $Time%format$ and $Time$ patterns
   result = replaceTimeTemplate(result, time)
   
   // Replace $RepresentationID%format$ and $RepresentationID$ patterns
   result = replaceRepresentationIDTemplate(result, representationID)
   
   return result
}

// replaceNumberTemplate handles $Number%format$ and $Number$ patterns
func replaceNumberTemplate(template string, number int) string {
   result := template
   
   // Handle $Number%format$ patterns
   for {
      start := strings.Index(result, "$Number%")
      if start == -1 {
         break
      }
      
      // Look for the closing $ after the %
      formatStart := start + 7 // position after "$Number%"
      closingDollar := strings.Index(result[formatStart:], "$")
      if closingDollar == -1 {
         break
      }
      closingDollar += formatStart
      
      formatPart := result[formatStart:closingDollar] // Extract format part
      formattedNumber := formatNumber(number, formatPart)
      result = result[:start] + formattedNumber + result[closingDollar+1:]
   }
   
   // Handle simple $Number$ patterns
   result = strings.ReplaceAll(result, "$Number$", fmt.Sprintf("%d", number))
   
   // Handle simple $Number patterns (without trailing $)
   result = strings.ReplaceAll(result, "$Number", fmt.Sprintf("%d", number))
   
   return result
}

// replaceTimeTemplate handles $Time%format$ and $Time$ patterns
func replaceTimeTemplate(template string, time int) string {
   result := template
   
   // Handle $Time%format$ patterns
   for {
      start := strings.Index(result, "$Time%")
      if start == -1 {
         break
      }
      
      // Look for the closing $ after the %
      formatStart := start + 6 // position after "$Time%"
      closingDollar := strings.Index(result[formatStart:], "$")
      if closingDollar == -1 {
         break
      }
      closingDollar += formatStart
      
      formatPart := result[formatStart:closingDollar] // Extract format part
      formattedTime := formatNumber(time, formatPart)
      result = result[:start] + formattedTime + result[closingDollar+1:]
   }
   
   // Handle simple $Time$ patterns
   result = strings.ReplaceAll(result, "$Time$", fmt.Sprintf("%d", time))
   
   // Handle simple $Time patterns (without trailing $)
   result = strings.ReplaceAll(result, "$Time", fmt.Sprintf("%d", time))
   
   return result
}

// replaceRepresentationIDTemplate handles $RepresentationID%format$ and $RepresentationID$ patterns
func replaceRepresentationIDTemplate(template string, representationID string) string {
   result := template
   
   // Handle $RepresentationID%format$ patterns
   for {
      start := strings.Index(result, "$RepresentationID%")
      if start == -1 {
         break
      }
      
      // Look for the closing $ after the %
      formatStart := start + 17 // position after "$RepresentationID%"
      closingDollar := strings.Index(result[formatStart:], "$")
      if closingDollar == -1 {
         break
      }
      closingDollar += formatStart
      
      // For RepresentationID, we just use the ID as-is (no numeric formatting)
      formatPart := result[formatStart:closingDollar] // Extract format part
      formattedID := formatString(representationID, formatPart)
      result = result[:start] + formattedID + result[closingDollar+1:]
   }
   
   // Handle simple $RepresentationID$ patterns
   result = strings.ReplaceAll(result, "$RepresentationID$", representationID)
   
   // Handle simple $RepresentationID patterns (without trailing $)
   result = strings.ReplaceAll(result, "$RepresentationID", representationID)
   
   return result
}

// formatNumber formats a number according to the format specification
func formatNumber(number int, format string) string {
   // Handle common format patterns like 08d, 05d, d, etc.
   if strings.HasPrefix(format, "0") && strings.HasSuffix(format, "d") {
      // Handle 0Nd patterns (zero-padded)
      widthStr := format[1 : len(format)-1]
      if width, err := strconv.Atoi(widthStr); err == nil {
         return fmt.Sprintf("%0*d", width, number)
      }
   } else if format == "d" {
      // Handle d pattern
      return fmt.Sprintf("%d", number)
   }
   
   // Default fallback
   return fmt.Sprintf("%d", number)
}

// formatString formats a string according to the format specification
func formatString(str string, format string) string {
   // For strings, we typically don't apply numeric formatting
   // Just return the string as-is
   return str
}

// generateSegmentTimelineURLs generates URLs based on SegmentTimeline in SegmentList
func generateSegmentTimelineURLs(timeline *SegmentTimeline, baseURL *url.URL, segmentURLs []SegmentURL) []string {
   var result []string
   
   // Use the actual segment URLs from SegmentList if available
   for _, segmentURL := range segmentURLs {
      if segmentURL.Media != "" {
         mediaResolvedURL := resolveURL(baseURL, segmentURL.Media)
         if mediaResolvedURL != nil {
            result = append(result, mediaResolvedURL.String())
         } else {
            fmt.Fprintf(os.Stderr, "Error resolving segment URL: %s\n", segmentURL.Media)
         }
      }
   }
   
   return result
}

// generateSegmentTemplateTimelineURLs generates URLs based on SegmentTimeline in SegmentTemplate
func generateSegmentTemplateTimelineURLs(timeline *SegmentTimeline, mediaTemplate string, baseURL *url.URL, startNumber string, representationID string) []string {
   var result []string
   
   startNum := 1
   if startNumber != "" {
      if num, err := strconv.Atoi(startNumber); err == nil {
         startNum = num
      } else {
         fmt.Fprintf(os.Stderr, "Error parsing startNumber: %s\n", startNumber)
      }
   }
   
   segmentNumber := startNum
   timeValue := 0
   
   for _, segment := range timeline.Segments {
      // Handle presentation time if specified
      if segment.T != "" {
         if t, err := strconv.Atoi(segment.T); err == nil {
            timeValue = t
         } else {
            fmt.Fprintf(os.Stderr, "Error parsing segment time: %s\n", segment.T)
         }
      }
      
      // Get segment duration
      duration := 0
      if segment.D != "" {
         if d, err := strconv.Atoi(segment.D); err == nil {
            duration = d
         } else {
            fmt.Fprintf(os.Stderr, "Error parsing segment duration: %s\n", segment.D)
         }
      }
      
      // Handle repeat count
      repeatCount := 0
      if segment.R != "" {
         if r, err := strconv.Atoi(segment.R); err == nil {
            repeatCount = r
         } else if segment.R == "-1" {
            // Special case: repeat until end of period (simplified)
            repeatCount = 0 // For now, just treat as no repeat
         } else {
            fmt.Fprintf(os.Stderr, "Error parsing segment repeat count: %s\n", segment.R)
         }
      }
      
      // Generate URLs for this segment and its repeats
      for i := 0; i <= repeatCount; i++ {
         // Replace all template variables
         urlStr := replaceTemplateVariables(mediaTemplate, segmentNumber, timeValue, representationID)
         
         mediaResolvedURL := resolveURL(baseURL, urlStr)
         if mediaResolvedURL != nil {
            result = append(result, mediaResolvedURL.String())
         } else {
            fmt.Fprintf(os.Stderr, "Error resolving media URL: %s\n", urlStr)
         }
         segmentNumber++
         
         // Increment time by segment duration for next iteration
         timeValue += duration
      }
   }
   
   return result
}

// generateSegmentTemplateDurationURLs generates URLs based on duration in SegmentTemplate with proper segment count calculation
func generateSegmentTemplateDurationURLs(mediaTemplate string, baseURL *url.URL, startNumber string, endNumber string, duration string, timescale string, representationID string, periodDuration float64) []string {
   var result []string
   
   startNum := 1
   if startNumber != "" {
      if num, err := strconv.Atoi(startNumber); err == nil {
         startNum = num
      } else {
         fmt.Fprintf(os.Stderr, "Error parsing startNumber: %s\n", startNumber)
      }
   }
   
   // Determine end number
   var endNum int
   if endNumber != "" {
      // Use explicit end number if provided
      if num, err := strconv.Atoi(endNumber); err == nil {
         endNum = num
      } else {
         fmt.Fprintf(os.Stderr, "Error parsing endNumber: %s, using default\n", endNumber)
         endNum = startNum + 9 // Default fallback
      }
   } else {
      // Calculate number of segments based on duration, timescale, and period duration
      durationValue := int64(1000) // Default duration
      if duration != "" {
         if d, err := strconv.ParseInt(duration, 10, 64); err == nil {
            durationValue = d
         } else {
            fmt.Fprintf(os.Stderr, "Error parsing duration: %s\n", duration)
         }
      }
      
      timescaleValue := int64(1) // Default timescale to 1
      if timescale != "" {
         if t, err := strconv.ParseInt(timescale, 10, 64); err == nil {
            timescaleValue = t
         } else {
            fmt.Fprintf(os.Stderr, "Error parsing timescale: %s, using default value 1\n", timescale)
            timescaleValue = 1
         }
      }
      
      // Calculate number of segments: ceil(PeriodDurationInSeconds * timescale / duration)
      if periodDuration > 0 && durationValue > 0 {
         segmentCount := int64(math.Ceil(periodDuration * float64(timescaleValue) / float64(durationValue)))
         endNum = startNum + int(segmentCount) - 1
      } else {
         endNum = startNum + 9 // Default fallback
      }
   }
   
   // Get duration value for time calculation
   durationValue := int64(1000)
   if duration != "" {
      if d, err := strconv.ParseInt(duration, 10, 64); err == nil {
         durationValue = d
      } else {
         fmt.Fprintf(os.Stderr, "Error parsing duration for time calculation: %s\n", duration)
      }
   }
   
   timescaleValue := int64(1) // Default timescale to 1
   if timescale != "" {
      if t, err := strconv.ParseInt(timescale, 10, 64); err == nil {
         timescaleValue = t
      } else {
         fmt.Fprintf(os.Stderr, "Error parsing timescale for time calculation: %s, using default value 1\n", timescale)
         timescaleValue = 1
      }
   }
   
   // Generate segments from start to end number
   for i := startNum; i <= endNum; i++ {
      segmentNumber := i
      // Calculate time value: (segmentNumber - startNum) * duration * (1000/timescale)
      // This converts from timescale units to milliseconds
      timeValue := int(float64(i-startNum) * float64(durationValue) * (1000.0 / float64(timescaleValue)))
      
      // Replace all template variables
      urlStr := replaceTemplateVariables(mediaTemplate, segmentNumber, timeValue, representationID)
      
      mediaResolvedURL := resolveURL(baseURL, urlStr)
      if mediaResolvedURL != nil {
         result = append(result, mediaResolvedURL.String())
      } else {
         fmt.Fprintf(os.Stderr, "Error resolving media URL: %s\n", urlStr)
      }
   }
   
   return result
}

// resolveURL resolves a relative URL against a base URL
func resolveURL(base *url.URL, relative string) *url.URL {
   if relative == "" {
      return base
   }
   
   // Parse the relative URL
   relURL, err := url.Parse(relative)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing relative URL '%s': %v\n", relative, err)
      return nil
   }
   
   // If relative URL has a scheme, it's absolute
   if relURL.Scheme != "" {
      return relURL
   }
   
   // Resolve against base URL
   resolved := base.ResolveReference(relURL)
   if resolved == nil {
      fmt.Fprintf(os.Stderr, "Error resolving URL: base='%s', relative='%s'\n", base.String(), relative)
      return nil
   }
   
   return resolved
}
