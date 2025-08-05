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
   "time"
)

// MPD structures
type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   BaseURL                   string   `xml:"BaseURL"`
   Periods                   []Period `xml:"Period"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
}

type Period struct {
   BaseURL        string          `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
   Duration       string          `xml:"duration,attr"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   Representations []Representation `xml:"Representation"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
   Bandwidth       int              `xml:"bandwidth,attr"`
   Width           int              `xml:"width,attr"`
   Height          int              `xml:"height,attr"`
}

type SegmentTemplate struct {
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   Duration        int              `xml:"duration,attr"`
   Timescale       int              `xml:"timescale,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
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
   Duration       int             `xml:"duration,attr"`
   Timescale      int             `xml:"timescale,attr"`
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type SegmentBase struct {
   Initialization *Initialization `xml:"Initialization"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

func main() {
   if len(os.Args) < 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }

   mpdFilePath := os.Args[1]

   // Read MPD file
   data, err := ioutil.ReadFile(mpdFilePath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
      os.Exit(1)
   }

   // Parse MPD
   var mpd MPD
   err = xml.Unmarshal(data, &mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing XML: %v\n", err)
      os.Exit(1)
   }

   // Base MPD URL
   baseMPDURL := "http://test.test/test.mpd"

   // Process all representations
   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      for _, adaptationSet := range period.AdaptationSets {
         for _, representation := range adaptationSet.Representations {
            urls := processRepresentation(baseMPDURL, mpd, period, adaptationSet, representation)
            // Append to existing URLs for this representation ID
            result[representation.ID] = append(result[representation.ID], urls...)
         }
      }
   }

   // Output JSON
   jsonData, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error creating JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonData))
}

func processRepresentation(baseMPDURL string, mpd MPD, period Period, adaptationSet AdaptationSet, representation Representation) []string {
   var urls []string

   // Build base URL chain
   baseURL := buildBaseURLChain(baseMPDURL, mpd.BaseURL, period.BaseURL, adaptationSet.BaseURL, representation.BaseURL)

   // Get segment template (check representation first, then adaptation set)
   segmentTemplate := representation.SegmentTemplate
   if segmentTemplate == nil {
      segmentTemplate = adaptationSet.SegmentTemplate
   }

   // Get segment list (check representation first, then adaptation set)
   segmentList := representation.SegmentList
   if segmentList == nil {
      segmentList = adaptationSet.SegmentList
   }

   if segmentTemplate != nil {
      urls = processSegmentTemplate(baseURL, segmentTemplate, representation, period.Duration)
   } else if segmentList != nil {
      urls = processSegmentList(baseURL, segmentList)
   } else {
      // Single segment (SegmentBase or no segmentation)
      if baseURL != "" {
         urls = append(urls, baseURL)
      }
   }

   return urls
}

func buildBaseURLChain(baseMPDURL string, mpdBaseURL, periodBaseURL, asBaseURL, repBaseURL string) string {
   baseURL := baseMPDURL

   // Apply base URLs in order: MPD -> Period -> AdaptationSet -> Representation
   if mpdBaseURL != "" {
      baseURL = resolveURL(baseURL, mpdBaseURL)
   }
   if periodBaseURL != "" {
      baseURL = resolveURL(baseURL, periodBaseURL)
   }
   if asBaseURL != "" {
      baseURL = resolveURL(baseURL, asBaseURL)
   }
   if repBaseURL != "" {
      baseURL = resolveURL(baseURL, repBaseURL)
   }

   return baseURL
}

func resolveURL(baseURL, relativeURL string) string {
   if relativeURL == "" {
      return baseURL
   }

   // If relative URL is absolute, return it
   if strings.HasPrefix(relativeURL, "http://") || strings.HasPrefix(relativeURL, "https://") {
      return relativeURL
   }

   // Parse base URL
   base, err := url.Parse(baseURL)
   if err != nil {
      return relativeURL
   }

   // Parse relative URL
   rel, err := url.Parse(relativeURL)
   if err != nil {
      return baseURL
   }

   // Resolve relative to base
   resolved := base.ResolveReference(rel)
   return resolved.String()
}

func processSegmentTemplate(baseURL string, template *SegmentTemplate, representation Representation, periodDuration string) []string {
   var urls []string

   if template.Media == "" {
      return urls
   }

   // Add initialization segment if present
   if template.Initialization != "" {
      initURL := template.Initialization
      initURL = replaceTemplateVariable(initURL, "RepresentationID", representation.ID)
      initURL = replaceTemplateVariable(initURL, "Bandwidth", strconv.Itoa(representation.Bandwidth))
      urls = append(urls, resolveURL(baseURL, initURL))
   }

   // Process segments
   if template.SegmentTimeline != nil {
      // Timeline-based segments
      segmentNumber := 1
      if template.StartNumber > 0 {
         segmentNumber = template.StartNumber
      }

      currentTime := 0
      for _, s := range template.SegmentTimeline.S {
         if s.T > 0 {
            currentTime = s.T
         }

         repeats := 0
         if s.R > 0 {
            repeats = s.R
         }

         for i := 0; i <= repeats; i++ {
            mediaURL := template.Media
            mediaURL = replaceTemplateVariable(mediaURL, "RepresentationID", representation.ID)
            mediaURL = replaceTemplateVariable(mediaURL, "Number", strconv.Itoa(segmentNumber))
            mediaURL = replaceTemplateVariable(mediaURL, "Time", strconv.Itoa(currentTime))
            mediaURL = replaceTemplateVariable(mediaURL, "Bandwidth", strconv.Itoa(representation.Bandwidth))

            urls = append(urls, resolveURL(baseURL, mediaURL))

            segmentNumber++
            currentTime += s.D
         }
      }
   } else if template.Duration > 0 {
      // Duration-based segments
      timescale := template.Timescale
      if timescale == 0 {
         timescale = 1
      }

      startNumber := 1
      if template.StartNumber > 0 {
         startNumber = template.StartNumber
      }

      // Determine the number of segments to generate
      var numSegments int
      if template.EndNumber > 0 && template.EndNumber >= startNumber {
         // If endNumber is specified, use it to calculate the exact number of segments
         numSegments = template.EndNumber - startNumber + 1
      } else if periodDuration != "" {
         // Calculate based on period duration
         duration, err := parseDuration(periodDuration)
         if err == nil {
            // Calculate: ceil(PeriodDurationInSeconds * timescale / duration)
            periodSeconds := duration.Seconds()
            numSegments = int(math.Ceil(periodSeconds * float64(timescale) / float64(template.Duration)))
         } else {
            // Fall back to default if parsing fails
            numSegments = 100
         }
      } else {
         // Default fallback
         numSegments = 100
      }

      // Generate segments
      for i := 0; i < numSegments; i++ {
         segmentNumber := startNumber + i
         segmentTime := i * template.Duration

         mediaURL := template.Media
         mediaURL = replaceTemplateVariable(mediaURL, "RepresentationID", representation.ID)
         mediaURL = replaceTemplateVariable(mediaURL, "Number", strconv.Itoa(segmentNumber))
         mediaURL = replaceTemplateVariable(mediaURL, "Time", strconv.Itoa(segmentTime))
         mediaURL = replaceTemplateVariable(mediaURL, "Bandwidth", strconv.Itoa(representation.Bandwidth))

         urls = append(urls, resolveURL(baseURL, mediaURL))
      }
   }

   return urls
}

func processSegmentList(baseURL string, segmentList *SegmentList) []string {
   var urls []string

   // Add initialization segment if present
   if segmentList.Initialization != nil && segmentList.Initialization.SourceURL != "" {
      urls = append(urls, resolveURL(baseURL, segmentList.Initialization.SourceURL))
   }

   // Add all segment URLs
   for _, segmentURL := range segmentList.SegmentURLs {
      if segmentURL.Media != "" {
         urls = append(urls, resolveURL(baseURL, segmentURL.Media))
      }
   }

   return urls
}

// parseDuration parses ISO 8601 duration format (e.g., PT0H3M30.014S)
func parseDuration(duration string) (time.Duration, error) {
   if !strings.HasPrefix(duration, "PT") {
      return 0, fmt.Errorf("invalid duration format")
   }

   // Remove PT prefix
   duration = duration[2:]

   var totalSeconds float64
   var current string

   for _, char := range duration {
      switch char {
      case 'H':
         if hours, err := strconv.ParseFloat(current, 64); err == nil {
            totalSeconds += hours * 3600
         }
         current = ""
      case 'M':
         if minutes, err := strconv.ParseFloat(current, 64); err == nil {
            totalSeconds += minutes * 60
         }
         current = ""
      case 'S':
         if seconds, err := strconv.ParseFloat(current, 64); err == nil {
            totalSeconds += seconds
         }
         current = ""
      default:
         current += string(char)
      }
   }

   return time.Duration(totalSeconds * float64(time.Second)), nil
}

// replaceTemplateVariable replaces template variables with support for format strings
func replaceTemplateVariable(template, variable, value string) string {
   // First try simple replacement
   template = strings.ReplaceAll(template, "$"+variable+"$", value)

   // Then handle formatted replacements like $Number%08d$
   re := regexp.MustCompile(`\$` + variable + `(%[0-9]*d)\$`)
   matches := re.FindAllStringSubmatch(template, -1)

   for _, match := range matches {
      if len(match) > 1 {
         formatSpec := match[1]
         // Parse the numeric value
         if numVal, err := strconv.Atoi(value); err == nil {
            // Apply the format
            formatted := fmt.Sprintf(formatSpec, numVal)
            template = strings.ReplaceAll(template, match[0], formatted)
         }
      }
   }

   return template
}
