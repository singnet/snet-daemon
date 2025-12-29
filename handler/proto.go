package handler

import (
	"errors"
	"fmt"

	"github.com/bufbuild/protocompile/linker"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func jsonToProto(protoFiles linker.Files, json []byte, methodName string) (proto proto.Message, err error) {

	method := findMethodInProto(protoFiles, methodName)
	if method == nil {
		zap.L().Error("[jsonToProto] method not found in proto for http call")
		return proto, errors.New("method in proto not found")
	}

	output := method.Output()
	zap.L().Debug("output msg descriptor", zap.String("fullname", string(output.FullName())))
	proto = dynamicpb.NewMessage(output)
	err = protojson.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}.Unmarshal(json, proto)
	if err != nil {
		zap.L().Error("can't unmarshal json to proto", zap.Error(err))
		return proto, fmt.Errorf("invalid proto, can't convert json to proto msg: %+v", err)
	}

	return proto, nil
}

func protoToJson(protoFiles linker.Files, in []byte, methodName string) (json []byte, err error) {

	method := findMethodInProto(protoFiles, methodName)
	if method == nil {
		zap.L().Error("[protoToJson] method not found in proto for http call")
		return []byte("error, method in proto not found"), errors.New("method in proto not found")
	}

	input := method.Input()
	zap.L().Debug("[protoToJson]", zap.Any("methodName", input.FullName()))
	msg := dynamicpb.NewMessage(input)
	err = proto.Unmarshal(in, msg)
	if err != nil {
		zap.L().Error("Error in unmarshalling", zap.Error(err))
		return []byte("error, invalid proto file or input request"), fmt.Errorf("error in unmarshaling proto to json: %+v", err)
	}
	json, err = protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
	if err != nil {
		zap.L().Error("Error in marshaling", zap.Error(err))
		return []byte("error, invalid proto file or input request"), fmt.Errorf("error in marshaling proto to json: %+v", err)
	}
	zap.L().Debug("ProtoToJson result:", zap.String("json", string(json)))

	return json, nil
}

func findMethodInProto(protoFiles linker.Files, methodName string) (method protoreflect.MethodDescriptor) {
	for _, protoFile := range protoFiles {
		if protoFile.Services().Len() == 0 {
			continue
		}

		for i := 0; i < protoFile.Services().Len(); i++ {
			service := protoFile.Services().Get(i)
			if service == nil {
				continue
			}

			method = service.Methods().ByName(protoreflect.Name(methodName))
			if method != nil {
				return method
			}
		}
	}
	return nil
}
