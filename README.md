go-waifu2x
==========

*A nitro-powered fork of waifu2x.go*

Upscale your waifu.
go-waifu2x is a fork of waifu2x.go that originates from [waifu2x-js](https://github.com/takuyaa/waifu2x-js).


Improvements
------------

This fork has the following improvements:

 - Concurrency (Up to 7x faster than the original on MacBook Pro Late 2021, 14-inch, M1 Max)
 - Upscale the alpha channel
 - More customizable CLI
   - Enable/disable concurrency (enabled by default)
   - Choose noise reduction level
   - Choose mode (photo/anime)
 - `go.mod`


Note
----

This software includes a binary and/or source version of model data from waifu2x
which can be obtained from [here](https://github.com/nagadomi/waifu2x).


License
-------

See [LICENSE](LICENSE) for the copyright notice and the license.
