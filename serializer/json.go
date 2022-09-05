package serializer

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ProtobufToJSON converts protocol buffer message to JSON string
func ProtobufToJSON(message protoreflect.ProtoMessage) (string, error) {

	jsonBytes := protojson.MarshalOptions{
		Multiline: true,
		UseEnumNumbers: false,
		Indent:         "  ",
		UseProtoNames:  true,
	}

	return jsonBytes.Format(message), nil
}
