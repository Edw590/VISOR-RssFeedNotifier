module RssFeedNotifier

// Keep it on 1.20, so that it can be compiled for Windows 7 too if it's compiled with Go 1.20 (it's the last version
// supporting it).
go 1.20

require (
	github.com/mmcdole/gofeed v1.2.1
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9
)

require (
	github.com/PuerkitoBio/goquery v1.8.1 // indirect
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/dchest/jsmin v0.0.0-20220218165748-59f39799265f // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mmcdole/goxpp v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/ztrue/tracerr v0.4.0 // indirect
	golang.org/x/net v0.15.0 // indirect
	golang.org/x/text v0.13.0 // indirect
)

//require Utils v0.0.0-00010101000000-000000000000
//replace Utils => ./Utils
