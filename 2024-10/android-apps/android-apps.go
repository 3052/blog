package main

import (
   "41.neocities.org/google/play"
   "41.neocities.org/text"
   "fmt"
   "os"
   "slices"
   "time"
)

var apps = []application{
   {id: "air.ITVMobilePlayer"},
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

func main() {
   home, err := os.UserHomeDir()
   if err != nil {
      panic(err)
   }
   var token play.GoogleToken
   token.Raw, err = os.ReadFile(home + "/google-play/token.txt")
   if err != nil {
      panic(err)
   }
   err = token.Unmarshal()
   if err != nil {
      panic(err)
   }
   auth, err := token.Auth()
   if err != nil {
      panic(err)
   }
   var checkin play.GoogleCheckin
   checkin.Raw, err = os.ReadFile(home + "/google-play/x86.txt")
   if err != nil {
      panic(err)
   }
   err = checkin.Unmarshal()
   if err != nil {
      panic(err)
   }
   for i, app := range apps {
      fmt.Println(app.id)
      detail, err := auth.Details(&checkin, app.id, false)
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
