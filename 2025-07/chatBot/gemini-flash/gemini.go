package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "net/url"
   "os"
   "strconv"
   "strings"
)

// Define structs to unmarshal the XML MPD file
type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   XMLName xml.Name `xml:"Period"`
   ID string `xml:"id,attr"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   XMLName xml.Name `xml:"AdaptationSet"`
   Representations []Representation `xml:"Representation"`
   BaseURL string `xml:"BaseURL"` // AdaptationSet can also have BaseURL
}

type Representation struct {
   XMLName xml.Name `xml:"Representation"`
   ID string `xml:"id,attr"`
   Bandwidth int `xml:"bandwidth,attr"`
   MimeType string `xml:"mimeType,attr"`
   Codecs string `xml:"codecs,attr"`
   BaseURL string `xml:"BaseURL"` // Representation can have BaseURL (e.g., for single file)
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList *SegmentList `xml:"SegmentList"`
   SegmentBase *SegmentBase `xml:"SegmentBase"`
}

type SegmentTemplate struct {
   XMLName xml.Name `xml:"SegmentTemplate"`
   Media string `xml:"media,attr"`
   Initialization string `xml:"initialization,attr"`
   StartNumber *int `xml:"startNumber,attr"` // Changed to pointer to int
   Duration int `xml:"duration,attr"` // Duration in timescale units
   Timescale int `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   XMLName xml.Name `xml:"SegmentTimeline"`
   Ss []S `xml:"S"`
}

type S struct {
   T int `xml:"t,attr"` // Start time
   D int `xml:"d,attr"` // Duration
   R int `xml:"r,attr"` // Repeat count
}

type Initialization struct {
   XMLName xml.Name `xml:"Initialization"`
   Source string `xml:"sourceURL,attr"`
   Range string `xml:"range,attr"`
}

type SegmentURL struct {
   XMLName xml.Name `xml:"SegmentURL"`
   Source string `xml:"sourceURL,attr"`
   Range string `xml:"range,attr"`
}

type SegmentList struct {
   XMLName xml.Name `xml:"SegmentList"`
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs []SegmentURL `xml:"SegmentURL"`
   Timescale int `xml:"timescale,attr"`
   Duration int `xml:"duration,attr"`
}

type SegmentBase struct {
   XMLName xml.Name `xml:"SegmentBase"`
   Initialization *Initialization `xml:"Initialization"`
   IndexRange string `xml:"indexRange,attr"`
}

type URL struct {
   XMLName xml.Name `xml:"URL"`
   Source string `xml:"sourceURL,attr"`
   Range string `xml:"range,attr"`
}


// Function to resolve URLs
func resolveURL(base *url.URL, relativePath string) string {
   if relativePath == "" {
      return ""
   }
   rel, err := url.Parse(relativePath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing relative URL '%s': %v\n", relativePath, err)
      return relativePath // Return original if parsing fails
   }
   return base.ResolveReference(rel).String()
}

func main() {
   if len(os.Args) < 2 {
      fmt.Println("Usage: go run mpd_parser.go <path_to_mpd_file>")
      os.Exit(1)
   }

   mpdFilePath := os.Args[1]
   mpdBaseURLStr := "http://test.test/test.mpd" // User-specified base URL

   mpdBaseURL, err := url.Parse(mpdBaseURLStr)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD base URL '%s': %v\n", mpdBaseURLStr, err)
      os.Exit(1)
   }

   xmlFile, err := os.Open(mpdFilePath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error opening MPD file: %v\n", err)
      os.Exit(1)
   }
   defer xmlFile.Close()

   byteValue, _ := ioutil.ReadAll(xmlFile)

   var mpd MPD
   err = xml.Unmarshal(byteValue, &mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error unmarshaling XML: %v\n", err)
      os.Exit(1)
   }

   // Output structure
   output := make(map[string][]string)

   // Determine global BaseURL if present at MPD level
   currentBaseURL := mpdBaseURL
   if mpd.BaseURL != "" {
      parsedBase, err := url.Parse(mpd.BaseURL)
      if err == nil {
         currentBaseURL = currentBaseURL.ResolveReference(parsedBase)
      } else {
         fmt.Fprintf(os.Stderr, "Warning: Could not parse MPD-level BaseURL '%s'. Using default.\n", mpd.BaseURL)
      }
   }

   for _, period := range mpd.Periods {
      periodBaseURL := currentBaseURL // Inherit from MPD

      for _, as := range period.AdaptationSets {
         asBaseURL := periodBaseURL // Inherit from Period
         if as.BaseURL != "" {
            parsedBase, err := url.Parse(as.BaseURL)
            if err == nil {
               asBaseURL = asBaseURL.ResolveReference(parsedBase)
            } else {
               fmt.Fprintf(os.Stderr, "Warning: Could not parse AdaptationSet-level BaseURL '%s'. Using parent.\n", as.BaseURL)
            }
         }

         for _, rep := range as.Representations {
            repBaseURL := asBaseURL // Inherit from AdaptationSet
            if rep.BaseURL != "" {
               parsedBase, err := url.Parse(rep.BaseURL)
               if err == nil {
                  repBaseURL = repBaseURL.ResolveReference(parsedBase)
               } else {
                  fmt.Fprintf(os.Stderr, "Warning: Could not parse Representation-level BaseURL '%s'. Using parent.\n", rep.BaseURL)
               }
            }

            var segmentURLs []string

            // Handle SegmentTemplate
            if rep.SegmentTemplate != nil {
               template := rep.SegmentTemplate
               mediaTemplate := template.Media
               initTemplate := template.Initialization

               // Determine the effective startNumber
               effectiveStartNumber := 1 // Default value
               if template.StartNumber != nil {
                  effectiveStartNumber = *template.StartNumber // Use the value if present
               }


               // Resolve initialization URL
               if initTemplate != "" {
                  segmentURLs = append(segmentURLs, resolveURL(repBaseURL, initTemplate))
               }

               if template.SegmentTimeline != nil {
                  // SegmentTimeline based segments
                  var currentTime int // Initialize currentTime for the SegmentTimeline

                  if len(template.SegmentTimeline.Ss) > 0 {
                     if template.SegmentTimeline.Ss[0].T != 0 {
                        currentTime = template.SegmentTimeline.Ss[0].T
                     } else {
                        currentTime = 0
                     }
                  }

                  currentSegmentNumber := effectiveStartNumber // Use the effective startNumber

                  for _, s := range template.SegmentTimeline.Ss {
                     if s.T != 0 {
                        currentTime = s.T
                     }

                     count := 1
                     if s.R > 0 {
                        count += s.R
                     }

                     for i := 0; i < count; i++ {
                        segmentURL := strings.Replace(mediaTemplate, "$Number$", strconv.Itoa(currentSegmentNumber), -1)
                        segmentURL = strings.Replace(segmentURL, "$Time$", strconv.Itoa(currentTime), -1)
                        segmentURLs = append(segmentURLs, resolveURL(repBaseURL, segmentURL))
                        currentSegmentNumber++
                        currentTime += s.D
                     }
                  }
               } else if mediaTemplate != "" && template.Duration > 0 && template.Timescale > 0 {
                  // Simple SegmentTemplate (e.g., $Number$)
                  for i := 0; i < 5; i++ { // Generate 5 segments as an example
                     segmentURL := strings.Replace(mediaTemplate, "$Number$", strconv.Itoa(effectiveStartNumber+i), -1)
                     segmentURLs = append(segmentURLs, resolveURL(repBaseURL, segmentURL))
                  }
               }
            } else if rep.SegmentList != nil {
               // Handle SegmentList
               if rep.SegmentList.Initialization != nil {
                  segmentURLs = append(segmentURLs, resolveURL(repBaseURL, rep.SegmentList.Initialization.Source))
               }
               for _, segURL := range rep.SegmentList.SegmentURLs {
                  segmentURLs = append(segmentURLs, resolveURL(repBaseURL, segURL.Source))
               }
            } else if rep.SegmentBase != nil {
               // Handle SegmentBase (initialization segment, usually single file)
               if rep.SegmentBase.Initialization != nil {
                  segmentURLs = append(segmentURLs, resolveURL(repBaseURL, rep.SegmentBase.Initialization.Source))
               }
            } else if rep.BaseURL != "" && rep.SegmentTemplate == nil && rep.SegmentList == nil && rep.SegmentBase == nil {
               // Handle case where BaseURL inside Representation refers to a single segment/file
               segmentURLs = append(segmentURLs, resolveURL(asBaseURL, rep.BaseURL))
            }

            if len(segmentURLs) > 0 {
               output[rep.ID] = segmentURLs
            }
         }
      }
   }

   jsonData, err := json.MarshalIndent(output, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonData))
}
