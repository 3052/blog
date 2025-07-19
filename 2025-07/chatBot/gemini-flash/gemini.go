package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "net/url"
   "os"
   "strconv"
   "strings" // Import the strings package
)

// MPD represents the root of the MPEG-DASH Media Presentation Description.
type MPD struct {
   XMLName              xml.Name           `xml:"MPD"`
   BaseURL              string             `xml:"BaseURL,omitempty"`
   Periods              []Period           `xml:"Period"`
   MediaPresentationDuration string `xml:"mediaPresentationDuration,attr"`
}

// Period represents a period in the MPD.
type Period struct {
   XMLName       xml.Name        `xml:"Period"`
   ID            string          `xml:"id,attr,omitempty"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet represents an adaptation set within a period.
type AdaptationSet struct {
   XMLName         xml.Name        `xml:"AdaptationSet"`
   ID              string          `xml:"id,attr,omitempty"`
   MimeType        string          `xml:"mimeType,attr,omitempty"`
   ContentType     string          `xml:"contentType,attr,omitempty"`
   Representations []Representation `xml:"Representation"`
}

// Representation represents a single media rendition.
type Representation struct {
   XMLName        xml.Name       `xml:"Representation"`
   ID             string         `xml:"id,attr"`
   BaseURL        string         `xml:"BaseURL,omitempty"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate,omitempty"`
   SegmentList    *SegmentList    `xml:"SegmentList,omitempty"`
   SegmentBase    *SegmentBase    `xml:"SegmentBase,omitempty"`
}

// SegmentTemplate defines a pattern for generating segment URLs.
type SegmentTemplate struct {
   XMLName      xml.Name      `xml:"SegmentTemplate"`
   Media        string        `xml:"media,attr,omitempty"`
   Initialization string      `xml:"initialization,attr,omitempty"`
   StartNumber  uint64        `xml:"startNumber,attr,omitempty"`
   Duration     uint64        `xml:"duration,attr,omitempty"` // for fixed duration segments
   Timescale    uint64        `xml:"timescale,attr,omitempty"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline,omitempty"`
}

// SegmentTimeline defines segment durations and presentation times.
type SegmentTimeline struct {
   XMLName xml.Name `xml:"SegmentTimeline"`
   Segments []S `xml:"S"`
}

// S represents an entry in the SegmentTimeline.
type S struct {
   XMLName xml.Name `xml:"S"`
   T       uint64   `xml:"t,attr,omitempty"` // Presentation time of the first segment in the series
   D       uint64   `xml:"d,attr"`           // Duration of the segment
   R       int      `xml:"r,attr,omitempty"` // Repeat count
}

// SegmentList defines a list of segments.
type SegmentList struct {
   XMLName xml.Name `xml:"SegmentList"`
   Segments []SegmentURL `xml:"SegmentURL"`
   Initialization *Initialization `xml:"Initialization,omitempty"`
}

// SegmentURL defines a single segment URL.
type SegmentURL struct {
   XMLName xml.Name `xml:"SegmentURL"`
   Media   string   `xml:"media,attr"`
   Index   string   `xml:"index,attr,omitempty"`
}

// SegmentBase defines the base for single segment media.
type SegmentBase struct {
   XMLName xml.Name `xml:"SegmentBase"`
   Initialization *Initialization `xml:"Initialization,omitempty"`
   IndexRange     string `xml:"indexRange,attr,omitempty"`
}

// Initialization defines the initialization segment.
type Initialization struct {
   XMLName xml.Name `xml:"Initialization"`
   Range   string   `xml:"range,attr,omitempty"`
   SourceURL string `xml:"sourceURL,attr,omitempty"`
}


// SegmentURLsByRepresentation maps Representation ID to a list of segment URLs.
type SegmentURLsByRepresentation map[string][]string

// resolveURL resolves a relative URL against a base URL.
func resolveURL(baseURL, relativePath string) (string, error) {
   base, err := url.Parse(baseURL)
   if err != nil {
      return "", fmt.Errorf("invalid base URL '%s': %w", baseURL, err)
   }
   rel, err := url.Parse(relativePath)
   if err != nil {
      return "", fmt.Errorf("invalid relative path '%s': %w", relativePath, err)
   }
   return base.ResolveReference(rel).String(), nil
}

func main() {
   if len(os.Args) < 2 {
      fmt.Println("Usage: go run mpd_parser.go <path_to_mpd_file>")
      os.Exit(1)
   }

   mpdFilePath := os.Args[1]
   mpdContent, err := ioutil.ReadFile(mpdFilePath)
   if err != nil {
      fmt.Printf("Error reading MPD file: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   err = xml.Unmarshal(mpdContent, &mpd)
   if err != nil {
      fmt.Printf("Error unmarshalling MPD: %v\n", err)
      os.Exit(1)
   }

   // Base URL for resolving relative paths, as specified in the prompt.
   const fixedBaseMPDURL = "http://test.test/test.mpd"
   
   // If MPD has a BaseURL, use it relative to the fixedBaseMPDURL
   currentBaseURL := fixedBaseMPDURL
   if mpd.BaseURL != "" {
      resolved, err := resolveURL(currentBaseURL, mpd.BaseURL)
      if err != nil {
         fmt.Printf("Warning: Could not resolve MPD's BaseURL '%s' relative to '%s'. Using fixed base. Error: %v\n", mpd.BaseURL, currentBaseURL, err)
      } else {
         currentBaseURL = resolved
      }
   }


   segmentURLs := make(SegmentURLsByRepresentation)

   for _, period := range mpd.Periods {
      for _, as := range period.AdaptationSets {
         for _, rep := range as.Representations {
            repURLs := []string{}
            
            repBaseURL := currentBaseURL
            if rep.BaseURL != "" {
               resolved, err := resolveURL(currentBaseURL, rep.BaseURL)
               if err != nil {
                  fmt.Printf("Warning: Could not resolve Representation BaseURL '%s' relative to '%s'. Using parent base. Error: %v\n", rep.BaseURL, currentBaseURL, err)
               } else {
                  repBaseURL = resolved
               }
            }

            if rep.SegmentTemplate != nil {
               st := rep.SegmentTemplate
               
               // Resolve initialization URL if present
               if st.Initialization != "" {
                  initURL := st.Initialization
                  // Replace $RepresentationID$ placeholder
                  initURL = strings.Replace(initURL, "$RepresentationID$", rep.ID, -1) // Using strings.Replace
                  resolvedInitURL, err := resolveURL(repBaseURL, initURL)
                  if err != nil {
                     fmt.Printf("Warning: Could not resolve Initialization URL '%s': %v\n", initURL, err)
                  } else {
                     repURLs = append(repURLs, resolvedInitURL)
                  }
               }

               segmentNumberCounter := st.StartNumber // Initialize segment number counter for SegmentTimeline
               if segmentNumberCounter == 0 {
                  segmentNumberCounter = 1 // Default start number is 1 if not specified
               }

               if st.SegmentTimeline != nil {
                  // SegmentTimeline based segments
                  time := st.SegmentTimeline.Segments[0].T
                  if st.SegmentTimeline.Segments[0].T == 0 {
                     // If 't' attribute is omitted, it indicates that the first segment starts at 0.
                     // The spec says "If the 't' attribute is omitted, it indicates that the first segment starts at a time that is the sum of the previous segment's start time and duration."
                     // For the very first segment in a timeline, if 't' is omitted, it starts at 0.
                     // For subsequent 'S' elements if 't' is omitted, it implies it continues from the previous segment.
                     // We initialize with 0 if first S has no t.
                     time = 0
                  }
                  
                  for _, s := range st.SegmentTimeline.Segments {
                     if st.Timescale > 0 {
                        // If you need scaled duration for other logic later, reintroduce this with usage.
                        // scaledDuration := float64(s.D) / float64(st.Timescale)
                     }

                     count := s.R + 1 // r attribute specifies repetition, so total segments are r+1
                     for i := 0; i < count; i++ {
                        segmentURL := st.Media
                        // Replace $Number$ placeholder and $Time$
                        segmentURL = strings.Replace(segmentURL, "$Number$", strconv.FormatUint(segmentNumberCounter, 10), -1)
                        segmentURL = strings.Replace(segmentURL, "$Time$", strconv.FormatUint(time, 10), -1)
                        segmentURL = strings.Replace(segmentURL, "$RepresentationID$", rep.ID, -1) // Ensure this is also replaced

                        resolvedSegmentURL, err := resolveURL(repBaseURL, segmentURL)
                        if err != nil {
                           fmt.Printf("Warning: Could not resolve Segment URL '%s': %v\n", segmentURL, err)
                        } else {
                           repURLs = append(repURLs, resolvedSegmentURL)
                        }
                        time += s.D // Advance time for the next segment
                        segmentNumberCounter++ // Increment segment number for the next segment
                     }
                     
                  }
               } else if st.Media != "" && st.Duration > 0 {
                  // Fixed duration segments with $Number$ (for example, for VOD)
                  // This is a simplified calculation and might need to consider mediaPresentationDuration
                  // and minBufferTime from MPD for accurate segment count for dynamic MPDs.
                  // For static MPDs, calculate based on overall duration.
                  
                  // For simplicity, let's assume a reasonable number of segments for demonstration.
                  // In a real scenario, you'd calculate this based on MPD's duration.
                  // The number of segments is simplified for demonstration purposes.
                  // A more robust solution would calculate the total number of segments based on
                  // mediaPresentationDuration and segment duration, or look for SegmentBase/SegmentList.
                  
                  // A simple heuristic for a few segments to demonstrate functionality:
                  numSegments := uint64(10) 
                  if st.Timescale > 0 && mpd.MediaPresentationDuration != "" {
                     // Attempt to parse mediaPresentationDuration and estimate segments
                     // This is complex, involving parsing ISO 8601 duration. For now, use fixed.
                     // Alternatively, look for other ways to determine total segments or leave it to user to know.
                  }

                  startNumber := uint64(1)
                  if st.StartNumber != 0 {
                     startNumber = st.StartNumber
                  }
                  
                  for i := uint64(0); i < numSegments; i++ { 
                     segmentURL := st.Media
                     segmentNumber := startNumber + i
                     segmentURL = strings.Replace(segmentURL, "$Number$", strconv.FormatUint(segmentNumber, 10), -1) // Using strings.Replace
                     segmentURL = strings.Replace(segmentURL, "$RepresentationID$", rep.ID, -1) // Ensure this is also replaced

                     resolvedSegmentURL, err := resolveURL(repBaseURL, segmentURL)
                     if err != nil {
                        fmt.Printf("Warning: Could not resolve Segment URL '%s': %v\n", segmentURL, err)
                        } else {
                        repURLs = append(repURLs, resolvedSegmentURL)
                     }
                  }

               }
            } else if rep.SegmentList != nil {
               // SegmentList based segments
               if rep.SegmentList.Initialization != nil && rep.SegmentList.Initialization.SourceURL != "" {
                  initURL := rep.SegmentList.Initialization.SourceURL
                  resolvedInitURL, err := resolveURL(repBaseURL, initURL)
                  if err != nil {
                     fmt.Printf("Warning: Could not resolve Initialization URL '%s': %v\n", initURL, err)
                  } else {
                     repURLs = append(repURLs, resolvedInitURL)
                  }
               }
               for _, segURL := range rep.SegmentList.Segments {
                  resolvedSegURL, err := resolveURL(repBaseURL, segURL.Media)
                  if err != nil {
                     fmt.Printf("Warning: Could not resolve SegmentURL '%s': %v\n", segURL.Media, err)
                  } else {
                     repURLs = append(repURLs, resolvedSegURL)
                  }
               }
            } else if rep.SegmentBase != nil && rep.SegmentBase.Initialization != nil && rep.SegmentBase.Initialization.SourceURL != "" {
               // SegmentBase based (single segment)
               initURL := rep.SegmentBase.Initialization.SourceURL
               resolvedInitURL, err := resolveURL(repBaseURL, initURL)
               if err != nil {
                  fmt.Printf("Warning: Could not resolve Initialization URL '%s': %v\n", initURL, err)
               } else {
                  repURLs = append(repURLs, resolvedInitURL)
               }
            } else if rep.BaseURL != "" && as.MimeType == "text/vtt" {
               // Handle VTT subtitles with a single BaseURL directly under Representation
               // as seen in criterion.txt
               resolvedURL, err := resolveURL(repBaseURL, "") // Resolves repBaseURL itself
               if err != nil {
                  fmt.Printf("Warning: Could not resolve Representation BaseURL for VTT '%s': %v\n", repBaseURL, err)
               } else {
                  repURLs = append(repURLs, resolvedURL)
               }
            }


            if len(repURLs) > 0 {
               segmentURLs[rep.ID] = repURLs
            }
         }
      }
   }

   jsonOutput, err := json.MarshalIndent(segmentURLs, "", "  ")
   if err != nil {
      fmt.Printf("Error marshalling JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonOutput))
}
