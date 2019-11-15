# asyn-ws
 asynchronous request or sub/pub base on iris and websocket
 
# 异步WebSocket框架
 基于iris和websocket 开发的长连接异步请求request框架，基于iris和websocket 开发的长连接异步推送pub/sub框架。
 
 1. wsserver和wsservermgr是基于iris实现得服务端ws模块
    requester是服务端请求消息抽象，包含qid，uuid，conid，counter。qid为消息id，连接唯一且递增；uuid是全网消息唯一id；conid是标识程序连接得id，counter是本链接推送消息递增id。
    RegHandler(path string, f func(request Requester))注册回调函数。
    支持订阅、取消订阅功能。
    支持断线相关资源清理。
    订阅关系必须根据业务自行实现，参考wshandler_example
    
 2. ws和wsmgr是基于websocket实现得客户端ws模块。
    wshandler是回调处理程序。
    支持断线重连。
    支持重连后topic自动订阅。
    心跳消息自动发送。
    
 3. example是示例代码，使用前请先初始化日志模块。或者替换掉日志相关语句