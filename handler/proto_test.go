package handler

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// compileProtoFiles
func getDescriptors(t *testing.T, protoFiles map[string]string) linker.Files {
	accessor := protocompile.SourceAccessorFromMap(protoFiles)
	r := protocompile.WithStandardImports(&protocompile.SourceResolver{Accessor: accessor})
	compiler := protocompile.Compiler{
		Resolver:       r,
		SourceInfoMode: protocompile.SourceInfoStandard,
	}
	fds, err := compiler.Compile(context.Background(), slices.Collect(maps.Keys(protoFiles))...)
	require.Nil(t, err)
	require.NotNil(t, fds)
	return fds
}

func TestFindMethodInProto(t *testing.T) {
	protoTexts := map[string]string{
		"test.proto": `
			syntax = "proto3";
			package test;
			service TestService {
				rpc TestMethod (TestRequest) returns (TestResponse);
			}
			message TestRequest {}
			message TestResponse {}
		`,
	}

	protoFiles := getDescriptors(t, protoTexts)
	method := findMethodInProto(protoFiles, "TestMethod")
	require.NotNil(t, method, "Method should be found")
	require.Equal(t, protoreflect.Name("TestMethod"), method.Name(), "Method name should match")
}

func TestFindMethodInSeveralProtos(t *testing.T) {
	protoTexts := map[string]string{
		"first.proto": `
			syntax = "proto3";
			package test;
			service TestService {
				rpc TestMethod (TestRequest1) returns (TestResponse1);
			}
			message TestRequest1 {}
			message TestResponse1 {}
		`,
		"second.proto": `
			syntax = "proto3";
			package test;
			service TestService2 {
				rpc TestMethod2 (TestRequest2) returns (TestResponse2);
			}
			message TestRequest2 {}
			message TestResponse2 {}
		`,
		"third.proto": `
			syntax = "proto3";
			package test;
			service TestService3 {
				rpc TestMethod3 (TestRequest3) returns (TestResponse3);
			}	
			service TestService4 {
				rpc TestMethod4 (TestRequest3) returns (TestResponse3);
			}
			message TestRequest3 {}
			message TestResponse3 {}
		`,
	}

	protoFiles := getDescriptors(t, protoTexts)
	method := findMethodInProto(protoFiles, "TestMethod4")
	require.NotNil(t, method, "Method should be found")
	require.Equal(t, protoreflect.Name("TestMethod4"), method.Name(), "Method name should match")
}

func TestFindMethodInProto_NotFound(t *testing.T) {
	protoTexts := map[string]string{
		"test.proto": `
			syntax = "proto3";
			package test;
			service TestService {
				rpc ExistingMethod (TestRequest) returns (TestResponse);
			}
			message TestRequest {}
			message TestResponse {}
		`,
	}

	protoFiles := getDescriptors(t, protoTexts)
	method := findMethodInProto(protoFiles, "NonExistentMethod")
	require.Nil(t, method, "Method should not be found")
}

func TestJsonToProto(t *testing.T) {
	protoTexts := map[string]string{
		"test.proto": `
			syntax = "proto3";
			package test;
			service TestService {
				rpc ExistingMethod (TestRequest) returns (TestResponse);
			}
			message TestRequest {
				string foo = 1;
			}
			message TestResponse {
				string bar = 1;
			}
		`,
	}

	protoFiles := getDescriptors(t, protoTexts)

	protoMsg, err := jsonToProto(protoFiles, []byte(`{"bar":"1"}`), "ExistingMethod")
	require.Nil(t, err)
	require.NotNil(t, protoMsg, "proto msg should not be nil")

	protoMsg, err = jsonToProto(protoFiles, []byte(`{"bar":"1"}`), "NoMethodFound")
	require.NotNil(t, err)
	require.Nil(t, protoMsg, "proto msg should be nil")
}
