package godis

import (
	"errors"
)


type ProtoType map[interface{}]interface{}
type HandleServerFunc *func(args ProtoType) (int64, ProtoType, error)
type HandleClientFunc *func(args RespInfo) (interface{}, error)
var  MaxRetryCount = 2


var errMsgMap = map[int64]string{
	1000: "service not found",
	1001: "request timeOut",
	1002: "server has not this service",

}

var ConvertErr = errors.New("event interface conversion error")
var SCRepeatErr  = errors.New("the server or client id is register already")
var ServiceRepeatErr  = errors.New("the service is register already")



