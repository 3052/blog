package main

import (
   "encoding/json"
   "flag"
   "log"
   "math"
   "os/exec"
)

func main() {
   run := flag.String("r", "", "run")
   mpd := flag.String("m", "../check-mpd/ignore/", "mpd")
   flag.Parse()
   if *run == "" {
      flag.Usage()
      return
   }
   log.SetFlags(log.Ltime)
   for _, testVar := range tests {
      data, err := output(
         "go", "run", *run, *mpd+testVar.name,
      )
      if err != nil {
         log.Fatal(string(data))
      }
      var representsB map[string][]string
      err = json.Unmarshal(data, &representsB)
      if err != nil {
         log.Fatal(err)
      }
      for _, representA := range testVar.representation {
         representB := representsB[representA.id]
         if len(representB) != representA.length {
            log.Fatalln(
               representA.id,
               "pass", representA.length,
               "fail", len(representB),
            )
         }
         if representB[0] != representA.start {
            log.Fatalln(
               "\npass", representA.start,
               "\nfail", representB[0],
            )
         }
         if representB[len(representB)-1] != representA.end {
            log.Fatalln(
               "\npass", representA.end,
               "\nfail", representB[len(representB)-1],
            )
         }
      }
   }
}

const initialization = 1
var tests = []struct {
   name           string
   representation []representationA
}{
   {
      name: "canal.mpd",
      representation: []representationA{
         {
            content_type: type_video,
            id:           "video=3399914",
            length:       initialization + 1 + 1332 + 1,
            start:        prefix + "dash/appletvcz_A007300100102_2464C3BF9652075492E7CF48A400F243_HD-video=3399914.dash?serviceid=298f95e1bf91361258c44a2b1f4a2425",
            end:          prefix + "dash/appletvcz_A007300100102_2464C3BF9652075492E7CF48A400F243_HD-video=3399914-4798800.dash?serviceid=298f95e1bf91361258c44a2b1f4a2425",
         },
         {
            content_type: type_image,
            id:           "thumbnail",
            length:       80,
            start:        prefix + "dash/thumbnail/tile_1.jpeg?serviceid=298f95e1bf91361258c44a2b1f4a2425",
            end:          prefix + "dash/thumbnail/tile_80.jpeg?serviceid=298f95e1bf91361258c44a2b1f4a2425",
         },
      },
   },
   {
      name: "criterion.mpd",
      representation: []representationA{
         {
            content_type: type_video,
            id:           "video-31e3417b-d163-45ba-9d7d-c356ef7aca57",
            length:       initialization + 1084,
            start:        prefix + "drm/cenc,derived,1039101898,70c108dd0a77b48f8738a37ed402d1e3/range/prot/aW5pdF9yYW5nZT0wLTgwNCZyYW5nZT0wLTgwNA/avf/31e3417b-d163-45ba-9d7d-c356ef7aca57.mp4?init_range=0-804&pathsig=8c953e4f~0yotwnRXHn_MAwePXLjUCgWFaE37toTPlqP5_COJ7Fg&r=dXMtZWFzdDE%3D&range=0-804",
            end:          prefix + "drm/cenc,derived,1039101898,70c108dd0a77b48f8738a37ed402d1e3/range/prot/aW5pdF9yYW5nZT0wLTgwNCZyYW5nZT0xODYzNTk5MTk1LTE4NjM4MTU0Mzk/avf/31e3417b-d163-45ba-9d7d-c356ef7aca57.mp4?init_range=0-804&pathsig=8c953e4f~H4likJuUcpljeDjz1r1ky7d3kH28faTdejTPfwh_5DY&r=dXMtZWFzdDE%3D&range=1863599195-1863815439",
         },
         {
            content_type: type_text,
            id:           "subs-202936084",
            length:       1,
            start:        prefix + "texttrack/sub/202936084.vtt?pathsig=8c953e4f~VmhEmE4Ge9n9S9R4gdPqJo02oUhoM8-o1o0iBVypx3k&r=dXMtY2VudHJhbDE%3D",
            end:          prefix + "texttrack/sub/202936084.vtt?pathsig=8c953e4f~VmhEmE4Ge9n9S9R4gdPqJo02oUhoM8-o1o0iBVypx3k&r=dXMtY2VudHJhbDE%3D",
         },
      },
   },
   {
      name: "hboMax.mpd",
      representation: []representationA{
         {
            content_type: type_video,
            id:           "v31",
            length:       8, // allow duplicates
            start:        prefix + "v/1_584f52/v31.mp4",
            end:          prefix + "v/1_584f52/v31.mp4",
         },
         {
            content_type: type_text,
            id:           "t2",
            length:       8,
            start:        prefix + "t/0_d2d294/t2/1.vtt",
            end:          prefix + "t/0_d2d294/t2/8.vtt",
         },
         {
            content_type: type_image,
            id:           "images_1",
            length: func() int { // allow duplicates
               media := math.Ceil(1100.34925 / 5)         // 221, 0-220
               media += math.Ceil(1122.2044166666667 / 5) // 225, 220-
               media += math.Ceil(1104.1864166666664 / 5) // 221
               media += math.Ceil(1017.5582083333338 / 5) // 204
               media += math.Ceil(1058.9328749999995 / 5) // 212
               media += math.Ceil(1011.9692916666672 / 5) // 203
               media += math.Ceil(994.4934999999996 / 5)  // 199
               media += math.Ceil(1914.2873749999999 / 5) // 383
               return int(media)
            }(),
            start: prefix + "i/1_6c0f17/images_1_00000000.jpg",
            end:   prefix + "i/1_6c0f17/images_1_00001863.jpg",
         },
      },
   },
   {
      name: "molotov.mpd",
      representation: []representationA{
         {
            content_type: type_video,
            id:           "video=4800000",
            length:       initialization + 3555,
            start:        prefix + "dash/32e3c47902de4911dca77b0ad73e9ac34965a1d8-video=4800000.dash",
            end:          prefix + "dash/32e3c47902de4911dca77b0ad73e9ac34965a1d8-video=4800000-3555.m4s",
         },
         {
            content_type: type_text,
            id:           "3=1000",
            length:       initialization + 3339,
            start:        prefix + "dash/32e3c47902de4911dca77b0ad73e9ac34965a1d8-3=1000.dash",
            end:          prefix + "dash/32e3c47902de4911dca77b0ad73e9ac34965a1d8-3=1000-3339.m4s",
         },
      },
   },
   {
      name: "paramount.mpd",
      representation: []representationA{
         {
            content_type: type_video,
            id:           "5",
            length:       9*initialization + 539 + 1 + 1 + 29 + 1,
            start:        prefix + "TPIR_0722_100824_2997DF_1920x1080_178_2CH_PRORESHQ_2CH_2939373_4500/init.m4v",
            end:          prefix + "TPIR_0722_100824_2997DF_1920x1080_178_2CH_PRORESHQ_2CH_2939373_4500/seg_571.m4s",
         },
         {
            content_type: type_image,
            id:           "thumb_320x180",
            length:       11,
            start:        prefix + "thumb_320x180/tile_1.jpg",
            end:          prefix + "thumb_320x180/tile_11.jpg",
         },
         {
            content_type: type_text,
            id:           "8",
            length:       9*initialization + 540 + 1 + 22,
            start:        prefix + "TPIR_0722_2997_2CH_DF_1728406422/vtt_init.m4v",
            end:          prefix + "TPIR_0722_2997_2CH_DF_1728406422/seg_563.m4s",
         },
      },
   },
   {
      name: "rakuten.mpd",
      representation: []representationA{
         {
            content_type: type_video,
            id:           "video-avc1-6",
            length:       1,
            start:        "https://prod-avod-pmd-cdn77.cdn.rakuten.tv/3/1/8/318f7ece69afcfe3e96de31be6b77272-mc-0-164-0-0_DS2BB/video-avc1-6.ismv?streaming_id=630ed6ed-1137-473c-8858-23ba59d12675&st_country=CZ,AT,DE,PL,SK&st_valid=1752245624&secure=PcDPRLtVJ_-tDsEPMD5Hzg==,1752267224",
            end:          "https://prod-avod-pmd-cdn77.cdn.rakuten.tv/3/1/8/318f7ece69afcfe3e96de31be6b77272-mc-0-164-0-0_DS2BB/video-avc1-6.ismv?streaming_id=630ed6ed-1137-473c-8858-23ba59d12675&st_country=CZ,AT,DE,PL,SK&st_valid=1752245624&secure=PcDPRLtVJ_-tDsEPMD5Hzg==,1752267224",
         },
      },
   },
}

func output(name string, arg ...string) ([]byte, error) {
   command := exec.Command(name, arg...)
   log.Print(command.Args)
   return command.Output()
}

type content_type int

const (
   type_image content_type = iota
   type_text
   type_video
)

type representationA struct {
   content_type content_type
   id           string
   length       int
   start        string
   end          string
}

const prefix = "http://test.test/"
