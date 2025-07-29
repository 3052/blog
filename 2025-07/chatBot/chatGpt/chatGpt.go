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

const baseURL = "http://test.test/test.mpd"

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL *string  `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   BaseURL        *string         `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         *string          `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         *string          `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   Duration        int              `xml:"duration,attr"`
   Timescale       int              `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
   EndNumber       int              `xml:"endNumber,attr"` // custom extension
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T int64 `xml:"t,attr"`
   D int64 `xml:"d,attr"`
   R int   `xml:"r,attr"`
}

type SegmentList struct {
   Initialization *URL  `xml:"Initialization"`
   SegmentURLs    []URL `xml:"SegmentURL"`
}

type URL struct {
   Media string `xml:"media,attr"`
}

// resolveHierarchicalBaseURL resolves the BaseURLs from MPD -> Period -> AdaptationSet -> Representation properly.
func resolveHierarchicalBaseURL(mpd *MPD, period *Period, as *AdaptationSet, rep *Representation) string {
   u, err := url.Parse(baseURL)
   if err != nil {
      panic("invalid baseURL constant")
   }

   if mpd.BaseURL != nil {
      u = u.ResolveReference(urlMustParse(*mpd.BaseURL))
   }
   if period.BaseURL != nil {
      u = u.ResolveReference(urlMustParse(*period.BaseURL))
   }
   if as.BaseURL != nil {
      u = u.ResolveReference(urlMustParse(*as.BaseURL))
   }
   if rep.BaseURL != nil {
      u = u.ResolveReference(urlMustParse(*rep.BaseURL))
   }

   return u.String()
}

func urlMustParse(s string) *url.URL {
   u, err := url.Parse(s)
   if err != nil {
      panic(fmt.Sprintf("Invalid URL: %s", s))
   }
   return u
}

func buildSegmentURLs(rep Representation, as AdaptationSet, period Period, mpd MPD) []string {
   base := resolveHierarchicalBaseURL(&mpd, &period, &as, &rep)
   var urls []string

   // SegmentList takes precedence over SegmentTemplate
   if rep.SegmentList != nil {
      if rep.SegmentList.Initialization != nil {
         urls = append(urls, baseURLJoin(base, rep.SegmentList.Initialization.Media))
      }
      for _, s := range rep.SegmentList.SegmentURLs {
         urls = append(urls, baseURLJoin(base, s.Media))
      }
      return urls
   } else if as.SegmentList != nil {
      if as.SegmentList.Initialization != nil {
         urls = append(urls, baseURLJoin(base, as.SegmentList.Initialization.Media))
      }
      for _, s := range as.SegmentList.SegmentURLs {
         urls = append(urls, baseURLJoin(base, s.Media))
      }
      return urls
   }

   // SegmentTemplate fallback (Representation > AdaptationSet)
   st := rep.SegmentTemplate
   if st == nil {
      st = as.SegmentTemplate
   }

   if st == nil {
      // fallback to base URL only, if no segment info available
      return []string{base}
   }

   // Initialization segment
   if st.Initialization != "" {
      initURL := strings.ReplaceAll(st.Initialization, "$RepresentationID$", rep.ID)
      urls = append(urls, baseURLJoin(base, initURL))
   }

   // $Number$ template
   if strings.Contains(st.Media, "$Number$") {
      start := st.StartNumber
      if start == 0 {
         start = 1
      }
      end := st.EndNumber
      if end == 0 {
         // If no endNumber given, fallback to 5 segments max
         end = start + 4
      }
      for i := start; i <= end; i++ {
         segURL := strings.ReplaceAll(st.Media, "$Number$", strconv.Itoa(i))
         segURL = strings.ReplaceAll(segURL, "$RepresentationID$", rep.ID)
         urls = append(urls, baseURLJoin(base, segURL))
      }
      return urls
   }

   // $Time$ template with SegmentTimeline
   if strings.Contains(st.Media, "$Time$") && st.SegmentTimeline != nil {
      time := int64(0)
      for _, s := range st.SegmentTimeline.S {
         repeat := 0
         if s.R > 0 {
            repeat = s.R
         }
         if s.T != 0 {
            time = s.T
         }
         for i := 0; i <= repeat; i++ {
            segURL := strings.ReplaceAll(st.Media, "$Time$", fmt.Sprintf("%d", time))
            segURL = strings.ReplaceAll(segURL, "$RepresentationID$", rep.ID)
            urls = append(urls, baseURLJoin(base, segURL))
            time += s.D
         }
      }
      return urls
   }

   // If media attribute exists but no recognized template, just append it
   if st.Media != "" {
      segURL := strings.ReplaceAll(st.Media, "$RepresentationID$", rep.ID)
      urls = append(urls, baseURLJoin(base, segURL))
      return urls
   }

   // fallback: just return base URL
   return []string{base}
}

func baseURLJoin(base, ref string) string {
   u, err := url.Parse(base)
   if err != nil {
      return ""
   }
   r, err := url.Parse(ref)
   if err != nil {
      return ""
   }
   return u.ResolveReference(r).String()
}

func main() {
   if len(os.Args) != 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path>")
      return
   }

   filePath := os.Args[1]
   data, err := ioutil.ReadFile(filePath)
   if err != nil {
      fmt.Println("Error reading MPD file:", err)
      return
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Println("Error parsing MPD:", err)
      return
   }

   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      for _, as := range period.AdaptationSets {
         for _, rep := range as.Representations {
            urls := buildSegmentURLs(rep, as, period, mpd)
            result[rep.ID] = urls
         }
      }
   }

   output, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Println("Error encoding JSON:", err)
      return
   }
   fmt.Println(string(output))
}
