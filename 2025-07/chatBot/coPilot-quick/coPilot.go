package main

import (
    "encoding/json"
    "encoding/xml"
    "fmt"
    "io/ioutil"
    "math"
    "net/url"
    "os"
    "strconv"
    "strings"
)

type MPD struct {
    XMLName xml.Name `xml:"MPD"`
    Periods []Period `xml:"Period"`
    BaseURL string   `xml:"BaseURL"`
}

type Period struct {
    BaseURL        string           `xml:"BaseURL"`
    Duration       string           `xml:"duration,attr"`
    AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
    BaseURL        string           `xml:"BaseURL"`
    SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
    Representations []Representation `xml:"Representation"`
}

type Representation struct {
    ID              string            `xml:"id,attr"`
    BaseURL         string            `xml:"BaseURL"`
    SegmentTemplate *SegmentTemplate  `xml:"SegmentTemplate"`
    SegmentList     *SegmentList      `xml:"SegmentList"`
}

type SegmentList struct {
    SegmentURLs []SegmentURL `xml:"SegmentURL"`
}

type SegmentURL struct {
    Media string `xml:"media,attr"`
}

type SegmentTemplate struct {
    Timescale      int              `xml:"timescale,attr"`
    Duration       int              `xml:"duration,attr"`
    StartNumber    int              `xml:"startNumber,attr"`
    EndNumber      int              `xml:"endNumber,attr"`
    Media          string           `xml:"media,attr"`
    Initialization string           `xml:"initialization,attr"`
    SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
    Segments []Segment `xml:"S"`
}

type Segment struct {
    D int `xml:"d,attr"`
    R int `xml:"r,attr"`
}

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: go run main.go path/to/file.mpd")
        return
    }

    mpdPath := os.Args[1]
    data, err := ioutil.ReadFile(mpdPath)
    if err != nil {
        panic(err)
    }

    var mpd MPD
    if err := xml.Unmarshal(data, &mpd); err != nil {
        panic(err)
    }

    baseMPD := "http://test.test/test.mpd"
    baseURL, _ := url.Parse(baseMPD)

    output := make(map[string][]string)

    for _, period := range mpd.Periods {
        periodBase := resolveBase(baseURL, period.BaseURL)
        for _, aset := range period.AdaptationSets {
            asetBase := resolveBase(periodBase, aset.BaseURL)
            for _, rep := range aset.Representations {
                repBase := resolveBase(asetBase, rep.BaseURL)
                st := rep.SegmentTemplate
                if st == nil {
                    st = aset.SegmentTemplate
                }
                if rep.SegmentList != nil {
                    for _, s := range rep.SegmentList.SegmentURLs {
                        output[rep.ID] = append(output[rep.ID], resolveBase(repBase, s.Media).String())
                    }
                    continue
                }
                if st == nil {
                    output[rep.ID] = append(output[rep.ID], repBase.String())
                    continue
                }

                if st.Timescale == 0 {
                    st.Timescale = 1
                }
                start := st.StartNumber
                if start == 0 {
                    start = 1
                }
                count := 0
                if st.SegmentTimeline != nil {
                    for _, s := range st.SegmentTimeline.Segments {
                        r := s.R
                        for i := 0; i <= r; i++ {
                            count++
                        }
                    }
                } else if st.EndNumber > 0 {
                    count = st.EndNumber - start + 1
                } else {
                    secs := parseDuration(period.Duration)
                    count = int(math.Ceil(secs * float64(st.Timescale) / float64(st.Duration)))
                }
                for i := 0; i < count; i++ {
                    num := start + i
                    media := strings.ReplaceAll(st.Media, "$RepresentationID$", rep.ID)
                    media = strings.ReplaceAll(media, "$Number$", strconv.Itoa(num))
                    segmentURL := resolveBase(repBase, media)
                    output[rep.ID] = append(output[rep.ID], segmentURL.String())
                }
            }
        }
    }

    jsonBytes, _ := json.MarshalIndent(output, "", "  ")
    fmt.Println(string(jsonBytes))
}

func resolveBase(base *url.URL, ref string) *url.URL {
    if ref == "" {
        return base
    }
    u, err := url.Parse(ref)
    if err != nil {
        return base
    }
    return base.ResolveReference(u)
}

func parseDuration(duration string) float64 {
    // Very simplistic ISO 8601 duration parser: PT#H#M#S
    duration = strings.TrimPrefix(duration, "PT")
    h, m, s := 0.0, 0.0, 0.0
    if strings.Contains(duration, "H") {
        parts := strings.Split(duration, "H")
        h, _ = strconv.ParseFloat(parts[0], 64)
        duration = parts[1]
    }
    if strings.Contains(duration, "M") {
        parts := strings.Split(duration, "M")
        m, _ = strconv.ParseFloat(parts[0], 64)
        duration = parts[1]
    }
    if strings.Contains(duration, "S") {
        parts := strings.Split(duration, "S")
        s, _ = strconv.ParseFloat(parts[0], 64)
    }
    return h*3600 + m*60 + s
}
