package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "net/url"
   "os"
   "strconv"
   "strings"
)

// MPD structures
type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL []string `xml:"BaseURL"`
   Period  []Period `xml:"Period"`
}

type Period struct {
   Duration      string          `xml:"duration,attr"`
   BaseURL       []string        `xml:"BaseURL"`
   AdaptationSet []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         []string         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representation  []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         []string         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   StartNumber     *int             `xml:"startNumber,attr"`
   EndNumber       *int             `xml:"endNumber,attr"`
   Duration        *int             `xml:"duration,attr"`
   Timescale       *int             `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T *int64 `xml:"t,attr"`
   D int64  `xml:"d,attr"`
   R *int   `xml:"r,attr"`
}

type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURL     []SegmentURL    `xml:"SegmentURL"`
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
   data, err := os.ReadFile(mpdPath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
      os.Exit(1)
   }

   // Parse XML
   var mpd MPD
   err = xml.Unmarshal(data, &mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing XML: %v\n", err)
      os.Exit(1)
   }

   // Process MPD
   result := make(map[string][]string)
   baseURL := "http://test.test/test.mpd"

   // Process each period
   for _, period := range mpd.Period {
      periodBaseURL := resolveBaseURLs(baseURL, append(mpd.BaseURL, period.BaseURL...))
      periodDuration := parseDuration(period.Duration)

      for _, adaptationSet := range period.AdaptationSet {
         asBaseURL := resolveBaseURLs(periodBaseURL, adaptationSet.BaseURL)

         for _, representation := range adaptationSet.Representation {
            repBaseURL := resolveBaseURLs(asBaseURL, representation.BaseURL)

            // Get segments for this representation
            segments := getSegments(representation, adaptationSet.SegmentTemplate, repBaseURL, periodDuration)

            // Append to existing segments for this representation ID
            if existing, ok := result[representation.ID]; ok {
               result[representation.ID] = append(existing, segments...)
            } else {
               result[representation.ID] = segments
            }
         }
      }
   }

   // Output JSON
   output, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(output))
}

func resolveBaseURLs(base string, urls []string) string {
   result := base
   for _, url := range urls {
      result = resolveURL(result, url)
   }
   return result
}

func resolveURL(base, ref string) string {
   baseURL, err := url.Parse(base)
   if err != nil {
      return ref
   }

   refURL, err := url.Parse(ref)
   if err != nil {
      return ref
   }

   resolved := baseURL.ResolveReference(refURL)
   return resolved.String()
}

func parseDuration(duration string) float64 {
   if duration == "" {
      return 0
   }

   // Parse ISO 8601 duration (e.g., PT634.566S or PT10M30S)
   if !strings.HasPrefix(duration, "PT") {
      return 0
   }

   duration = duration[2:] // Remove "PT"
   totalSeconds := 0.0

   // Handle hours
   if idx := strings.Index(duration, "H"); idx != -1 {
      if hours, err := strconv.ParseFloat(duration[:idx], 64); err == nil {
         totalSeconds += hours * 3600
      }
      duration = duration[idx+1:]
   }

   // Handle minutes
   if idx := strings.Index(duration, "M"); idx != -1 {
      if minutes, err := strconv.ParseFloat(duration[:idx], 64); err == nil {
         totalSeconds += minutes * 60
      }
      duration = duration[idx+1:]
   }

   // Handle seconds
   if idx := strings.Index(duration, "S"); idx != -1 {
      if seconds, err := strconv.ParseFloat(duration[:idx], 64); err == nil {
         totalSeconds += seconds
      }
   }

   return totalSeconds
}

func getSegments(rep Representation, asTemplate *SegmentTemplate, baseURL string, periodDuration float64) []string {
   var segments []string

   // Determine which template to use
   template := rep.SegmentTemplate
   if template == nil {
      template = asTemplate
   }

   // Handle SegmentList
   if rep.SegmentList != nil {
      if rep.SegmentList.Initialization != nil && rep.SegmentList.Initialization.SourceURL != "" {
         segments = append(segments, resolveURL(baseURL, rep.SegmentList.Initialization.SourceURL))
      }
      for _, seg := range rep.SegmentList.SegmentURL {
         if seg.Media != "" {
            segments = append(segments, resolveURL(baseURL, seg.Media))
         }
      }
      return segments
   }

   // Handle SegmentTemplate
   if template != nil {
      // Add initialization segment if present
      if template.Initialization != "" {
         initURL := substituteVariables(template.Initialization, rep.ID, 0, 0)
         segments = append(segments, resolveURL(baseURL, initURL))
      }

      // Generate segment URLs
      if template.SegmentTimeline != nil {
         // Timeline-based segments
         segments = append(segments, generateTimelineSegments(template, rep.ID, baseURL)...)
      } else if template.Duration != nil {
         // Duration-based segments
         segments = append(segments, generateDurationSegments(template, rep.ID, baseURL, periodDuration)...)
      }

      return segments
   }

   // If no template or list, just use the base URL
   if len(segments) == 0 {
      segments = append(segments, baseURL)
   }

   return segments
}

func generateTimelineSegments(template *SegmentTemplate, repID, baseURL string) []string {
   var segments []string
   var currentTime int64 = 0
   segmentNumber := 1

   if template.StartNumber != nil {
      segmentNumber = *template.StartNumber
   }

   for _, s := range template.SegmentTimeline.S {
      // Update current time if t is specified
      if s.T != nil {
         currentTime = *s.T
      }

      // Generate segments
      repeatCount := 0
      if s.R != nil {
         repeatCount = *s.R
      }

      for i := 0; i <= repeatCount; i++ {
         // Check end number
         if template.EndNumber != nil && segmentNumber > *template.EndNumber {
            return segments
         }

         segURL := substituteVariables(template.Media, repID, segmentNumber, currentTime)
         segments = append(segments, resolveURL(baseURL, segURL))

         currentTime += s.D
         segmentNumber++
      }
   }

   return segments
}

func generateDurationSegments(template *SegmentTemplate, repID, baseURL string, periodDuration float64) []string {
   var segments []string

   startNumber := 1
   if template.StartNumber != nil {
      startNumber = *template.StartNumber
   }

   // Determine end number
   endNumber := startNumber + 9 // Default to 10 segments
   if template.EndNumber != nil {
      endNumber = *template.EndNumber
   } else if periodDuration > 0 && template.Duration != nil {
      // Calculate segment count from period duration
      timescale := 1
      if template.Timescale != nil {
         timescale = *template.Timescale
      }
      segmentCount := int((periodDuration * float64(timescale)) / float64(*template.Duration))
      if (periodDuration * float64(timescale)) > float64(segmentCount*(*template.Duration)) {
         segmentCount++ // ceil
      }
      endNumber = startNumber + segmentCount - 1
   }

   // Generate segments
   for i := startNumber; i <= endNumber; i++ {
      time := int64(0)
      if template.Duration != nil && i > startNumber {
         time = int64((i - startNumber) * (*template.Duration))
      }

      segURL := substituteVariables(template.Media, repID, i, time)
      segments = append(segments, resolveURL(baseURL, segURL))
   }

   return segments
}

func substituteVariables(template, repID string, number int, time int64) string {
   result := template

   // Replace $RepresentationID$
   result = strings.ReplaceAll(result, "$RepresentationID$", repID)

   // Replace $Number$ with padding
   if strings.Contains(result, "$Number") {
      // Check for padding format
      start := strings.Index(result, "$Number")
      end := strings.Index(result[start:], "$") + start
      end = strings.Index(result[end+1:], "$") + end + 2

      numberVar := result[start:end]
      if strings.Contains(numberVar, "%") {
         // Extract padding format
         formatStart := strings.Index(numberVar, "%")
         formatEnd := strings.LastIndex(numberVar, "d")
         if formatStart != -1 && formatEnd != -1 {
            format := numberVar[formatStart : formatEnd+1]
            paddedNumber := fmt.Sprintf(format, number)
            result = strings.Replace(result, numberVar, paddedNumber, 1)
         }
      } else {
         result = strings.Replace(result, "$Number$", strconv.Itoa(number), 1)
      }
   }

   // Replace $Time$ with padding
   if strings.Contains(result, "$Time") {
      // Check for padding format
      start := strings.Index(result, "$Time")
      end := strings.Index(result[start:], "$") + start
      end = strings.Index(result[end+1:], "$") + end + 2

      timeVar := result[start:end]
      if strings.Contains(timeVar, "%") {
         // Extract padding format
         formatStart := strings.Index(timeVar, "%")
         formatEnd := strings.LastIndex(timeVar, "d")
         if formatStart != -1 && formatEnd != -1 {
            format := timeVar[formatStart : formatEnd+1]
            paddedTime := fmt.Sprintf(format, time)
            result = strings.Replace(result, timeVar, paddedTime, 1)
         }
      } else {
         result = strings.Replace(result, "$Time$", strconv.FormatInt(time, 10), 1)
      }
   }

   return result
}
