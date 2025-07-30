package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "log"
   "net/url"
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
   T *int `xml:"t,attr"`
   D int  `xml:"d,attr"`
   R *int `xml:"r,attr"`
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
   if len(os.Args) < 2 {
      log.Fatal("Usage: go run main.go <mpd_file_path>")
   }

   mpdPath := os.Args[1]
   baseURL := "http://test.test/test.mpd"

   // Read MPD file
   content, err := os.ReadFile(mpdPath)
   if err != nil {
      log.Fatalf("Error reading file: %v", err)
   }

   // Parse MPD
   var mpd MPD
   if err := xml.Unmarshal(content, &mpd); err != nil {
      log.Fatalf("Error parsing XML: %v", err)
   }

   // Process and extract URLs
   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      for _, adaptationSet := range period.AdaptationSets {
         for _, representation := range adaptationSet.Representations {
            urls := processRepresentation(representation, adaptationSet, period, mpd, baseURL)
            // Append to existing URLs for this representation ID
            result[representation.ID] = append(result[representation.ID], urls...)
         }
      }
   }

   // Output JSON
   output, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      log.Fatalf("Error creating JSON: %v", err)
   }
   fmt.Println(string(output))
}

func processRepresentation(rep Representation, as AdaptationSet, period Period, mpd MPD, startBaseURL string) []string {
   var urls []string

   // Resolve base URL hierarchy
   baseURL := resolveBaseURL(startBaseURL, mpd.BaseURL, period.BaseURL, as.BaseURL, rep.BaseURL)

   // Determine which segment template/list to use (representation takes precedence)
   segmentTemplate := rep.SegmentTemplate
   if segmentTemplate == nil {
      segmentTemplate = as.SegmentTemplate
   }

   segmentList := rep.SegmentList
   if segmentList == nil {
      segmentList = as.SegmentList
   }

   if segmentTemplate != nil {
      urls = processSegmentTemplate(segmentTemplate, rep.ID, baseURL)
   } else if segmentList != nil {
      urls = processSegmentList(segmentList, baseURL)
   } else if len(rep.BaseURL) > 0 || len(as.BaseURL) > 0 || len(period.BaseURL) > 0 || len(mpd.BaseURL) > 0 {
      // Only BaseURL, single segment
      urls = append(urls, baseURL)
   }

   return urls
}

func processSegmentTemplate(st *SegmentTemplate, repID string, baseURL string) []string {
   var urls []string

   // Add initialization URL if present
   if st.Initialization != "" {
      initURL := substituteVariables(st.Initialization, repID, 0, 0)
      urls = append(urls, resolveURL(baseURL, initURL))
   }

   // Process segments
   if st.SegmentTimeline != nil {
      urls = append(urls, processSegmentTimeline(st, repID, baseURL)...)
   } else if st.Duration != nil {
      urls = append(urls, processDurationBasedTemplate(st, repID, baseURL)...)
   }

   return urls
}

func processSegmentTimeline(st *SegmentTemplate, repID string, baseURL string) []string {
   var urls []string
   time := 0
   number := 1
   if st.StartNumber != nil {
      number = *st.StartNumber
   }

   for _, s := range st.SegmentTimeline.S {
      if s.T != nil {
         time = *s.T
      }

      repeat := 0
      if s.R != nil {
         repeat = *s.R
      }

      for i := 0; i <= repeat; i++ {
         if st.EndNumber != nil && number > *st.EndNumber {
            break
         }

         mediaURL := substituteVariables(st.Media, repID, number, time)
         urls = append(urls, resolveURL(baseURL, mediaURL))

         time += s.D
         number++
      }
   }

   return urls
}

func processDurationBasedTemplate(st *SegmentTemplate, repID string, baseURL string) []string {
   var urls []string
   startNumber := 1
   if st.StartNumber != nil {
      startNumber = *st.StartNumber
   }

   endNumber := startNumber + 9 // Default to 10 segments
   if st.EndNumber != nil {
      endNumber = *st.EndNumber
   }

   for number := startNumber; number <= endNumber; number++ {
      mediaURL := substituteVariables(st.Media, repID, number, 0)
      urls = append(urls, resolveURL(baseURL, mediaURL))
   }

   return urls
}

func processSegmentList(sl *SegmentList, baseURL string) []string {
   var urls []string

   // Add initialization URL if present
   if sl.Initialization != nil && sl.Initialization.SourceURL != "" {
      urls = append(urls, resolveURL(baseURL, sl.Initialization.SourceURL))
   }

   // Add segment URLs
   for _, segURL := range sl.SegmentURLs {
      if segURL.Media != "" {
         urls = append(urls, resolveURL(baseURL, segURL.Media))
      }
   }

   return urls
}

func substituteVariables(template string, repID string, number int, time int) string {
   result := template

   // Replace $RepresentationID$
   result = strings.ReplaceAll(result, "$RepresentationID$", repID)

   // Replace $Number$ with padding
   numberRe := regexp.MustCompile(`\$Number(%0\d+d)?\$`)
   result = numberRe.ReplaceAllStringFunc(result, func(match string) string {
      if strings.Contains(match, "%") {
         // Extract padding format
         formatRe := regexp.MustCompile(`%0(\d+)d`)
         formatMatch := formatRe.FindStringSubmatch(match)
         if len(formatMatch) > 1 {
            width, _ := strconv.Atoi(formatMatch[1])
            return fmt.Sprintf("%0*d", width, number)
         }
      }
      return strconv.Itoa(number)
   })

   // Replace $Time$ with padding
   timeRe := regexp.MustCompile(`\$Time(%0\d+d)?\$`)
   result = timeRe.ReplaceAllStringFunc(result, func(match string) string {
      if strings.Contains(match, "%") {
         // Extract padding format
         formatRe := regexp.MustCompile(`%0(\d+)d`)
         formatMatch := formatRe.FindStringSubmatch(match)
         if len(formatMatch) > 1 {
            width, _ := strconv.Atoi(formatMatch[1])
            return fmt.Sprintf("%0*d", width, time)
         }
      }
      return strconv.Itoa(time)
   })

   return result
}

func resolveBaseURL(startURL string, mpdURLs, periodURLs, asURLs, repURLs []string) string {
   baseURL := startURL

   // Apply hierarchical BaseURL resolution
   for _, urls := range [][]string{mpdURLs, periodURLs, asURLs, repURLs} {
      if len(urls) > 0 {
         baseURL = resolveURL(baseURL, urls[0])
      }
   }

   return baseURL
}

func resolveURL(baseURL, relativeURL string) string {
   // If relativeURL is already absolute, return it
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
      return relativeURL
   }

   // Resolve relative to base
   resolved := base.ResolveReference(rel)
   return resolved.String()
}

func getBaseFromPath(filePath string) string {
   dir := path.Dir(filePath)
   if dir == "." {
      return ""
   }
   return dir + "/"
}
