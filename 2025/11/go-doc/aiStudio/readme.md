# aiStudio

## 0

~~~
panic: doc.NewFromFiles: option argument type must be doc.Mode [recovered, repanicked]

goroutine 6 [running]:
testing.tRunner.func1.2({0x7ff7949965c0, 0xc00017a990})
        D:/go/src/testing/testing.go:1872 +0x239
testing.tRunner.func1()
        D:/go/src/testing/testing.go:1875 +0x35b
panic({0x7ff7949965c0?, 0xc00017a990?})
        D:/go/src/runtime/panic.go:783 +0x132
go/doc.NewFromFiles(0xc00001cf60, {0xc00005c618, 0x1, 0xc00001cf30?}, {0x7ff7949df758, 0x2}, {0xc00018fe70?, 0x7ff7949b6b40?, 0x7ff794c100a0?})
        D:/go/src/go/doc/doc.go:221 +0x5e5
blog/11/go-doc/aiStudio.Parse({0x7ff7949e1ce6, 0x7})
        D:/git/blog/2025/11/go-doc/aiStudio/parser.go:30 +0xc5
blog/11/go-doc/aiStudio.Generate({0x7ff7949e1ce6?, 0xb?}, {0x7ff7949e48c2, 0xb})
        D:/git/blog/2025/11/go-doc/aiStudio/doc.go:10 +0x26
blog/11/go-doc/aiStudio.TestGenerate(0xc000003500)
        D:/git/blog/2025/11/go-doc/aiStudio/doc_test.go:21 +0xe5
~~~
