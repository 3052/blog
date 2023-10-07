# Encoding

this is attractive:

https://godocs.io/encoding#TextUnmarshaler

but in practice, text input is going to be a string, which means you will need
to cast to byte slice. Further, since its a method, you would need to initialize
the value:

~~~go
package main

import (
   "fmt"
   "time"
)

func main() {
   var t time.Time
   err := t.UnmarshalText([]byte("2006-01-02T15:04:05Z"))
   if err != nil {
      panic(err)
   }
   fmt.Println(t)
}
~~~

Implementing this interface does enable these functions:

- https://godocs.io/encoding/json#Unmarshal
- https://godocs.io/encoding/xml#Unmarshal
- https://godocs.io/flag#TextVar

but those are not useful to me at this time. one:

- https://godocs.io/encoding#BinaryMarshaler
- https://godocs.io/net/netip#Addr.AsSlice
- https://godocs.io/net/netip#Addr.MarshalBinary

two:

- https://godocs.io/encoding#BinaryUnmarshaler
- https://godocs.io/net/netip#Addr.UnmarshalBinary
- https://godocs.io/net/netip#AddrFromSlice

three:

- https://godocs.io/encoding#TextMarshaler
- https://godocs.io/net/netip#Addr.MarshalText
- https://godocs.io/net/netip#Addr.String

four:

- https://godocs.io/encoding#TextUnmarshaler
- https://godocs.io/net/netip#Addr.UnmarshalText
- https://godocs.io/net/netip#ParseAddr
