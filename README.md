[中文Readme](https://github.com/moqsien/gknet/blob/main/docs/ReadMe_CN.md)
---------------------------
## What's gknet?

---------------------------
It's a net library written in go applying the reactor pattern which is inspired by gnet. 
Certainly gknet benifts a lot from gnet, but it also brings something new.

## What's in gknet?

---------------------------
- gknet has nearly everything supported by [gnet](https://github.com/panjf2000/gnet).
- gknet has a builtin http server with TLS support.
- gknet has gkgin which makes benifts from the famous framework [gin](https://github.com/gin-gonic/gin). You can easily create your http server using the gin facilities.
- gknet supports both epoll on linux and kqueue on macos (no windows support). You can also easily create your own platform support by referring to the sys package.

---------------------------
## What does gknet do better than gnet?
- More readable code. 
- A ready-made http framework with TLS.
- Shared and configurable goroutine pool for events.
- More open, which means we do not need internals when sometimes are only interested in a special package or implementation.

## How to use
```bash
go get -u github.com/moqsien/gknet@latest
```
[examples](https://github.com/moqsien/gknet/tree/main/examples)

---------------------------
## License
[License](https://github.com/moqsien/gknet/blob/main/LICENSE)

---------------------------
## Thanks to
[panjf2000](https://github.com/panjf2000)

## Yet another go net library?
------------------------------
Actually gnet is a very excellent net library written in go. But the main cause for creating gknet is that I found it is really hard to implement 
a graceful-rebooting functionality for gnet apps. A special non-importable listener is applied by gnet, but there is no adapter for a Listener
from the standard library. It seems that the only way to achieve graceful-rebooting is the unsafe pointer trick, or we just have to create 
a issue or pull request under gnet project.

However, I also noticed that gnet itself does not support http and TLS, and the author Andy Pan has no plan to implement TLS or importable 
listener support recently. Besides, the code design for gnet becomes somehow redundant recently. Therefore, gknet is created taking advantage 
of gnet's well-designed part, like buffers in user space, lock-free queues, etc.

Gknet tries to keep code from redundancy, adapt Listener from the standard library, and also provie builtin http as well as TLS support.
More optimizations and functionalities are on the way:

- Pool for Conn objects;
- Support for io_uring on linux;
- More builtin framework support like gkgin;
- rpc support;
- addr and port reuse;
