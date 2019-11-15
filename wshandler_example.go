package base

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"reflect"
	"strings"
	"sync"
	"time"
	"errors"
	"github.com/golang/protobuf/proto"
)

type ResBtcExaSubI struct {
	Topic  string     `json:"topic,omitempty"`
	Status string     `json:"status, omitempty"`
	Type   string     `json:"type, omitempty"`
	Data   [][]string `json:"data,omitempty"`
}

type ResBtcExaSubU struct {
	Topic  string   `json:"topic,omitempty"`
	Status string   `json:"status, omitempty"`
	Type   string   `json:"type, omitempty"`
	Data   []string `json:"data,omitempty"`
}

type MSG struct {
	Id                      *int32              `json:"Id,omitempty"`
	ResDataQuoteKlineSingle []*QuoteKlineSingle `json:"RepDataQuoteKlineSingle,omitempty"`
}

type QuoteKlineSingle struct {
	Obj              *string  `json:"Obj,omitempty"`
	Data             []*KXian `json:"Data,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (this *QuoteKlineSingle) Append() {

}

func (m *QuoteKlineSingle) Reset() { *m = QuoteKlineSingle{} }

func (m *QuoteKlineSingle) GetObj() string {
	if m != nil && m.Obj != nil {
		return *m.Obj
	}
	return ""
}

func (m *QuoteKlineSingle) GetData() []*KXian {
	if m != nil {
		return m.Data
	}
	return nil
}

type KXian struct {
	ShiJian        *int64  `json:"ShiJian,omitempty"`
	KaiPanJia      *string `json:"KaiPanJia,omitempty"`
	ZuiGaoJia      *string `json:"ZuiGaoJia,omitempty"`
	ZuiDiJia       *string `json:"ZuiDiJia,omitempty"`
	ShouPanJia     *string `json:"ShouPanJia,omitempty"`
	ChengJiaoLiang *string `json:"ChengJiaoLiang,omitempty"`
	ChengJiaoE     *string `json:"ChengJiaoE,omitempty"`
	ChengJiaoBiShu *string `json:"ChengJiaoBiShu,omitempty"`
}

func (this *KXian) SetShiJian(s int64) {
	if this.ShiJian == nil {
		this.ShiJian = new(int64)
	}
	*this.ShiJian = s
}

func (this *KXian) SetKaiPanJia(s string) {
	if this.KaiPanJia == nil {
		this.KaiPanJia = new(string)
	}
	*this.KaiPanJia = s
}

func (this *KXian) SetZuiGaoJia(s string) {
	if this.ZuiGaoJia == nil {
		this.ZuiGaoJia = new(string)
	}
	*this.ZuiGaoJia = s
}

func (this *KXian) SetZuiDiJia(s string) {
	if this.ZuiDiJia == nil {
		this.ZuiDiJia = new(string)
	}
	*this.ZuiDiJia = s
}

func (this *KXian) SetShouPanJia(s string) {
	if this.ShouPanJia == nil {
		this.ShouPanJia = new(string)
	}
	*this.ShouPanJia = s
}

func (this *KXian) SetChengJiaoLiang(s string) {
	if this.ChengJiaoLiang == nil {
		this.ChengJiaoLiang = new(string)
	}
	*this.ChengJiaoLiang = s
}

func (this *KXian) SetChengJiaoE(s string) {
	if this.ChengJiaoE == nil {
		this.ChengJiaoE = new(string)
	}
	*this.ChengJiaoE = s
}

func (this *KXian) SetChengJiaoBiShu(s string) {
	if this.ChengJiaoBiShu == nil {
		this.ChengJiaoBiShu = new(string)
	}
	*this.ChengJiaoBiShu = s
}


type SubKeyInfo struct {
	Data   []byte              //发送消息，断线重发的时候使用
	SubMap map[string]*SubItem //uuid, version
}

func (this *SubKeyInfo) Add(sessionId string, ver uint64) {
	this.SubMap[sessionId] = &SubItem{ver}
}

func (this *SubKeyInfo) Del(sessionId string) {
	delete(this.SubMap, sessionId)
}

func (this *SubKeyInfo) Len() int {
	return len(this.SubMap)
}

func NewSubKeyInfo() *SubKeyInfo {
	s := new(SubKeyInfo)
	s.SubMap = make(map[string]*SubItem)
	return s
}

type SubItem struct {
	Ver uint64 //涉及到版本就是用
}

type BtcExaModelsWsHandler struct {
	mRecvResChan chan error
	mWsCon       *WsCon
	mSubKeyMap   map[string]*SubKeyInfo //topic
	ConId        string
	mSendLock    sync.Mutex
	sync.Mutex
}

func (this *BtcExaModelsWsHandler) IsRecvPingHander() bool {
	return true
}

func (this *BtcExaModelsWsHandler) IsRecvPongHander() bool {
	return true
}

func (this *BtcExaModelsWsHandler) RecvPingHander(message string) string {
	return ""
}

func (this *BtcExaModelsWsHandler) RecvPongHander(message string) string {
	return ""
}

func (this *BtcExaModelsWsHandler) SendPingHander() []byte {
	return []byte("{\"event\":\"ping\"}")
}

func (this *BtcExaModelsWsHandler) BeforeSendHandler() {
	this.mSendLock.Lock()
	ZapLog().Info("BeforeSendHandler lock")
	for {
		select {
		case <-this.mRecvResChan:
			break
		default:
			return
		}
	}
}

func (this *BtcExaModelsWsHandler) AfterSendHandler() error {
	defer this.mSendLock.Unlock()
	ZapLog().Info("AfterSendHandler Unlock")
	select {
	case err := <-this.mRecvResChan:
		return err
	case <-time.After(time.Second * 30):
		return errors.New("chan timeout 30s")
	}
	return nil
}

func (this *BtcExaModelsWsHandler) RecvHandler(conId string, message []byte) {
	slic := make([]interface{}, 0)
	err := json.Unmarshal(message, &slic)
	if err != nil {
		ZapLog().Error("unMarshal err", zap.Error(err), zap.String("btcexa_sub_msg", string(message)))
		return
	}
	if len(slic) <= 0 {
		ZapLog().Error("unMarshal err , no data")
		return
	}

	topic, ok := slic[0].(string)
	if !ok {
		ZapLog().Error("topic type err")
		return
	}
	ty, ok := slic[1].(string)
	if !ok {
		ZapLog().Error("ty type err")
		return
	}
	status, ok := slic[2].(string)
	if !ok {
		ZapLog().Error("status type err")
		return
	}

	newTopic := strings.TrimLeft(topic, "sub.")

	if ty == "r" {
		if strings.ToUpper(status) == "OK" {
			this.mRecvResChan <- nil
		} else {
			this.mRecvResChan <- fmt.Errorf("%s", ty)
		}
		return
	}

	resmsg := &MSG{}
	if strings.Contains(newTopic, "kline") {
		kline, err := this.RecvKxianHandler(ty, topic, status, slic)
		if err != nil {
			ZapLog().Error("RecvKxianHandler err", zap.Error(err))
			return
		}
		resmsg.ResDataQuoteKlineSingle = []*QuoteKlineSingle{kline}
	}  else {
		ZapLog().Error("GBtcExaModels unknown sub msg")
		return
	}
	// else 各种消息类型
	//推送消息
	this.Lock()
	defer this.Unlock()
	subKeyInfoList, ok := this.mSubKeyMap[newTopic]
	if !ok {
		ZapLog().Error("unknown topic", zap.String("topic", newTopic))
		return
	}
	for k, v := range subKeyInfoList.SubMap {
		ZapLog().Info("push uuid =" + k)
		GWsServerMgr.Push(k, v.Ver, resmsg)
	}
}

func (this *BtcExaModelsWsHandler) RecvKxianHandler(ty, topic, status string, slic []interface{}) (*QuoteKlineSingle, error) {
	newTopic := strings.TrimLeft(topic, "sub.")
	if ty == "i" {
		//return init的数据放开
		res := new(ResBtcExaSubI)
		res.Topic = topic
		res.Type = ty
		res.Status = status
		data1, ok := slic[3].([]interface{})
		if !ok {
			ZapLog().Error("change i data wrong1")
			return nil, fmt.Errorf("type err")
		}
		dataValue1 := make([][]string, 0)
		for i := 0; i < len(data1); i++ {
			data2, ok := data1[i].([]interface{})
			if !ok {
				ZapLog().Error("change i data wrong2 ", zap.Any("type", reflect.TypeOf(data1[i])))
				return nil, fmt.Errorf("type err")
			}

			dataValue2 := make([]string, 0)
			for j := 0; j < len(data2); j++ {
				data3 := data2[j].(string)
				dataValue2 = append(dataValue2, data3)
			}
			dataValue1 = append(dataValue1, dataValue2)
		}
		res.Data = dataValue1

		apiKxian := BtcExaSubToApiKXianI(res)
		ZapLog().Info("recv=", zap.Any("res=", apiKxian))
		topicArr := strings.Split(newTopic, ".")
		obj := ""
		if len(topicArr) >= 3 {
			obj = topicArr[1]
		}
		singleKXian := &QuoteKlineSingle{
			Obj:  proto.String(obj),
			Data: BtcExaSubToApiKXianI(res),
		}
		return singleKXian, nil

	}

	if ty == "u" {
		res := new(ResBtcExaSubU)
		res.Topic = topic
		res.Type = ty
		res.Status = status
		data, ok := slic[3].([]interface{})
		if !ok {
			ZapLog().Error("recv data wrong1")
			return nil, fmt.Errorf("type err")

		}
		dataValue := make([]string, 0)
		if data != nil {
			for i := 0; i < len(data); i++ {
				data, ok := data[i].(string)
				if !ok {
					ZapLog().Error("recv data wrong2")
					return nil, fmt.Errorf("type err")
				}
				dataValue = append(dataValue, data)
			}
			res.Data = dataValue
		} else {
			return nil, nil
		}

		topicArr := strings.Split(newTopic, ".")
		obj := ""
		if len(topicArr) >= 3 {
			obj = topicArr[1]
		}
		singleKXian := &QuoteKlineSingle{
			Obj:  proto.String(obj),
			Data: BtcExaSubToApiKXianU(res),
		}
		return singleKXian, nil

	}
	return nil, nil
}

func (this *BtcExaModelsWsHandler) ReSendAllRequest() {
	errMap := make(map[string]bool)
	subMap := this.GetSubMap()
	ZapLog().Info("start ReSendAllRequest", zap.Int("topicnum", len(subMap)))
	for topic, info := range subMap {
		ZapLog().Info("start topic " + topic)
		if err := this.mWsCon.Send(websocket.BinaryMessage, info); err != nil {
			ZapLog().Sugar().Errorf("ReSendAllRequest fail %v", topic)
			errMap[topic] = true
		} else {
			ZapLog().Info("ReSendAllRequest success", zap.String("topic", topic))
		}
		ZapLog().Info("end topic " + topic)
	} //重试3次
}

func (this *BtcExaModelsWsHandler) GetSubMap() map[string][]byte {
	mm := make(map[string][]byte)
	this.Lock()
	defer this.Unlock()
	for topic, info := range this.mSubKeyMap {
		mm[topic] = info.Data
	}
	return mm
}

func (this *BtcExaModelsWsHandler) Sub(topic, sessionId string, data []byte) bool {
	this.Lock()
	defer this.Unlock()
	subKeyInfo, ok := this.mSubKeyMap[topic]
	if !ok {
		subKeyInfo = NewSubKeyInfo()
		this.mSubKeyMap[topic] = subKeyInfo
	}
	subKeyInfo.Add(sessionId, 0)
	subKeyInfo.Data = data
	return !ok
}

func (this *BtcExaModelsWsHandler) UnSub(topic, sessionId string) bool {
	this.Lock()
	defer this.Unlock()
	subKeyInfo, ok := this.mSubKeyMap[topic]
	if !ok {
		return false
	}
	subKeyInfo.Del(sessionId)
	if subKeyInfo.Len() == 0 {
		delete(this.mSubKeyMap, topic)
		return true
	} else {
		return false
	}

}


func BtcExaSubToApiKXianI(k1 *ResBtcExaSubI) []*KXian {
	if len(k1.Data) <= 0 {
		ZapLog().Error("data length <= 0 btcexa I data nil")
		return nil
	}

	k2 := make([]*KXian, 0)
	for j := 0; j < len(k1.Data); j++ {
		tmp := new(KXian)
		for i := 0; i < len(k1.Data[j]); i++ {
			//类型转换
		}
		k2 = append(k2, tmp)
	}

	return k2
}

func BtcExaSubToApiKXianU(k1 *ResBtcExaSubU) []*KXian {
	if len(k1.Data) <= 0 {
		ZapLog().Error("data length <= 0 btcexa U data nil")
		return nil
	}

	k2 := make([]*KXian, 0)
	tmp := new(KXian)
	for i := 0; i < len(k1.Data); i++ {
		//类型转换
	}
	k2 = append(k2, tmp)

	return k2
}