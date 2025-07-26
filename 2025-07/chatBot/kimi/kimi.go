package main

import (
   "encoding/json"
   "encoding/xml"
   "errors"
   "fmt"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
   "time"
)

/* ---------- CLI entry ---------- */
func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "usage: %s /path/to/local.mpd\n", os.Args[0])
      os.Exit(1)
   }
   mpdPath := os.Args[1]

   var mpd MPD
   if err := loadMPD(mpdPath, &mpd); err != nil {
      fmt.Fprintf(os.Stderr, "load MPD: %v\n", err)
      os.Exit(1)
   }

   // Fixed MPD URL from the requirement
   mpdURL := "http://test.test/test.mpd"

   urls, err := expandMPD(&mpd, mpdURL)
   if err != nil {
      fmt.Fprintf(os.Stderr, "expand MPD: %v\n", err)
      os.Exit(1)
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetEscapeHTML(false)
   if err := enc.Encode(urls); err != nil {
      fmt.Fprintf(os.Stderr, "encode JSON: %v\n", err)
      os.Exit(1)
   }
}

/* ---------- XML types ---------- */
type MPD struct {
   XMLName   xml.Name  `xml:"MPD"`
   BaseURL   string    `xml:"BaseURL"`
   Type      string    `xml:"type,attr"`
   MediaPresentationDuration string `xml:"mediaPresentationDuration,attr"`
   Periods   []Period  `xml:"Period"`
}

type Period struct {
   BaseURL string `xml:"BaseURL"`
   Duration string `xml:"duration,attr"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL string `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList *SegmentList `xml:"SegmentList"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID string `xml:"id,attr"`
   BaseURL string `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList *SegmentList `xml:"SegmentList"`
}

type SegmentTemplate struct {
   Template    string `xml:"media,attr"`
   StartNumber *int   `xml:"startNumber,attr"`
   EndNumber   *int   `xml:"endNumber,attr"`
   Timescale   int    `xml:"timescale,attr"`
   Duration    int    `xml:"duration,attr"`
   Timeline    *Timeline `xml:"SegmentTimeline"`
}

type SegmentList struct {
   Timeline   *Timeline   `xml:"SegmentTimeline"`
   URLs       []SegmentURL `xml:"SegmentURL"`
   StartNumber *int `xml:"startNumber,attr"`
   EndNumber   *int `xml:"endNumber,attr"`
   Timescale  int `xml:"timescale,attr"`
   Duration   int `xml:"duration,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

type Timeline struct {
   S []S `xml:"S"`
}

type S struct {
   T int `xml:"t,attr"`
   D int `xml:"d,attr"`
   R int `xml:"r,attr"`
}

/* ---------- Helpers ---------- */
func loadMPD(file string, mpd *MPD) error {
   f, err := os.Open(file)
   if err != nil {
      return err
   }
   defer f.Close()
   return xml.NewDecoder(f).Decode(mpd)
}

/* ---------- URL resolution using only net/url ---------- */
func resolveBaseURL(parent string, bases ...string) (string, error) {
   base, err := url.Parse(parent)
   if err != nil {
      return "", err
   }
   for _, b := range bases {
      b = strings.TrimSpace(b)
      if b == "" {
         continue
      }
      ref, err := url.Parse(b)
      if err != nil {
         return "", err
      }
      base = base.ResolveReference(ref)
   }
   return base.String(), nil
}

/* ---------- MPD expansion ---------- */
func expandMPD(mpd *MPD, mpdURL string) (map[string][]string, error) {
   out := make(map[string][]string)

   globalDur, err := parseDuration(mpd.MediaPresentationDuration)
   if err != nil {
      return nil, fmt.Errorf("invalid mediaPresentationDuration: %w", err)
   }

   for _, period := range mpd.Periods {
      periodDur := globalDur
      if period.Duration != "" {
         d, err := parseDuration(period.Duration)
         if err != nil {
            return nil, fmt.Errorf("invalid Period@duration: %w", err)
         }
         periodDur = d
      }

      periodBase, err := resolveBaseURL(mpdURL, mpd.BaseURL, period.BaseURL)
      if err != nil {
         return nil, err
      }

      for _, as := range period.AdaptationSets {
         asBase, err := resolveBaseURL(periodBase, as.BaseURL)
         if err != nil {
            return nil, err
         }

         for _, rep := range as.Representations {
            repBase, err := resolveBaseURL(asBase, rep.BaseURL)
            if err != nil {
               return nil, err
            }

            segTempl := chooseSegTemplate(as.SegmentTemplate, rep.SegmentTemplate)
            segList := chooseSegList(as.SegmentList, rep.SegmentList)

            var segments []string
            switch {
            case segTempl != nil:
               segments, err = expandTemplate(rep.ID, segTempl, periodDur, repBase)
            case segList != nil:
               segments, err = expandList(repBase, segList, periodDur)
            default:
               segments = []string{repBase}
            }
            if err != nil {
               return nil, fmt.Errorf("representation %s: %w", rep.ID, err)
            }

            // Rule 8 & 10: append unique
            seen := make(map[string]bool)
            for _, u := range out[rep.ID] {
               seen[u] = true
            }
            for _, u := range segments {
               if !seen[u] {
                  out[rep.ID] = append(out[rep.ID], u)
                  seen[u] = true
               }
            }
         }
      }
   }
   return out, nil
}

func chooseSegTemplate(asT *SegmentTemplate, repT *SegmentTemplate) *SegmentTemplate {
   if repT != nil {
      return repT
   }
   return asT
}

func chooseSegList(asL *SegmentList, repL *SegmentList) *SegmentList {
   if repL != nil {
      return repL
   }
   return asL
}

/* ---------- template expansion ---------- */
func expandTemplate(repID string, st *SegmentTemplate, periodDur time.Duration, base string) ([]string, error) {
   tmpl := st.Template
   if tmpl == "" {
      return nil, errors.New("SegmentTemplate@media missing")
   }
   timescale := st.Timescale
   if timescale == 0 {
      timescale = 1
   }

   startNum := 1
   if st.StartNumber != nil {
      startNum = *st.StartNumber
   }

   // timeline mode
   if st.Timeline != nil {
      timeVal := 0
      var segments []string
      for _, s := range st.Timeline.S {
         if s.T != 0 {
            timeVal = s.T
         }
         repeats := 1 + s.R
         for i := 0; i < repeats; i++ {
            url := fillTemplate(tmpl, map[string]int{
               "RepresentationID": 0,
               "Number":           startNum,
               "Time":             timeVal,
            }, repID)
            abs, err := resolveBaseURL(base, url)
            if err != nil {
               return nil, err
            }
            segments = append(segments, abs)
            startNum++
            timeVal += s.D
         }
      }
      return segments, nil
   }

   // simple mode: @startNumber … @endNumber
   var endNum int
   if st.EndNumber != nil {
      endNum = *st.EndNumber
   } else {
      if st.Duration == 0 {
         return nil, errors.New("SegmentTemplate@duration missing and no SegmentTimeline nor @endNumber")
      }
      periodTicks := int(periodDur.Seconds() * float64(timescale))
      count := (periodTicks + st.Duration - 1) / st.Duration
      endNum = startNum + count - 1
   }

   var segments []string
   for num := startNum; num <= endNum; num++ {
      timeVal := (num - startNum) * st.Duration
      url := fillTemplate(tmpl, map[string]int{
         "RepresentationID": 0,
         "Number":           num,
         "Time":             timeVal,
      }, repID)
      abs, err := resolveBaseURL(base, url)
      if err != nil {
         return nil, err
      }
      segments = append(segments, abs)
   }
   return segments, nil
}

func expandList(base string, sl *SegmentList, periodDur time.Duration) ([]string, error) {
   // explicit <SegmentURL media="…">
   if len(sl.URLs) > 0 {
      var out []string
      for _, su := range sl.URLs {
         abs, err := resolveBaseURL(base, su.Media)
         if err != nil {
            return nil, err
         }
         out = append(out, abs)
      }
      return out, nil
   }

   // timeline mode
   if sl.Timeline != nil {
      startNum := 1
      if sl.StartNumber != nil {
         startNum = *sl.StartNumber
      }
      timeVal := 0
      var segments []string
      for _, s := range sl.Timeline.S {
         if s.T != 0 {
            timeVal = s.T
         }
         repeats := 1 + s.R
         for i := 0; i < repeats; i++ {
            seg := fmt.Sprintf("%d", startNum)
            abs, err := resolveBaseURL(base, seg)
            if err != nil {
               return nil, err
            }
            segments = append(segments, abs)
            startNum++
            timeVal += s.D
         }
      }
      return segments, nil
   }

   // simple @startNumber … @endNumber for SegmentList
   startNum := 1
   if sl.StartNumber != nil {
      startNum = *sl.StartNumber
   }
   var endNum int
   if sl.EndNumber != nil {
      endNum = *sl.EndNumber
   } else {
      if sl.Duration == 0 {
         return nil, errors.New("SegmentList missing @duration and no timeline nor @endNumber")
      }
      timescale := sl.Timescale
      if timescale == 0 {
         timescale = 1
      }
      periodTicks := int(periodDur.Seconds() * float64(timescale))
      count := (periodTicks + sl.Duration - 1) / sl.Duration
      endNum = startNum + count - 1
   }

   var segments []string
   for num := startNum; num <= endNum; num++ {
      seg := fmt.Sprintf("%d", num)
      abs, err := resolveBaseURL(base, seg)
      if err != nil {
         return nil, err
      }
      segments = append(segments, abs)
   }
   return segments, nil
}

var placeholderRE = regexp.MustCompile(`\$(\w+)(?:%0(\d+)d)?\$`)

func fillTemplate(tmpl string, vars map[string]int, repID string) string {
   return placeholderRE.ReplaceAllStringFunc(tmpl, func(token string) string {
      parts := placeholderRE.FindStringSubmatch(token)
      name := parts[1]
      width, _ := strconv.Atoi(parts[2])

      var val int
      switch name {
      case "RepresentationID":
         return repID
      case "Number":
         val = vars["Number"]
      case "Time":
         val = vars["Time"]
      default:
         return token
      }
      if width > 0 {
         return fmt.Sprintf("%0*d", width, val)
      }
      return fmt.Sprintf("%d", val)
   })
}

func parseDuration(d string) (time.Duration, error) {
	if d == "" {
		return 0, errors.New("empty duration")
	}
	if !strings.HasPrefix(d, "PT") {
		return 0, fmt.Errorf("duration must start with PT")
	}
	d = strings.TrimPrefix(d, "PT")

	var total float64
	remaining := d

	for remaining != "" {
		var (
			numStr string
			unit   byte
		)
		i := 0
		// collect digits and optional decimal point
		for i < len(remaining) && (remaining[i] >= '0' && remaining[i] <= '9' || remaining[i] == '.') {
			i++
		}
		if i == 0 {
			return 0, fmt.Errorf("invalid duration segment %q", remaining)
		}
		numStr = remaining[:i]
		if i >= len(remaining) {
			return 0, fmt.Errorf("duration missing unit after %q", numStr)
		}
		unit = remaining[i]
		remaining = remaining[i+1:]

		num, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number %q: %w", numStr, err)
		}

		switch unit {
		case 'H':
			total += num * 3600
		case 'M':
			total += num * 60
		case 'S':
			total += num
		default:
			return 0, fmt.Errorf("invalid unit %c", unit)
		}
	}

	return time.Duration(total * float64(time.Second)), nil
}
