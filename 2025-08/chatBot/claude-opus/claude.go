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
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   BaseURL         string           `xml:"BaseURL"`
   AdaptationSets  []AdaptationSet  `xml:"AdaptationSet"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   Representations []Representation `xml:"Representation"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   Bandwidth       int              `xml:"bandwidth,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
}

type SegmentTemplate struct {
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
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

type SegmentBase struct {
   Initialization *Initialization `xml:"Initialization"`
}

func resolveURL(base, relative string) string {
   if relative == "" {
      return base
   }

   // Check if relative is already absolute
   if strings.HasPrefix(relative, "http://") || strings.HasPrefix(relative, "https://") {
      return relative
   }

   // Parse base URL
   baseURL, err := url.Parse(base)
   if err != nil {
      return relative
   }

   // Parse relative URL
   relURL, err := url.Parse(relative)
   if err != nil {
      return relative
   }

   // Resolve relative to base
   resolved := baseURL.ResolveReference(relURL)
   return resolved.String()
}

func replaceTemplateVars(template string, repID string, number int, time int, bandwidth int) string {
   result := template

   // Replace RepresentationID
   result = strings.ReplaceAll(result, "$RepresentationID$", repID)

   // Replace Bandwidth
   result = strings.ReplaceAll(result, "$Bandwidth$", strconv.Itoa(bandwidth))

   // Replace Time
   result = strings.ReplaceAll(result, "$Time$", strconv.Itoa(time))

   // Handle padded Number format
   if strings.Contains(result, "$Number") {
      // Find padded format like $Number%05d$
      start := strings.Index(result, "$Number")
      if start != -1 {
         end := strings.Index(result[start:], "$")
         if end != -1 {
            end += start + 1
            numberFormat := result[start:end]

            if strings.Contains(numberFormat, "%") {
               // Extract padding format
               formatStart := strings.Index(numberFormat, "%")
               formatEnd := strings.Index(numberFormat, "d$")
               if formatStart != -1 && formatEnd != -1 {
                  paddingStr := numberFormat[formatStart+1 : formatEnd]
                  padding, err := strconv.Atoi(paddingStr)
                  if err == nil {
                     // Apply padding
                     paddedNumber := fmt.Sprintf("%0*d", padding, number)
                     result = strings.Replace(result, numberFormat, paddedNumber, 1)
                  }
               }
            } else {
               // Simple replacement
               result = strings.ReplaceAll(result, "$Number$", strconv.Itoa(number))
            }
         }
      }
   } else {
      // Simple replacement
      result = strings.ReplaceAll(result, "$Number$", strconv.Itoa(number))
   }

   return result
}

func processRepresentation(rep Representation, periodBaseURL string, periodTemplate *SegmentTemplate,
   periodSegList *SegmentList, adaptSetBaseURL string, adaptSetTemplate *SegmentTemplate,
   adaptSetSegList *SegmentList) []string {

   var urls []string

   // Resolve BaseURL hierarchy
   baseURL := periodBaseURL
   if adaptSetBaseURL != "" {
      baseURL = resolveURL(baseURL, adaptSetBaseURL)
   }
   if rep.BaseURL != "" {
      baseURL = resolveURL(baseURL, rep.BaseURL)
   }

   // Determine which segment method to use (Representation > AdaptationSet > Period)
   var template *SegmentTemplate
   var segList *SegmentList

   if rep.SegmentTemplate != nil {
      template = rep.SegmentTemplate
   } else if adaptSetTemplate != nil {
      template = adaptSetTemplate
   } else if periodTemplate != nil {
      template = periodTemplate
   }

   if rep.SegmentList != nil {
      segList = rep.SegmentList
   } else if adaptSetSegList != nil {
      segList = adaptSetSegList
   } else if periodSegList != nil {
      segList = periodSegList
   }

   // Process based on segment type
   if template != nil {
      // SegmentTemplate processing
      urls = processSegmentTemplate(template, rep.ID, rep.Bandwidth, baseURL)
   } else if segList != nil {
      // SegmentList processing
      urls = processSegmentList(segList, baseURL)
   } else if rep.SegmentBase != nil {
      // SegmentBase processing
      urls = processSegmentBase(rep.SegmentBase, baseURL)
   } else {
      // If representation has only BaseURL and no segment info, use the baseURL directly
      urls = append(urls, baseURL)
   }

   return urls
}

func processSegmentTemplate(template *SegmentTemplate, repID string, bandwidth int, baseURL string) []string {
   var urls []string

   // Add initialization segment if present
   if template.Initialization != "" {
      initURL := replaceTemplateVars(template.Initialization, repID, 0, 0, bandwidth)
      urls = append(urls, resolveURL(baseURL, initURL))
   }

   // Process media segments
   if template.SegmentTimeline != nil {
      // Timeline-based
      time := 0
      segmentNumber := 1
      if template.StartNumber != nil {
         segmentNumber = *template.StartNumber
      }

      for _, s := range template.SegmentTimeline.S {
         if s.T > 0 {
            time = s.T
         }

         // Process segment and repeats
         repeats := s.R
         if repeats < 0 {
            repeats = 0
         }

         for i := 0; i <= repeats; i++ {
            mediaURL := replaceTemplateVars(template.Media, repID, segmentNumber, time, bandwidth)
            urls = append(urls, resolveURL(baseURL, mediaURL))

            time += s.D
            segmentNumber++
         }
      }
   } else if template.Duration != nil && *template.Duration > 0 {
      // Duration-based
      startNumber := 1
      if template.StartNumber != nil {
         startNumber = *template.StartNumber
      }

      endNumber := startNumber + 10 // Default if not specified
      if template.EndNumber != nil {
         endNumber = *template.EndNumber
      }

      for i := startNumber; i <= endNumber; i++ {
         mediaURL := replaceTemplateVars(template.Media, repID, i, 0, bandwidth)
         urls = append(urls, resolveURL(baseURL, mediaURL))
      }
   }

   return urls
}

func processSegmentList(segList *SegmentList, baseURL string) []string {
   var urls []string

   // Add initialization segment if present
   if segList.Initialization != nil && segList.Initialization.SourceURL != "" {
      urls = append(urls, resolveURL(baseURL, segList.Initialization.SourceURL))
   }

   // Add all segment URLs
   for _, segURL := range segList.SegmentURLs {
      if segURL.Media != "" {
         urls = append(urls, resolveURL(baseURL, segURL.Media))
      }
   }

   return urls
}

func processSegmentBase(segBase *SegmentBase, baseURL string) []string {
   var urls []string

   // SegmentBase typically represents a single segment
   // Add initialization if present
   if segBase.Initialization != nil && segBase.Initialization.SourceURL != "" {
      urls = append(urls, resolveURL(baseURL, segBase.Initialization.SourceURL))
   }

   // The base URL itself is the media segment
   urls = append(urls, baseURL)

   return urls
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

   // Starting base URL
   baseURL := "http://test.test/test.mpd"
   if mpd.BaseURL != "" {
      baseURL = resolveURL(baseURL, mpd.BaseURL)
   }

   // Result map
   result := make(map[string][]string)

   // Process each period
   for _, period := range mpd.Periods {
      periodBaseURL := baseURL
      if period.BaseURL != "" {
         periodBaseURL = resolveURL(periodBaseURL, period.BaseURL)
      }

      // Process each adaptation set
      for _, adaptSet := range period.AdaptationSets {
         adaptSetBaseURL := adaptSet.BaseURL

         // Process each representation
         for _, rep := range adaptSet.Representations {
            urls := processRepresentation(rep, periodBaseURL, period.SegmentTemplate,
               period.SegmentList, adaptSetBaseURL, adaptSet.SegmentTemplate,
               adaptSet.SegmentList)

            if len(urls) > 0 {
               result[rep.ID] = urls
            }
         }
      }
   }

   // Output as JSON
   jsonOutput, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error creating JSON output: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonOutput))
}
