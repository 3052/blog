# encoding

~~~
> go test -bench . -benchmem -gcflags -N
goos: windows
goarch: amd64
pkg: blog/encoding
cpu: Intel(R) Core(TM) i7-8700 CPU @ 3.20GHz
BenchmarkJson-12         1228746     963.0 ns/op    344 B/op          9 allocs/op
BenchmarkAsn1-12          900691    1333 ns/op      480 B/op         25 allocs/op
BenchmarkGob-12            73160   16432 ns/op     7968 B/op        200 allocs/op
~~~
