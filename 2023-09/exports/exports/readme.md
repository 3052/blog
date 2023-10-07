# export

## rsc/rf

~~~
> Measure-Command { rf 'mv FinishedHash _FinishedHash' }
TotalSeconds      : 1.6925358

> Measure-Command { rf 'mv FinishedHash _FinishedHash' }
TotalSeconds      : 1.7182022
~~~

- https://github.com/rsc/rf
- https://godocs.io/rsc.io/rf

## golang/tools

~~~
> Measure-Command { gorename -from '\".\".FinishedHash' -to _FinishedHash }
TotalSeconds      : 3.4577449

> Measure-Command { gorename -from '\".\".FinishedHash' -to _FinishedHash }
TotalSeconds      : 3.5773823
~~~

- https://github.com/golang/tools
- https://golang.org/x/tools/cmd/gorename
