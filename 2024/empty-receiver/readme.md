# empty receiver

~~~
> go test -bench .
BenchmarkPointerNil-12          1000000000               0.2329 ns/op
BenchmarkValue-12               1000000000               0.2301 ns/op
BenchmarkPointer-12               117768             12537 ns/op

> go test -bench . -gcflags -N
BenchmarkPointerNil-12          814226710                1.444 ns/op
BenchmarkValue-12                 858459              1370 ns/op
BenchmarkPointer-12                99013             12895 ns/op
~~~
