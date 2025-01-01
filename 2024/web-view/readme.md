# webView

- https://learn.microsoft.com/microsoft-edge/webview2/reference/win32/icorewebview2httpresponseheaders
- https://wikipedia.org/wiki/WebView

here's an answer on how to get response header:
https://stackoverflow.com/a/65429432

the code is in c#, here's how it's translated into go using this lib: (you can
put these code in the hello_webview.go, before the line
`webviewWindow.Navigate("https://www.bing.com/")` )

```
pCoreWebview2 := wv2.NewICoreWebView2_2(webviewWindow.GetIUnknown(), false, false)
pCoreWebview2.Add_WebResourceResponseReceived(
	wv2.NewICoreWebView2WebResourceResponseReceivedEventHandlerByFunc(
		func(sender *wv2.ICoreWebView2, e *wv2.ICoreWebView2WebResourceResponseReceivedEventArgs) com.Error {
			defer com.NewScope().Leave()
			var pResponse *wv2.ICoreWebView2WebResourceResponseView
			e.GetResponse(&pResponse)
			var pHeaders *wv2.ICoreWebView2HttpResponseHeaders
			pResponse.GetHeaders(&pHeaders)
			var pwsDate win32.PWSTR
			pHeaders.GetHeader("Date", &pwsDate)
			win32.MessageBox(hWnd, pwsDate, win32.StrToPwstr("Date Header:"), win32.MB_OK)
			win32.CoTaskMemFree(unsafe.Pointer(pwsDate))
			return com.OK
		}, false), nil)
```

https://github.com/zzl/go-webview2/issues/1

## issues

1. https://github.com/jchv/go-webview2/issues/72
2. https://github.com/gioui-plugins/gio-plugins/issues/62
3. <https://github.com/webview/webview_go/issues/42>
4. https://github.com/inkeliz/gowebview/issues/25
5. https://github.com/inkeliz/giowebview/issues/10
6. https://github.com/suchipi/webview/issues/6
7. https://github.com/CarsonSlovoka/go-webview2/issues/5
8. https://github.com/wailsapp/go-webview2/issues/5
9. https://github.com/mekkanized/go-webview/issues/3
