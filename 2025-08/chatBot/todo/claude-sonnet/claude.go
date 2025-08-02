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

// MPD XML structures
type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL []string `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   XMLName        xml.Name        `xml:"Period"`
   BaseURL        []string        `xml:"BaseURL"`
   Duration       string          `xml:"duration,attr"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   XMLName         xml.Name         `xml:"AdaptationSet"`
   BaseURL         []string         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   XMLName         xml.Name         `xml:"Representation"`
   ID              string           `xml:"id,attr"`
   BaseURL         []string         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
   XMLName         xml.Name         `xml:"SegmentTemplate"`
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   Timescale       int              `xml:"timescale,attr"`
   Duration        int              `xml:"duration,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   XMLName xml.Name `xml:"SegmentTimeline"`
   S       []S      `xml:"S"`
}

type S struct {
   XMLName xml.Name `xml:"S"`
   T       *int     `xml:"t,attr"`
   D       int      `xml:"d,attr"`
   R       int      `xml:"r,attr"`
}

type SegmentList struct {
   XMLName        xml.Name       `xml:"SegmentList"`
   Initialization Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL   `xml:"SegmentURL"`
}

type Initialization struct {
   XMLName   xml.Name `xml:"Initialization"`
   SourceURL string   `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   XMLName xml.Name `xml:"SegmentURL"`
   Media   string   `xml:"media,attr"`
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

   // Parse XML
   var mpd MPD
   err = xml.Unmarshal(data, &mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing XML: %v\n", err)
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
   baseURL := "http://test.test/test.mpd"

   // Build hierarchical base URLs
   mpdBaseURL := resolveBaseURL(baseURL, mpd.BaseURL)

   for _, period := range mpd.Periods {
      periodBaseURL := resolveBaseURL(mpdBaseURL, period.BaseURL)
      periodDuration := parseDuration(period.Duration)

      for _, adaptationSet := range period.AdaptationSets {
         adaptationSetBaseURL := resolveBaseURL(periodBaseURL, adaptationSet.BaseURL)

         for _, representation := range adaptationSet.Representations {
            representationBaseURL := resolveBaseURL(adaptationSetBaseURL, representation.BaseURL)

            // Get the effective segment template (inheritance)
            segmentTemplate := getEffectiveSegmentTemplate(&adaptationSet, &representation)

            var urls []string

            if representation.SegmentList != nil {
               // Handle SegmentList
               urls = extractFromSegmentList(representation.SegmentList, representationBaseURL)
            } else if segmentTemplate != nil {
               // Handle SegmentTemplate
               urls = extractFromSegmentTemplate(segmentTemplate, representationBaseURL, representation.ID, periodDuration)
            } else if len(representation.BaseURL) > 0 {
               // Handle BaseURL-only representation
               urls = []string{representationBaseURL}
            }

            if len(urls) > 0 {
               // Aggregate segments for the same Representation ID across multiple Periods
               if existingURLs, exists := result[representation.ID]; exists {
                  // Append new segments to existing ones
                  result[representation.ID] = append(existingURLs, urls...)
               } else {
                  // First occurrence of this Representation ID
                  result[representation.ID] = urls
               }
            }
         }
      }
   }

   return result
}

func resolveBaseURL(base string, baseURLs []string) string {
   current := base
   for _, baseURL := range baseURLs {
      if baseURL != "" {
         resolved, err := resolveURL(current, baseURL)
         if err == nil {
            current = resolved
         }
      }
   }
   return current
}

func resolveURL(base, ref string) (string, error) {
   baseURL, err := url.Parse(base)
   if err != nil {
      return "", err
   }

   refURL, err := url.Parse(ref)
   if err != nil {
      return "", err
   }

   return baseURL.ResolveReference(refURL).String(), nil
}

func getEffectiveSegmentTemplate(adaptationSet *AdaptationSet, representation *Representation) *SegmentTemplate {
   // Representation level takes precedence
   if representation.SegmentTemplate != nil {
      return mergeSegmentTemplates(adaptationSet.SegmentTemplate, representation.SegmentTemplate)
   }

   // Fall back to AdaptationSet level
   return adaptationSet.SegmentTemplate
}

func mergeSegmentTemplates(parent, child *SegmentTemplate) *SegmentTemplate {
   if parent == nil {
      return child
   }
   if child == nil {
      return parent
   }

   // Create merged template with child taking precedence
   merged := &SegmentTemplate{
      Media:           child.Media,
      Initialization:  child.Initialization,
      StartNumber:     child.StartNumber,
      EndNumber:       child.EndNumber,
      Timescale:       child.Timescale,
      Duration:        child.Duration,
      SegmentTimeline: child.SegmentTimeline,
   }

   // Fill in missing values from parent
   if merged.Media == "" {
      merged.Media = parent.Media
   }
   if merged.Initialization == "" {
      merged.Initialization = parent.Initialization
   }
   if merged.StartNumber == 0 {
      merged.StartNumber = parent.StartNumber
   }
   if merged.EndNumber == 0 {
      merged.EndNumber = parent.EndNumber
   }
   if merged.Timescale == 0 {
      merged.Timescale = parent.Timescale
   }
   if merged.Duration == 0 {
      merged.Duration = parent.Duration
   }
   if merged.SegmentTimeline == nil {
      merged.SegmentTimeline = parent.SegmentTimeline
   }

   return merged
}

func extractFromSegmentList(segmentList *SegmentList, baseURL string) []string {
   var urls []string

   // Add initialization URL if present
   if segmentList.Initialization.SourceURL != "" {
      initURL, err := resolveURL(baseURL, segmentList.Initialization.SourceURL)
      if err == nil {
         urls = append(urls, initURL)
      }
   }

   // Add segment URLs
   for _, segmentURL := range segmentList.SegmentURLs {
      if segmentURL.Media != "" {
         segURL, err := resolveURL(baseURL, segmentURL.Media)
         if err == nil {
            urls = append(urls, segURL)
         }
      }
   }

   return urls
}

func extractFromSegmentTemplate(template *SegmentTemplate, baseURL, representationID string, periodDuration float64) []string {
   var urls []string

   // Add initialization URL if present
   if template.Initialization != "" {
      initTemplate := template.Initialization
      initTemplate = strings.ReplaceAll(initTemplate, "$RepresentationID$", representationID)
      initURL, err := resolveURL(baseURL, initTemplate)
      if err == nil {
         urls = append(urls, initURL)
      }
   }

   // Generate segment URLs
   if template.Media != "" {
      segmentURLs := generateSegmentURLs(template, baseURL, representationID, periodDuration)
      urls = append(urls, segmentURLs...)
   }

   return urls
}

func generateSegmentURLs(template *SegmentTemplate, baseURL, representationID string, periodDuration float64) []string {
   var urls []string

   if template.SegmentTimeline != nil {
      // Use SegmentTimeline
      urls = generateTimelineBasedURLs(template, baseURL, representationID)
   } else if template.Duration > 0 {
      // Use duration-based generation
      urls = generateDurationBasedURLs(template, baseURL, representationID, periodDuration)
   }

   return urls
}

func generateTimelineBasedURLs(template *SegmentTemplate, baseURL, representationID string) []string {
   var urls []string

   startNumber := template.StartNumber
   if startNumber == 0 {
      startNumber = 1
   }

   currentTime := 0
   segmentNumber := startNumber

   for _, s := range template.SegmentTimeline.S {
      // Use explicit time if provided
      if s.T != nil {
         currentTime = *s.T
      }

      // Generate segments for this S element
      repeatCount := s.R + 1 // R is additional repeats, so total is R+1
      for i := 0; i < repeatCount; i++ {
         // Check endNumber limit
         if template.EndNumber > 0 && segmentNumber > template.EndNumber {
            return urls
         }

         mediaTemplate := template.Media
         mediaTemplate = strings.ReplaceAll(mediaTemplate, "$RepresentationID$", representationID)
         mediaTemplate = substituteNumber(mediaTemplate, segmentNumber)
         mediaTemplate = strings.ReplaceAll(mediaTemplate, "$Time$", strconv.Itoa(currentTime))

         segURL, err := resolveURL(baseURL, mediaTemplate)
         if err == nil {
            urls = append(urls, segURL)
         }

         currentTime += s.D
         segmentNumber++
      }
   }

   return urls
}

func generateDurationBasedURLs(template *SegmentTemplate, baseURL, representationID string, periodDuration float64) []string {
   var urls []string

   startNumber := template.StartNumber
   if startNumber == 0 {
      startNumber = 1
   }

   endNumber := template.EndNumber
   if endNumber == 0 {
      // Calculate endNumber based on period duration if available
      if periodDuration > 0 && template.Duration > 0 {
         // Default timescale to 1 if missing
         timescale := template.Timescale
         if timescale == 0 {
            timescale = 1
         }
         // Calculate: ceil(PeriodDurationInSeconds * timescale / duration)
         segmentCount := int(math.Ceil(periodDuration * float64(timescale) / float64(template.Duration)))
         endNumber = startNumber + segmentCount - 1
      } else {
         // If no period duration or calculation not possible, use a reasonable default
         endNumber = startNumber + 100 // Arbitrary limit
      }
   }

   for segmentNumber := startNumber; segmentNumber <= endNumber; segmentNumber++ {
      mediaTemplate := template.Media
      mediaTemplate = strings.ReplaceAll(mediaTemplate, "$RepresentationID$", representationID)
      mediaTemplate = substituteNumber(mediaTemplate, segmentNumber)

      segURL, err := resolveURL(baseURL, mediaTemplate)
      if err == nil {
         urls = append(urls, segURL)
      }
   }

   return urls
}

func substituteNumber(template string, number int) string {
   // Handle formatted numbers like $Number%05d$
   re := regexp.MustCompile(`\$Number(%\d*d)?\$`)

   return re.ReplaceAllStringFunc(template, func(match string) string {
      // Extract format specifier if present
      formatRe := regexp.MustCompile(`\$Number(%\d*d)?\$`)
      matches := formatRe.FindStringSubmatch(match)

      if len(matches) > 1 && matches[1] != "" {
         // Use the format specifier
         format := matches[1]
         return fmt.Sprintf(format, number)
      } else {
         // Plain number replacement
         return strconv.Itoa(number)
      }
   })
}

// parseDuration parses ISO 8601 duration string and returns duration in seconds
func parseDuration(duration string) float64 {
   if duration == "" {
      return 0
   }

   // Handle ISO 8601 duration format (PT30.5S, PT1M30S, PT1H30M, etc.)
   if strings.HasPrefix(duration, "PT") {
      duration = duration[2:] // Remove "PT" prefix

      var totalSeconds float64

      // Parse hours
      if strings.Contains(duration, "H") {
         parts := strings.Split(duration, "H")
         if len(parts) >= 2 {
            if hours, err := strconv.ParseFloat(parts[0], 64); err == nil {
               totalSeconds += hours * 3600
            }
            duration = parts[1]
         }
      }

      // Parse minutes
      if strings.Contains(duration, "M") {
         parts := strings.Split(duration, "M")
         if len(parts) >= 2 {
            if minutes, err := strconv.ParseFloat(parts[0], 64); err == nil {
               totalSeconds += minutes * 60
            }
            duration = parts[1]
         }
      }

      // Parse seconds
      if strings.Contains(duration, "S") {
         parts := strings.Split(duration, "S")
         if len(parts) >= 1 && parts[0] != "" {
            if seconds, err := strconv.ParseFloat(parts[0], 64); err == nil {
               totalSeconds += seconds
            }
         }
      }

      return totalSeconds
   }

   // Try parsing as plain number (seconds)
   if seconds, err := strconv.ParseFloat(duration, 64); err == nil {
      return seconds
   }

   return 0
}
