package main

import (
    "encoding/xml"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/url"
    "os"
    "strings"
)

type MPD struct {
    XMLName xml.Name `xml:"MPD"`
    Period  Period   `xml:"Period"`
    BaseURL string   `xml:"BaseURL"`
}

type Period struct {
    AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
    BaseURL        string          `xml:"BaseURL"`
}

type AdaptationSet struct {
    SegmentTemplate *SegmentTemplate   `xml:"SegmentTemplate"`
    Representations []Representation   `xml:"Representation"`
}

type Representation struct {
    ID              string             `xml:"id,attr"`
    BaseURL         string             `xml:"BaseURL"`
    SegmentTemplate *SegmentTemplate   `xml:"SegmentTemplate"`
    SegmentList     *SegmentList       `xml:"SegmentList"`
}

type SegmentTemplate struct {
    Media         string              `xml:"media,attr"`
    StartNumber   int                 `xml:"startNumber,attr"`
    EndNumber     int                 `xml:"endNumber,attr"`
    Timescale     int                 `xml:"timescale,attr"`
    Initialization string             `xml:"initialization,attr"`
    SegmentTimeline SegmentTimeline   `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
    S []Segment `xml:"S"`
}

type Segment struct {
    D int `xml:"d,attr"`
    R int `xml:"r,attr"`
}

type SegmentList struct {
    SegmentURLs []SegmentURL `xml:"SegmentURL"`
}

type SegmentURL struct {
    Media string `xml:"media,attr"`
}

func resolveURL(base, ref string) string {
    u, err := url.Parse(base)
    if err != nil {
        return ref
    }
    resolved, err := u.Parse(ref)
    if err != nil {
        return ref
    }
    return resolved.String()
}

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: go run main.go <mpd file path>")
        return
    }

    data, err := ioutil.ReadFile(os.Args[1])
    if err != nil {
        panic(err)
    }

    var mpd MPD
    err = xml.Unmarshal(data, &mpd)
    if err != nil {
        panic(err)
    }

    result := make(map[string][]string)
    mpdBase := "http://test.test/test.mpd"
    if mpd.BaseURL != "" {
        mpdBase = resolveURL(mpdBase, mpd.BaseURL)
    }

    periodBase := mpdBase
    if mpd.Period.BaseURL != "" {
        periodBase = resolveURL(mpdBase, mpd.Period.BaseURL)
    }

    for _, as := range mpd.Period.AdaptationSets {
        for _, r := range as.Representations {
            segments := []string{}
            baseURL := periodBase
            if r.BaseURL != "" {
                baseURL = resolveURL(periodBase, r.BaseURL)
            }

            st := r.SegmentTemplate
            if st == nil {
                st = as.SegmentTemplate
            }

            if r.SegmentList != nil {
                for _, s := range r.SegmentList.SegmentURLs {
                    segments = append(segments, resolveURL(baseURL, s.Media))
                }
            } else if st != nil {
                start := 1
                if st.StartNumber > 0 {
                    start = st.StartNumber
                }

                end := st.EndNumber
                if st.SegmentTimeline.S != nil && len(st.SegmentTimeline.S) > 0 {
                    seq := start
                    for _, s := range st.SegmentTimeline.S {
                        count := 1
                        if s.R > 0 {
                            count += s.R
                        }
                        for i := 0; i < count; i++ {
                            url := strings.ReplaceAll(st.Media, "$RepresentationID$", r.ID)
                            url = strings.ReplaceAll(url, "$Number$", fmt.Sprintf("%d", seq))
                            segments = append(segments, resolveURL(baseURL, url))
                            seq++
                        }
                    }
                } else if end > 0 {
                    for i := start; i <= end; i++ {
                        url := strings.ReplaceAll(st.Media, "$RepresentationID$", r.ID)
                        url = strings.ReplaceAll(url, "$Number$", fmt.Sprintf("%d", i))
                        segments = append(segments, resolveURL(baseURL, url))
                    }
                }
            } else {
                segments = append(segments, baseURL)
            }

            result[r.ID] = segments
        }
    }

    jsonOutput, err := json.MarshalIndent(result, "", "  ")
    if err != nil {
        panic(err)
    }

    fmt.Println(string(jsonOutput))
}
