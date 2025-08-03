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

// MPD structure
type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   BaseURL                   string   `xml:"BaseURL"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
   Periods                   []Period `xml:"Period"`
}

type Period struct {
   BaseURL        string          `xml:"BaseURL"`
   Duration       string          `xml:"duration,attr"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   Representations []Representation `xml:"Representation"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
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
   S []S `xml:"S"`
}

type S struct {
   T int `xml:"t,attr"`
   D int `xml:"d,attr"`
   R int `xml:"r,attr"`
}

type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
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

   mpdFilePath := os.Args[1]
   baseURL := "http://test.test/test.mpd"

   // Read MPD file
   xmlData, err := ioutil.ReadFile(mpdFilePath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading MPD file: %v\n", err)
      os.Exit(1)
   }

   // Parse XML
   var mpd MPD
   err = xml.Unmarshal(xmlData, &mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD XML: %v\n", err)
      os.Exit(1)
   }

   // Extract segment URLs for each representation
   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      for _, adaptationSet := range period.AdaptationSets {
         for _, representation := range adaptationSet.Representations {
            segmentURLs := extractSegmentURLs(representation, adaptationSet, period, baseURL, &mpd)

            // Append segments to existing representation or create new entry
            if existing, exists := result[representation.ID]; exists {
               result[representation.ID] = append(existing, segmentURLs...)
            } else {
               result[representation.ID] = segmentURLs
            }
         }
      }
   }

   // Output JSON
   jsonOutput, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonOutput))
}

func extractSegmentURLs(rep Representation, adaptationSet AdaptationSet, period Period, baseURL string, mpd *MPD) []string {
   var segmentURLs []string

   // Determine the base URL for this representation
   repBaseURL := resolveBaseURL(baseURL, mpd.BaseURL, period.BaseURL, adaptationSet.BaseURL, rep.BaseURL)

   // Check for SegmentList first (explicit segment URLs)
   segmentList := rep.SegmentList
   if segmentList == nil {
      segmentList = adaptationSet.SegmentList
   }

   if segmentList != nil {
      // Add initialization segment if it exists
      if segmentList.Initialization != nil && segmentList.Initialization.SourceURL != "" {
         fullInitURL := resolveURL(repBaseURL, segmentList.Initialization.SourceURL)
         segmentURLs = append(segmentURLs, fullInitURL)
      }

      for _, segURL := range segmentList.SegmentURLs {
         fullURL := resolveURL(repBaseURL, segURL.Media)
         segmentURLs = append(segmentURLs, fullURL)
      }
      return segmentURLs
   }

   // Check for SegmentTemplate
   segmentTemplate := rep.SegmentTemplate
   if segmentTemplate == nil {
      segmentTemplate = adaptationSet.SegmentTemplate
   }

   if segmentTemplate != nil {
      segmentURLs = extractFromSegmentTemplate(segmentTemplate, rep.ID, repBaseURL, period, mpd)
      return segmentURLs
   }

   // If no segment information but has BaseURL, use the resolved BaseURL directly
   if rep.BaseURL != "" || adaptationSet.BaseURL != "" || period.BaseURL != "" || mpd.BaseURL != "" {
      segmentURLs = append(segmentURLs, repBaseURL)
      return segmentURLs
   }

   return segmentURLs
}

func extractFromSegmentTemplate(template *SegmentTemplate, repID, baseURL string, period Period, mpd *MPD) []string {
   var segmentURLs []string

   // Add initialization segment if it exists
   if template.Initialization != "" {
      initURL := strings.ReplaceAll(template.Initialization, "$RepresentationID$", repID)
      initURL = replaceNumberVariable(initURL, 0) // Use 0 for initialization segment
      fullInitURL := resolveURL(baseURL, initURL)
      segmentURLs = append(segmentURLs, fullInitURL)
   }

   if template.Media == "" {
      return segmentURLs
   }

   // If there's a SegmentTimeline, use it to determine segment count
   if template.SegmentTimeline != nil {
      segmentNumber := template.StartNumber
      if segmentNumber == 0 {
         segmentNumber = 1
      }

      currentTime := 0
      for _, s := range template.SegmentTimeline.S {
         if s.T != 0 {
            currentTime = s.T
         }

         repeatCount := s.R + 1
         if s.R == 0 {
            repeatCount = 1
         }

         for i := 0; i < repeatCount; i++ {
            mediaURL := strings.ReplaceAll(template.Media, "$RepresentationID$", repID)
            mediaURL = replaceNumberVariable(mediaURL, segmentNumber)
            mediaURL = strings.ReplaceAll(mediaURL, "$Time$", strconv.Itoa(currentTime))

            fullURL := resolveURL(baseURL, mediaURL)
            segmentURLs = append(segmentURLs, fullURL)
            segmentNumber++
            currentTime += s.D
         }
      }
   } else {
      // No SegmentTimeline, use endNumber if available, otherwise calculate from duration/timescale
      startNumber := template.StartNumber
      if startNumber == 0 {
         startNumber = 1
      }

      endNumber := template.EndNumber
      if endNumber == 0 {
         // No endNumber specified, try to calculate from duration and timescale
         if template.Duration > 0 {
            timescale := template.Timescale
            if timescale == 0 {
               timescale = 1 // Default timescale to 1 if missing
            }

            periodDuration := getPeriodDurationInSeconds(period, mpd)
            if periodDuration > 0 {
               segmentCount := int(math.Ceil(periodDuration * float64(timescale) / float64(template.Duration)))
               endNumber = startNumber + segmentCount - 1
            } else {
               // Fallback to 10 segments
               endNumber = startNumber + 9
            }
         } else {
            // No duration, generate 10 segments as fallback
            endNumber = startNumber + 9
         }
      }

      for segmentNumber := startNumber; segmentNumber <= endNumber; segmentNumber++ {
         mediaURL := strings.ReplaceAll(template.Media, "$RepresentationID$", repID)
         mediaURL = replaceNumberVariable(mediaURL, segmentNumber)

         fullURL := resolveURL(baseURL, mediaURL)
         segmentURLs = append(segmentURLs, fullURL)
      }
   }

   return segmentURLs
}

func replaceNumberVariable(template string, number int) string {
   // Handle $Number%08d$ format (zero-padded)
   re := regexp.MustCompile(`\$Number%(\d+)d\$`)
   result := re.ReplaceAllStringFunc(template, func(match string) string {
      // Extract padding width from the format
      matches := re.FindStringSubmatch(match)
      if len(matches) > 1 {
         if width, err := strconv.Atoi(matches[1]); err == nil {
            return fmt.Sprintf("%0*d", width, number)
         }
      }
      return strconv.Itoa(number)
   })

   // Handle simple $Number$ format
   result = strings.ReplaceAll(result, "$Number$", strconv.Itoa(number))

   return result
}

func getPeriodDurationInSeconds(period Period, mpd *MPD) float64 {
   // Try period duration first
   if period.Duration != "" {
      if duration, err := parseDuration(period.Duration); err == nil {
         return duration
      }
   }

   // Fall back to MPD mediaPresentationDuration
   if mpd.MediaPresentationDuration != "" {
      if duration, err := parseDuration(mpd.MediaPresentationDuration); err == nil {
         return duration
      }
   }

   return 0
}

func parseDuration(durationStr string) (float64, error) {
   // Parse ISO 8601 duration format (PT#H#M#S or PT#S)
   re := regexp.MustCompile(`^PT(?:(\d+(?:\.\d+)?)H)?(?:(\d+(?:\.\d+)?)M)?(?:(\d+(?:\.\d+)?)S)?$`)
   matches := re.FindStringSubmatch(durationStr)

   if matches == nil {
      return 0, fmt.Errorf("invalid duration format")
   }

   var totalSeconds float64

   // Hours
   if matches[1] != "" {
      if hours, err := strconv.ParseFloat(matches[1], 64); err == nil {
         totalSeconds += hours * 3600
      }
   }

   // Minutes
   if matches[2] != "" {
      if minutes, err := strconv.ParseFloat(matches[2], 64); err == nil {
         totalSeconds += minutes * 60
      }
   }

   // Seconds
   if matches[3] != "" {
      if seconds, err := strconv.ParseFloat(matches[3], 64); err == nil {
         totalSeconds += seconds
      }
   }

   return totalSeconds, nil
}

func resolveBaseURL(baseURL string, mpdBaseURL string, periodBaseURL string, adaptationSetBaseURL string, repBaseURL string) string {
   result := baseURL

   if mpdBaseURL != "" {
      result = resolveURL(result, mpdBaseURL)
   }

   if periodBaseURL != "" {
      result = resolveURL(result, periodBaseURL)
   }

   if adaptationSetBaseURL != "" {
      result = resolveURL(result, adaptationSetBaseURL)
   }

   if repBaseURL != "" {
      result = resolveURL(result, repBaseURL)
   }

   return result
}

func resolveURL(baseURL, relativeURL string) string {
   if relativeURL == "" {
      return baseURL
   }

   // If relativeURL is already absolute, return it
   if strings.HasPrefix(relativeURL, "http://") || strings.HasPrefix(relativeURL, "https://") {
      return relativeURL
   }

   base, err := url.Parse(baseURL)
   if err != nil {
      return relativeURL
   }

   relative, err := url.Parse(relativeURL)
   if err != nil {
      return relativeURL
   }

   resolved := base.ResolveReference(relative)
   return resolved.String()
}
