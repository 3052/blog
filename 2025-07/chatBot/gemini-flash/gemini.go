package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "net/url"
   "os"
   "strings"
)

// MPD represents the top-level DASH MPD structure
type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL,omitempty"`
   Periods []Period `xml:"Period"`
}

// Period represents a period within the MPD
type Period struct {
   XMLName        xml.Name        `xml:"Period"`
   BaseURL        string          `xml:"BaseURL,omitempty"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// SegmentURL represents a URL for a segment
type SegmentURL struct {
   XMLName xml.Name `xml:"SegmentURL"`
   Media   string   `xml:"media,attr"`
}

// SegmentList represents a segment list
type SegmentList struct {
   XMLName     xml.Name     `xml:"SegmentList"`
   Duration    int          `xml:"duration,attr"`
   StartNumber int          `xml:"startNumber,attr,omitempty"`
   EndNumber   int          `xml:"endNumber,attr,omitempty"`
   SegmentURLs []SegmentURL `xml:"SegmentURL"`
}

// SegmentTimeline represents a segment timeline
type SegmentTimeline struct {
   XMLName xml.Name `xml:"SegmentTimeline"`
   Ss      []S      `xml:"S"`
}

// S represents an 'S' element in SegmentTimeline
type S struct {
   XMLName xml.Name `xml:"S"`
   T       int      `xml:"t,attr"` // Optional start time (in timescale units)
   D       int      `xml:"d,attr"` // Duration (in timescale units)
   R       int      `xml:"r,attr"` // Optional repeat count
}

// SegmentTemplate represents a segment template
type SegmentTemplate struct {
   XMLName         xml.Name         `xml:"SegmentTemplate"`
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   Timescale       int              `xml:"timescale,attr"`
   Duration        int              `xml:"duration,attr"`              // For fixed duration segments (if no timeline/endNumber)
   StartNumber     int              `xml:"startNumber,attr,omitempty"` // startNumber is optional
   EndNumber       int              `xml:"endNumber,attr,omitempty"`   // Added EndNumber
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// Representation represents a representation within an adaptation set
// MOVED THIS DEFINITION BEFORE AdaptationSet
type Representation struct {
   XMLName         xml.Name         `xml:"Representation"`
   ID              string           `xml:"id,attr,omitempty"` // MANDATORY for desired output
   BaseURL         string           `xml:"BaseURL,omitempty"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

// AdaptationSet represents an adaptation set within a period
type AdaptationSet struct {
   XMLName         xml.Name         `xml:"AdaptationSet"`
   BaseURL         string           `xml:"BaseURL,omitempty"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

func main() {
   if len(os.Args) < 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <MPD_FILE_PATH>\n", os.Args[0])
      os.Exit(1)
   }

   mpdFilePath := os.Args[1]
   var mpdBytes []byte
   var err error

   // --- Hard-coded root for BaseURL resolution ---
   const hardcodedRootMPDURL = "http://test.test/test.mpd"
   initialBaseURL, err := url.Parse(hardcodedRootMPDURL)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing hardcoded root URL '%s': %v\n", hardcodedRootMPDURL, err)
      os.Exit(1)
   }

   fmt.Fprintf(os.Stderr, "Initial hardcoded Base URL for ALL resolution chains: %s\n", initialBaseURL.String())
   // --- End hard-coded root ---

   fmt.Fprintf(os.Stderr, "Reading MPD from local file: %s\n", mpdFilePath)
   mpdBytes, err = ioutil.ReadFile(mpdFilePath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading MPD file '%s': %v\n", mpdFilePath, err)
      os.Exit(1)
   }

   var mpd MPD
   err = xml.Unmarshal(mpdBytes, &mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error unmarshaling MPD: %v\n", err)
      os.Exit(1)
   }

   outputURLs := make(map[string][]string)

   currentBaseURL := initialBaseURL

   if mpd.BaseURL != "" {
      relativeMPDBase, err := url.Parse(mpd.BaseURL)
      if err != nil {
         fmt.Fprintf(os.Stderr, "Warning: Could not parse MPD BaseURL '%s': %v\n", mpd.BaseURL, err)
      } else {
         currentBaseURL = currentBaseURL.ResolveReference(relativeMPDBase)
      }
   }

   fmt.Fprintf(os.Stderr, "Effective MPD Base URL for resolutions: %s\n", currentBaseURL.String())

   for _, period := range mpd.Periods {
      currentPeriodBaseURL := currentBaseURL
      if period.BaseURL != "" {
         relativePeriodBase, err := url.Parse(period.BaseURL)
         if err != nil {
            fmt.Fprintf(os.Stderr, "Warning: Could not parse Period BaseURL '%s': %v\n", period.BaseURL, err)
         } else {
            currentPeriodBaseURL = currentPeriodBaseURL.ResolveReference(relativePeriodBase)
         }
      }

      for _, as := range period.AdaptationSets {
         currentASBaseURL := currentPeriodBaseURL
         if as.BaseURL != "" {
            relativeASBase, err := url.Parse(as.BaseURL)
            if err != nil {
               fmt.Fprintf(os.Stderr, "Warning: Could not parse AdaptationSet BaseURL '%s': %v\n", as.BaseURL, err)
            } else {
               currentASBaseURL = currentASBaseURL.ResolveReference(relativeASBase)
            }
         }

         effectiveASSegmentTemplate := as.SegmentTemplate

         for _, rep := range as.Representations {
            if rep.ID == "" {
               fmt.Fprintf(os.Stderr, "Warning: Representation found without an 'id' attribute. Skipping URLs for this representation.\n")
               continue
            }

            currentRepBaseURL := currentASBaseURL
            if rep.BaseURL != "" {
               relativeRepBase, err := url.Parse(rep.BaseURL)
               if err != nil {
                  fmt.Fprintf(os.Stderr, "Warning: Could not parse Representation BaseURL '%s': %v\n", rep.BaseURL, err)
               } else {
                  currentRepBaseURL = currentRepBaseURL.ResolveReference(relativeRepBase)
               }
            }

            var effectiveRepSegmentTemplate *SegmentTemplate
            if rep.SegmentTemplate != nil {
               effectiveRepSegmentTemplate = rep.SegmentTemplate
            } else {
               effectiveRepSegmentTemplate = effectiveASSegmentTemplate
            }

            if _, ok := outputURLs[rep.ID]; !ok {
               outputURLs[rep.ID] = []string{}
            }

            if rep.SegmentList != nil {
               // Handle SegmentList (highest precedence)
               for _, segURL := range rep.SegmentList.SegmentURLs {
                  relativeSeg, err := url.Parse(segURL.Media)
                  if err != nil {
                     fmt.Fprintf(os.Stderr, "Warning: Could not parse SegmentURL media '%s' for RepID '%s': %v\n", segURL.Media, rep.ID, err)
                     continue
                  }
                  resolvedURL := currentRepBaseURL.ResolveReference(relativeSeg)
                  outputURLs[rep.ID] = append(outputURLs[rep.ID], resolvedURL.String())
               }

            } else if effectiveRepSegmentTemplate != nil {
               // Handle SegmentTemplate (if SegmentList is not present)

               // Handle initialization URL FIRST
               if effectiveRepSegmentTemplate.Initialization != "" {
                  relativeInit, err := url.Parse(effectiveRepSegmentTemplate.Initialization)
                  if err != nil {
                     fmt.Fprintf(os.Stderr, "Warning: Could not parse Initialization URL '%s' for RepID '%s': %v\n", effectiveRepSegmentTemplate.Initialization, rep.ID, err)
                  } else {
                     resolvedURL := currentRepBaseURL.ResolveReference(relativeInit)
                     outputURLs[rep.ID] = append(outputURLs[rep.ID], resolvedURL.String())
                  }
               }

               // Handle media segments from SegmentTemplate
               if effectiveRepSegmentTemplate.Media != "" {
                  if effectiveRepSegmentTemplate.SegmentTimeline != nil {
                     // SegmentTimeline (most detailed)
                     var segmentTime int
                     currentSegmentNumber := effectiveRepSegmentTemplate.StartNumber
                     if currentSegmentNumber == 0 {
                        currentSegmentNumber = 1
                     }

                     for i, s := range effectiveRepSegmentTemplate.SegmentTimeline.Ss {
                        if s.T != 0 {
                           segmentTime = s.T
                        } else if i == 0 && effectiveRepSegmentTemplate.StartNumber > 0 {
                           segmentTime = 0
                        }

                        numSegments := 1
                        if s.R > 0 {
                           numSegments = s.R + 1
                        }

                        for j := 0; j < numSegments; j++ {
                           mediaURL := effectiveRepSegmentTemplate.Media
                           mediaURL = strings.Replace(mediaURL, "$Number$", fmt.Sprintf("%d", currentSegmentNumber), -1)
                           mediaURL = strings.Replace(mediaURL, "$Time$", fmt.Sprintf("%d", segmentTime), -1)
                           if rep.ID != "" {
                              mediaURL = strings.Replace(mediaURL, "$RepresentationID$", rep.ID, -1)
                           }

                           relativeMedia, err := url.Parse(mediaURL)
                           if err != nil {
                              fmt.Fprintf(os.Stderr, "Warning: Could not parse SegmentTemplate media '%s' for RepID '%s': %v\n", mediaURL, rep.ID, err)
                              continue
                           }
                           resolvedURL := currentRepBaseURL.ResolveReference(relativeMedia)
                           outputURLs[rep.ID] = append(outputURLs[rep.ID], resolvedURL.String())

                           currentSegmentNumber++
                           segmentTime += s.D
                        }
                     }
                  } else if effectiveRepSegmentTemplate.EndNumber > 0 {
                     // SegmentTemplate with @endNumber (fixed number of segments)
                     startNum := effectiveRepSegmentTemplate.StartNumber
                     if startNum == 0 {
                        startNum = 1
                     }

                     for segNum := startNum; segNum <= effectiveRepSegmentTemplate.EndNumber; segNum++ {
                        mediaURL := effectiveRepSegmentTemplate.Media
                        mediaURL = strings.Replace(mediaURL, "$Number$", fmt.Sprintf("%d", segNum), -1)
                        if strings.Contains(mediaURL, "$Time$") {
                           fmt.Fprintf(os.Stderr, "Warning: SegmentTemplate for RepID '%s' has @endNumber but no SegmentTimeline. '$Time$' placeholder might not be resolved correctly.\n", rep.ID)
                        }

                        if rep.ID != "" {
                           mediaURL = strings.Replace(mediaURL, "$RepresentationID$", rep.ID, -1)
                        }

                        relativeMedia, err := url.Parse(mediaURL)
                        if err != nil {
                           fmt.Fprintf(os.Stderr, "Warning: Could not parse SegmentTemplate media '%s' for RepID '%s': %v\n", mediaURL, rep.ID, err)
                           continue
                        }
                        resolvedURL := currentRepBaseURL.ResolveReference(relativeMedia)
                        outputURLs[rep.ID] = append(outputURLs[rep.ID], resolvedURL.String())
                     }
                  } else if effectiveRepSegmentTemplate.Duration > 0 && effectiveRepSegmentTemplate.Timescale > 0 {
                     // SegmentTemplate with @duration (fixed duration, but indeterminate count without total duration)
                     fmt.Fprintf(os.Stderr, "Warning: SegmentTemplate for RepID '%s' has Duration/Timescale but no SegmentTimeline or EndNumber. Outputting template string as is (cannot resolve all segments).\n", rep.ID)
                     outputURLs[rep.ID] = append(outputURLs[rep.ID], effectiveRepSegmentTemplate.Media)
                  } else {
                     // SegmentTemplate exists but has no sufficient attributes to generate segments
                     fmt.Fprintf(os.Stderr, "Warning: SegmentTemplate for RepID '%s' exists but lacks SegmentTimeline, EndNumber, or Duration/Timescale to generate segment URLs. Outputting template string as is.\n", rep.ID)
                     outputURLs[rep.ID] = append(outputURLs[rep.ID], effectiveRepSegmentTemplate.Media)
                  }
               }
            } else {
               fmt.Fprintf(os.Stderr, "Warning: Representation '%s' has no SegmentList or SegmentTemplate (neither direct nor inherited from AdaptationSet). No segments will be extracted for this representation.\n", rep.ID)
            }
         }
      }
   }

   jsonData, err := json.MarshalIndent(outputURLs, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonData)) // ONLY JSON to stdout
}
