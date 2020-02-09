package handler

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/configuration_service"
	"github.com/singnet/snet-daemon/metrics"
	"github.com/singnet/snet-daemon/ratelimit"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"math/big"
	"strings"
	"time"
)

const (
	// PaymentTypeHeader is a type of payment used to pay for a RPC call.
	// Supported types are: "escrow".
	// Note: "job" Payment type is deprecated
	PaymentTypeHeader = "snet-payment-type"
	//Client that calls the Daemon ( example can be "snet-cli","snet-dapp","snet-sdk")
	ClientTypeHeader = "snet-client-type"
	//Value is a user address , example "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207"
	UserInfoHeader = "snet-user-info"
	//User Agent details set in on the server stream info
	UserAgentHeader = "user-agent"
	// PaymentChannelIDHeader is a MultiPartyEscrow contract payment channel
	// id. Value is a string containing a decimal number.
	PaymentChannelIDHeader = "snet-payment-channel-id"
	// PaymentChannelNonceHeader is a payment channel nonce value. Value is a
	// string containing a decimal number.
	PaymentChannelNonceHeader = "snet-payment-channel-nonce"
	// PaymentChannelAmountHeader is an amount of payment channel value
	// which server is authorized to withdraw after handling the RPC call.
	// Value is a string containing a decimal number.
	PaymentChannelAmountHeader = "snet-payment-channel-amount"
	// PaymentChannelSignatureHeader is a signature of the client to confirm
	// amount withdrawing authorization. Value is an array of bytes.
	PaymentChannelSignatureHeader = "snet-payment-channel-signature-bin"

	//Added for free call support in Daemon

    //The user Id of the person making the call
	FreeCallUserIdHeader = "snet-free-call-user-id"

	//Will be used to check if the Signature is still valid
	CurrentBlockNumberHeader = "snet-current-block-number"

	//Place holder to set the free call Auth Token issued
	FreeCallAuthTokenHeader = "snet-free-call-auth-token-bin"
    //Block number on when the Token was issued , to track the expiry of the token , which is ~ 1 Month
    FreeCallAuthTokenBlockNumberHeader =  "snet-free-call-token-issue-block"


)

// GrpcStreamContext contains information about gRPC call which is used to
// validate payment and pricing.
type GrpcStreamContext struct {
	MD   metadata.MD
	Info *grpc.StreamServerInfo
}

func (context *GrpcStreamContext) String() string {
	return fmt.Sprintf("{MD: %v, Info: %v", context.MD, *context.Info)
}

// Payment represents payment handler specific data which is validated
// and used to complete payment.
type Payment interface{}

// Custom gRPC codes to return to the client
const (
	// IncorrectNonce is returned to client when payment recieved contains
	// incorrect nonce value. Client may use PaymentChannelStateService to get
	// latest channel state and correct nonce value.
	IncorrectNonce codes.Code = 1000
)

// GrpcError is an error which will be returned by interceptor via gRPC
// protocol. Part of information will be returned as header metadata.
type GrpcError struct {
	// Status is a gRPC call status
	Status *status.Status
}

// Err returns error to return correct gRPC error to the caller
func (err *GrpcError) Err() error {
	if err.Status == nil {
		return nil
	}
	return err.Status.Err()
}

// String converts GrpcError to string
func (err *GrpcError) String() string {
	return fmt.Sprintf("{Status: %v}", err.Status)
}

// NewGrpcError returns new error which contains gRPC status with provided code
// and message
func NewGrpcError(code codes.Code, message string) *GrpcError {
	return &GrpcError{
		Status: status.Newf(code, message),
	}
}

// NewGrpcErrorf returns new error which contains gRPC status with provided
// code and message formed from format string and args.
func NewGrpcErrorf(code codes.Code, format string, args ...interface{}) *GrpcError {
	return &GrpcError{
		Status: status.Newf(code, format, args...),
	}
}


// PaymentHandler interface which is used by gRPC interceptor to get, validate
// and complete payment. There are two payment handler implementations so far:
// jobPaymentHandler and escrowPaymentHandler. jobPaymentHandler is depreactted.
type PaymentHandler interface {
	// Type is a content of PaymentTypeHeader field which triggers usage of the
	// payment handler.
	Type() (typ string)
	// Payment extracts payment data from gRPC request context and checks
	// validity of payment data. It returns nil if data is valid or
	// appropriate gRPC status otherwise.
	Payment(context *GrpcStreamContext) (payment Payment, err *GrpcError)
	// Complete completes payment if gRPC call was successfully proceeded by
	// service.
	Complete(payment Payment) (err *GrpcError)
	// CompleteAfterError completes payment if service returns error.
	CompleteAfterError(payment Payment, result error) (err *GrpcError)

}

type rateLimitInterceptor struct {
	rateLimiter           rate.Limiter
	messageBroadcaster    *configuration_service.MessageBroadcaster
	processRequest        int
	requestProcessingNotification chan int
}

func GrpcRateLimitInterceptor(broadcast *configuration_service.MessageBroadcaster) grpc.StreamServerInterceptor {
	interceptor := &rateLimitInterceptor{
		rateLimiter:           ratelimit.NewRateLimiter(),
		messageBroadcaster:    broadcast,
		processRequest :       configuration_service.START_PROCESSING_ANY_REQUEST,
		requestProcessingNotification: broadcast.NewSubscriber(),
	}
	go interceptor.startOrStopProcessingAnyRequests()
	return interceptor.intercept
}

func (interceptor *rateLimitInterceptor) startOrStopProcessingAnyRequests () {
	for {
		interceptor.processRequest =<- interceptor.requestProcessingNotification
	}
}

func GrpcMeteringInterceptor() grpc.StreamServerInterceptor {
	return interceptMetering
}

//Monitor requests arrived and responses sent and publish these stats for Reporting
func interceptMetering(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	var e error
	var start time.Time
	start = time.Now()
	//Get the method name
	methodName, _ := grpc.MethodFromServerStream(ss)
	//Get the Context

	//Build common stats and use this to set request stats and response stats
	commonStats := metrics.BuildCommonStats(start, methodName)
	if context, err := getGrpcContext(ss, info); err == nil {
		setAdditionalDetails(context, commonStats)
	}

	defer func() {
		go metrics.PublishResponseStats(commonStats, time.Now().Sub(start), e)
	}()
	e = handler(srv, ss)
	if e != nil {
		log.WithError(e)
		return e
	}
	return nil
}

func (interceptor *rateLimitInterceptor) intercept(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

	if (interceptor.processRequest == configuration_service.STOP_PROCESING_ANY_REQUEST) {
		return status.New(codes.Unavailable, "No requests are currently being processed, please try again later").Err()
	}
	if !interceptor.rateLimiter.Allow() {
		log.WithField("rateLimiter.Burst()", interceptor.rateLimiter.Burst()).Info("rate limit reached, too many requests to handle")
		return status.New(codes.ResourceExhausted, "rate limiting , too many requests to handle").Err()
	}
	e := handler(srv, ss)
	if e != nil {
		log.WithError(e)
		return e
	}
	return nil
}

// GrpcStreamInterceptor returns gRPC interceptor to validate payment. If
// blockchain is disabled then noOpInterceptor is returned.
func GrpcPaymentValidationInterceptor(defaultPaymentHandler PaymentHandler, paymentHandler ...PaymentHandler) grpc.StreamServerInterceptor {
	interceptor := &paymentValidationInterceptor{
		defaultPaymentHandler: defaultPaymentHandler,
		paymentHandlers:       make(map[string]PaymentHandler),
	}

	interceptor.paymentHandlers[defaultPaymentHandler.Type()] = defaultPaymentHandler
	log.WithField("defaultPaymentType", defaultPaymentHandler.Type()).Info("Default payment handler registered")
	for _, handler := range paymentHandler {
		interceptor.paymentHandlers[handler.Type()] = handler
		log.WithField("paymentType", handler.Type()).Info("Payment handler for type registered")
	}

	return interceptor.intercept

}

type paymentValidationInterceptor struct {
	defaultPaymentHandler PaymentHandler
	paymentHandlers       map[string]PaymentHandler
}

func (interceptor *paymentValidationInterceptor) intercept(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (e error) {
	var err *GrpcError

	context, err := getGrpcContext(ss, info)
	if err != nil {
		return err.Err()
	}
	log.WithField("context", context).Debug("New gRPC call received")

	paymentHandler, err := interceptor.getPaymentHandler(context)
	if err != nil {
		return err.Err()
	}

	payment, err := paymentHandler.Payment(context)
	if err != nil {
		return err.Err()
	}

	defer func() {
		if r := recover(); r != nil {
			log.WithField("panicValue", r).Warn("Service handler called panic(panicValue)")
			paymentHandler.CompleteAfterError(payment, fmt.Errorf("Service handler called panic(%v)", r))
			panic("re-panic after payment handler error handling")
		} else if e == nil {
			err = paymentHandler.Complete(payment)
			if err != nil {
				// return err.Err()
				e = err.Err()
			}
		} else {
			err = paymentHandler.CompleteAfterError(payment, e)
			if err != nil {
				// return err.Err()
				e = err.Err()
			}
		}
	}()

	log.WithField("payment", payment).Debug("New payment received")

	e = handler(srv, ss)
	if e != nil {
		log.WithError(e).Warn("gRPC handler returned error")
		return e
	}

	return nil
}

func getGrpcContext(serverStream grpc.ServerStream, info *grpc.StreamServerInfo) (context *GrpcStreamContext, err *GrpcError) {
	md, ok := metadata.FromIncomingContext(serverStream.Context())
	if !ok {
		log.WithField("info", info).Error("Invalid metadata")
		return nil, NewGrpcError(codes.InvalidArgument, "missing metadata")
	}

	return &GrpcStreamContext{
		MD:   md,
		Info: info,
	}, nil
}

func (interceptor *paymentValidationInterceptor) getPaymentHandler(context *GrpcStreamContext) (handler PaymentHandler, err *GrpcError) {
	paymentTypeMd, ok := context.MD[PaymentTypeHeader]
	if !ok || len(paymentTypeMd) == 0 {
		log.WithField("defaultPaymentHandlerType", interceptor.defaultPaymentHandler.Type()).Debug("Payment type was not set by caller, return default payment handler")
		return interceptor.defaultPaymentHandler, nil
	}

	paymentType := paymentTypeMd[0]
	paymentHandler, ok := interceptor.paymentHandlers[paymentType]
	if !ok {
		log.WithField("paymentType", paymentType).Error("Unexpected payment type")
		return nil, NewGrpcErrorf(codes.InvalidArgument, "unexpected \"%v\", value: \"%v\"", PaymentTypeHeader, paymentType)
	}

	log.WithField("paymentType", paymentType).Debug("Return payment handler by type")
	return paymentHandler, nil
}

// GetBigInt gets big.Int value from gRPC metadata
func GetBigInt(md metadata.MD, key string) (value *big.Int, err *GrpcError) {
	str, err := GetSingleValue(md, key)
	if err != nil {
		return
	}

	value = big.NewInt(0)
	e := value.UnmarshalText([]byte(str))
	if e != nil {
		return nil, NewGrpcErrorf(codes.InvalidArgument, "incorrect format \"%v\": \"%v\"", key, str)
	}

	return
}

// GetBytes gets bytes array value from gRPC metadata for key with '-bin'
// suffix, internally this data is encoded as base64
func GetBytes(md metadata.MD, key string) (result []byte, err *GrpcError) {
	if !strings.HasSuffix(key, "-bin") {
		return nil, NewGrpcErrorf(codes.InvalidArgument, "incorrect binary key name \"%v\"", key)
	}

	str, err := GetSingleValue(md, key)
	if err != nil {
		return
	}

	return []byte(str), nil
}

// GetBytesFromHex gets bytes array value from gRPC metadata, bytes array is
// encoded as hex string
func GetBytesFromHex(md metadata.MD, key string) (value []byte, err *GrpcError) {
	str, err := GetSingleValue(md, key)
	if err != nil {
		return
	}
	return common.FromHex(str), nil
}

// GetSingleValue gets string value from gRPC metadata
func GetSingleValue(md metadata.MD, key string) (value string, err *GrpcError) {
	array := md.Get(key)

	if len(array) == 0 {
		return "", NewGrpcErrorf(codes.InvalidArgument, "missing \"%v\"", key)
	}

	if len(array) > 1 {
		return "", NewGrpcErrorf(codes.InvalidArgument, "too many values for key \"%v\": %v", key, array)
	}

	return array[0], nil
}

// NoOpInterceptor is a gRPC interceptor which doesn't do payment checking.
func NoOpInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {
	return handler(srv, ss)
}

//set Additional details on the metrics persisted , this is to keep track of how many calls were made per channel
func setAdditionalDetails(context *GrpcStreamContext, stats *metrics.CommonStats) {
	md := context.MD
	if str, err := GetSingleValue(md, ClientTypeHeader); err == nil {
		stats.ClientType = str
	}
	if str, err := GetSingleValue(md, UserInfoHeader); err == nil {
		stats.UserDetails = str
	}
	if str, err := GetSingleValue(md, UserAgentHeader); err == nil {
		stats.UserAgent = str
	}
	if str, err := GetSingleValue(md, PaymentChannelIDHeader); err == nil {
		stats.ChannelId = str
	}
	if str, err := GetSingleValue(md, FreeCallUserIdHeader); err == nil {
		stats.UserName = str
	}
	if str, err := GetSingleValue(md, PaymentTypeHeader); err == nil {
		stats.PaymentMode = str
	}
}
