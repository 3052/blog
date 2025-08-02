package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
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
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   XMLName         xml.Name         `xml:"AdaptationSet"`
   BaseURL         []string         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   XMLName         xml.Name         `xml:"Representation"`
   ID              string           `xml:"id,attr"`
   BaseURL         []string         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
   XMLName         xml.Name         `xml:"SegmentTemplate"`
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   StartNumber     *int             `xml:"startNumber,attr"`
   EndNumber       *int             `xml:"endNumber,attr"`
   Duration        *int             `xml:"duration,attr"`
   Timescale       *int             `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   XMLName xml.Name `xml:"SegmentTimeline"`
   S       []S      `xml:"S"`
}

type S struct {
   XMLName xml.Name `xml:"S"`
   T       *uint64  `xml:"t,attr"`
   D       uint64   `xml:"d,attr"`
   R       *int     `xml:"r,attr"`
}

type SegmentBase struct {
   XMLName        xml.Name        `xml:"SegmentBase"`
   Initialization *Initialization `xml:"Initialization"`
}

type SegmentList struct {
   XMLName        xml.Name        `xml:"SegmentList"`
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type Initialization struct {
   XMLName   xml.Name `xml:"Initialization"`
   SourceURL string   `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   XMLName xml.Name `xml:"SegmentURL"`
   Media   string   `xml:"media,attr"`
}

// Helper structures
type ResolvedSegmentTemplate struct {
   Media          string
   Initialization string
   StartNumber    int
   EndNumber      int
   Timeline       *SegmentTimeline
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }

   mpdPath := os.Args[1]

   // Read the MPD file
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

   // Resolve MPD-level BaseURL
   mpdBaseURL := resolveBaseURL(baseURL, mpd.BaseURL)

   for _, period := range mpd.Periods {
      // Resolve Period-level BaseURL
      periodBaseURL := resolveBaseURL(mpdBaseURL, period.BaseURL)

      for _, adaptationSet := range period.AdaptationSets {
         // Resolve AdaptationSet-level BaseURL
         asBaseURL := resolveBaseURL(periodBaseURL, adaptationSet.BaseURL)

         for _, representation := range adaptationSet.Representations {
            // Resolve Representation-level BaseURL
            repBaseURL := resolveBaseURL(asBaseURL, representation.BaseURL)

            segments := extractRepresentationSegments(&adaptationSet, &representation, repBaseURL)
            if len(segments) > 0 {
               result[representation.ID] = segments
            }
         }
      }
   }

   return result
}

func resolveBaseURL(base string, baseURLs []string) string {
   if len(baseURLs) == 0 {
      return base
   }

   // Use the first BaseURL element
   newBase := baseURLs[0]

   // Parse base URL
   baseU, err := url.Parse(base)
   if err != nil {
      return base
   }

   // Parse relative URL
   relU, err := url.Parse(newBase)
   if err != nil {
      return base
   }

   // Resolve using ResolveReference
   resolved := baseU.ResolveReference(relU)
   return resolved.String()
}

func extractRepresentationSegments(adaptationSet *AdaptationSet, representation *Representation, baseURL string) []string {
   var segments []string

   // Check SegmentBase
   if representation.SegmentBase != nil || adaptationSet.SegmentBase != nil {
      segBase := representation.SegmentBase
      if segBase == nil {
         segBase = adaptationSet.SegmentBase
      }

      if segBase.Initialization != nil && segBase.Initialization.SourceURL != "" {
         initURL := resolveURL(baseURL, segBase.Initialization.SourceURL)
         segments = append(segments, initURL)
      }
      return segments
   }

   // Check SegmentList
   if representation.SegmentList != nil || adaptationSet.SegmentList != nil {
      segList := representation.SegmentList
      if segList == nil {
         segList = adaptationSet.SegmentList
      }

      if segList.Initialization != nil && segList.Initialization.SourceURL != "" {
         initURL := resolveURL(baseURL, segList.Initialization.SourceURL)
         segments = append(segments, initURL)
      }

      for _, segURL := range segList.SegmentURLs {
         if segURL.Media != "" {
            mediaURL := resolveURL(baseURL, segURL.Media)
            segments = append(segments, mediaURL)
         }
      }
      return segments
   }

   // Check SegmentTemplate
   template := resolveSegmentTemplate(adaptationSet.SegmentTemplate, representation.SegmentTemplate)
   if template != nil {
      segments = generateTemplateSegments(template, representation.ID, baseURL)
      return segments
   }

   // Handle BaseURL-only Representations
   // If no segment information is found but we have BaseURL, treat it as a single segment
   if len(representation.BaseURL) > 0 || baseURL != "" {
      // The baseURL already includes resolved BaseURL from hierarchy
      segments = append(segments, baseURL)
   }

   return segments
}

func resolveSegmentTemplate(asTemplate, repTemplate *SegmentTemplate) *ResolvedSegmentTemplate {
   if asTemplate == nil && repTemplate == nil {
      return nil
   }

   resolved := &ResolvedSegmentTemplate{
      StartNumber: 1,  // Default start number
      EndNumber:   -1, // Will be calculated or set
   }

   // Start with AdaptationSet template
   if asTemplate != nil {
      if asTemplate.Media != "" {
         resolved.Media = asTemplate.Media
      }
      if asTemplate.Initialization != "" {
         resolved.Initialization = asTemplate.Initialization
      }
      if asTemplate.StartNumber != nil {
         resolved.StartNumber = *asTemplate.StartNumber
      }
      if asTemplate.EndNumber != nil {
         resolved.EndNumber = *asTemplate.EndNumber
      }
      if asTemplate.SegmentTimeline != nil {
         resolved.Timeline = asTemplate.SegmentTimeline
      }
   }

   // Override with Representation template
   if repTemplate != nil {
      if repTemplate.Media != "" {
         resolved.Media = repTemplate.Media
      }
      if repTemplate.Initialization != "" {
         resolved.Initialization = repTemplate.Initialization
      }
      if repTemplate.StartNumber != nil {
         resolved.StartNumber = *repTemplate.StartNumber
      }
      if repTemplate.EndNumber != nil {
         resolved.EndNumber = *repTemplate.EndNumber
      }
      if repTemplate.SegmentTimeline != nil {
         resolved.Timeline = repTemplate.SegmentTimeline
      }
   }

   return resolved
}

func generateTemplateSegments(template *ResolvedSegmentTemplate, representationID, baseURL string) []string {
   var segments []string

   // Add initialization segment if present
   if template.Initialization != "" {
      initURL := substituteTemplateVariables(template.Initialization, representationID, 0, 0)
      initURL = resolveURL(baseURL, initURL)
      segments = append(segments, initURL)
   }

   // Generate media segments
   if template.Media != "" {
      if template.Timeline != nil {
         // Use SegmentTimeline for precise segment generation
         segments = append(segments, generateTimelineSegments(template, representationID, baseURL)...)
      } else {
         // Generate segments from startNumber to endNumber
         endNumber := template.EndNumber
         if endNumber == -1 {
            endNumber = template.StartNumber + 99 // Default to 100 segments if no endNumber
         }

         for i := template.StartNumber; i <= endNumber; i++ {
            mediaURL := substituteTemplateVariables(template.Media, representationID, i, 0)
            mediaURL = resolveURL(baseURL, mediaURL)
            segments = append(segments, mediaURL)
         }
      }
   }

   return segments
}

func generateTimelineSegments(template *ResolvedSegmentTemplate, representationID, baseURL string) []string {
   var segments []string
   segmentNumber := template.StartNumber
   currentTime := uint64(0)

   for _, s := range template.Timeline.S {
      // Set absolute time if specified
      if s.T != nil {
         currentTime = *s.T
      }

      // Calculate repeat count
      repeatCount := 1
      if s.R != nil {
         repeatCount = *s.R + 1 // r=-1 means repeat infinitely, but we'll cap it
         if repeatCount < 0 {
            repeatCount = 100 // Cap infinite repeats
         }
      }

      // Generate segments for this S element
      for i := 0; i < repeatCount; i++ {
         mediaURL := substituteTemplateVariables(template.Media, representationID, segmentNumber, currentTime)
         mediaURL = resolveURL(baseURL, mediaURL)
         segments = append(segments, mediaURL)

         segmentNumber++
         currentTime += s.D
      }
   }

   return segments
}

func substituteTemplateVariables(template, representationID string, number int, time uint64) string {
   result := template

   // Substitute $RepresentationID$
   result = strings.ReplaceAll(result, "$RepresentationID$", representationID)

   // Substitute $Number$ with formatting
   numberRegex := regexp.MustCompile(`\$Number(?:%(\d+)d)?\$`)
   result = numberRegex.ReplaceAllStringFunc(result, func(match string) string {
      // Extract format if present
      matches := numberRegex.FindStringSubmatch(match)
      if len(matches) > 1 && matches[1] != "" {
         // Format with padding
         padding, _ := strconv.Atoi(matches[1])
         format := fmt.Sprintf("%%0%dd", padding)
         return fmt.Sprintf(format, number)
      }
      return strconv.Itoa(number)
   })

   // Substitute $Time$
   result = strings.ReplaceAll(result, "$Time$", strconv.FormatUint(time, 10))

   return result
}

func resolveURL(base, relative string) string {
   baseU, err := url.Parse(base)
   if err != nil {
      return relative
   }

   relU, err := url.Parse(relative)
   if err != nil {
      return relative
   }

   // Resolve using ResolveReference
   resolved := baseU.ResolveReference(relU)
   return resolved.String()
}
