package iso

import (
   "io"
   "net/http"
   "net/url"
   "strings"
   "fmt"
)

/*
> curl -s -O -w '%{size_download}' https://www.iso.org/home.html
90122

> curl -w '%{size_download}' -s -O https://www.iso.org/obp/ui
1735
*/
