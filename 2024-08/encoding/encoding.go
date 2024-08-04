package encoding

const raw_body = `
{
  "slideshow": {
    "author": "Yours Truly",
    "date": "date of publication",
    "slides": [
      {
        "title": "Wake up to WonderWidgets!",
        "type": "all"
      },
      {
        "items": [
          "Why <em>WonderWidgets</em> are great",
          "Who <em>buys</em> WonderWidgets"
        ],
        "title": "Overview",
        "type": "all"
      }
    ],
    "title": "Sample Slide Show"
  }
}
`

const raw_date = "Sun, 04 Aug 2024 03:48:36 GMT"

type body struct {
   Slideshow struct {
      Author string
      Date   string
      Slides []struct {
         Items []string
         Title string
         Type  string
      }
      Title string
   }
}
