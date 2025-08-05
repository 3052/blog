package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io"
   "math"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
)

// MPD represents the root of the MPEG-DASH MPD XML file.
type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   BaseURL                   string   `xml:"BaseURL"`
   Period                    []Period `xml:"Period"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
}

// Period represents a period in the MPD.
type Period struct {
   AdaptationSet []AdaptationSet `xml:"AdaptationSet"`
   BaseURL       string          `xml:"BaseURL"`
   Duration      string          `xml:"duration,attr"`
}

// AdaptationSet represents an adaptation set within a period.
type AdaptationSet struct {
   Representation  []Representation `xml:"Representation"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

// Representation represents a single representation.
type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
}

// SegmentTemplate represents a segment template.
type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   Timescale       int              `xml:"timescale,attr"`
   Duration        int              `xml:"duration,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline represents a segment timeline.
type SegmentTimeline struct {
   S []TimelineSegment `xml:"S"`
}

// TimelineSegment represents a single segment in a timeline.
type TimelineSegment struct {
   T int `xml:"t,attr"` // Start time of the segment
   D int `xml:"d,attr"` // Duration of the segment
   R int `xml:"r,attr"` // Repeat count
}

// SegmentList represents a segment list.
type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURL     []SegmentURL    `xml:"SegmentURL"`
}

// SegmentURL represents a single segment URL.
type SegmentURL struct {
   Media string `xml:"media,attr"`
}

// SegmentBase represents a segment base.
type SegmentBase struct {
   Initialization *Initialization `xml:"Initialization"`
}

// Initialization represents the initialization segment.
type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

// resolveURL resolves a relative URL against a base URL using url.URL.ResolveReference.
func resolveURL(baseURL *url.URL, relativeURL string) (*url.URL, error) {
   if relativeURL == "" {
      return baseURL, nil
   }

   rel, err := url.Parse(relativeURL)
   if err != nil {
      return nil, err
   }
   return baseURL.ResolveReference(rel), nil
}

// resolveSegmentURLTemplate resolves a segment URL template.
func resolveSegmentURLTemplate(baseURL *url.URL, template, representationID string, number int, time int) (*url.URL, error) {
   // Regular expression to find placeholders with optional format specifiers
   reNumber := regexp.MustCompile(`\$Number(%[0-9]*d)\$`)
   reTime := regexp.MustCompile(`\$Time(%[0-9]*d)\$`)

   resolvedTemplate := strings.ReplaceAll(template, "$RepresentationID$", representationID)

   // Replace $Number$ with optional format specifier
   matchNumber := reNumber.FindStringSubmatch(resolvedTemplate)
   if len(matchNumber) == 2 {
      format := matchNumber[1]
      formattedNumber := fmt.Sprintf(format, number)
      resolvedTemplate = strings.ReplaceAll(resolvedTemplate, matchNumber[0], formattedNumber)
   } else {
      resolvedTemplate = strings.ReplaceAll(resolvedTemplate, "$Number$", fmt.Sprintf("%d", number))
   }

   // Replace $Time$ with optional format specifier
   matchTime := reTime.FindStringSubmatch(resolvedTemplate)
   if len(matchTime) == 2 {
      format := matchTime[1]
      formattedTime := fmt.Sprintf(format, time)
      resolvedTemplate = strings.ReplaceAll(resolvedTemplate, matchTime[0], formattedTime)
   } else {
      resolvedTemplate = strings.ReplaceAll(resolvedTemplate, "$Time$", fmt.Sprintf("%d", time))
   }

   return resolveURL(baseURL, resolvedTemplate)
}

// parseISODuration parses a subset of ISO 8601 duration string (e.g., PT5M30S) into seconds.
func parseISODuration(d string) (float64, error) {
   if !strings.HasPrefix(d, "PT") {
      return 0, fmt.Errorf("unsupported duration format: %s", d)
   }

   d = d[2:] // Strip the "PT" prefix
   var seconds float64

   if strings.Contains(d, "H") {
      parts := strings.Split(d, "H")
      hours, err := strconv.ParseFloat(parts[0], 64)
      if err != nil {
         return 0, err
      }
      seconds += hours * 3600
      d = parts[1]
   }

   if strings.Contains(d, "M") {
      parts := strings.Split(d, "M")
      minutes, err := strconv.ParseFloat(parts[0], 64)
      if err != nil {
         return 0, err
      }
      seconds += minutes * 60
      d = parts[1]
   }

   if strings.Contains(d, "S") {
      parts := strings.Split(d, "S")
      sec, err := strconv.ParseFloat(parts[0], 64)
      if err != nil {
         return 0, err
      }
      seconds += sec
   }

   return seconds, nil
}

func main() {
   if len(os.Args) < 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path>")
      os.Exit(1)
   }

   mpdFilePath := os.Args[1]

   // Read the XML file
   xmlFile, err := os.Open(mpdFilePath)
   if err != nil {
      fmt.Printf("Error opening file: %v\n", err)
      os.Exit(1)
   }
   defer xmlFile.Close()

   byteValue, _ := io.ReadAll(xmlFile)

   var mpd MPD
   err = xml.Unmarshal(byteValue, &mpd)
   if err != nil {
      fmt.Printf("Error unmarshalling XML: %v\n", err)
      os.Exit(1)
   }

   // Use a fixed initial MPD URL for resolving relative BaseURLs.
   initialMPDURL := "http://test.test/test.mpd"
   initialBaseURL, err := url.Parse(initialMPDURL)
   if err != nil {
      fmt.Printf("Error parsing initial MPD URL: %v\n", err)
      os.Exit(1)
   }

   // Determine the base URL for the MPD itself by stripping the filename
   mpdBaseURL, err := resolveURL(initialBaseURL, ".")
   if err != nil {
      fmt.Printf("Error resolving initial base URL: %v\n", err)
      os.Exit(1)
   }

   // Resolve the top-level MPD@BaseURL if it exists
   if mpd.BaseURL != "" {
      mpdBaseURL, err = resolveURL(mpdBaseURL, mpd.BaseURL)
      if err != nil {
         fmt.Printf("Error resolving top-level BaseURL: %v\n", err)
         os.Exit(1)
      }
   }

   result := make(map[string][]string)

   for _, period := range mpd.Period {
      // Resolve the Period@BaseURL against the MPD's base URL
      periodBaseURL := mpdBaseURL
      if period.BaseURL != "" {
         periodBaseURL, err = resolveURL(periodBaseURL, period.BaseURL)
         if err != nil {
            fmt.Printf("Error resolving Period BaseURL: %v\n", err)
            os.Exit(1)
         }
      }

      // Determine the Period duration in seconds for calculation
      periodDurationSeconds := float64(0)
      if period.Duration != "" {
         periodDurationSeconds, err = parseISODuration(period.Duration)
         if err != nil {
            fmt.Printf("Warning: Could not parse Period duration: %v. Using MPD duration as fallback.\n", err)
         }
      }
      if periodDurationSeconds == 0 && mpd.MediaPresentationDuration != "" {
         periodDurationSeconds, err = parseISODuration(mpd.MediaPresentationDuration)
         if err != nil {
            fmt.Printf("Warning: Could not parse MPD duration: %v. Segment count will be a default value.\n", err)
         }
      }

      for _, adaptationSet := range period.AdaptationSet {
         // Resolve the AdaptationSet@BaseURL against the Period's base URL
         adaptationSetBaseURL := periodBaseURL
         if adaptationSet.BaseURL != "" {
            adaptationSetBaseURL, err = resolveURL(adaptationSetBaseURL, adaptationSet.BaseURL)
            if err != nil {
               fmt.Printf("Error resolving AdaptationSet BaseURL: %v\n", err)
               os.Exit(1)
            }
         }

         // Check for SegmentTemplate at the AdaptationSet level
         adaptationSetTemplate := adaptationSet.SegmentTemplate

         for _, rep := range adaptationSet.Representation {
            // Get the existing list of URLs for this Representation ID, or initialize a new one.
            segmentURLs, exists := result[rep.ID]
            if !exists {
               segmentURLs = []string{}
            }

            // Resolve the Representation@BaseURL against the AdaptationSet's base URL
            repBaseURL := adaptationSetBaseURL
            if rep.BaseURL != "" {
               repBaseURL, err = resolveURL(repBaseURL, rep.BaseURL)
               if err != nil {
                  fmt.Printf("Error resolving Representation BaseURL: %v\n", err)
                  os.Exit(1)
               }
            }

            // Prioritize SegmentTemplate at the Representation level, otherwise use the one from AdaptationSet
            var segmentTemplate *SegmentTemplate
            if rep.SegmentTemplate != nil {
               segmentTemplate = rep.SegmentTemplate
            } else if adaptationSetTemplate != nil {
               segmentTemplate = adaptationSetTemplate
            }

            // Check if there is any segment information. If not, use the resolved BaseURL directly.
            if segmentTemplate == nil && rep.SegmentList == nil && rep.SegmentBase == nil {
               if repBaseURL != nil {
                  segmentURLs = append(segmentURLs, repBaseURL.String())
               }
            }

            // Handle SegmentTemplate
            if segmentTemplate != nil {
               // Default startNumber to 1 if it is 0
               effectiveStartNumber := segmentTemplate.StartNumber
               if effectiveStartNumber == 0 {
                  effectiveStartNumber = 1
               }

               // Resolve Initialization URL
               if segmentTemplate.Initialization != "" {
                  initURL, err := resolveSegmentURLTemplate(repBaseURL, segmentTemplate.Initialization, rep.ID, effectiveStartNumber, 0)
                  if err != nil {
                     fmt.Printf("Error resolving SegmentTemplate Initialization URL for %s: %v\n", rep.ID, err)
                     os.Exit(1)
                  }
                  segmentURLs = append(segmentURLs, initURL.String())
               }

               // Check for SegmentTimeline
               if segmentTemplate.SegmentTimeline != nil {
                  // Use timeline to determine segments
                  segmentNumber := effectiveStartNumber
                  var currentTime int
                  for _, s := range segmentTemplate.SegmentTimeline.S {
                     if s.T > 0 {
                        currentTime = s.T
                     }
                     // The 'r' attribute indicates a repeat count
                     repeatCount := s.R
                     if repeatCount == -1 {
                        // If -1, repeat indefinitely. We'll just generate a few segments for this example.
                        repeatCount = 5
                     }
                     for i := 0; i <= repeatCount; i++ {
                        mediaURL, err := resolveSegmentURLTemplate(repBaseURL, segmentTemplate.Media, rep.ID, segmentNumber, currentTime)
                        if err != nil {
                           fmt.Printf("Error resolving SegmentTimeline Media URL for %s: %v\n", rep.ID, err)
                           os.Exit(1)
                        }
                        segmentURLs = append(segmentURLs, mediaURL.String())
                        segmentNumber++
                        currentTime += s.D
                     }
                  }
               } else {
                  // Handle regular SegmentTemplate (without SegmentTimeline)
                  if segmentTemplate.Media != "" {
                     // Determine the number of segments to generate
                     var endSegmentNumber int

                     if segmentTemplate.EndNumber > 0 {
                        endSegmentNumber = segmentTemplate.EndNumber
                     } else {
                        effectiveTimescale := segmentTemplate.Timescale
                        if effectiveTimescale == 0 {
                           effectiveTimescale = 1 // Default timescale to 1
                        }

                        if periodDurationSeconds > 0 && segmentTemplate.Duration > 0 && effectiveTimescale > 0 {
                           numSegments := int(math.Ceil(periodDurationSeconds * float64(effectiveTimescale) / float64(segmentTemplate.Duration)))
                           endSegmentNumber = effectiveStartNumber + numSegments - 1
                        } else {
                           // Default to 10 segments if no duration info is available
                           endSegmentNumber = effectiveStartNumber + 9
                        }
                     }

                     for segmentNumber := effectiveStartNumber; segmentNumber <= endSegmentNumber; segmentNumber++ {
                        // Calculate the segment start time. It's relative to the period's start.
                        segmentTime := (segmentNumber - effectiveStartNumber) * segmentTemplate.Duration
                        mediaURL, err := resolveSegmentURLTemplate(repBaseURL, segmentTemplate.Media, rep.ID, segmentNumber, segmentTime)
                        if err != nil {
                           fmt.Printf("Error resolving SegmentTemplate Media URL for %s: %v\n", rep.ID, err)
                           os.Exit(1)
                        }
                        segmentURLs = append(segmentURLs, mediaURL.String())
                     }
                  }
               }
            }

            // Handle SegmentList
            if rep.SegmentList != nil {
               // Check for and resolve Initialization URL first
               if rep.SegmentList.Initialization != nil {
                  initURL, err := resolveURL(repBaseURL, rep.SegmentList.Initialization.SourceURL)
                  if err != nil {
                     fmt.Printf("Error resolving SegmentList Initialization URL for %s: %v\n", rep.ID, err)
                     os.Exit(1)
                  }
                  segmentURLs = append(segmentURLs, initURL.String())
               }
               // Then resolve media segment URLs
               for _, segment := range rep.SegmentList.SegmentURL {
                  mediaURL, err := resolveURL(repBaseURL, segment.Media)
                  if err != nil {
                     fmt.Printf("Error resolving SegmentList Media URL for %s: %v\n", rep.ID, err)
                     os.Exit(1)
                  }
                  segmentURLs = append(segmentURLs, mediaURL.String())
               }
            }

            // Handle SegmentBase
            if rep.SegmentBase != nil && rep.SegmentBase.Initialization != nil {
               initURL, err := resolveURL(repBaseURL, rep.SegmentBase.Initialization.SourceURL)
               if err != nil {
                  fmt.Printf("Error resolving SegmentBase Initialization URL for %s: %v\n", rep.ID, err)
                  os.Exit(1)
               }
               segmentURLs = append(segmentURLs, initURL.String())
            }

            // Store the (potentially appended) slice back into the map
            result[rep.ID] = segmentURLs
         }
      }
   }

   // Output the JSON
   jsonOutput, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Printf("Error marshalling JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonOutput))
}
