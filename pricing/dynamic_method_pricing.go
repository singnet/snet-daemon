//go:generate protoc -I . ./pricing.proto --go_out=plugins=grpc:.
package pricing

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/codec"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/handler"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"math/big"
	"net/url"
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
	log.WithField("methodNameRetrieved", method)
	if !ok {
		return nil, fmt.Errorf("Unable to get the method Name from the incoming request")
	}
	//todo, get grpc options standardized rather than doing then everytime
	passThroughURL, err := url.Parse(config.GetString(config.PassthroughEndpointKey))
	if err != nil {
		log.WithError(err)
		return nil, err
	}
	options := grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(config.GetInt(config.MaxMessageSizeInMB)*1024*1024),
		grpc.MaxCallSendMsgSize(config.GetInt(config.MaxMessageSizeInMB)*1024*1024))

	conn, _ := grpc.Dial(passThroughURL.Host, grpc.WithInsecure(), options)
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
	pp := &PriceInCogs{}

	if err := proto.Unmarshal(responseMessage.Data, pp); err != nil {
		return nil, err
	}
	log.WithField("dynamic price received", pp.Price)
	return big.NewInt(0).SetUint64(pp.Price), nil
}
