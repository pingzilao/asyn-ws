package base

import (
	"fmt"
	"strings"

	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/logger"
	"github.com/kataras/iris/middleware/pprof"
	"github.com/kataras/iris/middleware/recover"
	"github.com/kataras/iris/websocket"
	"go.uber.org/zap"
	"io/ioutil"
	"errors"
	"encoding/json"
	"gopkg.in/yaml.v2"
)

type WebServer struct {
	mIris *iris.Application
}

func NewWebServer() *WebServer {
	web := new(WebServer)
	if err := web.Init(); err != nil {
		ZapLog().With(zap.Error(err)).Error("quote Init err")
		panic("quote Init err")
	}
	return web
}

func (this *WebServer) Init() error {
	app := iris.New()
	app.Use(recover.New())
	app.Use(logger.New())
	if GConfig.Server.Debug {
		app.Any("/debug/pprof/{action:path}", pprof.New())
	}
	this.mIris = app
	this.controller()
	ZapLog().Info("WebServer Init ok")
	return nil
}

func (this *WebServer) Run() error {
	ZapLog().Info("WebServer Run with port[" + GConfig.Server.Port + "]")
	err := this.mIris.Run(iris.Addr(":" + GConfig.Server.Port)) //阻塞模式
	if err != nil {
		if err == iris.ErrServerClosed {
			ZapLog().Sugar().Infof("Iris Run[%v] Stoped[%v]", GConfig.Server.Port, err)
		} else {
			ZapLog().Sugar().Errorf("Iris Run[%v] err[%v]", GConfig.Server.Port, err)
		}
	}
	return nil
}

func (this *WebServer) Stop() error { //这里要处理下，全部锁得再看看，还有就是qid
	return nil
}

/********************内部接口************************/
func (a *WebServer) controller() {
	app := a.mIris
	//app.UseGlobal(interceptorCtrl.Interceptor)
	crs := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Authorization", "X-Requested-With", "X_Requested_With", "Content-Type", "Access-Token", "Accept-Language"},
		AllowCredentials: true,
	})

	app.Any("/", func(ctx iris.Context) {
		ctx.JSON(map[string]interface{}{
			"code":    0,
			"message": "ok",
			"data":    "",
		})
	})

	/********http***********/
	v1 := app.Party("/api/v1/tvproxy", crs, func(ctx iris.Context) { ctx.Next() }).AllowMethods(iris.MethodOptions)
	{
		v1.Get("/coinmerit/quote/kxian", controllers.GCoinMerit.HandleHttpKXian)
		v1.Get("/coinmerit/quote/market", controllers.GCoinMerit.HandleHttpExa)
		v1.Get("/coinmerit/quote/objs", controllers.GCoinMerit.HandleHttpObjList)
		v1.Get("/btcexa/quote/kxian", controllers.GBtcExa.HandleHttpKXian)
		v1.Get("/btcexa/quote/objs", controllers.GBtcExa.HandleHttpObjList)
		v1.Get("/quote/kxian", controllers.GQuoteCtrl.HandleHttpKXian)
		v1.Get("/quote/kbspirit", controllers.GKBSpirit.HandleGetObjs)
		v1.Get("/quote/dyna", controllers.GBtcExa.HandleHttpDync)
		//v1.Get("/quote/market/list",   controllers.GKBSpirit.HandleGetObjs)
		v1.Any("/", a.defaultRoot)
	}

	//Ws 设置
	Ws := websocket.New(websocket.Config{})
	app.Get("/api/v1/tvproxy/ws", Ws.Handler()) //iris设置ws
	GWsServerMgr.Init()
	GWsServerMgr.RegPackHander(PackPushMsg, PackResMsg)
	//GWsServerMgr.RegNewRequesterHandler(NewRequester)
	Ws.OnConnection(GWsServerMgr.HandleWsConnection)

	//ws注册回调函数
	GWsServerMgr.RegHandler("/api/v1/tvproxy/coinmerit/quote/kxian", controllers.GCoinMerit.HandleWsKXian)
	GWsServerMgr.RegHandler("/api/v1/tvproxy/btcexa/quote/kxian", controllers.GBtcExa.HandleWsKXian)
	GWsServerMgr.RegHandler("/api/v1/tvproxy/quote/kxian", controllers.GQuoteCtrl.HandleWsKXian)
	GWsServerMgr.RegHandler("/api/v1/tvproxy/quote/dyna", controllers.GBtcExa.HandleWsDyna)

}

func (this *WebServer) defaultRoot(ctx iris.Context) {
	resMsg := NewResponse("")
	ctx.JSON(resMsg)
}

//*******配置文件 config*************************************************************************************//
var GConfig Config
var GPreConfig PreConfig

func LoadConfig(path string) *Config {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(fmt.Errorf("Read yml config[%s] err[%v]", path, err).Error())
	}

	err = yaml.Unmarshal([]byte(data), &GConfig)
	if err != nil {
		panic(fmt.Errorf("yml.Unmarshal config[%s] err[%v]", path, err).Error())
	}
	fmt.Println("data===", GConfig.Server.Port, GConfig.CoinMerit.HttpUrl)
	return &GConfig
}

func PreProcess() error {
	if GPreConfig.MarketMap == nil {
		GPreConfig.MarketMap = make(map[string]*Market)
	}
	if GPreConfig.CoinDetailMap == nil {
		GPreConfig.CoinDetailMap = make(map[string]*CoinDetail)
	}
	if GPreConfig.Markets == nil {
		GPreConfig.Markets = make([]string, 0)
	}
	marketArr := make([]*Market, 0)
	for i := 0; i < len(GConfig.Markets); i++ {
		AbbArr := strings.Split(strings.TrimRight(strings.ToUpper(strings.Replace(GConfig.Markets[i].Abb, " ", "", -1)), ","), ",")
		NameArr := strings.Split(strings.TrimRight(strings.ToUpper(strings.Replace(GConfig.Markets[i].Name, " ", "", -1)), ","), ",")
		zhNameArr := strings.Split(strings.TrimRight(strings.ToUpper(strings.Replace(GConfig.Markets[i].ZhName, " ", "", -1)), ","), ",")
		webInfoArr := strings.Split(strings.TrimRight(strings.ToUpper(strings.Replace(GConfig.Markets[i].WebInfo, " ", "", -1)), ","), ",")
		if len(AbbArr) != len(NameArr) || len(NameArr) != len(zhNameArr) || len(zhNameArr) != len(webInfoArr) {
			fmt.Println(AbbArr)
			fmt.Println(NameArr)
			fmt.Println(zhNameArr)
			fmt.Println(webInfoArr)

			return errors.New("Markets config err " + fmt.Sprintf("%v %v %v %v", len(AbbArr), len(NameArr), len(zhNameArr), len(webInfoArr)))
		}
		for j := 0; j < len(AbbArr); j++ {
			market := new(Market)
			market.Abb = AbbArr[j]
			market.Name = NameArr[j]
			market.ZhName = zhNameArr[j]
			market.WebInfo = webInfoArr[j]
			GPreConfig.MarketMap[market.Abb] = market
			marketArr = append(marketArr, market)
			GPreConfig.Markets = append(GPreConfig.Markets, AbbArr[j])
		}
	}
	GConfig.Markets = marketArr

	coinDetailArr := make([]*CoinDetail, 0)
	for i := 0; i < len(GConfig.CoinDetails); i++ {
		NameArr := strings.Split(strings.TrimRight(strings.ToUpper(strings.Replace(GConfig.CoinDetails[i].Name, " ", "", -1)), ","), ",")
		zhNameArr := strings.Split(strings.TrimRight(strings.ToUpper(strings.Replace(GConfig.CoinDetails[i].ZhName, " ", "", -1)), ","), ",")
		pinyinArr := strings.Split(strings.TrimRight(strings.ToUpper(strings.Replace(GConfig.CoinDetails[i].PinYin, " ", "", -1)), ","), ",")
		if len(NameArr) != len(zhNameArr) || len(zhNameArr) != len(pinyinArr) {
			fmt.Println(NameArr)
			fmt.Println(zhNameArr)
			fmt.Println(pinyinArr)
			return errors.New("CoinDetail config err " + fmt.Sprintf("%v %v %v", len(NameArr), len(zhNameArr), len(pinyinArr)))
		}
		for j := 0; j < len(NameArr); j++ {
			deatil := new(CoinDetail)
			deatil.Name = NameArr[j]
			deatil.ZhName = zhNameArr[j]
			deatil.PinYin = pinyinArr[j]
			GPreConfig.CoinDetailMap[deatil.Name] = deatil
			coinDetailArr = append(coinDetailArr, deatil)
		}
	}
	GConfig.CoinDetails = coinDetailArr

	return nil
}

type PreConfig struct {
	CoinDetailMap map[string]*CoinDetail //HBI
	MarketMap     map[string]*Market     //HBI
	Markets       []string               // HBI、BIC
}

type Config struct {
	Server      Server        `yaml:"server"`
	CoinMerit   CoinMerit     `yaml:"coinmerit"`
	BtcExa      BtcExa        `yaml:"btcexa"`
	Markets     []*Market     `yaml:"market"`
	CoinDetails []*CoinDetail `yaml:"coin_detail"`
}

type CoinMerit struct {
	Secret_key string `yaml:"secret_key"`
	ApiKey     string `yaml:"api_key"`
	WsUrl      string `yaml:"ws_url"`
	HttpUrl    string `yaml:"http_url"`
}

type Server struct {
	Port    string `yaml:"port"`
	Debug   bool   `yaml:"debug"`
	LogPath string `yaml:"log"`
}

type BtcExa struct {
	Secret_key string `yaml:"secret_key"`
	ApiKey     string `yaml:"api_key"`
	WsUrl      string `yaml:"ws_url"`
	HttpUrl    string `yaml:"http_url"`
}

type Market struct {
	Abb     string `yaml:"abb"`
	Name    string `yaml:"name"`
	ZhName  string `yaml:"zh_name"`
	WebInfo string `yaml:"web_info"`
}

type CoinDetail struct {
	Name   string `yaml:"name"`
	ZhName string `yaml:"zh_name"`
	PinYin string `yaml:"pinyin"`
}

///////////////////////////////////////////////////////////////////////////
func NewErrResponse(qid string, err int32) *Response {
	res := new(Response)
	res.Qid = new(string)
	*res.Qid = qid
	res.SetErr(err)
	return res
}

func NewResponse(qid string) *Response {
	res := new(Response)
	res.Qid = new(string)
	*res.Qid = qid
	return res
}

type Response struct {
	Qid     *string     `json:"Qid,omitempty"`     //请求id，流水号, 单次请求唯一
	Err     *int32      `json:"Err,omitempty"`     //错误号
	Counter *uint64     `json:"Counter,omitempty"` //推送序号, 0表示正常请求，>=1 表示推送
	Data    interface{} `json:"Data,omitempty"`    //数据
}

func (this *Response) SetCounter(e uint64) {
	if this.Counter == nil {
		this.Counter = new(uint64)
	}
	*this.Counter = e
}

func (this *Response) SetErr(e int32) {
	if this.Err == nil {
		this.Err = new(int32)
	}
	*this.Err = e
}

func (this *Response) Marshal(msg interface{}) ([]byte, error) {
	if msg != nil {
		this.Data = msg
	}
	return json.Marshal(this)
}


func PackPushMsg(reqster Requester, apiMsg interface{}) ([]byte, error) {
	apiResponse := NewResponse(reqster.GetQid())
	apiResponse.SetCounter(reqster.GetCounter())

	content, err := apiResponse.Marshal(apiMsg)
	if err != nil {
		ZapLog().Error("apiResponse Marshal err", zap.Error(err))
		return nil, err
	}
	return content, nil
}

func PackResMsg(reqster Requester, errCode int32, apiMsg interface{}) ([]byte, error) {
	apiResponse := NewResponse(reqster.GetQid())
	apiResponse.SetErr(errCode)
	content, err := apiResponse.Marshal(apiMsg)
	if err != nil {
		ZapLog().Error("apiResponse Marshal err", zap.Error(err))
		return nil, err
	}
	return content, nil
}