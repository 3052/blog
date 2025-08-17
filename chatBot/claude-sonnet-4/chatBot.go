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

// MPD represents the root Media Presentation Description
type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

// Period represents a period in the MPD
type Period struct {
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
   BaseURL        string          `xml:"BaseURL"`
   Duration       string          `xml:"duration,attr"`
}

// AdaptationSet represents an adaptation set
type AdaptationSet struct {
   Representations []Representation `xml:"Representation"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   BaseURL         string           `xml:"BaseURL"`
}

// Representation represents a representation
type Representation struct {
   ID              string           `xml:"id,attr"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   BaseURL         string           `xml:"BaseURL"`
}

// SegmentTemplate represents segment template
type SegmentTemplate struct {
   Media           string          `xml:"media,attr"`
   Initialization  string          `xml:"initialization,attr"`
   StartNumber     *int            `xml:"startNumber,attr"`
   EndNumber       int             `xml:"endNumber,attr"`
   Duration        int             `xml:"duration,attr"`
   Timescale       int             `xml:"timescale,attr"`
   SegmentTimeline SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline represents segment timeline
type SegmentTimeline struct {
   S []S `xml:"S"`
}

// S represents a segment in timeline
type S struct {
   T int `xml:"t,attr"`
   D int `xml:"d,attr"`
   R int `xml:"r,attr"`
}

// SegmentList represents segment list
type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

// Initialization represents initialization segment
type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

// SegmentURL represents a segment URL
type SegmentURL struct {
   Media string `xml:"media,attr"`
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }

   mpdFilePath := os.Args[1]
   baseURL := "http://test.test/test.mpd"

   // Read the MPD file
   data, err := ioutil.ReadFile(mpdFilePath)
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

   // Process and generate segment URLs
   result := make(map[string][]string)

   // Resolve MPD BaseURL against the initial MPD URL if present
   mpdBaseURL := baseURL
   if mpd.BaseURL != "" {
      mpdBaseURL = resolveURL(baseURL, mpd.BaseURL)
   }

   for _, period := range mpd.Periods {
      for _, adaptationSet := range period.AdaptationSets {
         for _, representation := range adaptationSet.Representations {
            segmentURLs := generateSegmentURLs(representation, adaptationSet, period, mpdBaseURL)
            if len(segmentURLs) > 0 {
               // Append to existing representation ID if it already exists
               result[representation.ID] = append(result[representation.ID], segmentURLs...)
            }
         }
      }
   }

   // Output JSON
   jsonData, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonData))
}

func generateSegmentURLs(rep Representation, adaptationSet AdaptationSet, period Period, baseURL string) []string {
   var segmentURLs []string

   // Determine which segment template or list to use
   var segmentTemplate *SegmentTemplate
   var segmentList *SegmentList

   // Priority: Representation > AdaptationSet
   if rep.SegmentTemplate != nil {
      segmentTemplate = rep.SegmentTemplate
   } else if adaptationSet.SegmentTemplate != nil {
      segmentTemplate = adaptationSet.SegmentTemplate
   }

   if rep.SegmentList != nil {
      segmentList = rep.SegmentList
   } else if adaptationSet.SegmentList != nil {
      segmentList = adaptationSet.SegmentList
   }

   // Build hierarchical base URL: MPD BaseURL -> Period BaseURL -> AdaptationSet BaseURL -> Representation BaseURL
   resolvedBaseURL := baseURL

   if period.BaseURL != "" {
      resolvedBaseURL = resolveURL(resolvedBaseURL, period.BaseURL)
   }

   if adaptationSet.BaseURL != "" {
      resolvedBaseURL = resolveURL(resolvedBaseURL, adaptationSet.BaseURL)
   }

   if rep.BaseURL != "" {
      resolvedBaseURL = resolveURL(resolvedBaseURL, rep.BaseURL)
   }

   // If representation has only BaseURL and no segment information, use resolved BaseURL directly
   if segmentTemplate == nil && segmentList == nil && rep.BaseURL != "" {
      segmentURLs = append(segmentURLs, resolvedBaseURL)
      return segmentURLs
   }

   if segmentTemplate != nil {
      segmentURLs = generateFromTemplate(segmentTemplate, rep.ID, resolvedBaseURL, period)
   } else if segmentList != nil {
      segmentURLs = generateFromList(segmentList, resolvedBaseURL)
   }

   return segmentURLs
}

func generateFromTemplate(template *SegmentTemplate, repID, baseURL string, period Period) []string {
   var segmentURLs []string

   // Add initialization segment if present
   if template.Initialization != "" {
      initURL := substituteTemplate(template.Initialization, repID, 0, 0)
      segmentURLs = append(segmentURLs, resolveURL(baseURL, initURL))
   }

   // Generate media segments
   if template.Media != "" {
      // Determine start number: default to 1 if missing, otherwise use specified value (including 0)
      startNumber := 1
      if template.StartNumber != nil {
         startNumber = *template.StartNumber
      }

      // Priority: SegmentTimeline > endNumber > duration calculation
      if len(template.SegmentTimeline.S) > 0 {
         // Use SegmentTimeline
         segmentNumber := startNumber
         currentTime := 0

         for _, s := range template.SegmentTimeline.S {
            // Use explicit time if provided
            if s.T > 0 {
               currentTime = s.T
            }

            repeat := s.R + 1
            if s.R < 0 {
               // Negative repeat means "until end of period"
               // For simplicity, generate a reasonable number
               repeat = 100
            }

            for i := 0; i < repeat; i++ {
               mediaURL := substituteTemplate(template.Media, repID, segmentNumber, currentTime)
               segmentURLs = append(segmentURLs, resolveURL(baseURL, mediaURL))
               segmentNumber++
               currentTime += s.D
            }
         }
      } else if template.EndNumber > 0 {
         // Use endNumber to determine range
         duration := template.Duration
         if duration == 0 {
            // Default duration assumption: 2 seconds in typical timescale (90000 units per second)
            duration = 180000
         }

         for segmentNumber := startNumber; segmentNumber <= template.EndNumber; segmentNumber++ {
            currentTime := (segmentNumber - startNumber) * duration
            mediaURL := substituteTemplate(template.Media, repID, segmentNumber, currentTime)
            segmentURLs = append(segmentURLs, resolveURL(baseURL, mediaURL))
         }
      } else if template.Duration > 0 && period.Duration != "" {
         // Calculate number of segments using period duration, timescale, and segment duration
         timescale := template.Timescale
         if timescale == 0 {
            timescale = 1 // Default timescale to 1 if missing
         }

         periodDurationSeconds := parseDuration(period.Duration)
         if periodDurationSeconds > 0 {
            numSegments := int(math.Ceil(periodDurationSeconds * float64(timescale) / float64(template.Duration)))

            for i := 0; i < numSegments; i++ {
               segmentNumber := startNumber + i
               currentTime := i * template.Duration
               mediaURL := substituteTemplate(template.Media, repID, segmentNumber, currentTime)
               segmentURLs = append(segmentURLs, resolveURL(baseURL, mediaURL))
            }
         } else {
            // Fallback if period duration parsing fails
            generateDefaultSegments(template, repID, baseURL, startNumber, &segmentURLs)
         }
      } else {
         // No timeline, endNumber, or sufficient duration info - generate default segments
         generateDefaultSegments(template, repID, baseURL, startNumber, &segmentURLs)
      }
   }

   return segmentURLs
}

func generateDefaultSegments(template *SegmentTemplate, repID, baseURL string, startNumber int, segmentURLs *[]string) {
   duration := template.Duration
   if duration == 0 {
      // Default duration assumption: 2 seconds in typical timescale (90000 units per second)
      duration = 180000
   }

   for i := 0; i < 100; i++ {
      segmentNumber := startNumber + i
      currentTime := i * duration
      mediaURL := substituteTemplate(template.Media, repID, segmentNumber, currentTime)
      *segmentURLs = append(*segmentURLs, resolveURL(baseURL, mediaURL))
   }
}

func parseDuration(duration string) float64 {
   // Parse ISO 8601 duration format (PT30S, PT1M30S, PT1H30M, etc.)
   if !strings.HasPrefix(duration, "PT") {
      return 0
   }

   // Remove PT prefix
   duration = duration[2:]

   var totalSeconds float64

   // Parse hours
   if strings.Contains(duration, "H") {
      parts := strings.Split(duration, "H")
      if hours, err := strconv.ParseFloat(parts[0], 64); err == nil {
         totalSeconds += hours * 3600
      }
      duration = parts[1]
   }

   // Parse minutes
   if strings.Contains(duration, "M") && !strings.Contains(duration, ".") {
      parts := strings.Split(duration, "M")
      if minutes, err := strconv.ParseFloat(parts[0], 64); err == nil {
         totalSeconds += minutes * 60
      }
      duration = parts[1]
   }

   // Parse seconds
   if strings.Contains(duration, "S") {
      parts := strings.Split(duration, "S")
      if seconds, err := strconv.ParseFloat(parts[0], 64); err == nil {
         totalSeconds += seconds
      }
   }

   return totalSeconds
}

func generateFromList(segmentList *SegmentList, baseURL string) []string {
   var segmentURLs []string

   // Add initialization segment if present
   if segmentList.Initialization != nil && segmentList.Initialization.SourceURL != "" {
      segmentURLs = append(segmentURLs, resolveURL(baseURL, segmentList.Initialization.SourceURL))
   }

   // Add media segments
   for _, segmentURL := range segmentList.SegmentURLs {
      if segmentURL.Media != "" {
         segmentURLs = append(segmentURLs, resolveURL(baseURL, segmentURL.Media))
      }
   }

   return segmentURLs
}

func substituteTemplate(template, repID string, number, time int) string {
   result := template

   // Replace $RepresentationID$
   result = strings.ReplaceAll(result, "$RepresentationID$", repID)

   // Replace $Number$ with proper formatting
   numberRegex := regexp.MustCompile(`\$Number(?:%(\d+)d)?\$`)
   result = numberRegex.ReplaceAllStringFunc(result, func(match string) string {
      matches := numberRegex.FindStringSubmatch(match)
      if len(matches) > 1 && matches[1] != "" {
         padding, _ := strconv.Atoi(matches[1])
         return fmt.Sprintf("%0*d", padding, number)
      }
      return strconv.Itoa(number)
   })

   // Replace $Time$ with proper formatting
   timeRegex := regexp.MustCompile(`\$Time(?:%(\d+)d)?\$`)
   result = timeRegex.ReplaceAllStringFunc(result, func(match string) string {
      matches := timeRegex.FindStringSubmatch(match)
      if len(matches) > 1 && matches[1] != "" {
         padding, _ := strconv.Atoi(matches[1])
         return fmt.Sprintf("%0*d", padding, time)
      }
      return strconv.Itoa(time)
   })

   // Replace other common template variables
   result = strings.ReplaceAll(result, "$Bandwidth$", "1000000")

   return result
}

func resolveURL(base, relative string) string {
   baseURL, err := url.Parse(base)
   if err != nil {
      return relative
   }

   relativeURL, err := url.Parse(relative)
   if err != nil {
      return relative
   }

   return baseURL.ResolveReference(relativeURL).String()
}
