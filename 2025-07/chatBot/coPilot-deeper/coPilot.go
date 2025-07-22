package main

import (
    "encoding/json"
    "encoding/xml"
    "fmt"
    "net/url"
    "os"
    "strconv"
    "strings"
)

// Structures to parse DASH MPD XML

type MPD struct {
    XMLName   xml.Name  `xml:"MPD"`
    BaseURLs  []string  `xml:"BaseURL"`
    Periods   []Period  `xml:"Period"`
}

type Period struct {
    BaseURLs        []string        `xml:"BaseURL"`
    AdaptationSets  []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
    SegmentTemplates []SegmentTemplate `xml:"SegmentTemplate"`
    Representations  []Representation  `xml:"Representation"`
}

type Representation struct {
    ID               string             `xml:"id,attr"`
    SegmentTemplates []SegmentTemplate  `xml:"SegmentTemplate"`
}

type SegmentTemplate struct {
    Timescale      uint64           `xml:"timescale,attr"`
    Media          string           `xml:"media,attr"`
    StartNumber    uint64           `xml:"startNumber,attr"`
    EndNumber      *uint64          `xml:"endNumber,attr"`
    SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
    S []TimelineS `xml:"S"`
}

type TimelineS struct {
    T *uint64 `xml:"t,attr"`
    D uint64  `xml:"d,attr"`
    R *int64  `xml:"r,attr"`
}

func main() {
    if len(os.Args) < 2 {
        fmt.Fprintf(os.Stderr, "Usage: %s <mpd-file-path>\n", os.Args[0])
        os.Exit(1)
    }
    mpdPath := os.Args[1]

    // 1. Parse local MPD file
    f, err := os.Open(mpdPath)
    if err != nil {
        panic(err)
    }
    defer f.Close()

    var mpd MPD
    if err := xml.NewDecoder(f).Decode(&mpd); err != nil {
        panic(err)
    }

    // 2. Resolve MPD@BaseURL against the known MPD URL
    baseURL, err := url.Parse("http://test.test/test.mpd")
    if err != nil {
        panic(err)
    }
    if len(mpd.BaseURLs) > 0 {
        ref, err := url.Parse(strings.TrimSpace(mpd.BaseURLs[0]))
        if err != nil {
            panic(err)
        }
        baseURL = baseURL.ResolveReference(ref)
    }

    result := make(map[string][]string)

    // 3. Iterate Periods
    for _, period := range mpd.Periods {
        periodBase := baseURL
        if len(period.BaseURLs) > 0 {
            ref, err := url.Parse(strings.TrimSpace(period.BaseURLs[0]))
            if err != nil {
                panic(err)
            }
            periodBase = baseURL.ResolveReference(ref)
        }

        // 4. Each AdaptationSet
        for _, aset := range period.AdaptationSets {
            for _, rep := range aset.Representations {
                // pick the SegmentTemplate: child of Representation if exists, else AdaptationSet
                var st *SegmentTemplate
                if len(rep.SegmentTemplates) > 0 {
                    st = &rep.SegmentTemplates[0]
                } else if len(aset.SegmentTemplates) > 0 {
                    st = &aset.SegmentTemplates[0]
                } else {
                    continue
                }

                var segURLs []string

                // 5. If SegmentTimeline exists, use it
                if st.SegmentTimeline != nil && len(st.SegmentTimeline.S) > 0 {
                    // track the "current" start time
                    var currentTime uint64
                    // iterate each <S>
                    for _, s := range st.SegmentTimeline.S {
                        // if the element defines a new t, adopt it
                        if s.T != nil {
                            currentTime = *s.T
                        }
                        // determine repeat count (default 0)
                        repeats := int64(0)
                        if s.R != nil {
                            repeats = *s.R
                        }
                        // generate segments: one for each repeat + the first
                        for i := int64(0); i <= repeats; i++ {
                            t := currentTime + uint64(i)*s.D
                            media := strings.ReplaceAll(st.Media, "$RepresentationID$", rep.ID)
                            media = strings.ReplaceAll(media, "$Time$", strconv.FormatUint(t, 10))

                            u, err := url.Parse(media)
                            if err != nil {
                                panic(err)
                            }
                            full := periodBase.ResolveReference(u).String()
                            segURLs = append(segURLs, full)
                        }
                        // advance currentTime past all repeats
                        currentTime += uint64(repeats+1) * s.D
                    }

                } else if st.EndNumber != nil {
                    // 6. If no timeline but endNumber exists, use number range
                    for num := st.StartNumber; num <= *st.EndNumber; num++ {
                        media := strings.ReplaceAll(st.Media, "$RepresentationID$", rep.ID)
                        media = strings.ReplaceAll(media, "$Number$", strconv.FormatUint(num, 10))

                        u, err := url.Parse(media)
                        if err != nil {
                            panic(err)
                        }
                        full := periodBase.ResolveReference(u).String()
                        segURLs = append(segURLs, full)
                    }
                }

                // 7. collect into result map
                if len(segURLs) > 0 {
                    result[rep.ID] = segURLs
                }
            }
        }
    }

    // 8. Output JSON
    out, err := json.Marshal(result)
    if err != nil {
        panic(err)
    }
    fmt.Println(string(out))
}
