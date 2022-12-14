[En](https://github.com/moqsien/gknet) | [中文](https://github.com/moqsien/gknet/blob/main/docs/ReadMe_CN.md)

---------------------------
## 什么是gknet？

---------------------------
gknet 是一个基于reactor模式的网络库。gknet的开发受益于gnet项目。但是gknet相比于gnet也有自身的特点。

## gknet有哪些功能？
---------------------------
- gknet支持[gnet](https://github.com/panjf2000/gnet)的几乎所有功能；
- gknet有内置的http server，并且支持TLS；
- gknet适配了著名的微框架[gin](https://github.com/gin-gonic/gin)，能够轻松使用gin的路由、上下文、中间件等所有功能；
- gknet支持epoll和kqueue，能在macos和linux上很好的工作(目前不支持windows)；

## gknet 相较于gnet在哪些方面做得更好？
---------------------------
- 代码可读性更好；
- 现成的http server和框架；
- 全局共享goroutine池，用于异步任务以及读写事件的处理，因此每次事件循环中的事件都能并发进行处理；
- 更加开放，没有internal限制，如果只想引入其中的一部分设计到自己的项目中，大概率你只需要导入相应的包就行了；如果你的操作系统尚未被gknet支持，那么你也可以很轻松的把系统调用接口按照已有的规则封装到sys模块中，然后提pr给gknet，这比分散的系统调用友好太多；

## gknet 使用示例
---------------------------
```bash
go get -u github.com/moqsien/gknet@latest
```
[examples](https://github.com/moqsien/gknet/tree/main/examples)

## 许可
---------------------------
[License](https://github.com/moqsien/gknet/blob/main/LICENSE)

## 感谢
---------------------------
[panjf2000](https://github.com/panjf2000)

## 写gknet的初衷
---------------------------
主要是有一次，我想实现一个支持优雅重启的应用，就想到要支持gnet这个异步网络库。但是看了gnet的代码，才遗憾地发现，gnet的很多代码都是不支持import的，
虽然可以用Dup获得listener的文件描述符，但是gnet的listener是自己实现的，和标准库的net.Listener没啥关系，而且不支持import，
并且不支持通过外部listener来构建gnet特有的listener，这让我在优雅重启的时候，在重建listener上面犯难了，又不太想用unsafe去强改。所以我决定看看，
我能不能提个issue或者pr试试看。后来，issue提了，作者似乎对修改的兴趣不大。看完代码，我也觉得自己无从下手，因为代码我不够熟悉，无法掌握全局。
而且我有一点洁癖，不太喜欢冗余代码，所以看了很久，竟然不知道从何改起，感觉怎么写都不是特别好。

但是gnet关于用户空间的缓存设计，所有操作都作为事件注册到循环，以及无锁事件队列，这些都非常好。于是乎，我决定参考和使用gnet的这些优点，自己做一个代码更优雅，
扩展更方便的网络库。并且gnet不直接支持http，跑去看issues，近期似乎也没有支持TLS的打算，所以，想了一下，我为什么不能在自己的网络库中做这些呢？

于是，有了gknet。gknet在代码上清爽了不少，它将各种平台的系统调用的封装都放到一个模块中，只暴露特定设计的参数、方法、接口给其他通用模块使用，这样就
不会显得杂乱了。需要支持什么平台，就在sys模块中实现相应的方法，修改条件编译，其他不用修改，理论上就能完美支持了。

既然已经支持了http和TLS，那么框架也顺便支持一个吧。所以，就有了gkgin，gkgin底层用gknet收发TCP消息，顶层用gin框架的路由、Context、中间件等，本身
就等于gin，只不过没有使用标准库的TCP收发而已。

gknet的所有设计都坚持不要有太多冗余，如果有标准库实现的，就拿过来用，尽量与标准库兼容。这样可以更方便地适配已有的框架和应用，从而方便用户代码迁移，以及做修改和提pr等。
gknet更开放，很多可以用到的方法、对象之类的，都支持import，这样的话，如果用户只对其中一部分代码感兴趣，也可以通过import在自己的项目中使用。
所以，gknet没有理会go项目所谓的目录规范，只是按照项目代码清晰的需要进行了安排。

后续gknet的功能和优化安排主要有如下几点：
- Conn考虑增加sync.Pool，从而减少创建和内存回收开销；
- linux下io_uring异步支持；
- 更多的框架适配；
- rpc相关适配，包括grpc等；
- addr和port重用支持；
