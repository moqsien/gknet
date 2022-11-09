[中文Readme](https://github.com/moqsien/gknet/blob/main/docs/ReadMe_CN.md)
---------------------------
## what's gknet?

---------------------------
It's a net library written in go applying the reactor pattern which is inspired by gnet. 
Certainly gknet benifts a lot from gnet, but it also brings something new.

## what's in gknet?

---------------------------
- gknet has nearly everything supported by [gnet](https://github.com/panjf2000/gnet).
- gknet has a builtin http server with TLS support.
- gknet has gkgin which makes benifts from the famous framework [gin](https://github.com/gin-gonic/gin). You can easily create your http server using the gin facilities.
- gknet supports both epoll on linux and kqueue on macos (no windows support). You can also easily create your own platform support by referring to the sys package.

---------------------------
## what does gknet do better than gnet?
- More readable code. 
- A ready-made http framework with TLS.
- More open, which means we do not need internals when sometimes are only interested in a special package or implementation.

## examples
[examples](https://github.com/moqsien/gknet/tree/main/examples)

---------------------------
## license
[License](https://github.com/moqsien/gknet/blob/main/LICENSE)

---------------------------
## thanks to
[panjf2000](https://github.com/panjf2000)
