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
)

// MPD represents the root element of the MPEG-DASH MPD.
type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   BaseURL                   string   `xml:"BaseURL"`
   Periods                   []Period `xml:"Period"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
   MinimumUpdatePeriod       string   `xml:"minimumUpdatePeriod,attr"`
   Type                      string   `xml:"type,attr"`
   Profiles                  string   `xml:"profiles,attr"`
   PublishTime               string   `xml:"publishTime,attr"`
}

// Period represents a Period element within the MPD.
type Period struct {
   XMLName        xml.Name        `xml:"Period"`
   BaseURL        string          `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
   ID             string          `xml:"id,attr"`
   Start          string          `xml:"start,attr"`
   Duration       string          `xml:"duration,attr"` // Period duration
}

// AdaptationSet represents an AdaptationSet element.
type AdaptationSet struct {
   XMLName         xml.Name         `xml:"AdaptationSet"`
   BaseURL         string           `xml:"BaseURL"`
   Representations []Representation `xml:"Representation"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   ID              string           `xml:"id,attr"`
   ContentType     string           `xml:"contentType,attr"`
   MimeType        string           `xml:"mimeType,attr"`
   Codecs          string           `xml:"codecs,attr"`
   Bitrate         string           `xml:"bitrate,attr"`
}

// Representation represents a Representation element.
type Representation struct {
   XMLName         xml.Name         `xml:"Representation"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   ID              string           `xml:"id,attr"`
   Bandwidth       string           `xml:"bandwidth,attr"`
   Width           string           `xml:"width,attr"`
   Height          string           `xml:"height,attr"`
   Codecs          string           `xml:"codecs,attr"`
   MimeType        string           `xml:"mimeType,attr"`
}

// SegmentList represents a SegmentList element.
type SegmentList struct {
   XMLName        xml.Name        `xml:"SegmentList"`
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

// Initialization represents an Initialization element within SegmentList or SegmentBase.
type Initialization struct {
   XMLName   xml.Name `xml:"Initialization"`
   SourceURL string   `xml:"sourceURL,attr"`
   Range     string   `xml:"range,attr"`
}

// SegmentURL represents a SegmentURL element.
type SegmentURL struct {
   XMLName xml.Name `xml:"SegmentURL"`
   Media   string   `xml:"media,attr"`
   Index   string   `xml:"index,attr"`
}

// SegmentTemplate represents a SegmentTemplate element.
type SegmentTemplate struct {
   XMLName         xml.Name         `xml:"SegmentTemplate"`
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   Duration        string           `xml:"duration,attr"` // Segment duration in timescale units
   Timescale       string           `xml:"timescale,attr"`
   StartNumber     string           `xml:"startNumber,attr"`
   EndNumber       string           `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline represents a SegmentTimeline element.
type SegmentTimeline struct {
   XMLName xml.Name `xml:"SegmentTimeline"`
   S       []S      `xml:"S"`
}

// S represents an S element within SegmentTimeline.
type S struct {
   XMLName xml.Name `xml:"S"`
   T       string   `xml:"t,attr"`
   D       string   `xml:"d,attr"`
   R       string   `xml:"r,attr"`
}

// ParsedSegments stores the resolved segment URLs for each Representation.
type ParsedSegments map[string][]string

// parseDuration parses an ISO 8601 duration string (e.g., "PT1H2M3S") and returns total seconds.
func parseDuration(isoDuration string) (float64, error) {
   if !strings.HasPrefix(isoDuration, "PT") {
      return 0, fmt.Errorf("invalid ISO 8601 duration format, must start with 'PT': %s", isoDuration)
   }

   totalSeconds := 0.0
   temp := isoDuration[2:]

   var h, m, s string
   var currentVal string

   for _, r := range temp {
      if r == 'H' {
         if currentVal != "" {
            val, err := strconv.ParseFloat(currentVal, 64)
            if err != nil {
               return 0, fmt.Errorf("invalid hours value in duration: %s", currentVal)
            }
            h = fmt.Sprintf("%f", val)
            currentVal = ""
         }
      } else if r == 'M' {
         if currentVal != "" {
            val, err := strconv.ParseFloat(currentVal, 64)
            if err != nil {
               return 0, fmt.Errorf("invalid minutes value in duration: %s", currentVal)
            }
            m = fmt.Sprintf("%f", val)
            currentVal = ""
         }
      } else if r == 'S' {
         if currentVal != "" {
            val, err := strconv.ParseFloat(currentVal, 64)
            if err != nil {
               return 0, fmt.Errorf("invalid seconds value in duration: %s", currentVal)
            }
            s = fmt.Sprintf("%f", val)
            currentVal = ""
         }
      } else {
         currentVal += string(r)
      }
   }

   if h != "" {
      val, _ := strconv.ParseFloat(h, 64)
      totalSeconds += val * 3600
   }
   if m != "" {
      val, _ := strconv.ParseFloat(m, 64)
      totalSeconds += val * 60
   }
   if s != "" {
      val, _ := strconv.ParseFloat(s, 64)
      totalSeconds += val
   }

   return totalSeconds, nil
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <path_to_mpd_file>\n", os.Args[0])
      os.Exit(1)
   }

   mpdFilePath := os.Args[1]

   data, err := ioutil.ReadFile(mpdFilePath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading MPD file: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   err = xml.Unmarshal(data, &mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error unmarshaling MPD XML: %v\n", err)
      os.Exit(1)
   }

   initialBaseURL, err := url.Parse("http://test.test/test.mpd")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Fatal: Could not parse initial base URL: %v\n", err)
      os.Exit(1)
   }

   allSegments := make(ParsedSegments)

   currentBaseURL := initialBaseURL
   if mpd.BaseURL != "" {
      refURL, parseErr := url.Parse(mpd.BaseURL)
      if parseErr != nil {
         fmt.Fprintf(os.Stderr, "Warning: MPD BaseURL '%s' is malformed and skipped: %v\n", mpd.BaseURL, parseErr)
      } else {
         currentBaseURL = currentBaseURL.ResolveReference(refURL)
      }
   }

   for _, period := range mpd.Periods {
      periodBaseURL := currentBaseURL
      if period.BaseURL != "" {
         refURL, parseErr := url.Parse(period.BaseURL)
         if parseErr != nil {
            fmt.Fprintf(os.Stderr, "Warning: Period BaseURL '%s' is malformed and skipped: %v\n", period.BaseURL, parseErr)
         } else {
            periodBaseURL = periodBaseURL.ResolveReference(refURL)
         }
      }

      periodDurationSeconds := 0.0
      if period.Duration != "" {
         parsedDur, err := parseDuration(period.Duration)
         if err != nil {
            fmt.Fprintf(os.Stderr, "Warning: Period '%s': Could not parse duration '%s': %v. Segment count calculation might be inaccurate.\n", period.ID, period.Duration, err)
         } else {
            periodDurationSeconds = parsedDur
         }
      } else if mpd.MediaPresentationDuration != "" {
         parsedDur, err := parseDuration(mpd.MediaPresentationDuration)
         if err != nil {
            fmt.Fprintf(os.Stderr, "Warning: MPD MediaPresentationDuration '%s': Could not parse duration: %v. Segment count calculation might be inaccurate.\n", mpd.MediaPresentationDuration, err)
         } else {
            periodDurationSeconds = parsedDur
         }
      }

      for _, as := range period.AdaptationSets {
         asBaseURL := periodBaseURL
         if as.BaseURL != "" {
            refURL, parseErr := url.Parse(as.BaseURL)
            if parseErr != nil {
               fmt.Fprintf(os.Stderr, "Warning: AdaptationSet BaseURL '%s' is malformed and skipped: %v\n", as.BaseURL, parseErr)
            } else {
               asBaseURL = asBaseURL.ResolveReference(refURL)
            }
         }

         effectiveSegmentTemplate := as.SegmentTemplate

         for _, rep := range as.Representations {
            repBaseURL := asBaseURL
            if rep.BaseURL != "" {
               refURL, parseErr := url.Parse(rep.BaseURL)
               if parseErr != nil {
                  fmt.Fprintf(os.Stderr, "Warning: Representation BaseURL '%s' is malformed and skipped: %v\n", rep.BaseURL, parseErr)
               } else {
                  repBaseURL = repBaseURL.ResolveReference(refURL)
               }
            }

            currentRepSegments := []string{}
            representationID := rep.ID
            if representationID == "" {
               representationID = fmt.Sprintf("representation-fallback-%p", &rep)
               fmt.Fprintf(os.Stderr, "Warning: Representation missing ID, using fallback: %s\n", representationID)
            }

            currentRepSegmentTemplate := rep.SegmentTemplate
            if currentRepSegmentTemplate == nil {
               currentRepSegmentTemplate = effectiveSegmentTemplate
            }

            if rep.SegmentList != nil {
               if rep.SegmentList.Initialization != nil && rep.SegmentList.Initialization.SourceURL != "" {
                  refURL, parseErr := url.Parse(rep.SegmentList.Initialization.SourceURL)
                  if parseErr != nil {
                     fmt.Fprintf(os.Stderr, "Warning: Representation %s: SegmentList Initialization SourceURL '%s' is malformed and skipped: %v\n", representationID, rep.SegmentList.Initialization.SourceURL, parseErr)
                  } else {
                     resolvedURL := repBaseURL.ResolveReference(refURL)
                     currentRepSegments = append(currentRepSegments, resolvedURL.String())
                  }
               }
               if len(rep.SegmentList.SegmentURLs) > 0 {
                  for _, segURL := range rep.SegmentList.SegmentURLs {
                     if segURL.Media != "" {
                        refURL, parseErr := url.Parse(segURL.Media)
                        if parseErr != nil {
                           fmt.Fprintf(os.Stderr, "Warning: Representation %s: SegmentURL media '%s' is malformed and skipped: %v\n", representationID, segURL.Media, parseErr)
                           continue
                        }
                        resolvedURL := repBaseURL.ResolveReference(refURL)
                        currentRepSegments = append(currentRepSegments, resolvedURL.String())
                     }
                  }
               }
            } else if currentRepSegmentTemplate != nil {
               if currentRepSegmentTemplate.Initialization != "" {
                  refURL, parseErr := url.Parse(currentRepSegmentTemplate.Initialization)
                  if parseErr != nil {
                     fmt.Fprintf(os.Stderr, "Warning: Representation %s: SegmentTemplate initialization URL '%s' is malformed and skipped: %v\n", representationID, currentRepSegmentTemplate.Initialization, parseErr)
                  } else {
                     resolvedURL := repBaseURL.ResolveReference(refURL)
                     currentRepSegments = append(currentRepSegments, resolvedURL.String())
                  }
               }

               segmentStartNumber := 1
               if currentRepSegmentTemplate.StartNumber != "" {
                  parsedStartNumber, err := strconv.Atoi(currentRepSegmentTemplate.StartNumber)
                  if err != nil {
                     fmt.Fprintf(os.Stderr, "Warning: Representation %s: Invalid 'startNumber' attribute '%s' in SegmentTemplate, defaulting to 1: %v\n", representationID, currentRepSegmentTemplate.StartNumber, err)
                  } else {
                     segmentStartNumber = parsedStartNumber
                  }
               }

               // Get effective timescale (default to 1)
               segTimescale := 1.0
               if currentRepSegmentTemplate.Timescale != "" {
                  parsedTimescale, errTimescale := strconv.ParseFloat(currentRepSegmentTemplate.Timescale, 64)
                  if errTimescale == nil && parsedTimescale > 0 {
                     segTimescale = parsedTimescale
                  } else {
                     fmt.Fprintf(os.Stderr, "Warning: Representation %s: Invalid or zero 'timescale' attribute '%s' in SegmentTemplate, defaulting to 1: %v\n", representationID, currentRepSegmentTemplate.Timescale, errTimescale)
                  }
               }

               if currentRepSegmentTemplate.SegmentTimeline != nil && len(currentRepSegmentTemplate.SegmentTimeline.S) > 0 {
                  segmentNumber := segmentStartNumber
                  currentTime := 0

                  for _, s := range currentRepSegmentTemplate.SegmentTimeline.S {
                     if s.T != "" {
                        tVal, err := strconv.Atoi(s.T)
                        if err != nil {
                           fmt.Fprintf(os.Stderr, "Warning: Representation %s: Invalid 't' attribute in SegmentTimeline 'S' element '%s', continuing from last time: %v\n", representationID, s.T, err)
                        } else {
                           currentTime = tVal
                        }
                     }

                     dVal, err := strconv.Atoi(s.D)
                     if err != nil {
                        fmt.Fprintf(os.Stderr, "Warning: Representation %s: Invalid 'd' attribute in SegmentTimeline 'S' element '%s', skipping segment: %v\n", representationID, s.D, err)
                        continue
                     }

                     rVal := 0
                     if s.R != "" {
                        rVal, err = strconv.Atoi(s.R)
                        if err != nil {
                           fmt.Fprintf(os.Stderr, "Warning: Representation %s: Invalid 'r' attribute in SegmentTimeline 'S' element '%s', defaulting to 0 repeats: %v\n", representationID, s.R, err)
                        }
                     }

                     for i := 0; i <= rVal; i++ {
                        segmentURLTemplate := currentRepSegmentTemplate.Media

                        segmentURLTemplate = strings.ReplaceAll(segmentURLTemplate, "$RepresentationID$", representationID)

                        for x := 9; x >= 2; x-- {
                           paddedNumPlaceholder := fmt.Sprintf("$Number%%0%dd$", x)
                           if strings.Contains(segmentURLTemplate, paddedNumPlaceholder) {
                              formatStr := fmt.Sprintf("%%0%dd", x)
                              segmentURLTemplate = strings.ReplaceAll(segmentURLTemplate, paddedNumPlaceholder, fmt.Sprintf(formatStr, segmentNumber))
                           }
                        }

                        segmentURLTemplate = strings.ReplaceAll(segmentURLTemplate, "$Number$", fmt.Sprintf("%d", segmentNumber))
                        segmentURLTemplate = strings.ReplaceAll(segmentURLTemplate, "$Time$", fmt.Sprintf("%d", currentTime))

                        refURL, parseErr := url.Parse(segmentURLTemplate)
                        if parseErr != nil {
                           fmt.Fprintf(os.Stderr, "Warning: Representation %s: Generated segment URL '%s' is malformed and skipped: %v\n", representationID, segmentURLTemplate, parseErr)
                           continue
                        }
                        resolvedURL := repBaseURL.ResolveReference(refURL)
                        currentRepSegments = append(currentRepSegments, resolvedURL.String())
                        segmentNumber++
                        currentTime += dVal
                     }
                  }
               } else if currentRepSegmentTemplate.EndNumber != "" {
                  endNumber, err := strconv.Atoi(currentRepSegmentTemplate.EndNumber)
                  if err != nil {
                     fmt.Fprintf(os.Stderr, "Warning: Representation %s: Invalid 'endNumber' attribute in SegmentTemplate '%s', cannot generate segments: %v\n", representationID, currentRepSegmentTemplate.EndNumber, err)
                     continue
                  }

                  if strings.Contains(currentRepSegmentTemplate.Media, "$Time$") {
                     fmt.Fprintf(os.Stderr, "Warning: Representation %s: SegmentTemplate media contains '$Time$' placeholder but no SegmentTimeline is present. Time cannot be precisely resolved and will be replaced with '0'.\n", representationID)
                  }

                  for i := segmentStartNumber; i <= endNumber; i++ {
                     segmentURLTemplate := currentRepSegmentTemplate.Media

                     segmentURLTemplate = strings.ReplaceAll(segmentURLTemplate, "$RepresentationID$", representationID)

                     for x := 9; x >= 2; x-- {
                        paddedNumPlaceholder := fmt.Sprintf("$Number%%0%dd$", x)
                        if strings.Contains(segmentURLTemplate, paddedNumPlaceholder) {
                           formatStr := fmt.Sprintf("%%0%dd", x)
                           segmentURLTemplate = strings.ReplaceAll(segmentURLTemplate, paddedNumPlaceholder, fmt.Sprintf(formatStr, i))
                        }
                     }

                     segmentURLTemplate = strings.ReplaceAll(segmentURLTemplate, "$Number$", fmt.Sprintf("%d", i))
                     segmentURLTemplate = strings.ReplaceAll(segmentURLTemplate, "$Time$", "0")

                     refURL, parseErr := url.Parse(segmentURLTemplate)
                     if parseErr != nil {
                        fmt.Fprintf(os.Stderr, "Warning: Representation %s: Generated segment URL '%s' is malformed and skipped: %v\n", representationID, segmentURLTemplate, parseErr)
                        continue
                     }
                     resolvedURL := repBaseURL.ResolveReference(refURL)
                     currentRepSegments = append(currentRepSegments, resolvedURL.String())
                  }
               } else if currentRepSegmentTemplate.Duration != "" {
                  segDuration, errDuration := strconv.ParseFloat(currentRepSegmentTemplate.Duration, 64)

                  // segTimescale is already determined above (default to 1 or parsed value)

                  if errDuration == nil && segDuration > 0 && periodDurationSeconds > 0 {
                     segmentDurationInSeconds := segDuration / segTimescale
                     count := int(math.Ceil(periodDurationSeconds / segmentDurationInSeconds))

                     if strings.Contains(currentRepSegmentTemplate.Media, "$Time$") {
                        fmt.Fprintf(os.Stderr, "Warning: Representation %s: SegmentTemplate media contains '$Time$' placeholder, but no SegmentTimeline. Time will be replaced with '0' for calculated segments.\n", representationID)
                     }

                     for i := 0; i < count; i++ {
                        segmentURLTemplate := currentRepSegmentTemplate.Media

                        segmentURLTemplate = strings.ReplaceAll(segmentURLTemplate, "$RepresentationID$", representationID)

                        for x := 9; x >= 2; x-- {
                           paddedNumPlaceholder := fmt.Sprintf("$Number%%0%dd$", x)
                           if strings.Contains(segmentURLTemplate, paddedNumPlaceholder) {
                              formatStr := fmt.Sprintf("%%0%dd", x)
                              segmentURLTemplate = strings.ReplaceAll(segmentURLTemplate, paddedNumPlaceholder, fmt.Sprintf(formatStr, segmentStartNumber+i))
                           }
                        }

                        segmentURLTemplate = strings.ReplaceAll(segmentURLTemplate, "$Number$", fmt.Sprintf("%d", segmentStartNumber+i))
                        segmentURLTemplate = strings.ReplaceAll(segmentURLTemplate, "$Time$", "0")

                        refURL, parseErr := url.Parse(segmentURLTemplate)
                        if parseErr != nil {
                           fmt.Fprintf(os.Stderr, "Warning: Representation %s: Generated segment URL '%s' is malformed and skipped: %v\n", representationID, segmentURLTemplate, parseErr)
                           continue
                        }
                        resolvedURL := repBaseURL.ResolveReference(refURL)
                        currentRepSegments = append(currentRepSegments, resolvedURL.String())
                     }
                  } else {
                     fmt.Fprintf(os.Stderr, "Warning: Representation %s: SegmentTemplate has 'duration'='%s' or 'timescale'='%s', but period duration (%.2f s) or segment parameters are invalid for calculating segment count. Outputting raw Media template: %s\n", representationID, currentRepSegmentTemplate.Duration, currentRepSegmentTemplate.Timescale, periodDurationSeconds, currentRepSegmentTemplate.Media)
                  }
               } else {
                  fmt.Fprintf(os.Stderr, "Warning: Representation %s: SegmentTemplate has insufficient attributes ('endNumber', 'SegmentTimeline', 'duration' or 'timescale' missing) to extract segments. Outputting raw Media template: %s\n", representationID, currentRepSegmentTemplate.Media)
               }
            } else {
               if repBaseURL.String() != "" && repBaseURL.String() != initialBaseURL.String() {
                  currentRepSegments = append(currentRepSegments, repBaseURL.String())
               } else {
                  fmt.Fprintf(os.Stderr, "Warning: Representation %s: No SegmentList, SegmentTemplate, or meaningful effective BaseURL found to derive playable URLs.\n", representationID)
               }
            }
            allSegments[representationID] = append(allSegments[representationID], currentRepSegments...)
         }
      }
   }

   jsonOutput, err := json.MarshalIndent(allSegments, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error marshaling JSON output: %v\n", err)
      os.Exit(1)
   }
   fmt.Println(string(jsonOutput))
}
