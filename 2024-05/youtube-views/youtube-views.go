package main

import (
   "154.pages.dev/platform/youtube"
   "fmt"
   "os"
   "strconv"
   "time"
)

func main() {
   if len(os.Args) == 2 {
      var req youtube.Request
      req.VideoId = os.Args[1]
      req.Web()
      var play youtube.Player
      err := play.Post(req, nil)
      if err != nil {
         panic(err)
      }
      views, err := views_per_year(&play)
      if err != nil {
         panic(err)
      }
      fmt.Println(views)
   } else {
      fmt.Println("youtube-views [video ID]")
   }
}

func views_per_year(play *youtube.Player) (string, error) {
   views, err := func() (float64, error) {
      t, err := play.Time()
      if err != nil {
         return 0, err
      }
      y := time.Since(t).Hours() / 24 / 365
      return float64(play.VideoDetails.ViewCount) / y, nil
   }()
   if err != nil {
      return "", err
   }
   var b []byte
   if views >= 10_000_000 {
      b = append(b, red...)
      b = append(b, " FAIL "...)
   } else {
      b = append(b, green...)
      b = append(b, " PASS "...)
   }
   b = append(b, reset...)
   b = append(b, "   "...)
   b = strconv.AppendFloat(b, views, 'f', 0, 64)
   b = append(b, "   "...)
   b = append(b, play.VideoDetails.VideoId...)
   return string(b), nil
}

const (
   reset = "\x1b[m"
   green = "\x1b[30;102m"
   red = "\x1b[30;101m"
)
