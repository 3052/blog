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

// MPD represents the root MPD element
type MPD struct {
   XMLName    xml.Name `xml:"MPD"`
   BaseURL    string   `xml:"BaseURL"`
   Periods    []Period `xml:"Period"`
   MediaPres  string   `xml:"mediaPresentationDuration,attr"`
   MinBufTime string   `xml:"minBufferTime,attr"`
}

// Period represents a Period element in MPD
type Period struct {
   XMLName    xml.Name        `xml:"Period"`
   BaseURL    string          `xml:"BaseURL"`
   Adaptation []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet represents an AdaptationSet element
type AdaptationSet struct {
   XMLName     xml.Name         `xml:"AdaptationSet"`
   BaseURL     string           `xml:"BaseURL"`
   Represent   []Representation `xml:"Representation"`
   SegmentTemp SegmentTemplate  `xml:"SegmentTemplate"`
   SegmentList SegmentList      `xml:"SegmentList"`
}

// Representation represents a Representation element
type Representation struct {
   XMLName     xml.Name        `xml:"Representation"`
   ID          string          `xml:"id,attr"`
   BaseURL     string          `xml:"BaseURL"`
   SegmentTemp SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList SegmentList     `xml:"SegmentList"`
}

// SegmentTemplate represents a SegmentTemplate element
type SegmentTemplate struct {
   XMLName       xml.Name        `xml:"SegmentTemplate"`
   BaseURL       string          `xml:"BaseURL"`
   Media         string          `xml:"media,attr"`
   Init          string          `xml:"initialization,attr"`
   StartNumber   int             `xml:"startNumber,attr"`
   EndNumber     int             `xml:"endNumber,attr"`
   Timescale     int             `xml:"timescale,attr"`
   Duration      int             `xml:"duration,attr"`
   SegmentTiming SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentList represents a SegmentList element
type SegmentList struct {
   XMLName     xml.Name     `xml:"SegmentList"`
   BaseURL     string       `xml:"BaseURL"`
   InitSegment InitSegment  `xml:"Initialization"`
   Segments    []SegmentURL `xml:"SegmentURL"`
}

// InitSegment represents initialization segment in SegmentList
type InitSegment struct {
   XMLName xml.Name `xml:"Initialization"`
   Source  string   `xml:"sourceURL,attr"`
}

// SegmentURL represents a segment URL in SegmentList
type SegmentURL struct {
   XMLName xml.Name `xml:"SegmentURL"`
   Media   string   `xml:"media,attr"`
}

// SegmentTimeline represents a SegmentTimeline element
type SegmentTimeline struct {
   XMLName xml.Name `xml:"SegmentTimeline"`
   S       []S      `xml:"S"`
}

// S represents an S element in SegmentTimeline
type S struct {
   XMLName xml.Name `xml:"S"`
   T       int      `xml:"t,attr"`
   D       int      `xml:"d,attr"`
   R       int      `xml:"r,attr"`
}

func main() {
   if len(os.Args) < 2 {
      fail("Usage: mpd_expander <path_to_mpd_file>")
   }

   mpdPath := os.Args[1]
   data, err := ioutil.ReadFile(mpdPath)
   if err != nil {
      fail(fmt.Sprintf("Error reading MPD file: %v", err))
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fail(fmt.Sprintf("Error parsing MPD XML: %v", err))
   }

   baseURL, err := url.Parse("http://test.test/test.mpd")
   if err != nil {
      fail(fmt.Sprintf("Error parsing base URL: %v", err))
   }

   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      periodBase := resolveURL(baseURL, period.BaseURL)

      for _, adapt := range period.Adaptation {
         adaptBase := resolveURL(periodBase, adapt.BaseURL)

         for _, repr := range adapt.Represent {
            reprBase := resolveURL(adaptBase, repr.BaseURL)

            // Determine which segment information to use
            segTemp, segList := getSegmentInfo(adapt, repr)

            // Get the appropriate base URL for segments
            segBase := reprBase
            if segTemp.BaseURL != "" {
               segBase = resolveURL(reprBase, segTemp.BaseURL)
            } else if segList.BaseURL != "" {
               segBase = resolveURL(reprBase, segList.BaseURL)
            }

            // Case 1: No segment info at all - use Representation's BaseURL as single segment
            if segTemp.Media == "" && len(segList.Segments) == 0 {
               if repr.BaseURL != "" {
                  result[repr.ID] = append(result[repr.ID], reprBase.String())
               } else {
                  warn(fmt.Sprintf("No segment information and no BaseURL for representation %s", repr.ID))
               }
               continue
            }

            // Process initialization segment
            processInitSegment(segBase, segTemp, segList, repr.ID, result)

            // Process media segments
            if len(segList.Segments) > 0 {
               // Case 2: SegmentList mode
               for _, seg := range segList.Segments {
                  absURL := resolveURL(segBase, seg.Media)
                  result[repr.ID] = append(result[repr.ID], absURL.String())
               }
            } else if segTemp.Media != "" {
               // Case 3: SegmentTemplate mode
               processSegmentTemplate(segBase, segTemp, repr.ID, mpd.MediaPres, result)
            }
         }
      }
   }

   jsonData, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fail(fmt.Sprintf("Error generating JSON: %v", err))
   }
   fmt.Println(string(jsonData))
}

// getSegmentInfo returns the most specific segment information
func getSegmentInfo(adapt AdaptationSet, repr Representation) (SegmentTemplate, SegmentList) {
   // Prefer representation-level segment info
   segTemp := repr.SegmentTemp
   segList := repr.SegmentList

   // Fall back to adaptation-level if not specified at representation level
   if segTemp.Media == "" && len(segList.Segments) == 0 {
      segTemp = adapt.SegmentTemp
      segList = adapt.SegmentList
   }

   return segTemp, segList
}

// processInitSegment handles initialization segment from either SegmentTemplate or SegmentList
func processInitSegment(base *url.URL, segTemp SegmentTemplate, segList SegmentList, reprID string, result map[string][]string) {
   // Try SegmentTemplate first
   if segTemp.Init != "" {
      initURL := resolveTemplate(segTemp.Init, reprID, 0, 0, 0)
      absURL := resolveURL(base, initURL)
      result[reprID] = append(result[reprID], absURL.String())
      return
   }

   // Try SegmentList if no SegmentTemplate init
   if segList.InitSegment.Source != "" {
      absURL := resolveURL(base, segList.InitSegment.Source)
      result[reprID] = append(result[reprID], absURL.String())
   }
}

// processSegmentTemplate handles SegmentTemplate media segments
func processSegmentTemplate(base *url.URL, segTemp SegmentTemplate, reprID string, mediaPres string, result map[string][]string) {
   if len(segTemp.SegmentTiming.S) > 0 {
      // SegmentTimeline mode
      timeValue := segTemp.SegmentTiming.S[0].T
      for _, s := range segTemp.SegmentTiming.S {
         repeat := s.R
         if repeat < 0 {
            repeat = 0
         }
         for i := 0; i <= repeat; i++ {
            mediaURL := resolveTemplate(segTemp.Media, reprID, 0, timeValue, segTemp.Timescale)
            absURL := resolveURL(base, mediaURL)
            result[reprID] = append(result[reprID], absURL.String())
            timeValue += s.D
         }
      }
   } else if segTemp.Duration > 0 {
      // Simple duration mode with startNumber/endNumber
      segmentCount := calculateSegmentCount(segTemp, mediaPres)
      for i := 0; i < segmentCount; i++ {
         segmentNumber := segTemp.StartNumber + i
         if segTemp.EndNumber > 0 && segmentNumber > segTemp.EndNumber {
            break
         }
         mediaURL := resolveTemplate(segTemp.Media, reprID, segmentNumber, 0, segTemp.Timescale)
         absURL := resolveURL(base, mediaURL)
         result[reprID] = append(result[reprID], absURL.String())
      }
   }
}

// calculateSegmentCount determines the number of segments
func calculateSegmentCount(segTemp SegmentTemplate, mediaPres string) int {
   if segTemp.EndNumber > 0 {
      return segTemp.EndNumber - segTemp.StartNumber + 1
   }

   mediaDuration, err := parseDuration(mediaPres)
   if err != nil {
      warn(fmt.Sprintf("Error parsing duration: %v", err))
      mediaDuration = 600 // Default fallback
   }
   return int(
      float64(mediaDuration) / (float64(segTemp.Duration) / float64(segTemp.Timescale)),
   )

}

// resolveURL strictly uses ResolveReference with no additional logic
func resolveURL(base *url.URL, ref string) *url.URL {
   if ref == "" {
      return base
   }

   refURL, err := url.Parse(ref)
   if err != nil {
      return base
   }

   return base.ResolveReference(refURL)
}

// resolveTemplate replaces template variables in URL patterns
func resolveTemplate(template, representationID string, number, time, timescale int) string {
   result := template

   // Replace RepresentationID
   result = strings.ReplaceAll(result, "$RepresentationID$", representationID)

   // Replace Number variables
   if strings.Contains(result, "$Number") {
      numStr := strconv.Itoa(number)
      if idx := strings.Index(result, "$Number%0"); idx != -1 {
         endIdx := strings.Index(result[idx:], "d$")
         if endIdx != -1 {
            padStr := result[idx+8 : idx+endIdx]
            padLen, err := strconv.Atoi(padStr)
            if err == nil {
               numStr = fmt.Sprintf("%0*d", padLen, number)
            }
            result = result[:idx] + numStr + result[idx+endIdx+2:]
         }
      } else {
         result = strings.ReplaceAll(result, "$Number$", numStr)
      }
   }

   // Replace Time variables
   if strings.Contains(result, "$Time") && time > 0 {
      timeStr := strconv.Itoa(time)
      if idx := strings.Index(result, "$Time%0"); idx != -1 {
         endIdx := strings.Index(result[idx:], "d$")
         if endIdx != -1 {
            padStr := result[idx+7 : idx+endIdx]
            padLen, err := strconv.Atoi(padStr)
            if err == nil {
               timeStr = fmt.Sprintf("%0*d", padLen, time)
            }
            result = result[:idx] + timeStr + result[idx+endIdx+2:]
         }
      } else {
         result = strings.ReplaceAll(result, "$Time$", timeStr)
      }
   }

   return result
}

// parseDuration converts ISO 8601 duration to seconds
func parseDuration(duration string) (int, error) {
   if !strings.HasPrefix(duration, "PT") {
      return 0, fmt.Errorf("invalid duration format")
   }

   duration = strings.TrimPrefix(duration, "PT")
   var totalSeconds int

   // Parse hours
   if hIdx := strings.Index(duration, "H"); hIdx != -1 {
      h, err := strconv.Atoi(duration[:hIdx])
      if err != nil {
         return 0, err
      }
      totalSeconds += h * 3600
      duration = duration[hIdx+1:]
   }

   // Parse minutes
   if mIdx := strings.Index(duration, "M"); mIdx != -1 {
      m, err := strconv.Atoi(duration[:mIdx])
      if err != nil {
         return 0, err
      }
      totalSeconds += m * 60
      duration = duration[mIdx+1:]
   }

   // Parse seconds
   if sIdx := strings.Index(duration, "S"); sIdx != -1 {
      s, err := strconv.Atoi(duration[:sIdx])
      if err != nil {
         return 0, err
      }
      totalSeconds += s
   }

   return totalSeconds, nil
}

func fail(msg string) {
   fmt.Fprintln(os.Stderr, msg)
   os.Exit(1)
}

func warn(msg string) {
   fmt.Fprintln(os.Stderr, "Warning:", msg)
}
