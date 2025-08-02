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
   "strings"
)

// MPD structure definitions
type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   Duration       string          `xml:"duration,attr"`
   BaseURL        string          `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentBase struct {
   Initialization *Initialization `xml:"Initialization"`
}

type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type SegmentTemplate struct {
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   StartNumber     *int             `xml:"startNumber,attr"`
   EndNumber       *int             `xml:"endNumber,attr"`
   Timescale       *uint64          `xml:"timescale,attr"`
   Duration        *uint64          `xml:"duration,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T *uint64 `xml:"t,attr"`
   D uint64  `xml:"d,attr"`
   R *int    `xml:"r,attr"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }

   mpdPath := os.Args[1]

   // Read MPD file
   data, err := ioutil.ReadFile(mpdPath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
      os.Exit(1)
   }

   // Parse MPD
   var mpd MPD
   err = xml.Unmarshal(data, &mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD: %v\n", err)
      os.Exit(1)
   }

   // Extract segments
   result := extractSegments(&mpd)

   // Output JSON
   jsonData, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonData))
}

func extractSegments(mpd *MPD) map[string][]string {
   result := make(map[string][]string)

   // Starting base URL
   baseURL, _ := url.Parse("http://test.test/test.mpd")

   // Resolve MPD-level BaseURL
   if mpd.BaseURL != "" {
      if resolved, err := baseURL.Parse(mpd.BaseURL); err == nil {
         baseURL = resolved
      }
   }

   // Process each period
   for _, period := range mpd.Periods {
      periodBaseURL := baseURL

      // Resolve Period-level BaseURL
      if period.BaseURL != "" {
         if resolved, err := periodBaseURL.Parse(period.BaseURL); err == nil {
            periodBaseURL = resolved
         }
      }

      // Process each adaptation set
      for _, adaptationSet := range period.AdaptationSets {
         adaptationSetBaseURL := periodBaseURL

         // Resolve AdaptationSet-level BaseURL
         if adaptationSet.BaseURL != "" {
            if resolved, err := adaptationSetBaseURL.Parse(adaptationSet.BaseURL); err == nil {
               adaptationSetBaseURL = resolved
            }
         }

         // Process each representation
         for _, representation := range adaptationSet.Representations {
            representationBaseURL := adaptationSetBaseURL

            // Resolve Representation-level BaseURL
            if representation.BaseURL != "" {
               if resolved, err := representationBaseURL.Parse(representation.BaseURL); err == nil {
                  representationBaseURL = resolved
               }
            }

            segments := extractRepresentationSegments(&representation, &adaptationSet, &period, representationBaseURL)
            if len(segments) > 0 {
               result[representation.ID] = append(result[representation.ID], segments...)
            }
         }
      }
   }

   return result
}

func extractRepresentationSegments(rep *Representation, adaptationSet *AdaptationSet, period *Period, baseURL *url.URL) []string {
   var segments []string

   // Handle different segment types
   switch {
   case rep.SegmentTemplate != nil:
      // Use representation-level SegmentTemplate
      segments = extractSegmentTemplateURLs(rep.SegmentTemplate, rep.ID, period, baseURL)

   case adaptationSet.SegmentTemplate != nil:
      // Use adaptation set-level SegmentTemplate
      segments = extractSegmentTemplateURLs(adaptationSet.SegmentTemplate, rep.ID, period, baseURL)

   case rep.SegmentList != nil:
      // Handle SegmentList
      segments = extractSegmentListURLs(rep.SegmentList, baseURL)

   case rep.SegmentBase != nil:
      // Handle SegmentBase
      segments = extractSegmentBaseURLs(rep.SegmentBase, baseURL)

   case rep.BaseURL != "":
      // BaseURL-only representation - treat as single segment
      // The baseURL parameter already has the Representation BaseURL resolved
      segments = []string{baseURL.String()}

   default:
      // No segments found
      return nil
   }

   return segments
}

func extractSegmentTemplateURLs(template *SegmentTemplate, repID string, period *Period, baseURL *url.URL) []string {
   var segments []string

   // Add initialization segment if present
   if template.Initialization != "" {
      initURL := substituteTemplate(template.Initialization, repID, 0, 0)
      if resolved, err := baseURL.Parse(initURL); err == nil {
         segments = append(segments, resolved.String())
      }
   }

   // Generate media segments
   if template.SegmentTimeline != nil {
      // Use SegmentTimeline for precise segment generation
      segments = append(segments, generateTimelineSegments(template, repID, baseURL)...)
   } else {
      // Generate segments from startNumber to endNumber
      startNum := 1
      if template.StartNumber != nil {
         startNum = *template.StartNumber
      }

      var endNum int
      if template.EndNumber != nil {
         endNum = *template.EndNumber
      } else if template.Duration != nil && period.Duration != "" {
         // Calculate endNumber using duration formula
         endNum = calculateEndNumberFromDuration(template, period, startNum)
      } else {
         endNum = startNum + 10 // Default if no endNumber specified
      }

      for i := startNum; i <= endNum; i++ {
         mediaURL := substituteTemplate(template.Media, repID, i, 0)
         if resolved, err := baseURL.Parse(mediaURL); err == nil {
            segments = append(segments, resolved.String())
         }
      }
   }

   return segments
}

func generateTimelineSegments(template *SegmentTemplate, repID string, baseURL *url.URL) []string {
   var segments []string

   startNum := 1
   if template.StartNumber != nil {
      startNum = *template.StartNumber
   }

   segmentNumber := startNum
   currentTime := uint64(0)

   for _, s := range template.SegmentTimeline.S {
      // Handle absolute timestamp
      if s.T != nil {
         currentTime = *s.T
      }

      // Calculate repeat count
      repeatCount := 1
      if s.R != nil {
         repeatCount = *s.R + 1
      }

      // Generate segments for this S element
      for i := 0; i < repeatCount; i++ {
         // Use raw currentTime for $Time$ template substitution
         mediaURL := substituteTemplate(template.Media, repID, segmentNumber, currentTime)
         if resolved, err := baseURL.Parse(mediaURL); err == nil {
            segments = append(segments, resolved.String())
         }

         segmentNumber++
         currentTime += s.D
      }
   }

   return segments
}

func extractSegmentListURLs(segmentList *SegmentList, baseURL *url.URL) []string {
   var segments []string

   // Add initialization segment if present
   if segmentList.Initialization != nil && segmentList.Initialization.SourceURL != "" {
      if resolved, err := baseURL.Parse(segmentList.Initialization.SourceURL); err == nil {
         segments = append(segments, resolved.String())
      }
   }

   // Add segment URLs
   for _, segmentURL := range segmentList.SegmentURLs {
      if resolved, err := baseURL.Parse(segmentURL.Media); err == nil {
         segments = append(segments, resolved.String())
      }
   }

   return segments
}

func extractSegmentBaseURLs(segmentBase *SegmentBase, baseURL *url.URL) []string {
   var segments []string

   // Add initialization segment if present
   if segmentBase.Initialization != nil && segmentBase.Initialization.SourceURL != "" {
      if resolved, err := baseURL.Parse(segmentBase.Initialization.SourceURL); err == nil {
         segments = append(segments, resolved.String())
      }
   }

   // For SegmentBase, the main content is usually the base URL itself
   segments = append(segments, baseURL.String())

   return segments
}

func substituteTemplate(template, repID string, number int, time uint64) string {
   result := template

   // Replace $RepresentationID$
   result = strings.ReplaceAll(result, "$RepresentationID$", repID)

   // Replace $Number$ patterns (with optional formatting)
   result = replaceNumberTemplate(result, number)

   // Replace $Time$
   result = strings.ReplaceAll(result, "$Time$", strconv.FormatUint(time, 10))

   return result
}

func replaceNumberTemplate(template string, number int) string {
   // Handle $Number$ without formatting
   template = strings.ReplaceAll(template, "$Number$", strconv.Itoa(number))

   // Handle $Number%XXd$ formatting patterns
   for {
      start := strings.Index(template, "$Number%")
      if start == -1 {
         break
      }

      // Find the closing $ (skip the opening $)
      end := strings.Index(template[start+1:], "$")
      if end == -1 {
         break
      }
      end += start + 1

      // Extract format pattern (between % and final $)
      formatPattern := template[start+8 : end] // Skip "$Number%"
      if strings.HasSuffix(formatPattern, "d") {
         // Parse width (e.g., "05d" -> width 5, zero-padded)
         widthStr := formatPattern[:len(formatPattern)-1]
         if width, err := strconv.Atoi(strings.TrimLeft(widthStr, "0")); err == nil {
            var formatted string
            if strings.HasPrefix(widthStr, "0") {
               // Zero-padded
               formatted = fmt.Sprintf("%0*d", width, number)
            } else {
               // Space-padded
               formatted = fmt.Sprintf("%*d", width, number)
            }

            // Replace the pattern
            pattern := template[start : end+1]
            template = strings.ReplaceAll(template, pattern, formatted)
         }
      }
   }

   return template
}

func calculateEndNumberFromDuration(template *SegmentTemplate, period *Period, startNum int) int {
   // Parse period duration (ISO 8601 format)
   periodDurationSeconds, err := parseDuration(period.Duration)
   if err != nil {
      return startNum + 10 // Fallback default
   }

   // Get timescale, default to 1
   timescale := uint64(1)
   if template.Timescale != nil {
      timescale = *template.Timescale
   }

   // Calculate number of segments: ceil(PeriodDurationInSeconds * timescale / duration)
   numSegments := math.Ceil(periodDurationSeconds * float64(timescale) / float64(*template.Duration))

   return startNum + int(numSegments) - 1
}

func parseDuration(duration string) (float64, error) {
   // Simple ISO 8601 duration parser for PT format (e.g., PT30S, PT1M30S, PT1H2M3S)
   if !strings.HasPrefix(duration, "PT") {
      return 0, fmt.Errorf("invalid duration format")
   }

   duration = duration[2:] // Remove "PT"
   var totalSeconds float64

   // Parse hours
   re := regexp.MustCompile(`(\d+(?:\.\d+)?)H`)
   if matches := re.FindStringSubmatch(duration); len(matches) > 1 {
      if hours, err := strconv.ParseFloat(matches[1], 64); err == nil {
         totalSeconds += hours * 3600
      }
      duration = re.ReplaceAllString(duration, "")
   }

   // Parse minutes
   re = regexp.MustCompile(`(\d+(?:\.\d+)?)M`)
   if matches := re.FindStringSubmatch(duration); len(matches) > 1 {
      if minutes, err := strconv.ParseFloat(matches[1], 64); err == nil {
         totalSeconds += minutes * 60
      }
      duration = re.ReplaceAllString(duration, "")
   }

   // Parse seconds
   re = regexp.MustCompile(`(\d+(?:\.\d+)?)S`)
   if matches := re.FindStringSubmatch(duration); len(matches) > 1 {
      if seconds, err := strconv.ParseFloat(matches[1], 64); err == nil {
         totalSeconds += seconds
      }
   }

   return totalSeconds, nil
}
