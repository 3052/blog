package main

import (
    "encoding/json"
    "encoding/xml"
    "fmt"
    "log"
    "net/url"
    "os"
    "strconv"
    "strings"
)

type MPD struct {
    XMLName xml.Name `xml:"MPD"`
    BaseURL string   `xml:"BaseURL"`
    Periods []Period `xml:"Period"`
}

type Period struct {
    BaseURL        string          `xml:"BaseURL"`
    AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
    BaseURL         string           `xml:"BaseURL"`
    SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
    Representations []Representation `xml:"Representation"`
}

type Representation struct {
    ID              string           `xml:"id,attr"`
    BaseURL         string           `xml:"BaseURL"`
    SegmentList     *SegmentList     `xml:"SegmentList"`
    SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentList struct {
    BaseURL     string       `xml:"BaseURL"`
    SegmentURLs []SegmentURL `xml:"SegmentURL"`
}

type SegmentURL struct {
    Media string `xml:"media,attr"`
}

type SegmentTemplate struct {
    Media           string           `xml:"media,attr"`
    Initialization  string           `xml:"initialization,attr"`
    StartNumber     int              `xml:"startNumber,attr"`
    EndNumber       int              `xml:"endNumber,attr"`
    SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
    S []STimeline `xml:"S"`
}

type STimeline struct {
    T *uint64 `xml:"t,attr"`
    D uint64  `xml:"d,attr"`
    R int64   `xml:"r,attr"`
}

func main() {
    if len(os.Args) != 2 {
        fmt.Fprintf(os.Stderr, "Usage: go run main.go <mpd_file_path>\n")
        os.Exit(1)
    }
    mpdFile := os.Args[1]
    data, err := os.ReadFile(mpdFile)
    if err != nil {
        log.Fatalf("reading MPD file %q: %v", mpdFile, err)
    }

    var mpd MPD
    if err := xml.Unmarshal(data, &mpd); err != nil {
        log.Fatalf("parsing MPD XML: %v", err)
    }

    base, err := url.Parse("http://test.test/test.mpd")
    if err != nil {
        log.Fatalf("invalid initial base URL: %v", err)
    }

    result := make(map[string][]string)

    for _, period := range mpd.Periods {
        pBase := applyBase(base, mpd.BaseURL)
        pBase = applyBase(pBase, period.BaseURL)

        for _, ad := range period.AdaptationSets {
            aBase := applyBase(pBase, ad.BaseURL)

            for _, rep := range ad.Representations {
                rBase := applyBase(aBase, rep.BaseURL)
                segs, err := collectSegments(rep, ad.SegmentTemplate, rBase)
                if err != nil {
                    log.Fatalf("representation %q: %v", rep.ID, err)
                }
                result[rep.ID] = segs
            }
        }
    }

    enc, err := json.MarshalIndent(result, "", "  ")
    if err != nil {
        log.Fatalf("encoding JSON: %v", err)
    }
    fmt.Println(string(enc))
}

// applyBase resolves child against parent if child is non-empty.
func applyBase(parent *url.URL, child string) *url.URL {
    if child == "" {
        return parent
    }
    u, err := url.Parse(strings.TrimSpace(child))
    if err != nil {
        log.Fatalf("invalid BaseURL %q: %v", child, err)
    }
    return parent.ResolveReference(u)
}

// collectSegments handles SegmentList, then Rep-level or AdaptationSet-level SegmentTemplate.
// It replaces $RepresentationID$, $Number$, and $Time$, includes initialization if provided,
// and respects EndNumber when no timeline is present.
func collectSegments(
    rep Representation,
    adTmpl *SegmentTemplate,
    base *url.URL,
) ([]string, error) {
    // 1) explicit SegmentList
    if sl := rep.SegmentList; sl != nil && len(sl.SegmentURLs) > 0 {
        b := applyBase(base, sl.BaseURL)
        out := make([]string, len(sl.SegmentURLs))
        for i, s := range sl.SegmentURLs {
            u, err := url.Parse(s.Media)
            if err != nil {
                return nil, fmt.Errorf("invalid media URL %q: %v", s.Media, err)
            }
            out[i] = b.ResolveReference(u).String()
        }
        return out, nil
    }

    // 2) choose template: rep overrides ad
    tmpl := rep.SegmentTemplate
    if tmpl == nil {
        tmpl = adTmpl
    }
    if tmpl == nil {
        return nil, fmt.Errorf("no SegmentList or SegmentTemplate for representation %q", rep.ID)
    }

    if tmpl.Media == "" {
        return nil, fmt.Errorf("segmentTemplate missing media attribute")
    }
    start := tmpl.StartNumber
    if start <= 0 {
        start = 1
    }

    // build base media and initialization strings with RepresentationID
    mediaTemplate := strings.ReplaceAll(tmpl.Media, "$RepresentationID$", rep.ID)
    initTemplate := strings.ReplaceAll(tmpl.Initialization, "$RepresentationID$", rep.ID)

    var urls []string
    // include initialization if present
    if initTemplate != "" {
        u, err := url.Parse(initTemplate)
        if err != nil {
            return nil, fmt.Errorf("invalid initialization URL %q: %v", initTemplate, err)
        }
        urls = append(urls, base.ResolveReference(u).String())
    }

    // timeline-driven
    if tl := tmpl.SegmentTimeline; tl != nil && len(tl.S) > 0 {
        var seq int64 = int64(start)
        var curTime uint64
        if tl.S[0].T != nil {
            curTime = *tl.S[0].T
        }
        for _, e := range tl.S {
            if e.T != nil {
                curTime = *e.T
            }
            reps := e.R
            if reps < 0 {
                reps = 0
            }
            for i := int64(0); i < reps+1; i++ {
                uri := mediaTemplate
                uri = strings.ReplaceAll(uri, "$Number$", strconv.FormatInt(seq, 10))
                uri = strings.ReplaceAll(uri, "$Time$", strconv.FormatUint(curTime, 10))
                u, err := url.Parse(uri)
                if err != nil {
                    return nil, fmt.Errorf("invalid templated media %q: %v", uri, err)
                }
                urls = append(urls, base.ResolveReference(u).String())
                seq++
                curTime += e.D
            }
        }
        return urls, nil
    }

    // no timeline: respect EndNumber if set
    if tmpl.EndNumber > 0 {
        for num := start; num <= tmpl.EndNumber; num++ {
            uri := mediaTemplate
            uri = strings.ReplaceAll(uri, "$Number$", strconv.Itoa(num))
            uri = strings.ReplaceAll(uri, "$Time$", "0")
            u, err := url.Parse(uri)
            if err != nil {
                return nil, fmt.Errorf("invalid templated media %q: %v", uri, err)
            }
            urls = append(urls, base.ResolveReference(u).String())
        }
        return urls, nil
    }

    // single template segment: time = 0
    uri := mediaTemplate
    uri = strings.ReplaceAll(uri, "$Number$", strconv.Itoa(start))
    uri = strings.ReplaceAll(uri, "$Time$", "0")
    u, err := url.Parse(uri)
    if err != nil {
        return nil, fmt.Errorf("invalid templated media %q: %v", uri, err)
    }
    urls = append(urls, base.ResolveReference(u).String())
    return urls, nil
}
