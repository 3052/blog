# aiStudio

## 0

~~~
panic: runtime error: slice bounds out of range [:-482] [recovered, repanicked]

goroutine 19 [running]:
testing.tRunner.func1.2({0x7ff6a829fc60, 0xc0000ae510})
        D:/go/src/testing/testing.go:1872 +0x239
testing.tRunner.func1()
        D:/go/src/testing/testing.go:1875 +0x35b
panic({0x7ff6a829fc60?, 0xc0000ae510?})
        D:/go/src/runtime/panic.go:783 +0x132
blog/11/go-doc/aiStudio.syntaxHighlight({0xc0000fc400, 0x1e6})
        D:/git/blog/2025/11/go-doc/aiStudio/highlighter.go:35 +0x76f
blog/11/go-doc/aiStudio.formatAndHighlight({0x7ff6a827ce00, 0xc00011fe40}, 0xc00008af30)
        D:/git/blog/2025/11/go-doc/aiStudio/parser.go:106 +0xe7
blog/11/go-doc/aiStudio.Parse({0x7ff6a82bc074, 0x7})
        D:/git/blog/2025/11/go-doc/aiStudio/parser.go:53 +0x487
blog/11/go-doc/aiStudio.Generate({0x7ff6a82bc074?, 0xb?}, {0x7ff6a82becb7, 0xb})
        D:/git/blog/2025/11/go-doc/aiStudio/doc.go:11 +0x2f
blog/11/go-doc/aiStudio.TestGenerate(0xc000086540)
        D:/git/blog/2025/11/go-doc/aiStudio/doc_test.go:21 +0xf5
~~~
