package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "os"
   "path"
   "regexp"
   "strconv"
   "strings"
)

// MPD structures
type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL []string `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   BaseURL        []string        `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         []string         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         []string         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
}

type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
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

   mpdPath := os.Args[1]

   // Read MPD file
   xmlData, err := ioutil.ReadFile(mpdPath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
      os.Exit(1)
   }

   // Parse MPD
   var mpd MPD
   err = xml.Unmarshal(xmlData, &mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD: %v\n", err)
      os.Exit(1)
   }

   // Process and extract segment URLs
   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      for _, adaptationSet := range period.AdaptationSets {
         for _, representation := range adaptationSet.Representations {
            urls := extractSegmentURLs(representation, adaptationSet, period, mpd, mpdPath)
            if len(urls) > 0 {
               result[representation.ID] = urls
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

func extractSegmentURLs(rep Representation, adaptSet AdaptationSet, period Period, mpd MPD, mpdPath string) []string {
   var urls []string

   // Get base URL hierarchy
   baseURL := resolveBaseURL(rep.BaseURL, adaptSet.BaseURL, period.BaseURL, mpd.BaseURL, mpdPath)

   // Check for SegmentTemplate (priority: Representation > AdaptationSet)
   segmentTemplate := rep.SegmentTemplate
   if segmentTemplate == nil {
      segmentTemplate = adaptSet.SegmentTemplate
   }

   if segmentTemplate != nil {
      // Handle SegmentTemplate
      urls = processSegmentTemplate(segmentTemplate, rep.ID, baseURL)
   } else if rep.SegmentList != nil {
      // Handle SegmentList
      urls = processSegmentList(rep.SegmentList, baseURL)
   } else if adaptSet.SegmentList != nil {
      // Handle SegmentList from AdaptationSet
      urls = processSegmentList(adaptSet.SegmentList, baseURL)
   } else if len(rep.BaseURL) > 0 || len(baseURL) > 0 {
      // BaseURL only case
      if len(rep.BaseURL) > 0 {
         urls = append(urls, resolveURL(baseURL, rep.BaseURL[0]))
      } else {
         urls = append(urls, baseURL)
      }
   }

   return urls
}

func resolveBaseURL(repBaseURLs, adaptBaseURLs, periodBaseURLs, mpdBaseURLs []string, mpdPath string) string {
   // Start with MPD location as base URL
   // Convert file path to URL format
   var baseURL string
   if strings.HasPrefix(mpdPath, "http://") || strings.HasPrefix(mpdPath, "https://") {
      // If MPD path is already a URL, use it
      baseURL = mpdPath
   } else {
      // Convert file path to URL format: http://test.test/test.mpd
      baseURL = "http://test.test/" + path.Base(mpdPath)
   }

   // Remove the filename to get the directory URL
   lastSlash := strings.LastIndex(baseURL, "/")
   if lastSlash != -1 {
      baseURL = baseURL[:lastSlash+1]
   }

   // Apply MPD BaseURL if exists
   if len(mpdBaseURLs) > 0 {
      baseURL = resolveURL(baseURL, mpdBaseURLs[0])
   }

   // Apply Period BaseURL if exists
   if len(periodBaseURLs) > 0 {
      baseURL = resolveURL(baseURL, periodBaseURLs[0])
   }

   // Apply AdaptationSet BaseURL if exists
   if len(adaptBaseURLs) > 0 {
      baseURL = resolveURL(baseURL, adaptBaseURLs[0])
   }

   return baseURL
}

func resolveURL(base, relative string) string {
   // Handle absolute URLs
   if strings.HasPrefix(relative, "http://") || strings.HasPrefix(relative, "https://") {
      return relative
   }

   // Handle absolute paths (starting with /)
   if strings.HasPrefix(relative, "/") {
      // Extract protocol and host from base URL
      if strings.HasPrefix(base, "http://") || strings.HasPrefix(base, "https://") {
         // Find the third slash (after http://)
         count := 0
         for i, ch := range base {
            if ch == '/' {
               count++
               if count == 3 {
                  return base[:i] + relative
               }
            }
         }
         // If no third slash found, append to base
         return strings.TrimSuffix(base, "/") + relative
      }
      return relative
   }

   // Handle relative URLs
   // Ensure base ends with /
   if !strings.HasSuffix(base, "/") {
      base += "/"
   }

   // Handle ../ in relative path
   for strings.HasPrefix(relative, "../") {
      relative = relative[3:]
      // Remove last directory from base
      base = strings.TrimSuffix(base, "/")
      lastSlash := strings.LastIndex(base, "/")
      if lastSlash != -1 {
         base = base[:lastSlash+1]
      }
   }

   // Handle ./ in relative path
   relative = strings.TrimPrefix(relative, "./")

   // Combine base and relative
   return base + relative
}

func processSegmentTemplate(template *SegmentTemplate, repID string, baseURL string) []string {
   var urls []string

   // Add initialization URL if exists
   if template.Initialization != "" {
      initURL := substituteVariables(template.Initialization, repID, 0, 0)
      urls = append(urls, resolveURL(baseURL, initURL))
   }

   // Process media segments
   if template.Media == "" {
      return urls
   }

   if template.SegmentTimeline != nil {
      // Timeline-based segments
      segmentNumber := template.StartNumber
      if segmentNumber == 0 {
         segmentNumber = 1
      }

      var currentTime int
      for i, s := range template.SegmentTimeline.S {
         // Check if we've reached endNumber
         if template.EndNumber > 0 && segmentNumber > template.EndNumber {
            break
         }

         // Use explicit time if provided, otherwise continue from last time
         if s.T != 0 || i == 0 {
            currentTime = s.T
         }
         duration := s.D
         repeat := s.R

         // Add segment for initial occurrence
         mediaURL := substituteVariables(template.Media, repID, segmentNumber, currentTime)
         urls = append(urls, resolveURL(baseURL, mediaURL))
         segmentNumber++
         currentTime += duration

         // Add repeated segments
         for j := 0; j < repeat; j++ {
            // Check if we've reached endNumber
            if template.EndNumber > 0 && segmentNumber > template.EndNumber {
               break
            }
            mediaURL := substituteVariables(template.Media, repID, segmentNumber, currentTime)
            urls = append(urls, resolveURL(baseURL, mediaURL))
            segmentNumber++
            currentTime += duration
         }
      }
   } else if template.Duration > 0 {
      // Duration-based segments
      startNumber := template.StartNumber
      if startNumber == 0 {
         startNumber = 1
      }

      // Calculate time based on duration
      timescale := template.Timescale
      if timescale == 0 {
         timescale = 1
      }

      // Determine how many segments to generate
      maxSegments := 10 // Default if no endNumber
      if template.EndNumber > 0 {
         maxSegments = template.EndNumber - startNumber + 1
      }

      // Generate segments up to endNumber or default limit
      for i := 0; i < maxSegments; i++ {
         segmentNumber := startNumber + i
         // Stop if we've reached endNumber
         if template.EndNumber > 0 && segmentNumber > template.EndNumber {
            break
         }
         time := i * template.Duration
         mediaURL := substituteVariables(template.Media, repID, segmentNumber, time)
         urls = append(urls, resolveURL(baseURL, mediaURL))
      }
   }

   return urls
}

func processSegmentList(segmentList *SegmentList, baseURL string) []string {
   var urls []string

   // Add initialization URL if exists
   if segmentList.Initialization != nil && segmentList.Initialization.SourceURL != "" {
      urls = append(urls, resolveURL(baseURL, segmentList.Initialization.SourceURL))
   }

   // Add segment URLs
   for _, segURL := range segmentList.SegmentURLs {
      if segURL.Media != "" {
         urls = append(urls, resolveURL(baseURL, segURL.Media))
      }
   }

   return urls
}

func substituteVariables(template string, repID string, number int, time int) string {
   // Replace $RepresentationID$
   result := strings.ReplaceAll(template, "$RepresentationID$", repID)

   // Replace $Number$ with padding
   numberPattern := regexp.MustCompile(`\$Number(%0(\d+)d)?\$`)
   matches := numberPattern.FindStringSubmatch(result)

   if len(matches) > 0 {
      if len(matches) > 2 && matches[2] != "" {
         // Padded number
         width, _ := strconv.Atoi(matches[2])
         paddedNumber := fmt.Sprintf("%0*d", width, number)
         result = numberPattern.ReplaceAllString(result, paddedNumber)
      } else {
         // Unpadded number
         result = numberPattern.ReplaceAllString(result, strconv.Itoa(number))
      }
   }

   // Replace $Time$ with padding support
   timePattern := regexp.MustCompile(`\$Time(%0(\d+)d)?\$`)
   timeMatches := timePattern.FindStringSubmatch(result)

   if len(timeMatches) > 0 {
      if len(timeMatches) > 2 && timeMatches[2] != "" {
         // Padded time
         width, _ := strconv.Atoi(timeMatches[2])
         paddedTime := fmt.Sprintf("%0*d", width, time)
         result = timePattern.ReplaceAllString(result, paddedTime)
      } else {
         // Unpadded time
         result = timePattern.ReplaceAllString(result, strconv.Itoa(time))
      }
   }

   return result
}
