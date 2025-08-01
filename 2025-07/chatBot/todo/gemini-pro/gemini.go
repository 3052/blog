package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io"
   "log"
   "math"
   "net/url"
   "os"
   "regexp"
   "strconv"
)

// topLevelBaseURL is the hardcoded top-level base for resolving all relative paths.
const topLevelBaseURL = "http://test.test/test.mpd"

// #############################################################################
// ## 📜 XML Data Structures
// #############################################################################

// MPD is the root element of the MPEG-DASH manifest.
type MPD struct {
   XMLName                   xml.Name  `xml:"MPD"`
   MediaPresentationDuration string    `xml:"mediaPresentationDuration,attr"`
   Type                      string    `xml:"type,attr"`
   BaseURL                   []BaseURL `xml:"BaseURL"`
   Periods                   []Period  `xml:"Period"`
}

// Period represents a single period in the manifest.
type Period struct {
   ID             string          `xml:"id,attr"`
   Duration       string          `xml:"duration,attr"`
   BaseURL        []BaseURL       `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet groups representations of one or more media content components.
type AdaptationSet struct {
   ID              string           `xml:"id,attr"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Initialization  *Initialization  `xml:"Initialization"`
   Representations []Representation `xml:"Representation"`
}

// Representation describes a specific version of the content.
type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         []BaseURL        `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Initialization  *Initialization  `xml:"Initialization"`
}

// BaseURL specifies a base URL at various levels of the manifest.
type BaseURL struct {
   Value string `xml:",chardata"`
}

// SegmentTemplate defines properties for segment URL generation using a template.
type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   Timescale       uint64           `xml:"timescale,attr"`
   StartNumber     uint64           `xml:"startNumber,attr"`
   EndNumber       uint64           `xml:"endNumber,attr"`
   Duration        uint64           `xml:"duration,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline provides an explicit list of segments and their properties.
type SegmentTimeline struct {
   S []S `xml:"S"`
}

// S represents a segment in the SegmentTimeline.
type S struct {
   T uint64 `xml:"t,attr"` // Time
   D uint64 `xml:"d,attr"` // Duration
   R int64  `xml:"r,attr"` // Repeat count
}

// SegmentList provides an explicit list of segment URLs.
type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

// Initialization specifies the initialization segment URL.
type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

// SegmentURL provides the URL for a single media segment.
type SegmentURL struct {
   Media string `xml:"media,attr"`
}

// #############################################################################
// ## ⚙️ Main Execution
// #############################################################################

func main() {
   // 1. Check for command-line argument
   if len(os.Args) < 2 {
      log.Fatalf("Usage: go run main.go <path_to_mpd_file>")
   }
   filePath := os.Args[1]

   // 2. Open and read the MPD file
   xmlFile, err := os.Open(filePath)
   if err != nil {
      log.Fatalf("Error opening file %s: %v", filePath, err)
   }
   defer xmlFile.Close()

   byteValue, err := io.ReadAll(xmlFile)
   if err != nil {
      log.Fatalf("Error reading file %s: %v", filePath, err)
   }

   // 3. Unmarshal the XML content into structs
   var mpd MPD
   if err := xml.Unmarshal(byteValue, &mpd); err != nil {
      log.Fatalf("Error unmarshalling XML: %v", err)
   }

   // 4. Process the MPD to generate segment URLs
   result, err := processMPD(&mpd)
   if err != nil {
      log.Fatalf("Error processing MPD: %v", err)
   }

   // 5. Marshal the result map into indented JSON
   jsonOutput, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      log.Fatalf("Error marshalling result to JSON: %v", err)
   }

   // 6. Print the final JSON to standard output
   fmt.Println(string(jsonOutput))
}

// #############################################################################
// ## 🛠️ Processing and Helper Functions
// #############################################################################

// processMPD orchestrates the parsing of the MPD structure to generate segment URLs.
func processMPD(mpd *MPD) (map[string][]string, error) {
   results := make(map[string][]string)

   // Start with the hardcoded top-level base URL.
   rootURL, err := url.Parse(topLevelBaseURL)
   if err != nil {
      return nil, fmt.Errorf("invalid top-level base URL: %w", err)
   }

   // Resolve MPD-level BaseURL.
   mpdBaseURL := resolveBaseURL(rootURL, mpd.BaseURL)

   for _, period := range mpd.Periods {
      // Resolve Period-level BaseURL against the MPD base.
      periodBaseURL := resolveBaseURL(mpdBaseURL, period.BaseURL)

      for _, adaptSet := range period.AdaptationSets {
         for _, rep := range adaptSet.Representations {
            // Resolve Representation-level BaseURL against the Period base.
            repBaseURL := resolveBaseURL(periodBaseURL, rep.BaseURL)

            // Get effective elements by handling inheritance from AdaptationSet.
            effSegmentTemplate := getEffectiveSegmentTemplate(rep, adaptSet)
            effSegmentList := getEffectiveSegmentList(rep, adaptSet)

            // Always get the start number from the current period's context.
            startNumberForThisPeriod := getStartNumber(effSegmentTemplate)

            // Generate media segments for the current period.
            mediaURLs, _ := generateSegmentURLs(
               rep,
               &period,
               mpd,
               effSegmentTemplate,
               effSegmentList,
               repBaseURL.String(),
               periodBaseURL, // Pass the correct parent base URL for the fallback case
               startNumberForThisPeriod,
            )

            // Find the initialization segment URL for the current period's representation.
            initURL, err := getInitURL(rep, adaptSet, effSegmentTemplate, effSegmentList, repBaseURL.String())
            if err != nil {
               log.Printf("Warning: could not get init URL for rep %s: %v", rep.ID, err)
            }

            // Assemble the list of URLs for this period (init + media).
            var currentPeriodSegments []string
            if initURL != "" {
               currentPeriodSegments = append(currentPeriodSegments, initURL)
            }
            currentPeriodSegments = append(currentPeriodSegments, mediaURLs...)

            // Append this period's list to the main results for the representation ID.
            results[rep.ID] = append(results[rep.ID], currentPeriodSegments...)
         }
      }
   }
   return results, nil
}

// --- URL and Inheritance Helpers ---

// resolveBaseURL computes the new base URL for a given level.
func resolveBaseURL(parentBase *url.URL, baseURLElements []BaseURL) *url.URL {
   currentBase := parentBase
   // Per DASH-IF guidelines, only the first BaseURL element is used for base resolution.
   if len(baseURLElements) > 0 {
      resolved, err := parentBase.Parse(baseURLElements[0].Value)
      if err == nil {
         currentBase = resolved
      }
   }
   return currentBase
}

// getEffectiveSegmentTemplate returns the Representation's SegmentTemplate or inherits from the AdaptationSet.
func getEffectiveSegmentTemplate(rep Representation, as AdaptationSet) *SegmentTemplate {
   if rep.SegmentTemplate != nil {
      return rep.SegmentTemplate
   }
   return as.SegmentTemplate
}

// getEffectiveSegmentList returns the Representation's SegmentList or inherits from the AdaptationSet.
func getEffectiveSegmentList(rep Representation, as AdaptationSet) *SegmentList {
   if rep.SegmentList != nil {
      return rep.SegmentList
   }
   return as.SegmentList
}

// getEffectiveInitialization returns the Representation's Initialization or inherits from the AdaptationSet.
func getEffectiveInitialization(rep Representation, as AdaptationSet) *Initialization {
   if rep.Initialization != nil {
      return rep.Initialization
   }
   return as.Initialization
}

// --- Initialization Segment Logic ---

// getInitURL finds the initialization segment URL by checking definitions in order of precedence.
func getInitURL(rep Representation, as AdaptationSet, st *SegmentTemplate, sl *SegmentList, repBaseURL string) (string, error) {
   var initPath string

   // 1. Direct <Initialization> on <Representation>
   effInit := getEffectiveInitialization(rep, as)
   if effInit != nil && effInit.SourceURL != "" {
      initPath = effInit.SourceURL
   }

   // 2. <Initialization> child of effective <SegmentList>
   if initPath == "" && sl != nil && sl.Initialization != nil && sl.Initialization.SourceURL != "" {
      initPath = sl.Initialization.SourceURL
   }

   // 3. `initialization` attribute on effective <SegmentTemplate>
   if initPath == "" && st != nil && st.Initialization != "" {
      initPath = st.Initialization
   }

   if initPath == "" {
      return "", nil // No initialization segment found.
   }

   // Substitute placeholders and resolve the final URL.
   replacements := map[string]interface{}{"RepresentationID": rep.ID}
   finalPath := substitutePlaceholders(initPath, replacements)

   baseURL, err := url.Parse(repBaseURL)
   if err != nil {
      return "", err
   }
   resolvedURL, err := baseURL.Parse(finalPath)
   if err != nil {
      return "", err
   }

   return resolvedURL.String(), nil
}

// --- Media Segment Generation Logic ---

// generateSegmentURLs creates the list of media segments using the first available method by precedence.
func generateSegmentURLs(rep Representation, period *Period, mpd *MPD, st *SegmentTemplate, sl *SegmentList, repBaseURL string, parentForFallbackURL *url.URL, startNum uint64) ([]string, uint64) {
   urls := []string{}
   segmentNumber := startNum

   // Pre-calculate base URL object.
   baseURL, err := url.Parse(repBaseURL)
   if err != nil {
      log.Printf("Warning: invalid base URL for rep %s: %s", rep.ID, repBaseURL)
      return urls, segmentNumber
   }

   // 1. Primary: <SegmentTimeline>
   if st != nil && st.SegmentTimeline != nil && len(st.SegmentTimeline.S) > 0 {
      currentTime := uint64(0)
      if len(st.SegmentTimeline.S) > 0 {
         // First S element might not have 't', in which case time starts at 0.
         currentTime = st.SegmentTimeline.S[0].T
      }

      for _, s := range st.SegmentTimeline.S {
         if s.T > 0 {
            currentTime = s.T
         }
         // Repeat count `r` means r+1 total instances. `r=-1` is not handled per prompt.
         for i := int64(0); i <= s.R; i++ {
            replacements := map[string]interface{}{
               "RepresentationID": rep.ID,
               "Number":           segmentNumber,
               "Time":             currentTime,
            }
            segmentPath := substitutePlaceholders(st.Media, replacements)
            resolvedURL, _ := baseURL.Parse(segmentPath)
            urls = append(urls, resolvedURL.String())

            currentTime += s.D
            segmentNumber++
         }
      }
      return urls, segmentNumber
   }

   // 2. Secondary: <SegmentList>
   if sl != nil && len(sl.SegmentURLs) > 0 {
      for _, segURL := range sl.SegmentURLs {
         resolvedURL, _ := baseURL.Parse(segURL.Media)
         urls = append(urls, resolvedURL.String())
      }
      return urls, segmentNumber
   }

   if st != nil {
      // 3. Tertiary: Number-based Template
      if st.EndNumber > 0 {
         start := getStartNumber(st)
         for i := start; i <= st.EndNumber; i++ {
            replacements := map[string]interface{}{
               "RepresentationID": rep.ID,
               "Number":           i,
            }
            segmentPath := substitutePlaceholders(st.Media, replacements)
            resolvedURL, _ := baseURL.Parse(segmentPath)
            urls = append(urls, resolvedURL.String())
            segmentNumber++
         }
         return urls, segmentNumber
      }

      // 4. Quaternary: Duration-based Template
      if st.Duration > 0 {
         timescale := getTimescale(st)
         start := getStartNumber(st)

         // Get period duration in seconds from Period@duration or MPD@mediaPresentationDuration.
         var totalDurationSec float64
         if period.Duration != "" {
            totalDurationSec, _ = parseISODuration(period.Duration)
         } else if mpd.MediaPresentationDuration != "" {
            totalDurationSec, _ = parseISODuration(mpd.MediaPresentationDuration)
         }

         if totalDurationSec > 0 {
            segmentCount := math.Ceil((totalDurationSec * float64(timescale)) / float64(st.Duration))
            for i := 0; i < int(segmentCount); i++ {
               currentNumber := start + uint64(i)
               currentTime := uint64(i) * st.Duration
               replacements := map[string]interface{}{
                  "RepresentationID": rep.ID,
                  "Number":           currentNumber,
                  "Time":             currentTime,
               }
               segmentPath := substitutePlaceholders(st.Media, replacements)
               resolvedURL, _ := baseURL.Parse(segmentPath)
               urls = append(urls, resolvedURL.String())
               segmentNumber++
            }
            return urls, segmentNumber
         }
      }
   }

   // 5. Final Fallback: <BaseURL> list on Representation
   if len(rep.BaseURL) > 0 {
      // They are resolved against the *parent* base URL, which is parentForFallbackURL.
      for _, bu := range rep.BaseURL {
         resolvedURL, _ := parentForFallbackURL.Parse(bu.Value)
         urls = append(urls, resolvedURL.String())
      }
      return urls, segmentNumber
   }

   return urls, segmentNumber
}

// --- Utility Functions ---

var placeholderRegex = regexp.MustCompile(`\$(\w+?)(?:(%.*?d))?\$`)

// substitutePlaceholders replaces identifiers like $RepresentationID$ and $Number%05d$ in a template string.
func substitutePlaceholders(template string, replacements map[string]interface{}) string {
   return placeholderRegex.ReplaceAllStringFunc(template, func(match string) string {
      submatches := placeholderRegex.FindStringSubmatch(match)
      identifier := submatches[1]
      formatSpecifier := submatches[2]

      if val, ok := replacements[identifier]; ok {
         if formatSpecifier != "" {
            return fmt.Sprintf(formatSpecifier, val)
         }
         return fmt.Sprint(val)
      }
      return match // Return original placeholder if no replacement is found
   })
}

var durationRegex = regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)`)

// parseISODuration converts an ISO 8601 duration string (like "PT1M30.5S") to total seconds.
func parseISODuration(duration string) (float64, error) {
   matches := durationRegex.FindStringSubmatch(duration)
   if matches == nil {
      return 0, fmt.Errorf("invalid ISO 8601 duration format: %s", duration)
   }

   var totalSeconds float64
   // Hours
   if matches[1] != "" {
      h, _ := strconv.Atoi(matches[1])
      totalSeconds += float64(h) * 3600
   }
   // Minutes
   if matches[2] != "" {
      m, _ := strconv.Atoi(matches[2])
      totalSeconds += float64(m) * 60
   }
   // Seconds
   if matches[3] != "" {
      s, _ := strconv.ParseFloat(matches[3], 64)
      totalSeconds += s
   }

   return totalSeconds, nil
}

// getStartNumber returns the StartNumber from a SegmentTemplate, defaulting to 1.
func getStartNumber(st *SegmentTemplate) uint64 {
   if st != nil && st.StartNumber > 0 {
      return st.StartNumber
   }
   return 1
}

// getTimescale returns the Timescale from a SegmentTemplate, defaulting to 1.
func getTimescale(st *SegmentTemplate) uint64 {
   if st != nil && st.Timescale > 0 {
      return st.Timescale
   }
   return 1
}
