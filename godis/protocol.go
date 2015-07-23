package godis

import
(
	"errors"
	uuid "github.com/nu7hatch/gouuid"
	"github.com/ugorji/go/codec"
)




type ProtoType map[interface{}]interface{}
type HandleServerFunc *func(args ProtoType) (ProtoType, error)
type HandleClientFunc *func(args RespInfo) (interface{}, error)
var  MaxRetryCount = 2
// Event representation
type Event struct {
	MId   string //机器id
	MsgId string //
	Name  string //对应服务名称
	Args  ProtoType
}

type RespInfo struct {
	Code   int64
	Data   ProtoType
	ErrMsg string
}
type Resp  struct {
	MsgId    string
	RespInfo RespInfo
}



// Returns a pointer to a new event,
// a UUID V4 message_id is generated
func newEvent(mId string, name string, args ProtoType) (Event, error) {
	id, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	e := Event{
		MId: mId,
		MsgId: id.String(),
		Name: name,
		Args: args,
	}
	return e, nil
}


func newResp(msgId string, code int64, err string, data ProtoType) (Resp) {
	r := RespInfo{
		Code: code,
		Data: data,
	}
	if err!="" {
		r.ErrMsg=err
	}
	resp := Resp{
		MsgId:msgId,
		RespInfo:r,
	}
	return resp
}


// Packs an event into MsgPack bytes
func (r *Resp) packBytes() ([]byte, error) {
	data := make([]interface{}, 3)
	data[0] = r.MsgId
	data[1] = r.RespInfo.Code
	data[2] = r.RespInfo.Data
	if len(r.RespInfo.ErrMsg)>0 {
		data = append(data, r.RespInfo.ErrMsg)
	}
	return encode(data)
}



// Packs an event into MsgPack bytes
func (e *Event) packBytes() ([]byte, error) {
	data := make([]interface{}, 4)
	data[0] =e.MId
	data[1] = e.MsgId
	data[2] = e.Name
	data[3] = e.Args
	return encode(data)
}

func encode(data []interface{}) ([]byte, error) {

	var buf []byte

	enc := codec.NewEncoderBytes(&buf, &codec.MsgpackHandle{})
	if err := enc.Encode(data); err != nil {
		return nil, err
	}

	return buf, nil
}

func decode(b []byte) (interface{}, error) {

	var mh codec.MsgpackHandle
	var v interface{}
	dec := codec.NewDecoderBytes(b, &mh)

	err := dec.Decode(&v)
	if err != nil {
		return nil, err
	}
	return v, nil
}


//// Unpacks an event fom MsgPack bytes
func unPackEventBytes(b []byte) (*Event, error) {

	v, err := decode(b)
	if err != nil {
		return nil, err
	}

	// get the event headers
	m, ok := v.([]interface{})[0].([]byte)
	if !ok {
		return nil, errors.New("zerorpc/event interface conversion error")
	}
	// get the event headers
	h, ok := v.([]interface{})[1].([]byte)
	if !ok {
		return nil, errors.New("zerorpc/event interface conversion error")
	}

	// get the event headers
	n, ok := v.([]interface{})[2].([]byte)
	if !ok {
		return nil, errors.New("zerorpc/event interface conversion error")
	}
	// get the event args
	args := convertValue(v.([]interface{})[3])

	e := Event{
		MId:   string(m),
		MsgId: string(h),
		Name:  string(n),
		Args:  args.(map[interface{}]interface{}),
	}

	return &e, nil
}


func unPackRespByte(b []byte) (*Resp, error) {

	v, err := decode(b)
	if err != nil {
		return nil, err
	}
	msgId := convertValue(v.([]interface{})[0])
	code := convertValue(v.([]interface{})[1])
	data := convertValue(v.([]interface{})[2])
	r := RespInfo{
		Code: code.(int64),
	}
	if (data!=nil) {
		r.Data = data.(map[interface{}]interface{})
	}
	if (len(v.([]interface{}))>3) {
		er := convertValue(v.([]interface{})[3])
		r.ErrMsg = er.(string)
	}
	resp := Resp{
		MsgId: msgId.(string),
		RespInfo: r,
	}
	return &resp, nil
}
//
//// Returns a pointer to a new heartbeat event
//func newHeartbeatEvent() (*Event, error) {
//	ev, err := newEvent("_zpc_hb", nil)
//	if err != nil {
//		return nil, err
//	}
//
//	return ev, nil
//}

// converts an interface{} to a type
func convertValue(v interface{}) interface{} {
	var out interface{}

	switch t := v.(type) {
		case []byte:
		out = string(t)

		case []interface{}:
		for i, x := range t {
			t[i] = convertValue(x)
		}

		out = t

		case map[interface{}]interface{}:
		for key, val := range v.(map[interface{}]interface{}) {
			t[key] = convertValue(val)
		}
		out = t

		default:
		out = t
	}

	return out
}
