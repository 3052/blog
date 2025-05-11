package main

import (
   "bytes"
   "encoding/json"
   "flag"
   "fmt"
   "net/http"
   "net/url"
   "path"
   "strconv"
   "time"
)

const (
   android = "ANDROID"
   android_embedded_player = "ANDROID_EMBEDDED_PLAYER"
   android_version = "19.33.35"
   web = "WEB"
   web_version = "2.20231219.04.00"
)

func main() {
   var tube InnerTube
   tube.Context.Client.ClientName = web
   flag.StringVar(&tube.VideoId, "v", "", "video ID")
   flag.Parse()
   if tube.VideoId != "" {
      play, err := tube.Player()
      if err != nil {
         panic(err)
      }
      views, err := views_per_year(play)
      if err != nil {
         panic(err)
      }
      fmt.Println(views)
   } else {
      flag.Usage()
   }
}

func views_per_year(play *Player) (string, error) {
   views := func() float64 {
      date := play.Microformat.PlayerMicroformatRenderer.PublishDate.Time
      fmt.Println(date)
      years := time.Since(date).Hours() / 24 / 365
      return float64(play.VideoDetails.ViewCount) / years
   }()
   var data []byte
   if views >= 10_000_000 {
      data = append(data, red...)
      data = append(data, " FAIL "...)
   } else {
      data = append(data, green...)
      data = append(data, " PASS "...)
   }
   data = append(data, reset...)
   data = append(data, "   "...)
   data = strconv.AppendFloat(data, views, 'f', 0, 64)
   data = append(data, "   "...)
   data = append(data, play.VideoDetails.VideoId...)
   return string(data), nil
}

const (
   reset = "\x1b[m"
   green = "\x1b[30;102m"
   red = "\x1b[30;101m"
)

type Player struct {
   Microformat struct {
      PlayerMicroformatRenderer struct {
         PublishDate Date
      }
   }
   PlayabilityStatus struct {
      Status string
      Reason string
   }
   VideoDetails struct {
      Author string
      LengthSeconds int64 `json:",string"`
      ShortDescription string
      Title string
      VideoId string
      ViewCount int64 `json:",string"`
   }
}

// need `osVersion` this to get the correct:
// This video requires payment to watch
// instead of the invalid:
// This video can only be played on newer versions of Android or other
// supported devices.
type InnerTube struct {
   ContentCheckOk bool `json:"contentCheckOk,omitempty"`
   Context struct {
      Client struct {
         AndroidSdkVersion int `json:"androidSdkVersion"`
         ClientName string `json:"clientName"`
         ClientVersion string `json:"clientVersion"`
         OsVersion string `json:"osVersion"`
      } `json:"client"`
   } `json:"context"`
   RacyCheckOk bool `json:"racyCheckOk,omitempty"`
   VideoId string `json:"videoId"`
}

func (i *InnerTube) Player() (*Player, error) {
   i.Context.Client.AndroidSdkVersion = 32
   i.Context.Client.OsVersion = "12"
   switch i.Context.Client.ClientName {
   case android:
      i.ContentCheckOk = true
      i.Context.Client.ClientVersion = android_version
      i.RacyCheckOk = true
   case android_embedded_player:
      i.Context.Client.ClientVersion = android_version
   case web:
      i.Context.Client.ClientVersion = web_version
   }
   data, err := json.Marshal(i)
   if err != nil {
      return nil, err
   }
   req, err := http.NewRequest(
      "POST", "https://www.youtube.com/youtubei/v1/player",
      bytes.NewReader(data),
   )
   if err != nil {
      return nil, err
   }
   req.Header.Set("user-agent", user_agent + i.Context.Client.ClientVersion)
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   play := &Player{}
   err = json.NewDecoder(resp.Body).Decode(play)
   if err != nil {
      return nil, err
   }
   return play, nil
}

const user_agent = "com.google.android.youtube/"

type Date struct {
   Time time.Time
}

func (d *Date) UnmarshalText(data []byte) error {
   var err error
   d.Time, err = time.Parse(time.RFC3339, string(data))
   if err != nil {
      return err
   }
   return nil
}

type VideoId string

func (v *VideoId) Set(data string) error {
   address, err := url.Parse(data)
   if err != nil {
      return err
   }
   id := address.Query().Get("v")
   if id == "" {
      id = path.Base(address.Path)
   }
   *v = VideoId(id)
   return nil
}

func (v VideoId) String() string {
   return string(v)
}
