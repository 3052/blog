# go doc

~~~
parser.go:210
highlighter.go:61
doc.go:36
types.go:33
renderer.go:22
template.tmpl:77
style.css:38

477 matched lines
~~~

Go language, I would like to create a package that creates HTML documentation
for a Go package

1. package will be called "doc"
2. do not include any "main.go" or "package main"
3. use a separate file for each type
4. any templates should be a separate file not a string
5. put all package files in the top directory not a subfolder
6. exclude unexported items
7. include a test file in the top folder
8. test file should only use the "example" directory, assuming the user has
   provided it, not create it
9. test output should remain after test is complete
10. do not include any other test files
11. when sending updates, send the complete file, for only new or updated
   files

https://github.com/golang/go/issues/2381
