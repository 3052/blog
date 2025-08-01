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
    "strings"
    "time"
)

const defaultBase = "http://test.test/test.mpd"

// === XML Structs ===
type MPD struct {
    XMLName                   xml.Name `xml:"MPD"`
    BaseURL                   *string  `xml:"BaseURL"`
    MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
    Periods                   []Period `xml:"Period"`
}

type Period struct {
    BaseURL        *string         `xml:"BaseURL"`
    AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
    BaseURL         *string           `xml:"BaseURL"`
    SegmentTemplate *SegmentTemplate  `xml:"SegmentTemplate"`
    Representations []Representation  `xml:"Representation"`
}

type Representation struct {
    ID              string           `xml:"id,attr"`
    BaseURL         *string          `xml:"BaseURL"`
    SegmentList     *SegmentList     `xml:"SegmentList"`
    SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentList struct {
    Initialization *SegmentURL  `xml:"Initialization"`
    SegmentURLs    []SegmentURL `xml:"SegmentURL"`
}

type SegmentURL struct {
    Media string `xml:"media,attr"`
}

type SegmentTemplate struct {
    Initialization  string           `xml:"initialization,attr"`
    Media           string           `xml:"media,attr"`
    Duration        uint64           `xml:"duration,attr"`
    Timescale       uint64           `xml:"timescale,attr"`
    StartNumber     uint64           `xml:"startNumber,attr"`
    EndNumber       uint64           `xml:"endNumber,attr"`
    SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
    S []S `xml:"S"`
}

type S struct {
    T uint64 `xml:"t,attr"`
    D uint64 `xml:"d,attr"`
    R int64  `xml:"r,attr"`
}

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: go run main.go <mpd_file_path>")
        os.Exit(1)
    }
    data, err := ioutil.ReadFile(os.Args[1])
    if err != nil {
        fmt.Println("Error reading file:", err)
        os.Exit(1)
    }

    var mpd MPD
    if err := xml.Unmarshal(data, &mpd); err != nil {
        fmt.Println("Error parsing MPD XML:", err)
        os.Exit(1)
    }

    base, _ := url.Parse(defaultBase)
    if mpd.BaseURL != nil {
        mpdBase, _ := url.Parse(*mpd.BaseURL)
        base = base.ResolveReference(mpdBase)
    }

    periodDuration := parseDuration(mpd.MediaPresentationDuration)
    result := map[string][]string{}

    for _, period := range mpd.Periods {
        pBase := base
        if period.BaseURL != nil {
            periodURL, _ := url.Parse(*period.BaseURL)
            pBase = pBase.ResolveReference(periodURL)
        }

        for _, aset := range period.AdaptationSets {
            aBase := pBase
            if aset.BaseURL != nil {
                asetURL, _ := url.Parse(*aset.BaseURL)
                aBase = aBase.ResolveReference(asetURL)
            }

            for _, rep := range aset.Representations {
                rBase := aBase
                if rep.BaseURL != nil {
                    repURL, _ := url.Parse(*rep.BaseURL)
                    rBase = rBase.ResolveReference(repURL)
                }

                segmentTemplate := rep.SegmentTemplate
                if segmentTemplate == nil {
                    segmentTemplate = aset.SegmentTemplate
                }

                var urls []string

                if rep.SegmentList != nil {
                    if rep.SegmentList.Initialization != nil {
                        initURL, _ := url.Parse(rep.SegmentList.Initialization.Media)
                        urls = append(urls, rBase.ResolveReference(initURL).String())
                    }
                    for _, seg := range rep.SegmentList.SegmentURLs {
                        segURL, _ := url.Parse(seg.Media)
                        urls = append(urls, rBase.ResolveReference(segURL).String())
                    }
                } else if segmentTemplate != nil {
                    start := segmentTemplate.StartNumber
                    if start == 0 {
                        start = 1
                    }
                    if segmentTemplate.Initialization != "" {
                        initPath := applyVars(segmentTemplate.Initialization, start, 0, rep.ID)
                        initURL, _ := url.Parse(initPath)
                        urls = append(urls, rBase.ResolveReference(initURL).String())
                    }
                    if segmentTemplate.SegmentTimeline != nil {
                        var timeVal uint64 = 0
                        for _, s := range segmentTemplate.SegmentTimeline.S {
                            r := s.R
                            if r < 0 {
                                r = 0
                            }
                            if s.T != 0 {
                                timeVal = s.T
                            }
                            for i := int64(0); i <= r; i++ {
                                t := timeVal
                                media := applyVars(segmentTemplate.Media, 0, t, rep.ID)
                                mediaURL, _ := url.Parse(media)
                                urls = append(urls, rBase.ResolveReference(mediaURL).String())
                                timeVal += s.D
                            }
                        }
                    } else {
                        count := segmentTemplate.EndNumber
                        if count == 0 && segmentTemplate.Duration > 0 && segmentTemplate.Timescale > 0 {
                            count = uint64(math.Ceil(periodDuration.Seconds() * float64(segmentTemplate.Timescale) / float64(segmentTemplate.Duration)))
                        }
                        for i := start; i <= count; i++ {
                            media := applyVars(segmentTemplate.Media, i, 0, rep.ID)
                            mediaURL, _ := url.Parse(media)
                            urls = append(urls, rBase.ResolveReference(mediaURL).String())
                        }
                    }
                }

                result[rep.ID] = urls
            }
        }
    }

    out, err := json.MarshalIndent(result, "", "  ")
    if err != nil {
        fmt.Println("Error generating JSON:", err)
        os.Exit(1)
    }
    fmt.Println(string(out))
}

// Substitute $RepresentationID$, $Number$, $Time$, with optional zero-padding
func applyVars(tmpl string, number uint64, timeVal uint64, repID string) string {
    tmpl = strings.ReplaceAll(tmpl, "$RepresentationID$", repID)
    re := regexp.MustCompile(`\$(Number|Time)(%0(\d+)d)?\$`)
    return re.ReplaceAllStringFunc(tmpl, func(m string) string {
        g := re.FindStringSubmatch(m)
        var val uint64
        if g[1] == "Number" {
            val = number
        } else {
            val = timeVal
        }
        if g[3] != "" {
            width, _ := strconv.Atoi(g[3])
            return fmt.Sprintf("%0*d", width, val)
        }
        return fmt.Sprintf("%d", val)
    })
}

// Parse MPEG-DASH duration format like PT4M12.16S
func parseDuration(s string) time.Duration {
    s = strings.TrimPrefix(s, "PT")
    sec := 0.0
    if strings.Contains(s, "H") {
        parts := strings.SplitN(s, "H", 2)
        h, _ := strconv.ParseFloat(parts[0], 64)
        sec += h * 3600
        s = parts[1]
    }
    if strings.Contains(s, "M") {
        parts := strings.SplitN(s, "M", 2)
        m, _ := strconv.ParseFloat(parts[0], 64)
        sec += m * 60
        s = parts[1]
    }
    if strings.Contains(s, "S") {
        parts := strings.SplitN(s, "S", 2)
        sv, _ := strconv.ParseFloat(parts[0], 64)
        sec += sv
    }
    return time.Duration(sec * float64(time.Second))
}

