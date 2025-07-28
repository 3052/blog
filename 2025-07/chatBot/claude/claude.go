package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io"
   "net/http"
   "net/url"
   "os"
   "strconv"
   "strings"
   "time"
)

// MPD structure definitions
type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   BaseURL                   []string `xml:"BaseURL"`
   Periods                   []Period `xml:"Period"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
}

type Period struct {
   XMLName        xml.Name        `xml:"Period"`
   ID             string          `xml:"id,attr"`
   BaseURL        []string        `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   XMLName         xml.Name         `xml:"AdaptationSet"`
   ID              string           `xml:"id,attr"`
   BaseURL         []string         `xml:"BaseURL"`
   Representations []Representation `xml:"Representation"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type Representation struct {
   XMLName         xml.Name         `xml:"Representation"`
   ID              string           `xml:"id,attr"`
   BaseURL         []string         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   Duration        int              `xml:"duration,attr"`
   Timescale       int              `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []SegmentTimelineEntry `xml:"S"`
}

type SegmentTimelineEntry struct {
   T int `xml:"t,attr"`
   D int `xml:"d,attr"`
   R int `xml:"r,attr"`
}

type SegmentList struct {
   SegmentURLs []SegmentURL `xml:"SegmentURL"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

// Result structure for JSON output
type Result map[string][]string

func main() {
   if len(os.Args) < 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path_or_url>")
      os.Exit(1)
   }

   mpdPath := os.Args[1]
   baseURL := "http://test.test/test.mpd"

   // Parse the MPD file
   mpd, err := parseMPD(mpdPath)
   if err != nil {
      fmt.Printf("Error parsing MPD: %v\n", err)
      os.Exit(1)
   }

   // Extract segment URLs
   result := extractSegmentURLs(mpd, baseURL)

   // Output as JSON
   jsonOutput, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Printf("Error marshaling JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonOutput))
}

func parseMPD(mpdPath string) (*MPD, error) {
   var data []byte
   var err error

   // Check if it's a URL or file path
   if strings.HasPrefix(mpdPath, "http://") || strings.HasPrefix(mpdPath, "https://") {
      resp, err := http.Get(mpdPath)
      if err != nil {
         return nil, fmt.Errorf("failed to fetch MPD from URL: %v", err)
      }
      defer resp.Body.Close()

      data, err = io.ReadAll(resp.Body)
      if err != nil {
         return nil, fmt.Errorf("failed to read MPD response: %v", err)
      }
   } else {
      data, err = os.ReadFile(mpdPath)
      if err != nil {
         return nil, fmt.Errorf("failed to read MPD file: %v", err)
      }
   }

   var mpd MPD
   err = xml.Unmarshal(data, &mpd)
   if err != nil {
      return nil, fmt.Errorf("failed to unmarshal MPD XML: %v", err)
   }

   return &mpd, nil
}

func extractSegmentURLs(mpd *MPD, baseURL string) Result {
   result := make(Result)

   // Parse the base URL
   parsedBaseURL, err := url.Parse(baseURL)
   if err != nil {
      fmt.Printf("Error parsing base URL: %v\n", err)
      return result
   }

   // Resolve MPD-level BaseURL
   mpdBaseURL := resolveBaseURLs(parsedBaseURL, mpd.BaseURL)

   for _, period := range mpd.Periods {
      // Resolve Period-level BaseURL
      periodBaseURL := resolveBaseURLs(mpdBaseURL, period.BaseURL)

      for _, adaptationSet := range period.AdaptationSets {
         // Resolve AdaptationSet-level BaseURL
         asBaseURL := resolveBaseURLs(periodBaseURL, adaptationSet.BaseURL)

         for _, representation := range adaptationSet.Representations {
            // Resolve Representation-level BaseURL
            repBaseURL := resolveBaseURLs(asBaseURL, representation.BaseURL)

            // Extract segments for this representation
            segments := extractRepresentationSegments(representation, adaptationSet, repBaseURL)
            if len(segments) > 0 {
               result[representation.ID] = segments
            }
         }
      }
   }

   return result
}

func resolveBaseURLs(baseURL *url.URL, baseURLs []string) *url.URL {
   currentURL := baseURL

   for _, baseURLStr := range baseURLs {
      if baseURLStr == "" {
         continue
      }

      // Parse the BaseURL
      newBaseURL, err := url.Parse(baseURLStr)
      if err != nil {
         continue
      }

      // Resolve against current URL
      currentURL = currentURL.ResolveReference(newBaseURL)
   }

   return currentURL
}

func extractRepresentationSegments(rep Representation, adaptationSet AdaptationSet, baseURL *url.URL) []string {
   var segments []string

   // Check for SegmentList first
   if rep.SegmentList != nil {
      for _, segmentURL := range rep.SegmentList.SegmentURLs {
         if segmentURL.Media != "" {
            resolvedURL := resolveSegmentURL(baseURL, segmentURL.Media)
            segments = append(segments, resolvedURL)
         }
      }
      return segments
   }

   // Check for SegmentTemplate at representation level
   segmentTemplate := rep.SegmentTemplate
   if segmentTemplate == nil {
      // Fall back to AdaptationSet level SegmentTemplate
      segmentTemplate = adaptationSet.SegmentTemplate
   }

   if segmentTemplate != nil {
      // Add initialization URL as first segment if present
      if segmentTemplate.Initialization != "" {
         initURL := segmentTemplate.Initialization
         initURL = strings.ReplaceAll(initURL, "$RepresentationID$", rep.ID)
         resolvedInitURL := resolveSegmentURL(baseURL, initURL)
         segments = append(segments, resolvedInitURL)
      }

      // Generate and append media segments
      mediaSegments := generateSegmentsFromTemplate(segmentTemplate, rep.ID, baseURL)
      segments = append(segments, mediaSegments...)
   }

   return segments
}

func generateSegmentsFromTemplate(template *SegmentTemplate, repID string, baseURL *url.URL) []string {
   var segments []string

   if template.Media == "" {
      return segments
   }

   // Default values
   startNumber := template.StartNumber
   if startNumber == 0 {
      startNumber = 1
   }

   // Generate segments based on SegmentTimeline if available
   if template.SegmentTimeline != nil {
      segmentNumber := startNumber
      currentTime := 0

      for _, s := range template.SegmentTimeline.S {
         // If 't' attribute is present, use it as the starting time for this S element
         if s.T > 0 {
            currentTime = s.T
         }

         repeat := s.R
         if repeat < 0 {
            repeat = 0 // Handle infinite repeat by limiting to 0 for safety
         }

         for i := 0; i <= repeat; i++ {
            mediaURL := template.Media
            mediaURL = strings.ReplaceAll(mediaURL, "$RepresentationID$", repID)
            mediaURL = strings.ReplaceAll(mediaURL, "$Number$", strconv.Itoa(segmentNumber))
            mediaURL = strings.ReplaceAll(mediaURL, "$Time$", strconv.Itoa(currentTime))

            resolvedURL := resolveSegmentURL(baseURL, mediaURL)
            segments = append(segments, resolvedURL)

            segmentNumber++
            currentTime += s.D // Add duration to current time for next segment
         }
      }
   } else {
      // Generate segments based on startNumber and endNumber
      segmentCount := 10 // Default fallback

      if template.EndNumber > 0 {
         // Use endNumber if specified
         segmentCount = template.EndNumber - startNumber + 1
      } else if template.Duration > 0 && template.Timescale > 0 {
         // This is a simplified calculation - in practice, you'd need the total duration
         segmentDurationSeconds := float64(template.Duration) / float64(template.Timescale)
         totalDurationSeconds := 3600.0 // Assume 1 hour as default
         segmentCount = int(totalDurationSeconds / segmentDurationSeconds)
         if segmentCount > 100 {
            segmentCount = 100 // Limit for safety
         }
      }

      for i := 0; i < segmentCount; i++ {
         segmentNumber := startNumber + i
         mediaURL := template.Media
         mediaURL = strings.ReplaceAll(mediaURL, "$RepresentationID$", repID)
         mediaURL = strings.ReplaceAll(mediaURL, "$Number$", strconv.Itoa(segmentNumber))

         resolvedURL := resolveSegmentURL(baseURL, mediaURL)
         segments = append(segments, resolvedURL)
      }
   }

   return segments
}

func resolveSegmentURL(baseURL *url.URL, segmentPath string) string {
   // Parse the segment path
   segmentURL, err := url.Parse(segmentPath)
   if err != nil {
      return segmentPath
   }

   // If it's already absolute, return as is
   if segmentURL.IsAbs() {
      return segmentPath
   }

   // Resolve against base URL
   resolvedURL := baseURL.ResolveReference(segmentURL)
   return resolvedURL.String()
}

// Helper function to parse ISO 8601 duration (simplified)
func parseDuration(duration string) time.Duration {
   // This is a simplified parser - in production, use a proper ISO 8601 parser
   if strings.HasPrefix(duration, "PT") {
      duration = strings.TrimPrefix(duration, "PT")
      if strings.HasSuffix(duration, "S") {
         duration = strings.TrimSuffix(duration, "S")
         if seconds, err := strconv.ParseFloat(duration, 64); err == nil {
            return time.Duration(seconds * float64(time.Second))
         }
      }
   }
   return 0
}
