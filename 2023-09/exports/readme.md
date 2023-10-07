# exports

> This what you wanted? To be a janitor? Live like this? All this? Do what you
> do? It can’t be. It’s a burden, that’s what I’m trying to tell you. That’s
> how it feels.
>
> Michael Clayton (2007)

## what

The exports module prints exported identifiers in Go source code. 

## why

Starting with StaticCheck 2020.2, whole-program mode was removed. it now
operates like this:

> The normal mode of 'unused' now considers all exported package-level
> identifiers as used

So we need to unexport all symbols using some tactic. This will then give us
agency. StaticCheck will then report symbols as unused, which we can respond to
by either exporting them or removing them.

## how

If you use a command line this:

~~~
gofmt -w -r 'NewBuffer -> _NewBuffer' .
~~~

It will also catch imported functions like `bytes.NewBuffer`. You could try to
fix like this:

~~~
gofmt -w -r 'a._NewBuffer -> a.NewBuffer' .
~~~

but it will also catch method calls:

~~~
hello.Len()
~~~

we could try hardcoding the exceptions:

~~~
gofmt -w -r 'bytes._NewBuffer -> bytes.NewBuffer' .
~~~

but the imported method calls will still be broken:

~~~
bytes.Buffer._Len()
~~~

we can use this:

~~~
rf 'mv FinishedHash _FinishedHash'
~~~

but it only works for a single identifier. Can we print out all identifiers?

- https://github.com/rsc/rf
- https://godocs.io/rsc.io/rf

## where

[where.md](where.md)

## when

May 10, 2020:

https://github.com/dominikh/go-tools/commit/5cfc85b

## who

<dl>
   <dt>
   email
   </dt>
   <dd>
   srpen6@gmail.com
   </dd>
   <dt>
   Discord
   </dt>
   <dd>
   srpen6
   </dd>
   <dd>
   https://discord.com/invite/WWq6rFb8Rf
   </dd>
</dl>
