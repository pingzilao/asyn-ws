package base

type WsHandler interface {
	RecvPingHander(message string) string
	RecvPongHander(message string) string
	SendPingHander() []byte
	AfterSendHandler() error
	BeforeSendHandler()
	RecvHandler(conId string, message []byte)
	IsRecvPingHander() bool
	IsRecvPongHander() bool
	ReSendAllRequest()
}
