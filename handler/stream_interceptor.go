package handler

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/configuration_service"
	"github.com/singnet/snet-daemon/v6/metrics"
	"github.com/singnet/snet-daemon/v6/ratelimit"
	"go.uber.org/zap"

	"math/big"
	"strings"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	// PaymentTypeHeader selects a registered payment handler. The daemon registers
	// escrow, free-call, prepaid-call, and train-call; absence selects the default.
	PaymentTypeHeader = "snet-payment-type"
	// ClientTypeHeader identifies the client application for reporting, not authorization.
	ClientTypeHeader = "snet-client-type"
	// UserInfoHeader carries caller-supplied user details for metering.
	UserInfoHeader = "snet-user-info"
	// UserAgentHeader carries the client user agent used in metering records.
	UserAgentHeader = "user-agent"
	// PaymentChannelIDHeader is a MultiPartyEscrow contract payment channel
	// id. Value is a string containing a decimal number.
	PaymentChannelIDHeader = "snet-payment-channel-id"
	// PaymentChannelNonceHeader is a payment channel nonce value. Value is a
	// string containing a decimal number.
	PaymentChannelNonceHeader = "snet-payment-channel-nonce"
	// PaymentChannelAmountHeader carries the cumulative signed amount for the
	// channel nonce, in token base units as a decimal string, not the per-call price.
	PaymentChannelAmountHeader = "snet-payment-channel-amount"
	// PaymentChannelSignatureHeader carries raw signature bytes. Escrow/train calls
	// use an MPE claim signature; free calls use their caller-request signature.
	PaymentChannelSignatureHeader = "snet-payment-channel-signature-bin"
	// PaymentMultiPartyEscrowAddressHeader is the hex MPE address used by the
	// signature-only escrow handler in blockchain-disabled curation mode. Normal
	// channel validation takes the contract address from service metadata instead.
	PaymentMultiPartyEscrowAddressHeader = "snet-payment-mpe-address"

	// FreeCallUserIdHeader is the optional end-user identity bound to a free-call token.
	FreeCallUserIdHeader = "snet-free-call-user-id"
	// FreeCallUserAddressHeader identifies the wallet or trusted backend signing a free call.
	FreeCallUserAddressHeader = "snet-free-call-user-address"
	// SnetUserAddressHeader is populated downstream with the address reported by
	// an authenticated payment handle implementing SenderProvider.
	SnetUserAddressHeader = "snet-user-address"

	// CurrentBlockNumberHeader carries the decimal request block for freshness
	// checks, notably free-call requests. It is separate from token expiration.
	CurrentBlockNumberHeader = "snet-current-block-number"

	// FreeCallAuthTokenHeader carries the complete provider-signed token bytes,
	// including the underscore and decimal expiration-block suffix.
	FreeCallAuthTokenHeader = "snet-free-call-auth-token-bin"

	// PrePaidAuthTokenHeader carries the prepaid JWT as raw bytes. Its wall-clock
	// expiry and usage accounting are separate from free-call token block deadlines.
	PrePaidAuthTokenHeader = "snet-prepaid-auth-token-bin"

	DynamicPriceDerived = "snet-derived-dynamic-price-cost"

	TrainingModelId = "snet-train-model-id"
)

// GrpcStreamContext contains information about gRPC call which is used to
// validate payment and pricing.
type GrpcStreamContext struct {
	MD       metadata.MD
	Info     *grpc.StreamServerInfo
	InStream grpc.ServerStream
}

func (context *GrpcStreamContext) String() string {
	return fmt.Sprintf("{MD: %v, Info: %v}", context.MD, *context.Info)
}

// Payment represents payment handler specific data which is validated
// and used to complete payment.
type Payment any

// Custom gRPC codes to return to the client
const (
	// IncorrectNonce is returned to client when payment received contains
	// incorrect nonce value. Client may use PaymentChannelStateService to get
	// latest channel state and correct nonce value.
	IncorrectNonce codes.Code = 1000
)

// GrpcError wraps a gRPC status for payment and metadata failures. Interceptors
// return Err() to preserve its status code; this wrapper does not send headers.
type GrpcError struct {
	// Status is a gRPC call status
	Status *status.Status
}

func (err *GrpcError) Error() string {
	return err.String()
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
		Status: status.New(code, message),
	}
}

// NewGrpcErrorf returns new error which contains gRPC status with provided
// code and message formed from format string and args.
func NewGrpcErrorf(code codes.Code, format string, args ...any) *GrpcError {
	if len(args) == 0 {
		return &GrpcError{
			Status: status.New(code, format),
		}
	}
	return &GrpcError{
		Status: status.Newf(code, format, args...),
	}
}

// StreamPaymentHandler separates payment policy from request forwarding. Escrow,
// free-call, prepaid, and training handlers return their own opaque payment state.
// The interceptor invokes a completion hook only after Payment succeeds.
type StreamPaymentHandler interface {
	// Type is a content of PaymentTypeHeader field which triggers usage of the
	// payment handler.
	Type() (typ string)
	// Payment validates credentials and returns payment-specific state with a
	// nil error on success. On failure it must release any acquired resources,
	// because neither completion hook will be called.
	Payment(context *GrpcStreamContext) (payment Payment, err *GrpcError)
	// Complete finalizes a payment after the service handler returns nil.
	Complete(payment Payment) (err *GrpcError)
	// CompleteAfterError handles a service error or panic. A non-nil returned
	// status replaces the service error; panic cleanup is followed by a new panic.
	CompleteAfterError(payment Payment, result error) (err *GrpcError)
}

type rateLimitInterceptor struct {
	rateLimiter                   rate.Limiter
	messageBroadcaster            *configuration_service.MessageBroadcaster
	processRequest                int
	requestProcessingNotification chan int
}

func GrpcRateLimitInterceptor(broadcast *configuration_service.MessageBroadcaster) grpc.StreamServerInterceptor {
	interceptor := &rateLimitInterceptor{
		rateLimiter:                   *ratelimit.NewRateLimiter(),
		messageBroadcaster:            broadcast,
		processRequest:                configuration_service.StartProcessingAnyRequest,
		requestProcessingNotification: broadcast.NewSubscriber(),
	}
	go interceptor.startOrStopProcessingAnyRequests()
	return interceptor.intercept
}

func (interceptor *rateLimitInterceptor) startOrStopProcessingAnyRequests() {
	for {
		interceptor.processRequest = <-interceptor.requestProcessingNotification
	}
}

func GrpcMeteringInterceptor(currentBlock func() (*big.Int, error)) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return interceptMetering(srv, ss, info, handler, currentBlock)
	}
}

// interceptMetering records call duration and the final handler error, then
// publishes response statistics asynchronously using the latest available block.
func interceptMetering(
	srv any,
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
	currentBlock func() (*big.Int, error),
) error {
	var (
		err   error
		start time.Time
	)
	start = time.Now()

	methodName, _ := grpc.MethodFromServerStream(ss)
	commonStats := metrics.BuildCommonStats(start, methodName)

	if ctx, err := getGrpcContext(ss, info); err == nil {
		setAdditionalDetails(ctx, commonStats)
	}

	defer func() {
		var block *big.Int
		if currentBlock != nil {
			block, _ = currentBlock()
		}
		go metrics.PublishResponseStats(commonStats, time.Since(start), err, block)
	}()

	err = handler(srv, ss)
	if err != nil {
		zap.L().Error(err.Error())
		return err
	}
	return nil
}

func (interceptor *rateLimitInterceptor) intercept(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

	if interceptor.processRequest == configuration_service.StopProcessingAnyRequest {
		return status.New(codes.Unavailable, "No requests are currently being processed, please try again later").Err()
	}
	if !interceptor.rateLimiter.Allow() {
		zap.L().Info("rate limit reached, too many requests to handle", zap.Any("rateLimiter.Burst()", interceptor.rateLimiter.Burst()))
		return status.New(codes.ResourceExhausted, "rate limiting , too many requests to handle").Err()
	}
	err := handler(srv, ss)
	if err != nil {
		zap.L().Error(err.Error())
		return err
	}
	return nil
}

// GrpcPaymentValidationInterceptor registers the supplied payment policies and
// selects one from request metadata, falling back to the default when absent.
// Blockchain-enabled/disabled policy is selected by snetd components, not here.
func GrpcPaymentValidationInterceptor(serviceData *blockchain.ServiceMetadata, defaultPaymentHandler StreamPaymentHandler, paymentHandler ...StreamPaymentHandler) grpc.StreamServerInterceptor {
	interceptor := &paymentValidationInterceptor{
		defaultPaymentHandler: defaultPaymentHandler,
		paymentHandlers:       make(map[string]StreamPaymentHandler),
		serviceMetadata:       serviceData,
	}

	interceptor.paymentHandlers[defaultPaymentHandler.Type()] = defaultPaymentHandler
	zap.L().Info("Default payment handler registered", zap.Any("defaultPaymentType", defaultPaymentHandler.Type()))
	for _, handler := range paymentHandler {
		interceptor.paymentHandlers[handler.Type()] = handler
		zap.L().Info("Payment handler for type registered", zap.Any("paymentType", handler.Type()))
	}
	return interceptor.streamIntercept
}

type paymentValidationInterceptor struct {
	serviceMetadata       *blockchain.ServiceMetadata
	defaultPaymentHandler StreamPaymentHandler
	paymentHandlers       map[string]StreamPaymentHandler
}

func (interceptor *paymentValidationInterceptor) streamIntercept(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (e error) {
	var err *GrpcError

	// Work with copied metadata; the original incoming context may be shared.
	grpcCtx, err := getGrpcContext(ss, info)
	if err != nil {
		return err.Err()
	}

	baseCtx := metadata.NewIncomingContext(ss.Context(), grpcCtx.MD)

	// Buffer the first frame for price estimation, then replay it to the backend.
	wrapperStream, streamErr := NewWrapperServerStream(ss, baseCtx)
	if streamErr != nil {
		return streamErr
	}

	grpcCtx.InStream = wrapperStream

	paymentHandler, err := interceptor.getPaymentHandler(grpcCtx)
	if err != nil {
		return err.Err()
	}

	zap.L().Debug("[streamIntercept] New gRPC call received", zap.Any("context", grpcCtx))

	payment, err := paymentHandler.Payment(grpcCtx)
	if err != nil {
		return err.Err()
	}

	if sp, ok := payment.(SenderProvider); ok {
		// Propagate the payment handle's sender rather than trusting a caller's
		// existing snet-user-address header.
		outMD := grpcCtx.MD.Copy()
		ethAddr := sp.GetSender().Hex()
		outMD.Set(SnetUserAddressHeader, ethAddr)
		outMD.Set("snet-daemon-debug", "streamIntercept")
		grpcCtx.MD = outMD

		// Keep the payment context and forwarded stream metadata consistent.
		if ws, ok := wrapperStream.(*WrapperServerStream); ok {
			ws.Ctx = metadata.NewIncomingContext(ws.Ctx, outMD)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			zap.L().Warn("Service handler called panic(panicValue)", zap.Any("panicValue", r))
			paymentHandler.CompleteAfterError(payment, fmt.Errorf("service handler called panic(%v)", r))
			panic("re-panic after payment handler error handling")
		} else if e == nil {
			err = paymentHandler.Complete(payment)
			if err != nil {
				e = err.Err()
			}
		} else {
			err = paymentHandler.CompleteAfterError(payment, e)
			if err != nil {
				e = err.Err()
			}
		}
	}()

	zap.L().Debug("[streamIntercept] New payment received", zap.Any("payment", payment))

	e = handler(srv, wrapperStream)
	if e != nil {
		zap.L().Warn("[streamIntercept] gRPC handler returned error", zap.Error(e))
		return e
	}

	return nil
}

func getGrpcContext(
	serverStream grpc.ServerStream,
	info *grpc.StreamServerInfo,
) (*GrpcStreamContext, *GrpcError) {
	md, ok := metadata.FromIncomingContext(serverStream.Context())
	if !ok {
		zap.L().Error("Invalid metadata", zap.Any("info", info))
		return nil, NewGrpcError(codes.InvalidArgument, "missing metadata")
	}

	// Copy metadata before adding pricing or authenticated-sender fields.
	mdCopy := md.Copy()

	return &GrpcStreamContext{
		MD:       mdCopy,
		Info:     info,
		InStream: serverStream,
	}, nil
}

func (interceptor *paymentValidationInterceptor) getPaymentHandler(context *GrpcStreamContext) (handler StreamPaymentHandler, err *GrpcError) {
	paymentTypeMd, ok := context.MD[PaymentTypeHeader]
	if !ok || len(paymentTypeMd) == 0 {
		zap.L().Debug("Payment type was not set by caller, return default payment handler",
			zap.String("defaultPaymentHandlerType", interceptor.defaultPaymentHandler.Type()))
		return interceptor.defaultPaymentHandler, nil
	}

	paymentType := paymentTypeMd[0]
	zap.L().Debug("Payment metadata", zap.String("paymentType", paymentType), zap.Any("paymentTypeMd", paymentTypeMd))
	paymentHandler, ok := interceptor.paymentHandlers[paymentType]
	if !ok {
		zap.L().Error("Unexpected payment type", zap.String("paymentType", paymentType))
		return nil, NewGrpcErrorf(codes.InvalidArgument, "unexpected \"%v\", value: \"%v\"", PaymentTypeHeader, paymentType)
	}

	zap.L().Debug("Return payment handler by type", zap.Any("paymentType", paymentType))
	return paymentHandler, nil
}

// GetBigInt parses exactly one metadata value with big.Int.UnmarshalText.
// Protocol-specific sign and uint256-width checks are left to callers.
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

// GetBytes returns exactly one raw binary metadata value for a key ending in
// -bin. The gRPC transport handles base64 on the wire; this helper does not decode
// application-level base64 or hexadecimal text.
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

// GetBytesFromHex requires one metadata value and applies go-ethereum's
// permissive hex decoder; malformed hex is not returned as a decoding error.
func GetBytesFromHex(md metadata.MD, key string) (value []byte, err *GrpcError) {
	str, err := GetSingleValue(md, key)
	if err != nil {
		return
	}
	return common.FromHex(str), nil
}

// GetSingleValue requires exactly one metadata entry. An empty string is allowed;
// missing or repeated entries produce InvalidArgument.
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
func NoOpInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {
	return handler(srv, ss)
}

// setAdditionalDetails copies optional client, user, channel, and payment-mode
// reporting fields. These metering hints are not themselves payment authorization.
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
