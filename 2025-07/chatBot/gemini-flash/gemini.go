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

// Define simplified structs to parse common MPD elements
type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   XMLName        xml.Name        `xml:"Period"`
   ID             string          `xml:"id,attr"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
   BaseURL        string          `xml:"BaseURL"`
}

type AdaptationSet struct {
   XMLName         xml.Name         `xml:"AdaptationSet"`
   ContentType     string           `xml:"contentType,attr"`
   Representations []Representation `xml:"Representation"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"` // Inherited SegmentTemplate
}

type Representation struct {
   XMLName         xml.Name         `xml:"Representation"`
   ID              string           `xml:"id,attr"`
   MimeType        string           `xml:"mimeType,attr"`
   Codecs          string           `xml:"codecs,attr"`
   Bandwidth       uint64           `xml:"bandwidth,attr"`
   Initialization  *Initialization  `xml:"Initialization"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"` // Specific SegmentTemplate
   SegmentList     *SegmentList     `xml:"SegmentList"`
   BaseURL         string           `xml:"BaseURL"`
}

type Initialization struct {
   XMLName   xml.Name `xml:"Initialization"`
   SourceURL string   `xml:"sourceURL,attr"`
   Range     string   `xml:"range,attr"`
}

type SegmentTemplate struct {
   XMLName         xml.Name         `xml:"SegmentTemplate"`
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   StartNumber     uint64           `xml:"startNumber,attr"`
   EndNumber       uint64           `xml:"endNumber,attr"`
   Timescale       uint64           `xml:"timescale,attr"`
   Duration        uint64           `xml:"duration,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   XMLName xml.Name `xml:"SegmentTimeline"`
   Ss      []S      `xml:"S"`
}

type S struct {
   XMLName xml.Name `xml:"S"`
   T       *int64   `xml:"t,attr"` // Use pointer to differentiate between absent and zero
   D       uint64   `xml:"d,attr"`
   R       int      `xml:"r,attr"`
}

type SegmentList struct {
   XMLName        xml.Name        `xml:"SegmentList"`
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type SegmentURL struct {
   XMLName xml.Name `xml:"SegmentURL"`
   Media   string   `xml:"media,attr"`
}

// Global base URL for initial resolution
var initialBaseURL *url.URL

func init() {
   var err error
   initialBaseURL, err = url.Parse("http://test.test/test.mpd")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Fatal error parsing initial base URL: %v\n", err)
      os.Exit(1)
   }
}

// resolveURL resolves a relative URL against a base URL.
func resolveURL(base *url.URL, ref string) (*url.URL, error) {
   parsedRef, err := url.Parse(ref)
   if err != nil {
      return nil, fmt.Errorf("error parsing reference URL '%s': %w", ref, err)
   }
   return base.ResolveReference(parsedRef), nil
}

func main() {
   if len(os.Args) < 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <path_to_mpd_file>\n", os.Args[0])
      os.Exit(1)
   }

   mpdFilePath := os.Args[1]

   // Read the MPD file
   xmlFile, err := os.Open(mpdFilePath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error opening MPD file '%s': %v\n", mpdFilePath, err)
      os.Exit(1)
   }
   defer xmlFile.Close()

   byteValue, err := ioutil.ReadAll(xmlFile)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading MPD file '%s': %v\n", mpdFilePath, err)
      os.Exit(1)
   }

   var mpd MPD
   err = xml.Unmarshal(byteValue, &mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error unmarshalling MPD: %v\n", err)
      os.Exit(1)
   }

   outputURLs := make(map[string][]string)

   currentBaseURL := initialBaseURL

   // Resolve MPD's BaseURL
   if mpd.BaseURL != "" {
      resolvedURL, err := resolveURL(currentBaseURL, mpd.BaseURL)
      if err != nil {
         fmt.Fprintf(os.Stderr, "Warning: Could not resolve MPD BaseURL '%s': %v\n", mpd.BaseURL, err)
      } else {
         currentBaseURL = resolvedURL
      }
   }

   for _, period := range mpd.Periods {
      periodBaseURL := currentBaseURL
      if period.BaseURL != "" {
         resolvedURL, err := resolveURL(currentBaseURL, period.BaseURL)
         if err != nil {
            fmt.Fprintf(os.Stderr, "Warning: Could not resolve Period BaseURL '%s' (Period ID: %s): %v\n", period.BaseURL, period.ID, err)
         } else {
            periodBaseURL = resolvedURL
         }
      }

      for _, as := range period.AdaptationSets {
         asBaseURL := periodBaseURL
         if as.BaseURL != "" {
            resolvedURL, err := resolveURL(periodBaseURL, as.BaseURL)
            if err != nil {
               fmt.Fprintf(os.Stderr, "Warning: Could not resolve AdaptationSet BaseURL '%s' (Period ID: %s, ContentType: %s): %v\n", as.BaseURL, period.ID, as.ContentType, err)
            } else {
               asBaseURL = resolvedURL
            }
         }

         // Inherited SegmentTemplate from AdaptationSet
         inheritedSegmentTemplate := as.SegmentTemplate

         for _, rep := range as.Representations {
            repBaseURL := asBaseURL
            if rep.BaseURL != "" {
               resolvedURL, err := resolveURL(asBaseURL, rep.BaseURL)
               if err != nil {
                  fmt.Fprintf(os.Stderr, "Warning: Could not resolve Representation BaseURL '%s' (Period ID: %s, AdaptationSet ContentType: %s, Representation ID: %s): %v\n", rep.BaseURL, period.ID, as.ContentType, rep.ID, err)
               } else {
                  repBaseURL = resolvedURL
               }
            }

            var urls []string

            // Prioritize SegmentList
            if rep.SegmentList != nil {
               // Handle Initialization for SegmentList
               if rep.SegmentList.Initialization != nil && rep.SegmentList.Initialization.SourceURL != "" {
                  resolvedInitURL, err := resolveURL(repBaseURL, rep.SegmentList.Initialization.SourceURL)
                  if err != nil {
                     fmt.Fprintf(os.Stderr, "Warning: Could not resolve SegmentList Initialization URL '%s' (Rep ID: %s): %v\n", rep.SegmentList.Initialization.SourceURL, rep.ID, err)
                  } else {
                     urls = append(urls, resolvedInitURL.String())
                  }
               }
               // Handle SegmentURLs
               for _, segURL := range rep.SegmentList.SegmentURLs {
                  if segURL.Media == "" {
                     fmt.Fprintf(os.Stderr, "Warning: SegmentList SegmentURL 'media' attribute is empty (Rep ID: %s)\n", rep.ID)
                     continue
                  }
                  resolvedSegURL, err := resolveURL(repBaseURL, segURL.Media)
                  if err != nil {
                     fmt.Fprintf(os.Stderr, "Warning: Could not resolve SegmentList SegmentURL '%s' (Rep ID: %s): %v\n", segURL.Media, rep.ID, err)
                  } else {
                     urls = append(urls, resolvedSegURL.String())
                  }
               }
            } else if rep.SegmentTemplate != nil || inheritedSegmentTemplate != nil {
               var currentST *SegmentTemplate
               if rep.SegmentTemplate != nil {
                  currentST = rep.SegmentTemplate
               } else { // Use inherited
                  currentST = inheritedSegmentTemplate
               }

               if currentST.Media == "" {
                  fmt.Fprintf(os.Stderr, "Warning: SegmentTemplate 'media' attribute is empty for Representation ID: %s. No segments can be extracted.\n", rep.ID)
                  outputURLs[rep.ID] = urls
                  continue
               }

               // Handle Initialization for SegmentTemplate
               if currentST.Initialization != "" {
                  initTemplate := currentST.Initialization
                  initTemplate = strings.ReplaceAll(initTemplate, "$RepresentationID$", rep.ID)
                  initTemplate = strings.ReplaceAll(initTemplate, "$Bandwidth$", strconv.FormatUint(rep.Bandwidth, 10))

                  resolvedInitURL, err := resolveURL(repBaseURL, initTemplate)
                  if err != nil {
                     fmt.Fprintf(os.Stderr, "Warning: Could not resolve SegmentTemplate Initialization URL '%s' (Rep ID: %s): %v\n", initTemplate, rep.ID, err)
                  } else {
                     urls = append(urls, resolvedInitURL.String())
                  }
               }

               // Handle Segment URLs based on SegmentTemplate
               mediaTemplate := currentST.Media
               mediaTemplate = strings.ReplaceAll(mediaTemplate, "$RepresentationID$", rep.ID)
               mediaTemplate = strings.ReplaceAll(mediaTemplate, "$Bandwidth$", strconv.FormatUint(rep.Bandwidth, 10))

               if currentST.SegmentTimeline != nil {
                  time := uint64(0)
                  if currentST.StartNumber != 0 {
                     time = currentST.StartNumber * currentST.Timescale // Use startNumber if present for initial time
                  }
                  currentSegmentNumber := currentST.StartNumber
                  if currentSegmentNumber == 0 {
                     currentSegmentNumber = 1 // Default startNumber is 1 if not specified
                  }

                  for idx, s := range currentST.SegmentTimeline.Ss {
                     // If 't' attribute is present, it explicitly sets the start time
                     if s.T != nil {
                        time = uint64(*s.T)
                     }

                     numSegments := s.R + 1 // r="0" means 1 segment, r="1" means 2 segments, etc.
                     for i := 0; i < numSegments; i++ {
                        segmentURLStr := strings.ReplaceAll(mediaTemplate, "$Time$", strconv.FormatUint(time, 10))
                        segmentURLStr = strings.ReplaceAll(segmentURLStr, "$Number$", strconv.FormatUint(currentSegmentNumber, 10))

                        resolvedSegURL, err := resolveURL(repBaseURL, segmentURLStr)
                        if err != nil {
                           fmt.Fprintf(os.Stderr, "Warning: Could not resolve SegmentTemplate Segment URL '%s' (Rep ID: %s, SegmentTimeline S index: %d, repeat: %d): %v\n", segmentURLStr, rep.ID, idx, i, err)
                        } else {
                           urls = append(urls, resolvedSegURL.String())
                        }
                        time += s.D
                        currentSegmentNumber++
                     }
                  }
               } else if currentST.EndNumber != 0 {
                  startNumber := currentST.StartNumber
                  if startNumber == 0 {
                     startNumber = 1 // Default startNumber is 1 if not specified
                  }
                  for i := startNumber; i <= currentST.EndNumber; i++ {
                     segmentURLStr := strings.ReplaceAll(mediaTemplate, "$Number$", strconv.FormatUint(i, 10))
                     if strings.Contains(segmentURLStr, "$Time$") {
                        fmt.Fprintf(os.Stderr, "Warning: $Time$ placeholder found in SegmentTemplate media without SegmentTimeline (Rep ID: %s). May not be resolved correctly.\n", rep.ID)
                     }
                     resolvedSegURL, err := resolveURL(repBaseURL, segmentURLStr)
                     if err != nil {
                        fmt.Fprintf(os.Stderr, "Warning: Could not resolve SegmentTemplate Segment URL '%s' (Rep ID: %s, Segment Number: %d): %v\n", segmentURLStr, rep.ID, i, err)
                     } else {
                        urls = append(urls, resolvedSegURL.String())
                     }
                  }
               } else if currentST.Duration != 0 && currentST.Timescale != 0 {
                  fmt.Fprintf(os.Stderr, "Warning: SegmentTemplate has @duration and @timescale but no SegmentTimeline or @endNumber for Representation ID: %s. Cannot resolve all segments. Raw Media template: %s\n", rep.ID, mediaTemplate)
                  // Output the raw media template as per requirement if segments cannot be fully resolved
                  resolvedSegURL, err := resolveURL(repBaseURL, mediaTemplate)
                  if err != nil {
                     fmt.Fprintf(os.Stderr, "Warning: Could not resolve raw media template '%s' (Rep ID: %s): %v\n", mediaTemplate, rep.ID, err)
                  } else {
                     urls = append(urls, resolvedSegURL.String())
                  }
               } else {
                  fmt.Fprintf(os.Stderr, "Warning: Insufficient SegmentTemplate attributes (no SegmentTimeline, @endNumber, @duration/@timescale) for Representation ID: %s. No segments can be extracted. Raw Media template: %s\n", rep.ID, mediaTemplate)
                  // Output the raw media template as per requirement if segments cannot be fully extracted
                  resolvedSegURL, err := resolveURL(repBaseURL, mediaTemplate)
                  if err != nil {
                     fmt.Fprintf(os.Stderr, "Warning: Could not resolve raw media template '%s' (Rep ID: %s): %v\n", mediaTemplate, rep.ID, err)
                  } else {
                     urls = append(urls, resolvedSegURL.String())
                  }
               }
            } else {
               // Case: Representation has neither SegmentList nor SegmentTemplate
               // Emit the absolute URL derived from its effective BaseURL
               if repBaseURL != nil {
                  fmt.Fprintf(os.Stderr, "Warning: Representation ID: %s has neither SegmentList nor SegmentTemplate. Emitting effective BaseURL as the playable URL.\n", rep.ID)
                  urls = append(urls, repBaseURL.String())
               } else {
                  fmt.Fprintf(os.Stderr, "Warning: Representation ID: %s has neither SegmentList nor SegmentTemplate, and no effective BaseURL to derive a playable URL.\n", rep.ID)
               }
            }
            outputURLs[rep.ID] = urls
         }
      }
   }

   jsonOutput, err := json.MarshalIndent(outputURLs, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error marshalling JSON output: %v\n", err)
      os.Exit(1)
   }

   fmt.Fprintln(os.Stdout, string(jsonOutput))
}
