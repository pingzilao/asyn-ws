package base

type EnumID int32

const (
	EVENT_nonesub = "0"
	EVENT_sub     = "1"
	EVENT_unsub   = "2"
	EVENT_SubMax  = "3"

	EnumID_IDNone             EnumID = 0
	EnumID_IDId               EnumID = 1
	EnumID_IDQuoteKlineSingle EnumID = 20
	EnumID_IDMarket           EnumID = 21
	EnumID_IDJianPanBaoShuChu EnumID = 22
)

const (
	ErrCode_Success    = 0
	ErrCode_Param      = 10001
	ErrCode_InerServer = 10002
	ErrCode_UrlPath    = 10003
)
