package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io"
   "net/url"
   "os"
   "path"
   "regexp"
   "strings"
)

type MPD struct {
   XMLName         xml.Name         `xml:"MPD"`
   BaseURL         string           `xml:"BaseURL"`
   Periods         []Period         `xml:"Period"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
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
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   Timescale       int64            `xml:"timescale,attr"`
   StartNumber     int64            `xml:"startNumber,attr"`
   EndNumber       int64            `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T *int64 `xml:"t,attr"` // start time
   D int64  `xml:"d,attr"` // duration
   R *int64 `xml:"r,attr"` // repeat count
}

type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
   Timescale      int64           `xml:"timescale,attr"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: go run main.go <mpd_file_path>\n")
      os.Exit(1)
   }
   mpdFile := os.Args[1]

   f, err := os.Open(mpdFile)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error opening MPD file: %v\n", err)
      os.Exit(1)
   }
   defer f.Close()

   mpdData, err := io.ReadAll(f)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading MPD file: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(mpdData, &mpd); err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD XML: %v\n", err)
      os.Exit(1)
   }

   baseMPDURL, err := url.Parse("http://test.test/test.mpd")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Invalid initial base URL: %v\n", err)
      os.Exit(1)
   }

   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      periodBaseURL := resolveBaseURL(baseMPDURL, mpd.BaseURL, period.BaseURL)

      for _, as := range period.AdaptationSets {
         asBaseURL := resolveBaseURL(periodBaseURL, "", as.BaseURL)

         for _, rep := range as.Representations {
            repBaseURL := resolveBaseURL(asBaseURL, "", rep.BaseURL)

            // SegmentList inheritance
            sl := inheritSegmentList(rep.SegmentList, as.SegmentList, period.SegmentList, mpd.SegmentList)
            if sl != nil {
               segments := buildSegmentsFromSegmentList(sl, repBaseURL)
               result[rep.ID] = segments
               continue
            }

            // SegmentTemplate inheritance
            st := inheritSegmentTemplate(rep.SegmentTemplate, as.SegmentTemplate, period.SegmentTemplate, mpd.SegmentTemplate)
            if st != nil {
               segments := buildSegmentsFromSegmentTemplate(st, rep.ID, repBaseURL)
               result[rep.ID] = segments
               continue
            }

            // Fallback: Use Representation BaseURL as segments (fully resolved)
            baseURLSegments := []string{}
            if rep.BaseURL != "" {
               baseURLSegments = append(baseURLSegments, repBaseURL.String())
            }
            result[rep.ID] = baseURLSegments

         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   if err := enc.Encode(result); err != nil {
      fmt.Fprintf(os.Stderr, "Error encoding JSON output: %v\n", err)
      os.Exit(1)
   }
}

func buildSegmentsFromSegmentList(sl *SegmentList, baseURL *url.URL) []string {
   var segments []string

   if sl.Initialization != nil && sl.Initialization.SourceURL != "" {
      initURL, err := resolveURL(baseURL, sl.Initialization.SourceURL)
      if err == nil {
         segments = append(segments, initURL.String())
      }
   }

   for _, seg := range sl.SegmentURLs {
      if seg.Media == "" {
         continue
      }
      segURL, err := resolveURL(baseURL, seg.Media)
      if err == nil {
         segments = append(segments, segURL.String())
      }
   }

   return segments
}

func buildSegmentsFromSegmentTemplate(st *SegmentTemplate, repID string, baseURL *url.URL) []string {
   if st.Timescale == 0 {
      st.Timescale = 1
   }
   startNumber := st.StartNumber
   if startNumber == 0 {
      startNumber = 1
   }

   segments := []string{}
   if st.Initialization != "" {
      initURL, err := resolveURL(baseURL, substituteTemplate(st.Initialization, repID, 0, 0))
      if err == nil {
         segments = append(segments, initURL.String())
      }
   }
   if st.Media == "" {
      return segments
   }

   if st.SegmentTimeline != nil && len(st.SegmentTimeline.S) > 0 {
      timeline := expandSegmentTimeline(st.SegmentTimeline)

      endNumber := st.EndNumber
      if endNumber == 0 {
         endNumber = startNumber + int64(len(timeline)) - 1
      }

      count := endNumber - startNumber + 1
      if count > int64(len(timeline)) {
         count = int64(len(timeline))
      }
      if count < 0 {
         count = 0
      }

      for i := int64(0); i < count; i++ {
         segmentNumber := startNumber + i
         segmentTime := timeline[i]
         urlStr := substituteTemplate(st.Media, repID, segmentNumber, segmentTime)
         fullURL, err := resolveURL(baseURL, urlStr)
         if err != nil {
            continue
         }
         segments = append(segments, fullURL.String())
      }
   } else {
      endNumber := st.EndNumber
      if endNumber == 0 {
         endNumber = startNumber + 999
      }
      for num := startNumber; num <= endNumber; num++ {
         urlStr := substituteTemplate(st.Media, repID, num, 0)
         fullURL, err := resolveURL(baseURL, urlStr)
         if err != nil {
            continue
         }
         segments = append(segments, fullURL.String())
      }
   }
   return segments
}

func resolveBaseURL(parentBase *url.URL, higherLevelBase, lowerLevelBase string) *url.URL {
   lower := strings.TrimSpace(lowerLevelBase)
   if lower != "" {
      u, err := url.Parse(lower)
      if err == nil {
         return parentBase.ResolveReference(u)
      }
   }
   higher := strings.TrimSpace(higherLevelBase)
   if higher != "" {
      u, err := url.Parse(higher)
      if err == nil {
         return parentBase.ResolveReference(u)
      }
   }
   return parentBase
}

func inheritSegmentList(repSL, asSL, periodSL, mpdSL *SegmentList) *SegmentList {
   if repSL != nil {
      return repSL
   }
   if asSL != nil {
      return asSL
   }
   if periodSL != nil {
      return periodSL
   }
   return mpdSL
}

func inheritSegmentTemplate(repST, asST, periodST, mpdST *SegmentTemplate) *SegmentTemplate {
   if repST != nil {
      return mergeSegmentTemplate(repST, asST)
   }
   if asST != nil {
      return mergeSegmentTemplate(asST, periodST)
   }
   if periodST != nil {
      return mergeSegmentTemplate(periodST, mpdST)
   }
   return mpdST
}

func mergeSegmentTemplate(child, parent *SegmentTemplate) *SegmentTemplate {
   if child == nil && parent == nil {
      return nil
   }
   if child == nil {
      return parent
   }
   if parent == nil {
      return child
   }

   merged := *parent

   if child.Initialization != "" {
      merged.Initialization = child.Initialization
   }
   if child.Media != "" {
      merged.Media = child.Media
   }
   if child.Timescale != 0 {
      merged.Timescale = child.Timescale
   }
   if child.StartNumber != 0 {
      merged.StartNumber = child.StartNumber
   }
   if child.EndNumber != 0 {
      merged.EndNumber = child.EndNumber
   }
   if child.SegmentTimeline != nil {
      merged.SegmentTimeline = child.SegmentTimeline
   }

   return &merged
}

func expandSegmentTimeline(timeline *SegmentTimeline) []int64 {
   var result []int64
   var lastT int64 = 0
   for i, s := range timeline.S {
      t := int64(0)
      if s.T != nil {
         t = *s.T
         lastT = t
      } else if i == 0 {
         t = 0
         lastT = 0
      } else {
         t = lastT
      }

      r := int64(0)
      if s.R != nil {
         r = *s.R
      }

      for j := int64(0); j <= r; j++ {
         result = append(result, t)
         t += s.D
      }
      lastT = t
   }
   return result
}

func substituteTemplate(template, repID string, number, time int64) string {
   re := regexp.MustCompile(`\$(RepresentationID|Number|Time)(%[^$]+)?\$`)
   return re.ReplaceAllStringFunc(template, func(match string) string {
      parts := re.FindStringSubmatch(match)
      if len(parts) < 2 {
         return match
      }
      varName := parts[1]
      format := "%d"
      if len(parts) == 3 && parts[2] != "" {
         format = parts[2]
      }
      switch varName {
      case "RepresentationID":
         return repID
      case "Number":
         return fmt.Sprintf(format, number)
      case "Time":
         return fmt.Sprintf(format, time)
      default:
         return match
      }
   })
}

func resolveURL(baseURL *url.URL, ref string) (*url.URL, error) {
   refURL, err := url.Parse(ref)
   if err != nil {
      return nil, err
   }
   resolved := baseURL.ResolveReference(refURL)
   resolved.Path = path.Clean(resolved.Path)
   return resolved, nil
}
