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

// MPD root element
type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

// Period element
type Period struct {
   BaseURL        string          `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet element
type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

// Representation element
type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

// SegmentBase element
type SegmentBase struct {
   Initialization *Initialization `xml:"Initialization"`
}

// SegmentList element
type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

// SegmentTemplate element
type SegmentTemplate struct {
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline element
type SegmentTimeline struct {
   S []S `xml:"S"`
}

// S element (segment timeline entry)
type S struct {
   T int `xml:"t,attr"` // start time
   D int `xml:"d,attr"` // duration
   R int `xml:"r,attr"` // repeat count
}

// Initialization element
type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

// SegmentURL element
type SegmentURL struct {
   Media string `xml:"media,attr"`
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }

   mpdFilePath := os.Args[1]

   // Read the MPD file
   data, err := ioutil.ReadFile(mpdFilePath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
      os.Exit(1)
   }

   // Parse the XML
   var mpd MPD
   err = xml.Unmarshal(data, &mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing XML: %v\n", err)
      os.Exit(1)
   }

   // Extract segment URLs
   result := extractSegmentURLs(&mpd)

   // Output as JSON
   jsonOutput, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonOutput))
}

func extractSegmentURLs(mpd *MPD) map[string][]string {
   result := make(map[string][]string)
   baseURL := "http://test.test/test.mpd"

   // Resolve MPD-level BaseURL
   mpdBaseURL := resolveBaseURL(baseURL, mpd.BaseURL)

   for _, period := range mpd.Periods {
      // Resolve Period-level BaseURL
      periodBaseURL := resolveBaseURL(mpdBaseURL, period.BaseURL)

      for _, adaptationSet := range period.AdaptationSets {
         // Resolve AdaptationSet-level BaseURL
         adaptationSetBaseURL := resolveBaseURL(periodBaseURL, adaptationSet.BaseURL)

         for _, representation := range adaptationSet.Representations {
            // Resolve Representation-level BaseURL
            representationBaseURL := resolveBaseURL(adaptationSetBaseURL, representation.BaseURL)

            // Merge SegmentTemplate from AdaptationSet and Representation
            mergedTemplate := mergeSegmentTemplates(adaptationSet.SegmentTemplate, representation.SegmentTemplate)

            // Extract segments for this representation
            segmentURLs := extractRepresentationSegments(&representation, representationBaseURL, mergedTemplate)
            if len(segmentURLs) > 0 {
               result[representation.ID] = segmentURLs
            }
         }
      }
   }

   return result
}

func extractRepresentationSegments(rep *Representation, baseURL string, segmentTemplate *SegmentTemplate) []string {
   var segments []string

   if rep.SegmentBase != nil {
      // Handle SegmentBase
      if rep.SegmentBase.Initialization != nil {
         initURL := resolveBaseURL(baseURL, rep.SegmentBase.Initialization.SourceURL)
         segments = append(segments, initURL)
      }
   } else if rep.SegmentList != nil {
      // Handle SegmentList
      if rep.SegmentList.Initialization != nil {
         initURL := resolveBaseURL(baseURL, rep.SegmentList.Initialization.SourceURL)
         segments = append(segments, initURL)
      }

      for _, segmentURL := range rep.SegmentList.SegmentURLs {
         segURL := resolveBaseURL(baseURL, segmentURL.Media)
         segments = append(segments, segURL)
      }
   } else if segmentTemplate != nil {
      // Handle SegmentTemplate (either from Representation or inherited from AdaptationSet)
      template := segmentTemplate

      // Add initialization segment
      if template.Initialization != "" {
         initTemplate := template.Initialization
         initURL := resolveTemplateURL(initTemplate, rep.ID, 0, baseURL, 0)
         segments = append(segments, initURL)
      }

      // Generate segment URLs
      if template.SegmentTimeline != nil {
         // Use SegmentTimeline with proper time accumulation
         segmentNumber := template.StartNumber
         if segmentNumber == 0 {
            segmentNumber = 1
         }

         currentTime := int64(0)

         for _, s := range template.SegmentTimeline.S {
            // If T attribute is present, it sets the absolute start time
            if s.T != 0 {
               currentTime = int64(s.T)
            }

            repeatCount := s.R + 1
            if s.R < 0 {
               repeatCount = 1 // Handle special case where R is not specified
            }

            for i := 0; i < repeatCount; i++ {
               segURL := resolveTemplateURL(template.Media, rep.ID, segmentNumber, baseURL, currentTime)
               segments = append(segments, segURL)
               segmentNumber++

               // Accumulate time by duration for next segment
               currentTime += int64(s.D)
            }
         }
      } else {
         // Fallback: generate segments using startNumber and endNumber
         segmentNumber := template.StartNumber
         if segmentNumber == 0 {
            segmentNumber = 1
         }

         endNumber := template.EndNumber
         if endNumber == 0 {
            // If no endNumber specified, generate first 10 segments as fallback
            endNumber = segmentNumber + 9
         }

         // Generate segments from startNumber to endNumber (inclusive)
         for segmentNumber <= endNumber {
            segURL := resolveTemplateURL(template.Media, rep.ID, segmentNumber, baseURL, 0)
            segments = append(segments, segURL)
            segmentNumber++
         }
      }
   }

   return segments
}

func resolveBaseURL(base, relative string) string {
   if relative == "" {
      return base
   }

   baseURL, err := url.Parse(base)
   if err != nil {
      return relative
   }

   relativeURL, err := url.Parse(relative)
   if err != nil {
      return relative
   }

   resolvedURL := baseURL.ResolveReference(relativeURL)
   return resolvedURL.String()
}

func resolveTemplateURL(template, representationID string, segmentNumber int, baseURL string, segmentTime int64) string {
   // Replace template identifiers
   resolved := template
   resolved = strings.ReplaceAll(resolved, "$RepresentationID$", representationID)
   resolved = strings.ReplaceAll(resolved, "$Number$", strconv.Itoa(segmentNumber))
   resolved = strings.ReplaceAll(resolved, "$Time$", strconv.FormatInt(segmentTime, 10))

   // Handle formatted Number patterns
   resolved = strings.ReplaceAll(resolved, "$Number%01d$", fmt.Sprintf("%01d", segmentNumber))
   resolved = strings.ReplaceAll(resolved, "$Number%02d$", fmt.Sprintf("%02d", segmentNumber))
   resolved = strings.ReplaceAll(resolved, "$Number%03d$", fmt.Sprintf("%03d", segmentNumber))
   resolved = strings.ReplaceAll(resolved, "$Number%04d$", fmt.Sprintf("%04d", segmentNumber))
   resolved = strings.ReplaceAll(resolved, "$Number%05d$", fmt.Sprintf("%05d", segmentNumber))

   // Handle formatted Time patterns
   resolved = strings.ReplaceAll(resolved, "$Time%01d$", fmt.Sprintf("%01d", segmentTime))
   resolved = strings.ReplaceAll(resolved, "$Time%02d$", fmt.Sprintf("%02d", segmentTime))
   resolved = strings.ReplaceAll(resolved, "$Time%03d$", fmt.Sprintf("%03d", segmentTime))
   resolved = strings.ReplaceAll(resolved, "$Time%04d$", fmt.Sprintf("%04d", segmentTime))
   resolved = strings.ReplaceAll(resolved, "$Time%05d$", fmt.Sprintf("%05d", segmentTime))

   // Handle other common template patterns for Number
   if strings.Contains(resolved, "$Number%") {
      // Handle generic $Number%Xd$ patterns
      for i := 1; i <= 10; i++ {
         pattern := fmt.Sprintf("$Number%%%02dd$", i)
         replacement := fmt.Sprintf("%0"+strconv.Itoa(i)+"d", segmentNumber)
         resolved = strings.ReplaceAll(resolved, pattern, replacement)
      }
   }

   // Handle other common template patterns for Time
   if strings.Contains(resolved, "$Time%") {
      // Handle generic $Time%Xd$ patterns
      for i := 6; i <= 15; i++ { // Time values are typically longer
         pattern := fmt.Sprintf("$Time%%%02dd$", i)
         replacement := fmt.Sprintf("%0"+strconv.Itoa(i)+"d", segmentTime)
         resolved = strings.ReplaceAll(resolved, pattern, replacement)
      }
   }

   // Resolve against base URL
   return resolveBaseURL(baseURL, resolved)
}

// mergeSegmentTemplates combines AdaptationSet and Representation level SegmentTemplates
// Representation level attributes override AdaptationSet level attributes
func mergeSegmentTemplates(adaptationSetTemplate, representationTemplate *SegmentTemplate) *SegmentTemplate {
   // If no templates exist, return nil
   if adaptationSetTemplate == nil && representationTemplate == nil {
      return nil
   }

   // If only one template exists, return it
   if adaptationSetTemplate == nil {
      return representationTemplate
   }
   if representationTemplate == nil {
      return adaptationSetTemplate
   }

   // Merge templates with Representation taking precedence
   merged := &SegmentTemplate{
      Media:           adaptationSetTemplate.Media,
      Initialization:  adaptationSetTemplate.Initialization,
      StartNumber:     adaptationSetTemplate.StartNumber,
      SegmentTimeline: adaptationSetTemplate.SegmentTimeline,
   }

   // Override with Representation values if they exist
   if representationTemplate.Media != "" {
      merged.Media = representationTemplate.Media
   }
   if representationTemplate.Initialization != "" {
      merged.Initialization = representationTemplate.Initialization
   }
   if representationTemplate.StartNumber != 0 {
      merged.StartNumber = representationTemplate.StartNumber
   }
   if representationTemplate.SegmentTimeline != nil {
      merged.SegmentTimeline = representationTemplate.SegmentTimeline
   }

   return merged
}
