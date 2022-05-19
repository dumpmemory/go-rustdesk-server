package server

import (
	logs "github.com/danbai225/go-logs"
	"go-rustdesk-server/common"
	"go-rustdesk-server/model/model_proto"
	"google.golang.org/protobuf/proto"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type ringMsg struct {
	ID      string //消息发送者ID
	Type    string
	TimeOut uint32
	InsTime time.Time
	Val     interface{}
}

func getMsgForm(id, Type string, timeOut uint) interface{} {
	if timeOut == 0 {
		timeOut = 3
	}
	after := time.After(time.Second * time.Duration(timeOut))
	for {
		select {
		case <-after:
			return nil
		default:
			next := r.Next()
			val := next.Val()
			now := time.Now()
			if val != nil {
				if v, ok := val.(*ringMsg); ok {
					if now.Add(time.Second * time.Duration(v.TimeOut)).Before(now) {
						next.Set(nil)
					} else if v.ID == id && v.Type == Type {
						next.Set(nil)
						return v.Val
					}
				}
			}
		}
	}
}
func handlerMsg(msg []byte, writer *common.Writer) {
	message := model_proto.RendezvousMessage{}
	err := proto.Unmarshal(msg, &message)
	if err != nil {
		logs.Err(err)
	}
	logs.Info(writer.Type(), writer.GetAddrStr(), reflect.TypeOf(message.Union).String())
	var response proto.Message
	switch reflect.TypeOf(message.Union).String() {
	case model_proto.TypeRendezvousMessagePunchHoleRequest:
		//打洞
		HoleRequest := message.GetPunchHoleRequest()
		if HoleRequest == nil {
			return
		}
		response = model_proto.NewRendezvousMessage(RendezvousMessagePunchHoleRequest(HoleRequest, writer))
	case model_proto.TypeRendezvousMessageRegisterPk:
		//注册公钥
		RegisterPk := message.GetRegisterPk()
		if RegisterPk == nil {
			return
		}
		response = model_proto.NewRendezvousMessage(RendezvousMessageRegisterPk(RegisterPk))
	case model_proto.TypeRendezvousMessageRegisterPeer:
		//注册id
		RegisterPeer := message.GetRegisterPeer()
		if RegisterPeer == nil {
			return
		}
		peer := RendezvousMessageRegisterPeer(RegisterPeer)
		response = model_proto.NewRendezvousMessage(peer)
		if !peer.RequestPk {
			writer.SetKey(RegisterPeer.GetId())
		}
	case model_proto.TypeRendezvousMessageSoftwareUpdate:
		//软件更新
		SoftwareUpdate := message.GetSoftwareUpdate()
		if SoftwareUpdate == nil {
			return
		}
		response = model_proto.NewRendezvousMessage(RendezvousMessageSoftwareUpdate(SoftwareUpdate))
	case model_proto.TypeRendezvousMessageTestNatRequest:
		//网络类型测试
		TestNatRequest := message.GetTestNatRequest()
		if TestNatRequest == nil {
			return
		}
		request := RendezvousMessageTestNatRequest(TestNatRequest)
		str := writer.GetAddrStr()
		split := strings.Split(str, ":")
		parseUint, _ := strconv.ParseUint(split[1], 10, 32)
		request.Port = int32(parseUint)
		response = model_proto.NewRendezvousMessage(request)
	case model_proto.TypeRendezvousMessageLocalAddr:
		//本地地址返回
		LocalAddr := message.GetLocalAddr()
		if LocalAddr == nil {
			return
		}
		RendezvousMessageLocalAddr(LocalAddr, writer)
	case model_proto.TypeRendezvousMessageRequestRelay:
		//请求继中
		RequestRelay := message.GetRequestRelay()
		if RequestRelay == nil {
			return
		}
		response = model_proto.NewRendezvousMessage(RendezvousMessageRequestRelay(RequestRelay))
	case model_proto.TypeRendezvousMessageRelayResponse:
		//请求继中
		RelayResponse := message.GetRelayResponse()
		if RelayResponse == nil {
			return
		}
		RendezvousMessageRelayResponse(RelayResponse)
	default:
		logs.Info(reflect.TypeOf(message.Union).String())
	}
	if response != nil {
		marshal, err2 := proto.Marshal(response)
		if err2 != nil {
			logs.Err(err2)
			return
		}
		_, err2 = writer.Write(marshal)
		if err2 != nil {
			logs.Err(err2)
		}
	}
}