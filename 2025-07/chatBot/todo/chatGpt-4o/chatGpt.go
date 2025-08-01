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
)

type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   BaseURL                   *string  `xml:"BaseURL"`
   MediaPresentationDuration *string  `xml:"mediaPresentationDuration,attr"`
   Periods                   []Period `xml:"Period"`
}

type Period struct {
   BaseURL        *string         `xml:"BaseURL"`
   Duration       *string         `xml:"duration,attr"`
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
   Timescale       *int             `xml:"timescale,attr"`
   Duration        *int64           `xml:"duration,attr"`
   Initialization  *string          `xml:"initialization,attr"`
   Media           *string          `xml:"media,attr"`
   StartNumber     *int             `xml:"startNumber,attr"`
   EndNumber       *int             `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   Segments []S `xml:"S"`
}

type S struct {
   T *int64 `xml:"t,attr"`
   D int64  `xml:"d,attr"`
   R *int   `xml:"r,attr"`
}

type SegmentList struct {
   Initialization *URL  `xml:"Initialization"`
   SegmentURLs    []URL `xml:"SegmentURL"`
}

type URL struct {
   Media string `xml:"media,attr"`
}

func resolveURL(base *url.URL, parts ...*string) *url.URL {
   u := base
   for _, p := range parts {
      if p != nil {
         ref, err := url.Parse(*p)
         if err == nil {
            u = u.ResolveReference(ref)
         }
      }
   }
   return u
}

func substituteTemplate(template string, values map[string]string) string {
   re := regexp.MustCompile(`\$(\w+)(%0(\d+)d)?\$`)
   return re.ReplaceAllStringFunc(template, func(match string) string {
      m := re.FindStringSubmatch(match)
      key, format := m[1], m[2]
      val, ok := values[key]
      if !ok {
         return match
      }
      if format != "" {
         width, _ := strconv.Atoi(m[3])
         num, _ := strconv.Atoi(val)
         return fmt.Sprintf("%0*d", width, num)
      }
      return val
   })
}

func getSegmentTemplate(rep Representation, aset AdaptationSet) SegmentTemplate {
   if rep.SegmentTemplate != nil {
      return *rep.SegmentTemplate
   }
   if aset.SegmentTemplate != nil {
      return *aset.SegmentTemplate
   }
   return SegmentTemplate{}
}

func getBaseURL(mpd *MPD, period Period, aset AdaptationSet, rep Representation) *url.URL {
   base := "http://test.test/test.mpd"
   u, _ := url.Parse(base)
   return resolveURL(u, mpd.BaseURL, period.BaseURL, aset.BaseURL, rep.BaseURL)
}

func generateSegmentsFromTimeline(tpl SegmentTemplate, base *url.URL, repID string) []string {
   timeline := tpl.SegmentTimeline
   if timeline == nil || tpl.Media == nil {
      return nil
   }

   startNumber := 1
   if tpl.StartNumber != nil {
      startNumber = *tpl.StartNumber
   }

   var urls []string
   var currentTime int64
   for i, s := range timeline.Segments {
      repeat := 0
      if s.R != nil {
         repeat = *s.R
      }
      if s.T != nil {
         currentTime = *s.T
      } else if i == 0 {
         currentTime = 0
      }
      for r := 0; r <= repeat; r++ {
         num := startNumber + len(urls)
         values := map[string]string{
            "RepresentationID": repID,
            "Number":           strconv.Itoa(num),
            "Time":             strconv.FormatInt(currentTime, 10),
         }
         media := substituteTemplate(*tpl.Media, values)
         fullURL := resolveURL(base, &media)
         urls = append(urls, fullURL.String())
         currentTime += s.D
      }
   }
   return urls
}

func generateSegmentsFromTemplate(tpl SegmentTemplate, base *url.URL, repID string, periodDuration float64) []string {
   if tpl.SegmentTimeline != nil {
      return generateSegmentsFromTimeline(tpl, base, repID)
   }
   if tpl.Media == nil {
      return nil
   }

   start := 1
   if tpl.StartNumber != nil {
      start = *tpl.StartNumber
   }

   end := start + 4 // default
   if tpl.EndNumber != nil {
      end = *tpl.EndNumber

   } else if tpl.Duration != nil && periodDuration > 0 {
      dur := float64(*tpl.Duration)
      scale := 1.0
      if tpl.Timescale != nil {
         scale = float64(*tpl.Timescale)
      }
      count := int(math.Ceil(periodDuration * scale / dur))
      end = start + count - 1
   }

   var urls []string
   for i := start; i <= end; i++ {
      values := map[string]string{
         "RepresentationID": repID,
         "Number":           strconv.Itoa(i),
      }
      media := substituteTemplate(*tpl.Media, values)
      fullURL := resolveURL(base, &media)
      urls = append(urls, fullURL.String())
   }
   return urls
}

func parseDurationISO8601(s string) float64 {
   re := regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?`)
   m := re.FindStringSubmatch(s)
   if m == nil {
      return 0
   }
   hours, _ := strconv.Atoi(defaultStr(m[1], "0"))
   mins, _ := strconv.Atoi(defaultStr(m[2], "0"))
   secs, _ := strconv.ParseFloat(defaultStr(m[3], "0"), 64)
   return float64(hours*3600+mins*60) + secs
}

func defaultStr(s, d string) string {
   if s == "" {
      return d
   }
   return s
}

func processRepresentation(mpd *MPD, period Period, aset AdaptationSet, rep Representation) []string {
   base := getBaseURL(mpd, period, aset, rep)
   var urls []string

   if rep.SegmentList != nil || aset.SegmentList != nil {
      slist := rep.SegmentList
      if slist == nil {
         slist = aset.SegmentList
      }
      if slist.Initialization != nil {
         initURL := resolveURL(base, &slist.Initialization.Media)
         urls = append(urls, initURL.String())
      }
      for _, seg := range slist.SegmentURLs {
         segURL := resolveURL(base, &seg.Media)
         urls = append(urls, segURL.String())
      }
      return urls
   }

   tpl := getSegmentTemplate(rep, aset)
   if tpl.Initialization != nil {
      values := map[string]string{
         "RepresentationID": rep.ID,
      }
      init := substituteTemplate(*tpl.Initialization, values)
      initURL := resolveURL(base, &init)
      urls = append(urls, initURL.String())
   }

   periodDur := 0.0
   if period.Duration != nil {
      periodDur = parseDurationISO8601(*period.Duration)
   } else if mpd.MediaPresentationDuration != nil {
      periodDur = parseDurationISO8601(*mpd.MediaPresentationDuration)
   }

   urls = append(urls, generateSegmentsFromTemplate(tpl, base, rep.ID, periodDur)...)

   if len(urls) == 0 {
      urls = append(urls, base.String())
   }
   return urls
}

func main() {
   if len(os.Args) < 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path>")
      os.Exit(1)
   }
   filePath := os.Args[1]
   data, err := ioutil.ReadFile(filePath)
   if err != nil {
      panic(err)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      panic(err)
   }

   result := make(map[string][]string)
   for _, period := range mpd.Periods {
      for _, aset := range period.AdaptationSets {
         for _, rep := range aset.Representations {
            segments := processRepresentation(&mpd, period, aset, rep)
            result[rep.ID] = append(result[rep.ID], segments...)
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   enc.Encode(result)
}
