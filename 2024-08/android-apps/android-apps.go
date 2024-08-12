package main

import (
   "154.pages.dev/google/play"
   "154.pages.dev/text"
   "fmt"
   "os"
   "slices"
   "time"
)

func main() {
   home, err := os.UserHomeDir()
   if err != nil {
      panic(err)
   }
   var token play.GoogleToken
   token.Data, err = os.ReadFile(home + "/google-play/token.txt")
   if err != nil {
      panic(err)
   }
   token.Unmarshal()
   var auth play.GoogleAuth
   if err := auth.Auth(token); err != nil {
      panic(err)
   }
   var checkin play.GoogleCheckin
   checkin.Data, err = os.ReadFile(home + "/google-play/x86.txt")
   if err != nil {
      panic(err)
   }
   checkin.Unmarshal()
   for i, app := range apps {
      fmt.Println(app.id)
      detail, err := checkin.Details(auth, app.id, false)
      if err != nil {
         panic(err)
      }
      apps[i].installs, _ = detail.Downloads()
      apps[i].name, _ = detail.Name()
      time.Sleep(99 * time.Millisecond)
   }
   slices.SortFunc(apps, func(a, b application) int {
      return int(b.installs - a.installs)
   })
   for i, app := range apps {
      fmt.Printf("%v. %v\n", i+1, app.name)
      fmt.Printf("\t- %v\n", text.Cardinal(app.installs))
   }
}

type application struct {
   id string
   name string
   installs uint64
}

var apps = []application{
   {id: "au.com.stan.and"},
   {id: "be.rtbf.auvio"},
   {id: "ca.ctv.ctvgo"},
   {id: "com.amcplus.amcfullepisodes"},
   {id: "com.cbs.app"},
   {id: "com.criterionchannel"},
   {id: "com.draken.android"},
   {id: "com.hulu.plus"},
   {id: "com.mubi"},
   {id: "com.nbcuni.nbc"},
   {id: "com.plexapp.android"},
   {id: "com.roku.web.trc"},
   {id: "com.tubitv"},
   {id: "com.wbd.stream"},
   {id: "nl.peoplesplayground.audienceplayer.cinemember"},
   {id: "tv.pluto.android"},
   {id: "tv.wuaki"},
}
