//go:generate protoc -I . ./pricing.proto --go-grpc_out=. --go_out=.
package pricing

import (
	"fmt"
	"google.golang.org/grpc/credentials/insecure"
	"math/big"
	"net/url"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/codec"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/handler"

	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type DynamicMethodPrice struct {
	serviceMetaData *blockchain.ServiceMetadata
}

func (priceType DynamicMethodPrice) GetPrice(derivedContext *handler.GrpcStreamContext) (price *big.Int, err error) {

	return priceType.checkForDynamicPricing(derivedContext)
}

func (priceType DynamicMethodPrice) GetPriceType() string {
	return DYNAMIC_PRICING
}

func (priceType *DynamicMethodPrice) checkForDynamicPricing(
	derivedContext *handler.GrpcStreamContext) (price *big.Int, e error) {

	method, ok := grpc.MethodFromServerStream(derivedContext.InStream)
	methodNameField := zap.Any("methodNameRetrieved", method)
	if !ok {
		return nil, fmt.Errorf("Unable to get the method Name from the incoming request")
	}
	//[TODO]: get grpc options standardized rather than doing then everytime
	passThroughURL, err := url.Parse(config.GetString(config.ServiceEndpointKey))
	if err != nil {
		zap.L().Error(err.Error(), methodNameField)
		return nil, err
	}
	options := grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(config.GetInt(config.MaxMessageSizeInMB)*1024*1024),
		grpc.MaxCallSendMsgSize(config.GetInt(config.MaxMessageSizeInMB)*1024*1024))

	conn, _ := grpc.NewClient(passThroughURL.Host, grpc.WithTransportCredentials(insecure.NewCredentials()), options)
	md, ok := metadata.FromIncomingContext(derivedContext.InStream.Context())

	if !ok {
		return nil, status.Errorf(codes.Internal, "could not get metadata from incoming context")
	}
	outCtx, outCancel := context.WithCancel(derivedContext.InStream.Context())
	defer func() { outCancel() }()
	outCtx = metadata.NewOutgoingContext(outCtx, md.Copy())
	pricingMethod, ok := priceType.serviceMetaData.GetDynamicPricingMethodAssociated(method)
	if !ok {
		return nil, fmt.Errorf("Umable to determine the pricing method")
	}
	clientStream, err := conn.NewStream(outCtx,
		&grpc.StreamDesc{ServerStreams: true, ClientStreams: true}, pricingMethod,
		grpc.CallContentSubtype("proto"))
	if err != nil {
		zap.L().Error(err.Error())
	}
	return priceType.getPriceFromPricingMethod(derivedContext.InStream, clientStream)

}

func (priceType *DynamicMethodPrice) getPriceFromPricingMethod(ss grpc.ServerStream, clientStream grpc.ClientStream) (price *big.Int, err error) {
	reqMessage := ss.(*handler.WrapperServerStream).OriginalRecvMsg()

	err = clientStream.SendMsg(reqMessage)
	if err != nil {
		return nil, err
	}
	responseMessage := &codec.GrpcFrame{}
	err = clientStream.RecvMsg(responseMessage)
	if err != nil {
		return nil, err
	}
	priceInCogs := &PriceInCogs{}

	if err := proto.Unmarshal(responseMessage.Data, priceInCogs); err != nil {
		return nil, err
	}
	zap.L().Info("dynamic price received", zap.Uint64("Price", priceInCogs.Price))
	return big.NewInt(0).SetUint64(priceInCogs.Price), nil
}
