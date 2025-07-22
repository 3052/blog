package main

import (
    "encoding/json"
    "encoding/xml"
    "fmt"
    "io/ioutil"
    "log"
    "math"
    "net/url"
    "os"
    "regexp"
    "strconv"
    "strings"
)

// MPD represents the DASH manifest
type MPD struct {
    XMLName xml.Name `xml:"MPD"`
    BaseURL string   `xml:"BaseURL"`
    Periods []Period `xml:"Period"`
}

// Period is one DASH period
type Period struct {
    BaseURL        string          `xml:"BaseURL"`
    Duration       string          `xml:"duration,attr"`
    AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet groups Representations
type AdaptationSet struct {
    BaseURL         string           `xml:"BaseURL"`
    SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
    Representations []Representation `xml:"Representation"`
}

// Representation is one quality/language variant
type Representation struct {
    ID              string           `xml:"id,attr"`
    BaseURL         string           `xml:"BaseURL"`
    SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

// SegmentTemplate uses pointers so we can detect missing attrs
type SegmentTemplate struct {
    Timescale   *int64           `xml:"timescale,attr"`
    Duration    *int64           `xml:"duration,attr"`
    StartNumber *int64           `xml:"startNumber,attr"`
    EndNumber   *int64           `xml:"endNumber,attr"`
    Media       string           `xml:"media,attr"`
    Timeline    *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline holds one or more <S> entries
type SegmentTimeline struct {
    S []STimeline `xml:"S"`
}

// STimeline is one timeline entry
type STimeline struct {
    T int64 `xml:"t,attr"` // start time
    D int64 `xml:"d,attr"` // duration
    R int64 `xml:"r,attr"` // repeat count
}

func main() {
    if len(os.Args) < 2 {
        log.Fatalln("Usage: dash_segments <path/to/manifest.mpd>")
    }
    data, err := ioutil.ReadFile(os.Args[1])
    if err != nil {
        log.Fatalln("Error reading MPD:", err)
    }
    var mpd MPD
    if err := xml.Unmarshal(data, &mpd); err != nil {
        log.Fatalln("XML unmarshal error:", err)
    }

    // Starting point for BaseURL resolution
    mpdURL, _ := url.Parse("http://test.test/test.mpd")
    if mpd.BaseURL != "" {
        if b, err := url.Parse(strings.TrimSpace(mpd.BaseURL)); err == nil {
            mpdURL = mpdURL.ResolveReference(b)
        }
    }

    result := make(map[string][]string)
    for _, period := range mpd.Periods {
        // Resolve Period@BaseURL
        periodBase := mpdURL
        if period.BaseURL != "" {
            if b, err := url.Parse(strings.TrimSpace(period.BaseURL)); err == nil {
                periodBase = mpdURL.ResolveReference(b)
            }
        }
        // Parse Period duration
        periodSec, _ := parseISODuration(period.Duration)

        for _, ad := range period.AdaptationSets {
            // Resolve AdaptationSet@BaseURL
            adBase := periodBase
            if ad.BaseURL != "" {
                if b, err := url.Parse(strings.TrimSpace(ad.BaseURL)); err == nil {
                    adBase = periodBase.ResolveReference(b)
                }
            }

            for _, rep := range ad.Representations {
                // Single‐segment via Representation@BaseURL
                if rep.BaseURL != "" {
                    if b, err := url.Parse(strings.TrimSpace(rep.BaseURL)); err == nil {
                        seg := adBase.ResolveReference(b).String()
                        result[rep.ID] = appendUnique(result[rep.ID], seg)
                    }
                    continue
                }

                // Choose the SegmentTemplate (Representation > AdaptationSet)
                tpl := ad.SegmentTemplate
                if rep.SegmentTemplate != nil {
                    tpl = rep.SegmentTemplate
                }
                if tpl == nil || tpl.Media == "" {
                    continue
                }

                // Default values when the XML attribute is missing
                timescale := int64(1)
                if tpl.Timescale != nil {
                    timescale = *tpl.Timescale
                }
                duration := int64(0)
                if tpl.Duration != nil {
                    duration = *tpl.Duration
                }
                startNumber := int64(1)
                if tpl.StartNumber != nil {
                    startNumber = *tpl.StartNumber
                }
                endNumber := int64(0)
                if tpl.EndNumber != nil {
                    endNumber = *tpl.EndNumber
                }

                segs := buildSegments(
                    adBase,
                    tpl.Media,
                    tpl.Timeline,
                    timescale,
                    duration,
                    startNumber,
                    endNumber,
                    rep.ID,
                    periodSec,
                )
                for _, s := range segs {
                    result[rep.ID] = appendUnique(result[rep.ID], s)
                }
            }
        }
    }

    out, err := json.Marshal(result)
    if err != nil {
        log.Fatalln("JSON marshal error:", err)
    }
    fmt.Println(string(out))
}

// buildSegments handles timeline‐based or count‐based segment lists
func buildSegments(
    base *url.URL,
    media string,
    timeline *SegmentTimeline,
    timescale, duration, startNumber, endNumber int64,
    repID string,
    periodSec float64,
) []string {
    var urls []string

    // Timeline case
    if timeline != nil && len(timeline.S) > 0 {
        timeCursor := int64(0)
        numCursor := startNumber
        for i, entry := range timeline.S {
            if i == 0 && entry.T > 0 {
                timeCursor = entry.T
            }
            repeats := entry.R + 1
            for r := int64(0); r < repeats; r++ {
                u := buildMediaURL(media, repID, numCursor, timeCursor)
                if rel, err := url.Parse(u); err == nil {
                    urls = append(urls, base.ResolveReference(rel).String())
                }
                numCursor++
                timeCursor += entry.D
            }
        }
        return urls
    }

    // Count‐based case: use endNumber or compute from Period duration
    start := startNumber
    end := endNumber
    if end == 0 {
        var count int
        if duration > 0 {
            count = int(math.Ceil(periodSec*float64(timescale) / float64(duration)))
        } else {
            count = 1
        }
        end = start + int64(count) - 1
    }

    for n := start; n <= end; n++ {
        u := buildMediaURL(media, repID, n, 0)
        if rel, err := url.Parse(u); err == nil {
            urls = append(urls, base.ResolveReference(rel).String())
        }
    }
    return urls
}

// buildMediaURL replaces $…$ tokens in the media template
func buildMediaURL(media, repID string, number, timeVal int64) string {
    // Handle $Number%0Nd$ → format width immediately
    reNumFmt := regexp.MustCompile(`\$Number%0?(\d*)d\$`)
    media = reNumFmt.ReplaceAllStringFunc(media, func(tok string) string {
        parts := reNumFmt.FindStringSubmatch(tok)
        width := parts[1]
        spec := "%0" + width + "d"
        return fmt.Sprintf(spec, number)
    })

    media = strings.ReplaceAll(media, "$RepresentationID$", repID)
    media = strings.ReplaceAll(media, "$Time$", strconv.FormatInt(timeVal, 10))
    media = strings.ReplaceAll(media, "$Number$", strconv.FormatInt(number, 10))
    return media
}

// parseISODuration converts an ISO8601 duration into seconds
func parseISODuration(iso string) (float64, error) {
    s := strings.TrimPrefix(strings.TrimPrefix(iso, "P"), "T")
    re := regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)([HMS])`)
    var h, m, sec float64
    for _, mch := range re.FindAllStringSubmatch(s, -1) {
        v, _ := strconv.ParseFloat(mch[1], 64)
        switch mch[2] {
        case "H":
            h = v
        case "M":
            m = v
        case "S":
            sec = v
        }
    }
    return h*3600 + m*60 + sec, nil
}

// appendUnique appends s only if it isn’t already in the slice
func appendUnique(arr []string, s string) []string {
    for _, v := range arr {
        if v == s {
            return arr
        }
    }
    return append(arr, s)
}
