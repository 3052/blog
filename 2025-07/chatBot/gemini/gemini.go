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
)

// MPD represents the top-level structure of a DASH MPD file
type MPD struct {
   XMLName                    xml.Name     `xml:"MPD"`
   BaseURL                    string       `xml:"BaseURL,omitempty"`
   MediaPresentationDuration  string       `xml:"mediaPresentationDuration,attr"`
   MinBufferTime              string       `xml:"minBufferTime,attr"`
   Type                       string       `xml:"type,attr"`
   Profiles                   string       `xml:"profiles,attr"`
   Timescale                  uint64       `xml:"timescale,attr"` // Added Timescale to MPD
   Periods                    []Period     `xml:"Period"`
}

// Period represents a Period element in the MPD
type Period struct {
   XMLName      xml.Name       `xml:"Period"`
   ID           string         `xml:"id,attr"`
   Duration     string         `xml:"duration,attr"`
   BaseURL      string         `xml:"BaseURL,omitempty"`
   Timescale    uint64         `xml:"timescale,attr"` // Added Timescale to Period
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet represents an AdaptationSet element in the MPD
type AdaptationSet struct {
   XMLName      xml.Name       `xml:"AdaptationSet"`
   ID           string         `xml:"id,attr"`
   ContentType  string         `xml:"contentType,attr"`
   MimeType     string         `xml:"mimeType,attr"`
   Codecs       string         `xml:"codecs,attr"`
   Timescale    uint64         `xml:"timescale,attr"` // Added Timescale to AdaptationSet
   Representations []Representation `xml:"Representation"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"` // This can be at AdaptationSet level
}

// Representation represents a Representation element in the MPD
type Representation struct {
   XMLName         xml.Name         `xml:"Representation"`
   ID              string           `xml:"id,attr"`
   Bandwidth       string           `xml:"bandwidth,attr"`
   Width           string           `xml:"width,attr"`
   Height          string           `xml:"height,attr"`
   Codecs          string           `xml:"codecs,attr"`
   MimeType        string           `xml:"mimeType,attr"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"` // This can be at Representation level
   BaseURL         string           `xml:"BaseURL,omitempty"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
}

// SegmentTemplate represents a SegmentTemplate element in the MPD
type SegmentTemplate struct {
   XMLName            xml.Name        `xml:"SegmentTemplate"`
   Timescale          uint64          `xml:"timescale,attr"`
   Initialization     string          `xml:"initialization,attr"`
   Media              string          `xml:"media,attr"`
   StartNumber        uint64          `xml:"startNumber,attr"`
   Duration           uint64          `xml:"duration,attr"`
   EndNumber          uint64          `xml:"endNumber,attr"`
   SegmentTimeline    *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline represents a SegmentTimeline element in the MPD
type SegmentTimeline struct {
   XMLName xml.Name `xml:"SegmentTimeline"`
   Ss      []S      `xml:"S"`
}

// S represents an S element within a SegmentTimeline
type S struct {
   XMLName xml.Name `xml:"S"`
   T       uint64   `xml:"t,attr"`
   D       uint64   `xml:"d,attr"`
   R       int      `xml:"r,attr"` // r can be -1
}

// SegmentBase represents a SegmentBase element in the MPD
type SegmentBase struct {
   XMLName      xml.Name    `xml:"SegmentBase"`
   IndexRange   string      `xml:"indexRange,attr"`
   Initialization *Initialization `xml:"Initialization"`
}

// Initialization represents an Initialization element within SegmentBase
type Initialization struct {
   XMLName xml.Name `xml:"Initialization"`
   Range   string   `xml:"range,attr"`
}


func resolveURL(base *url.URL, relativePath string) string {
   relativeURL, err := url.Parse(relativePath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing relative URL %s: %v\n", relativePath, err)
      return relativePath // Return original path if parsing fails
   }
   return base.ResolveReference(relativeURL).String()
}

// ParseDurationToSeconds parses an ISO 8601 duration string (e.g., "PT1H37M14.320S") into seconds.
func ParseDurationToSeconds(duration string) (float64, error) {
   re := regexp.MustCompile(`P(?:(\d+)Y)?(?:(\d+)M)?(?:(\d+)D)?T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?`)
   matches := re.FindStringSubmatch(duration)

   if len(matches) == 0 {
      return 0, fmt.Errorf("invalid duration format: %s", duration)
   }

   var totalSeconds float64

   // Extract hours
   if matches[4] != "" {
      h, err := strconv.ParseFloat(matches[4], 64)
      if err != nil { return 0, err }
      totalSeconds += h * 3600
   }
   // Extract minutes
   if matches[5] != "" {
      m, err := strconv.ParseFloat(matches[5], 64)
      if err != nil { return 0, err }
      totalSeconds += m * 60
   }
   // Extract seconds
   if matches[6] != "" {
      s, err := strconv.ParseFloat(matches[6], 64)
      if err != nil { return 0, err }
      totalSeconds += s
   }

   return totalSeconds, nil
}

// getInheritedTimescale determines the effective timescale for a SegmentTemplate
func getInheritedTimescale(mpd *MPD, period *Period, as *AdaptationSet, st *SegmentTemplate) uint64 {
   if st != nil && st.Timescale != 0 {
      return st.Timescale
   }
   if as != nil && as.Timescale != 0 {
      return as.Timescale
   }
   if period != nil && period.Timescale != 0 {
      return period.Timescale
   }
   if mpd != nil && mpd.Timescale != 0 {
      return mpd.Timescale
   }
   return 1 // Default to 1 if not specified anywhere
}

// deduplicateStrings removes duplicate strings from a slice, preserving order of first appearance.
func deduplicateStrings(slice []string) []string {
    seen := make(map[string]struct{})
    var result []string
    for _, item := range slice {
        if _, exists := seen[item]; !exists {
            seen[item] = struct{}{}
            result = append(result, item)
        }
    }
    return result
}


func main() {
   if len(os.Args) < 2 {
      fmt.Println("Usage: go run main.go <path_to_mpd_file>")
      return
   }

   mpdFilePath := os.Args[1]
   mpdContent, err := ioutil.ReadFile(mpdFilePath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading MPD file: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   err = xml.Unmarshal(mpdContent, &mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error unmarshalling MPD XML: %v\n", err)
      os.Exit(1)
   }

   // Base URL for resolving relative paths.
   // This is a placeholder; replace with the actual base URL if your MPD uses relative paths
   // and is not being served from the specified base.
   baseURLStr := "http://test.test/test.mpd" 
   parsedBaseURL, err := url.Parse(baseURLStr)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing initial base URL: %v\n", err)
      os.Exit(1)
   }

   // Apply MPD-level BaseURL if present
   currentBaseURL := parsedBaseURL
   if mpd.BaseURL != "" {
      mpdBaseURL, err := url.Parse(mpd.BaseURL)
      if err != nil {
         fmt.Fprintf(os.Stderr, "Warning: Error parsing MPD's BaseURL '%s': %v. Using derived base URL.\n", mpd.BaseURL, err)
      } else {
         currentBaseURL = currentBaseURL.ResolveReference(mpdBaseURL)
      }
   }

   output := make(map[string][]string)

   for _, period := range mpd.Periods {
      // Period-level BaseURL overrides parent (MPD)
      periodBaseURL := currentBaseURL
      if period.BaseURL != "" {
         pb, err := url.Parse(period.BaseURL)
         if err != nil {
            fmt.Fprintf(os.Stderr, "Warning: Error parsing Period BaseURL '%s': %v. Using parent.\n", period.BaseURL, err)
         } else {
            periodBaseURL = periodBaseURL.ResolveReference(pb)
         }
      }

      for _, as := range period.AdaptationSets {
         // AdaptationSet itself doesn't have BaseURL in standard schema,
         // but its children (Representations) inherit from Period.
         asBaseURL := periodBaseURL

         for _, rep := range as.Representations {
            // Representation-level BaseURL overrides parent (Period/AdaptationSet)
            repBaseURL := asBaseURL
            if rep.BaseURL != "" {
               rb, err := url.Parse(rep.BaseURL)
               if err != nil {
                  fmt.Fprintf(os.Stderr, "Warning: Error parsing Representation BaseURL '%s': %v. Using parent.\n", rep.BaseURL, err)
               } else {
                  repBaseURL = repBaseURL.ResolveReference(rb)
               }
            }

            representationID := rep.ID
            var currentPeriodSegments []string // Segments generated in the current period

            segmentTemplate := rep.SegmentTemplate
            if segmentTemplate == nil { // Fallback to AdaptationSet level SegmentTemplate
               segmentTemplate = as.SegmentTemplate
            }

            if segmentTemplate != nil {
               // Handle initialization segment
               if segmentTemplate.Initialization != "" {
                  initPath := segmentTemplate.Initialization
                  initPath = replaceTemplateVars(initPath, representationID, 0, 0, 0)
                  currentPeriodSegments = append(currentPeriodSegments, resolveURL(repBaseURL, initPath))
               }

               if segmentTemplate.SegmentTimeline != nil {
                  currentSegmentTime := segmentTemplate.Timescale * segmentTemplate.StartNumber
                  if segmentTemplate.StartNumber == 0 {
                     currentSegmentTime = 0
                  }

                  segmentNumber := segmentTemplate.StartNumber
                  if segmentNumber == 0 {
                     segmentNumber = 1
                  }

                  for _, s := range segmentTemplate.SegmentTimeline.Ss {
                     if s.T != 0 {
                        currentSegmentTime = s.T
                     }

                     count := 1
                     if s.R > 0 {
                        count = s.R + 1
                     } else if s.R == -1 {
                        fmt.Fprintf(os.Stderr, "Warning: Segment 'r=-1' found for Representation '%s'. Generating a limited number of segments (e.g., 5) for demonstration. Exact count requires full duration calculation.\n", representationID)
                        count = 5
                     }

                     for i := 0; i < count; i++ {
                        segmentPath := segmentTemplate.Media
                        segmentPath = replaceTemplateVars(segmentPath, representationID, currentSegmentTime, i, segmentNumber)
                        currentPeriodSegments = append(currentPeriodSegments, resolveURL(repBaseURL, segmentPath))

                        currentSegmentTime += s.D
                        segmentNumber++
                     }
                  }
               } else if segmentTemplate.Media != "" { // SegmentTemplate without SegmentTimeline
                  startNum := segmentTemplate.StartNumber
                  if startNum == 0 {
                     startNum = 1
                  }

                  if segmentTemplate.EndNumber > 0 { // If EndNumber is specified, use it
                     fmt.Fprintf(os.Stderr, "Generating segments from %d to %d for Representation '%s' using EndNumber.\n", startNum, segmentTemplate.EndNumber, representationID)
                     for i := startNum; i <= segmentTemplate.EndNumber; i++ {
                        segmentPath := segmentTemplate.Media
                        segmentPath = replaceTemplateVars(segmentPath, representationID, 0, 0, i)
                        currentPeriodSegments = append(currentPeriodSegments, resolveURL(repBaseURL, segmentPath))
                     }
                  } else { // Apply the user's SegmentCount formula
                     periodDurationSeconds, err := ParseDurationToSeconds(period.Duration)
                     if err != nil {
                        fmt.Fprintf(os.Stderr, "Error parsing Period duration '%s' for Representation '%s': %v. Cannot calculate SegmentCount. Generating 3 segments as fallback.\n", period.Duration, representationID, err)
                        // Fallback to generating 3 segments if duration parsing fails
                        for i := 0; i < 3; i++ {
                           segmentPath := segmentTemplate.Media
                           segmentPath = replaceTemplateVars(segmentPath, representationID, 0, i, startNum+uint64(i))
                           currentPeriodSegments = append(currentPeriodSegments, resolveURL(repBaseURL, segmentPath))
                        }
                     } else {
                        segmentTimescale := getInheritedTimescale(&mpd, &period, &as, segmentTemplate)
                        segmentDurationInSeconds := float64(segmentTemplate.Duration) / float64(segmentTimescale)

                        if segmentDurationInSeconds > 0 {
                           segmentCount := int(math.Ceil(periodDurationSeconds / segmentDurationInSeconds))
                           fmt.Fprintf(os.Stderr, "Calculated %d segments for Representation '%s' in Period '%s' using the provided formula (PeriodDuration: %.2f s, SegmentDuration: %.2f s, Timescale: %d).\n", segmentCount, representationID, period.ID, periodDurationSeconds, segmentDurationInSeconds, segmentTimescale)

                           for i := 0; i < segmentCount; i++ {
                              segmentPath := segmentTemplate.Media
                              segmentPath = replaceTemplateVars(segmentPath, representationID, 0, 0, startNum+uint64(i))
                              currentPeriodSegments = append(currentPeriodSegments, resolveURL(repBaseURL, segmentPath))
                           }
                        } else {
                           fmt.Fprintf(os.Stderr, "Warning: Calculated segment duration is zero for Representation '%s'. Cannot generate segments based on formula. Generating 3 segments as fallback.\n", representationID)
                           for i := 0; i < 3; i++ {
                              segmentPath := segmentTemplate.Media
                              segmentPath = replaceTemplateVars(segmentPath, representationID, 0, i, startNum+uint64(i))
                              currentPeriodSegments = append(currentPeriodSegments, resolveURL(repBaseURL, segmentPath))
                           }
                        }
                     }
                  }
               }
            } else if rep.SegmentBase != nil {
               if rep.BaseURL != "" {
                  currentPeriodSegments = append(currentPeriodSegments, repBaseURL.String())
               } else {
                  fmt.Fprintf(os.Stderr, "Note: Representation '%s' uses SegmentBase and no explicit Representation BaseURL. No discrete segment URLs generated.\n", representationID)
               }
            }

            if len(currentPeriodSegments) == 0 && rep.BaseURL != "" {
               currentPeriodSegments = append(currentPeriodSegments, repBaseURL.String())
            }
            
            // Accumulate segments for this representation across all periods
            output[representationID] = append(output[representationID], currentPeriodSegments...)
         }
      }
   }

   // Deduplicate URLs for each representation
   for repID, urls := range output {
      output[repID] = deduplicateStrings(urls)
   }

   jsonData, err := json.MarshalIndent(output, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error marshalling JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonData))
}

// Helper function to replace template variables
func replaceTemplateVars(template string, representationID string, time uint64, index int, number uint64) string {
   // Replace $RepresentationID$
   template = replaceAll(template, "$RepresentationID$", representationID)

   // Replace $Number$
   if findString(template, "$Number%0") != -1 && findString(template, "d$") != -1 {
      start := findString(template, "$Number%0") + len("$Number%0")
      end := findString(template[start:], "d$")
      if end != -1 {
         paddingStr := template[start : start+end]
         padding, err := strconv.Atoi(paddingStr)
         if err == nil {
            format := fmt.Sprintf("%%0%dd", padding)
            paddedNum := fmt.Sprintf(format, number)
            template = replaceAll(template, fmt.Sprintf("$Number%%0%dd$", padding), paddedNum)
         } else {
            template = replaceAll(template, "$Number$", strconv.FormatUint(number, 10))
         }
      } else {
         template = replaceAll(template, "$Number$", strconv.FormatUint(number, 10))
      }
   } else {
      template = replaceAll(template, "$Number$", strconv.FormatUint(number, 10))
   }


   // Replace $Time$
   template = replaceAll(template, "$Time$", strconv.FormatUint(time, 10))

   return template
}

// Simple string replacement function for string.ReplaceAll compatibility
func replaceAll(s, old, new string) string {
   for {
      idx := findString(s, old)
      if idx == -1 {
         break
      }
      s = s[:idx] + new + s[idx+len(old):]
   }
   return s
}

// findString finds the first instance of a substring.
func findString(s, substr string) int {
   for i := 0; i+len(substr) <= len(s); i++ {
      if s[i:i+len(substr)] == substr {
         return i
      }
   }
   return -1
}
