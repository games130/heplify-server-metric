package decoder

import (
	"time"

	proto "github.com/games130/protoMetric"
)

// The first 4 bytes are the string "HEP3". The next 2 bytes are the length of the
// whole message (len("HEP3") + length of all the chucks we have. The next bytes
// are all the chuncks created by makeChuncks()
// Bytes: 0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31......
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//        | "HEP3"|len|chuncks(0x0001|0x0002|0x0003|0x0004|0x0007|0x0008|0x0009|0x000a|0x000b|......)
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

// HEP represents HEP packet
type HEP struct {
	Version     uint32 `protobuf:"varint,1,opt,name=Version,proto3" json:"Version,omitempty"`
	Protocol    uint32 `protobuf:"varint,2,opt,name=Protocol,proto3" json:"Protocol,omitempty"`
	SrcIP       string `protobuf:"bytes,3,opt,name=SrcIP,proto3" json:"SrcIP,omitempty"`
	DstIP       string `protobuf:"bytes,4,opt,name=DstIP,proto3" json:"DstIP,omitempty"`
	SrcPort     uint32 `protobuf:"varint,5,opt,name=SrcPort,proto3" json:"SrcPort,omitempty"`
	DstPort     uint32 `protobuf:"varint,6,opt,name=DstPort,proto3" json:"DstPort,omitempty"`
	Tsec        uint32 `protobuf:"varint,7,opt,name=Tsec,proto3" json:"Tsec,omitempty"`
	Tmsec       uint32 `protobuf:"varint,8,opt,name=Tmsec,proto3" json:"Tmsec,omitempty"`
	ProtoType   uint32 `protobuf:"varint,9,opt,name=ProtoType,proto3" json:"ProtoType,omitempty"`
	NodeID      uint32 `protobuf:"varint,10,opt,name=NodeID,proto3" json:"NodeID,omitempty"`
	NodePW      string `protobuf:"bytes,11,opt,name=NodePW,proto3" json:"NodePW,omitempty"`
	Payload     string `protobuf:"bytes,12,opt,name=Payload,proto3" json:"Payload,omitempty"`
	CID         string `protobuf:"bytes,13,opt,name=CID,proto3" json:"CID,omitempty"`
	Vlan        uint32 `protobuf:"varint,14,opt,name=Vlan,proto3" json:"Vlan,omitempty"`
	CseqMethod  string `protobuf:"bytes,15,opt,name=CseqMethod,proto3" json:"CseqMethod,omitempty"`
	FirstMethod string `protobuf:"bytes,16,opt,name=FirstMethod,proto3" json:"FirstMethod,omitempty"`
	CallID      string `protobuf:"bytes,17,opt,name=CallID,proto3" json:"CallID,omitempty"`
	FromUser    string `protobuf:"bytes,18,opt,name=FromUser,proto3" json:"FromUser,omitempty"`
	Expires     string `protobuf:"bytes,19,opt,name=Expires,proto3" json:"Expires,omitempty"`
	ReasonVal   string `protobuf:"bytes,20,opt,name=ReasonVal,proto3" json:"ReasonVal,omitempty"`
	RTPStatVal  string `protobuf:"bytes,21,opt,name=RTPStatVal,proto3" json:"RTPStatVal,omitempty"`
	ToUser      string `protobuf:"bytes,22,opt,name=ToUser,proto3" json:"ToUser,omitempty"`
	ProtoString string
	Timestamp   time.Time
	HostTag     string
	NodeName    string
}

// DecodeHEP returns a parsed HEP message
func DecodeHEP(packet *proto.Event) (*HEP, error) {
	hep := &HEP{}
	
	hep.Version      = packet.GetVersion()
	hep.Protocol     = packet.GetProtocol()
	hep.SrcIP        = packet.GetSrcIP()
	hep.DstIP        = packet.GetDstIP()
	hep.SrcPort      = packet.GetSrcPort()
	hep.DstPort      = packet.GetDstPort()
	hep.Tsec         = packet.GetTsec()
	hep.Tmsec        = packet.GetTmsec()
	hep.ProtoType    = packet.GetProtoType()
	hep.NodeID       = packet.GetNodeID()
	hep.NodePW       = packet.GetNodePW()
	hep.Payload      = packet.GetPayload()
	hep.CID          = packet.GetCID()
	hep.Vlan         = packet.GetVlan()
	hep.CseqMethod   = packet.GetCseqMethod()
	hep.FirstMethod  = packet.GetFirstMethod()
	hep.CallID       = packet.GetCallID()
	hep.FromUser     = packet.GetFromUser()
	hep.Expires      = packet.GetExpires()
	hep.ReasonVal    = packet.GetReasonVal()
	hep.RTPStatVal   = packet.GetRTPStatVal()
	hep.ToUser       = packet.GetToUser()
	hep.ProtoString  = packet.GetProtoString()
	hep.Timestamp    = packet.GetTimestamp()	
	hep.HostTag      = packet.GetHostTag()
	hep.NodeName     = packet.GetNodeName()

	return hep, nil
}
