[![Go Reference](https://pkg.go.dev/badge/github.com/ikawaha/waifu2x.go.svg)](https://pkg.go.dev/github.com/ikawaha/waifu2x.go)
waifu2x.go
===

Image super-resolution using deep convolutional neural network (CNN).
waifu2x.go is a clone of waifu2x-js.

waifu2x-js: https://github.com/takuyaa/waifu2x-js

Changes
---
* 2022-02-09: Imported changes from [go-waifu2x](https://github.com/puhitaku/go-waifu2x), a fork of this repository. This is an excellent job done by @puhitaku and @orisano. It is 14x faster than the original in the non-GPU case.

Install
---

```shell
go install github.com/ikawaha/waifu2x.go@latest
```

Usage
---

```shell
$ waifu2x.go --help
Usage of waifu2x:
  -i string
    	input file (default stdin)
  -m string
    	waifu2x mode, choose from 'anime' and 'photo' (default "anime")
  -n int
    	noise reduction level 0 <= n <= 3
  -o string
    	output file (default stdout)
  -p int
    	concurrency (default 8)
  -s float
    	scale multiplier >= 1.0 (default 2)
  -v	verbose
```

<img width="542" alt="image" src="https://user-images.githubusercontent.com/4232165/155845021-83a90df6-5324-4511-94fc-2d9d4a00273c.png">

The Go gopher was designed by [Renée French](https://reneefrench.blogspot.com/).

Note
---

This software includes a binary and/or source version of model data from waifu2x which can be obtained from https://github.com/nagadomi/waifu2x.

License
---

MIT
