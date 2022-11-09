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

## 写gknet的初衷
---------------------------
主要是有一次，我想实现一个支持优雅重启的应用，就想到要支持gnet这个异步网络库。但是看了gnet的代码，才遗憾地发现，gnet的很多代码都是不支持import的，
虽然可以用Dup获得listener的文件描述符，但是gnet的listener是自己实现的，和标准库的net.Listener没啥关系，而且不支持import，
并且不支持通过外部listener来构建gnet特有的listener，这让我在优雅重启的时候，在重建listener上面犯难了，又不太想用unsafe去强改。所以我决定看看，
我能不能提个issue或者pr试试看。后来，issue提了，作者似乎对修改的兴趣不大。看完代码，我也觉得自己无从下手，因为代码我不够熟悉，无法掌握全局。
而且我有一点洁癖，不太喜欢冗余代码，所以看了很久，竟然不知道从何改起，感觉怎么写都不是特别好。

但是gnet关于用户空间的缓存设计，所有操作都作为事件注册到循环，以及无锁事件队列，这些都非常好。于是乎，我决定参考和适用gnet的这些优点，自己做一个代码更优雅，
扩展更方便的网络库。并且gnet不直接支持http，跑去看issues，近期似乎也没有支持TLS的打算，所以，想了一下，我为什么不能在自己的网络库中做这些呢？

于是，有了gknet。gknet在代码上清爽了不少，它将各种平台的系统调用的封装都放到一个模块中，只暴露特定设计的参数、方法、接口给其他通用模块使用，这样就
不会显得杂乱了。需要支持什么平台，就在sys模块中实现相应的方法，修改条件编译，其他不用修改，理论上就能完美支持了。

既然已经支持了http和TLS，那么框架也顺便支持一个吧。所以，就有了gkgin，gkgin底层用gknet收发TCP消息，顶层用gin框架的路由、Context、中间件等，本身
就等于gin，只不过没有使用标准库的TCP收发而已。

gknet的所有设计都坚持不要有太多冗余，如果有标准库实现的，就拿过来用，尽量与标准库兼容。这样可以更方便地适配已有的框架和应用，从而方便用户代码迁移，以及做修改和提pr等。
gknet更开放，很多可以用到的方法、对象之类的，都支持import，这样的话，如果用户只对其中一部分代码感兴趣，也可以通过import在自己的项目中使用。
所以，gknet没有理会go项目所谓的目录规范，只是按照项目代码清晰的需要进行了安排。

后续gknet的优化安排主要有如下几点：
1、事件循环中增加统一的可配置的goroutine池，以此来在资源和性能之间寻求合理的平衡；
2、Conn考虑增加sync.Pool；
3、linux下io_uring异步支持；
