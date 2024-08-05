package method

func (response_body) value() string {
   return raw_body
}

func (*response_body) pointer() string {
   return raw_body
}

type response_body struct {
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
