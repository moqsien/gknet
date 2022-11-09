## 什么是gknet？

---------------------------
gknet 是一个基于reactor模式的网络库。gknet的开发受益于gnet项目。但是gknet相比于gnet也有自身的特点。

## gknet有哪些功能？
---------------------------
- gknet支持[gnet](https://github.com/panjf2000/gnet)的几乎所有功能；
- gknet有内置的http server，并且支持TLS；
- gknet适配了著名的微框架gin，能够轻松使用gin的路由、上下文、中间件等所有功能；
- gknet支持epoll和kqueue，能在macos和linux上很好的工作(目前不支持windows)；

## gknet 相较于gnet在哪些方面做得更好？
---------------------------
- 代码可读性更好；
- 现成的http server和框架；
- 更加开放，没有internal限制，如果只想引入其中的一部分设计到自己的项目中，大概率你只需要导入相应的包就行了；如果你的操作系统尚未被gknet支持，那么你也可以很轻松的把系统调用接口按照已有的规则封装到sys模块中，然后提pr给gknet，这比分散的系统调用友好太多；

## gknet 使用示例
[examples](https://github.com/moqsien/gknet/tree/main/examples)

## 许可
[License]()

## 感谢
[panjf2000](https://github.com/panjf2000)
