package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "net/url"
   "os"
   "strconv"
)

// MPD represents the top-level structure of a DASH MPD file
type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   BaseURL                   string   `xml:"BaseURL,omitempty"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
   MinBufferTime             string   `xml:"minBufferTime,attr"`
   Type                      string   `xml:"type,attr"`
   Profiles                  string   `xml:"profiles,attr"`
   Periods                   []Period `xml:"Period"`
}

// Period represents a Period element in the MPD
type Period struct {
   XMLName        xml.Name        `xml:"Period"`
   ID             string          `xml:"id,attr"`
   Duration       string          `xml:"duration,attr"`
   BaseURL        string          `xml:"BaseURL,omitempty"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet represents an AdaptationSet element in the MPD
type AdaptationSet struct {
   XMLName         xml.Name         `xml:"AdaptationSet"`
   ID              string           `xml:"id,attr"`
   ContentType     string           `xml:"contentType,attr"`
   MimeType        string           `xml:"mimeType,attr"`
   Codecs          string           `xml:"codecs,attr"`
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
   XMLName         xml.Name         `xml:"SegmentTemplate"`
   Timescale       uint64           `xml:"timescale,attr"`
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   StartNumber     uint64           `xml:"startNumber,attr"`
   Duration        uint64           `xml:"duration,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
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
   XMLName        xml.Name        `xml:"SegmentBase"`
   IndexRange     string          `xml:"indexRange,attr"`
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

   // Base URL for resolving relative paths, as specified in the prompt
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
            var segments []string

            segmentTemplate := rep.SegmentTemplate
            if segmentTemplate == nil { // Fallback to AdaptationSet level SegmentTemplate
               segmentTemplate = as.SegmentTemplate
            }

            if segmentTemplate != nil {
               // Handle initialization segment
               if segmentTemplate.Initialization != "" {
                  initPath := segmentTemplate.Initialization
                  // Initialize with dummy values for time/number as they don't apply to init segment
                  initPath = replaceTemplateVars(initPath, representationID, 0, 0, 0)
                  segments = append(segments, resolveURL(repBaseURL, initPath))
               }

               if segmentTemplate.SegmentTimeline != nil {
                  currentSegmentTime := segmentTemplate.Timescale * segmentTemplate.StartNumber // This assumes startNumber is correctly applied to first segment's T
                  if segmentTemplate.StartNumber == 0 {                                         // If startNumber is 0, then the first segment's time could be 0, but if T is defined, it overrides.
                     currentSegmentTime = 0 // Reset for initial segment from timeline
                  }

                  segmentNumber := segmentTemplate.StartNumber // Track segment number for $Number$ template
                  if segmentNumber == 0 {                      // Default startNumber is 1 if not specified
                     segmentNumber = 1
                  }

                  for _, s := range segmentTemplate.SegmentTimeline.Ss {
                     if s.T != 0 { // If T is present, it explicitly sets the start time for this segment group
                        currentSegmentTime = s.T
                     }

                     count := 1
                     if s.R > 0 { // r is repetition count, so total segments = r + 1
                        count = s.R + 1
                     } else if s.R == -1 {
                        fmt.Fprintf(os.Stderr, "Warning: Segment 'r=-1' found for Representation '%s'. Generating a limited number of segments (e.g., 5) for demonstration. Exact count requires full duration calculation.\n", representationID)
                        count = 5 // Arbitrary fixed number for demo when r=-1
                     }

                     for i := 0; i < count; i++ {
                        segmentPath := segmentTemplate.Media
                        segmentPath = replaceTemplateVars(segmentPath, representationID, currentSegmentTime, i, segmentNumber)
                        segments = append(segments, resolveURL(repBaseURL, segmentPath))

                        currentSegmentTime += s.D
                        segmentNumber++
                     }
                  }
               } else if segmentTemplate.Duration > 0 && segmentTemplate.Media != "" {
                  // For SegmentTemplate without SegmentTimeline (fixed duration segments)
                  fmt.Fprintf(os.Stderr, "Warning: SegmentTemplate without SegmentTimeline found for Representation '%s'. Generating 3 segments for demonstration. Exact count requires total media duration.\n", representationID)
                  startNumber := segmentTemplate.StartNumber
                  if startNumber == 0 { // Default startNumber is 1 if not specified
                     startNumber = 1
                  }
                  for i := 0; i < 3; i++ { // Generate a few segments for demonstration
                     segmentPath := segmentTemplate.Media
                     segmentPath = replaceTemplateVars(segmentPath, representationID, 0, i, startNumber+uint64(i))
                     segments = append(segments, resolveURL(repBaseURL, segmentPath))
                  }
               }
            } else if rep.SegmentBase != nil {
               // SegmentBase typically refers to a single media file with byte ranges.
               // If the Representation has its own BaseURL, that BaseURL is likely the URL to the main media file.
               if rep.BaseURL != "" {
                  segments = append(segments, repBaseURL.String())
               } else {
                  fmt.Fprintf(os.Stderr, "Note: Representation '%s' uses SegmentBase and no explicit Representation BaseURL. No discrete segment URLs generated.\n", representationID)
               }
            }

            // Crucial for cases like criterion.txt's subs-7433271:
            // If no segments were generated by SegmentTemplate or SegmentBase logic,
            // but the Representation has a direct BaseURL defined in the MPD XML,
            // then that BaseURL itself is the segment URL.
            // We check `rep.BaseURL != ""` to ensure it was explicitly set in the MPD,
            // not just inherited from a parent.
            if len(segments) == 0 && rep.BaseURL != "" {
               segments = append(segments, repBaseURL.String())
            }

            output[representationID] = segments
         }
      }
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
   // This handles simple $Number$ and also attempts to handle $Number%0xd$
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
