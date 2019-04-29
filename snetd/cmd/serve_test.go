package cmd

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/escrow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"math/big"
	"net"
	"os/signal"
	"reflect"
	"syscall"
	"testing"
	"time"
)

//todo
func TestDaemonPort(t *testing.T) {
	assert.Equal(t, config.GetString(config.DaemonEndPoint), "127.0.0.1:8080")
}

func Test_newDaemon(t *testing.T) {
	type args struct {
		components *Components
	}
	tests := []struct {
		name    string
		args    args
		want    daemon
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newDaemon(tt.args.components)
			if (err != nil) != tt.wantErr {
				t.Errorf("newDaemon() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newDaemon() = %v, want %v", got, tt.want)
			}
		})
	}
}

var defercheck bool



// NoOpInterceptor is a gRPC interceptor which doesn't do payment checking.
func MockInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {
		time.Sleep(1000)
		defer func()  {
			time.Sleep(1000)
			defercheck = true
		}()
	return nil
}

func StartMockService() {
	go func() {
		flag.Parse()
		lis, err := net.Listen("tcp", config.GetString(config.PassthroughEndpointKey))
		if err != nil {
			fmt.Sprintf("failed to listen: %v", err)
		}
		var opts []grpc.ServerOption

		fmt.Printf("Starting Service.....\n")
		grpcServer := grpc.NewServer(opts...)
		RegisterExampleServiceServer(grpcServer, &ServiceMock{output:&Output{"Hello from Service"},err:nil})
		ch <- 0
		grpcServer.Serve(lis)
		fmt.Printf("Started.....")
	}()
}

func Test_daemon_start(t *testing.T) {

	config.Vip().Set(config.PaymentChannelStorageTypeKey,"")
	config.Vip().Set(config.PassthroughEndpointKey,"localhost:8086")
	config.Vip().Set(config.DaemonEndPoint,"localhost:8085")
	lis, _ := net.Listen("tcp", config.GetString(config.DaemonEndPoint))

	testSuite := &PaymentChannelServiceSuite{}
	testSuite.SetupSuite()

	var testJsonData = "{\"version\": 1, \"display_name\": \"Example1\", \"encoding\": \"grpc\", \"service_type\": \"grpc\", " +
		"\"payment_expiration_threshold\": 4032000, \"model_ipfs_hash\": \"QmQC9EoVdXRWmg8qm25Hkj4fG79YAgpNJCMDoCnknZ6VeJ\"," +
		" \"mpe_address\": \"0xf25186b5081ff5ce73482ad761db0eb0d25abfbf\", \"pricing\":" +
		" {\"price_model\": \"fixed_price\", \"price_in_cogs\": 12300}, \"groups\": [{\"group_name\": \"default_group\", " +
		"\"group_id\": \"nXzNEetD1kzU3PZqR4nHPS8erDkrUK0hN4iCBQ4vH5U=\", \"payment_address\": \"" +
		blockchain.AddressToHex(&testSuite.recipientAddress) +
		"\"}], " +
		"\"endpoints\": [{\"group_name\": \"default_group\", \"endpoint\": \"\"}]}"

	serviceMetadata, _ := blockchain.InitServiceMetaDataFromJson(testJsonData)


	//serviceMetadata.Groups = &
	comp := &Components{}

	comp.serviceMetadata = serviceMetadata

	comp.paymentChannelService = testSuite.service

	type fields struct {
		autoSSLDomain string
		acmeListener  net.Listener
		grpcServer    *grpc.Server
		blockProc     blockchain.Processor
		lis           net.Listener
		sslCert       *tls.Certificate
		components    *Components
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{"test1",fields{"",nil,nil,blockchain.Processor{},lis,nil,comp,}},
	}

	StartMockService()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &daemon{
				autoSSLDomain: tt.fields.autoSSLDomain,
				acmeListener:  tt.fields.acmeListener,
				grpcServer:    tt.fields.grpcServer,
				blockProc:     tt.fields.blockProc,
				lis:           tt.fields.lis,
				sslCert:       tt.fields.sslCert,
				components:    tt.fields.components,
			}

			go func(){
				d.start()
				signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
				<-sigChan
				d.stop()
			}()

			//go func () {
				flag.Parse()
				var opts []grpc.DialOption
				opts = append(opts, grpc.WithInsecure())
				serverAddr := flag.String("server_addr", config.GetString(config.DaemonEndPoint), "The server address in the format of host:port")
				conn, err := grpc.Dial(*serverAddr, opts...)
				if err != nil {
					fmt.Sprintf("fail to dial: %v", err)
				}
				ctx, _ := context.WithTimeout(context.Background(), 1000*time.Second)
				ctx = metadata.AppendToOutgoingContext(ctx, "snet-payment-type", "escrow",
					"snet-payment-channel-id", "42",
					"snet-payment-channel-nonce", "3",
					"snet-payment-channel-amount","12300",
					"snet-payment-channel-signature-bin",string(testSuite.payment().Signature) )
				//defer cancel()
				//Wait till the service has started
				_ = <-ch
				go func() {
					time.Sleep(time.Second*10)
					ch <- 0
					sigChan <- syscall.SIGTERM
				}()
				exampleClient := NewExampleServiceClient(conn)
				output, err := exampleClient.LongCall(ctx, &Input{Message: "Hello from Client"})
				assert.NotNil(t,output)
				//Check if the channel is not locked here
				channel, _, errC := testSuite.storage.Get(testSuite.channelKey())
				assert.Nil(t,errC)
				assert.NotNil(t,channel)
		//	}()

		})
	}
}


type paymentChannelServiceMock struct {
	err  error
	key  *escrow.PaymentChannelKey
	data *escrow.PaymentChannelData
}

func (p *paymentChannelServiceMock) PaymentChannel(key *escrow.PaymentChannelKey) (*escrow.PaymentChannelData, bool, error) {
	if p.err != nil {
		return nil, false, p.err
	}
	if p.key == nil || p.key.ID.Cmp(key.ID) != 0 {
		return nil, false, nil
	}
	return p.data, true, nil
}

func (p *paymentChannelServiceMock) Put(key *escrow.PaymentChannelKey, data *escrow.PaymentChannelData) {
	p.key = key
	p.data = data
}

func (p *paymentChannelServiceMock) SetError(err error) {
	p.err = err
}

func (p *paymentChannelServiceMock) Clear() {
	p.key = nil
	p.data = nil
	p.err = nil
}

func (p *paymentChannelServiceMock) StartPaymentTransaction(payment *escrow.Payment) (escrow.PaymentTransaction, error) {
	if p.err != nil {
		return nil, p.err
	}

	return &paymentTransactionMock{
		channel: p.data,
		err:     p.err,
	}, nil
}

type paymentTransactionMock struct {
	channel *escrow.PaymentChannelData
	err     error
}

func (transaction *paymentTransactionMock) Channel() *escrow.PaymentChannelData {
	return transaction.channel
}

func (transaction *paymentTransactionMock) Commit() error {
	return transaction.err
}

func (transaction *paymentTransactionMock) Rollback() error {
	return transaction.err
}

type PaymentChannelServiceSuite struct {
	suite.Suite

	senderAddress      common.Address
	signerPrivateKey   *ecdsa.PrivateKey
	signerAddress      common.Address
	recipientAddress   common.Address
	mpeContractAddress common.Address
	memoryStorage      *memoryStorage
	storage            *escrow.PaymentChannelStorage
	paymentStorage     *escrow.PaymentStorage

	service escrow.PaymentChannelService
}


func GenerateTestPrivateKey() (privateKey *ecdsa.PrivateKey) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Sprintf("Cannot generate private key for test: %v", err))
	}
	return
}


func (suite *PaymentChannelServiceSuite) SetupSuite() {

	suite.senderAddress = crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)
	suite.signerPrivateKey = GenerateTestPrivateKey()
	suite.signerAddress = crypto.PubkeyToAddress(suite.signerPrivateKey.PublicKey)
	suite.recipientAddress = crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)
	suite.mpeContractAddress = blockchain.HexToAddress("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf")
	suite.memoryStorage = NewMemStorage()
	suite.storage = escrow.NewPaymentChannelStorage(suite.memoryStorage)
	suite.paymentStorage = escrow.NewPaymentStorage(suite.memoryStorage)

	err := suite.storage.Put(suite.channelKey(), suite.channel())
	if err != nil {
		panic(fmt.Errorf("Cannot put value into test storage: %v", err))
	}

		readChannelFromBlockchainfunc:= func(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
			return suite.mpeChannel(), true, nil
		}
		recipientPaymentAddressfunc:= func() common.Address {
			return suite.recipientAddress
		}
	bkreader := escrow.NewBlockChainChannelReader(readChannelFromBlockchainfunc,recipientPaymentAddressfunc)
	currentBlockFunc :=  func() (*big.Int, error) { return big.NewInt(99), nil }
	paymentExpirationThresholdFunc := func() *big.Int { return big.NewInt(0) }

	suite.service = escrow.NewPaymentChannelService(
		suite.storage,
		suite.paymentStorage,
		bkreader,
		escrow.NewEtcdLocker(suite.memoryStorage),
		escrow.NewChannelPaymentValidatorT(currentBlockFunc,paymentExpirationThresholdFunc),
		func() ([32]byte, error) {
			return [32]byte{123}, nil
		},
	)
}

func (suite *PaymentChannelServiceSuite) SetupTest() {
	suite.memoryStorage.Clear()
}

func TestPaymentChannelServiceSuite(t *testing.T) {
	suite.Run(t, new(PaymentChannelServiceSuite))
}

func (suite *PaymentChannelServiceSuite) mpeChannel() *blockchain.MultiPartyEscrowChannel {
	return &blockchain.MultiPartyEscrowChannel{
		Sender:     suite.senderAddress,
		Recipient:  suite.recipientAddress,
		GroupId:    blockchain.StringToBytes32("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf"),
		Value:      big.NewInt(12345),
		Nonce:      big.NewInt(3),
		Expiration: big.NewInt(100),
		Signer:     suite.signerAddress,
	}
}


func bigIntToBytes(value *big.Int) []byte {
	return common.BigToHash(value).Bytes()
}

func SignPayment(payment *escrow.Payment, privateKey *ecdsa.PrivateKey) {
	payment.MpeContractAddress = blockchain.HexToAddress("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf")
	message := bytes.Join([][]byte{
		payment.MpeContractAddress.Bytes(),
		bigIntToBytes(payment.ChannelID),
		bigIntToBytes(payment.ChannelNonce),
		bigIntToBytes(payment.Amount),
	}, nil)
	payment.Signature = getSignature(message, privateKey)
}


func getSignature(message []byte, privateKey *ecdsa.PrivateKey) (signature []byte) {
	hash := crypto.Keccak256(
		blockchain.HashPrefix32Bytes,
		crypto.Keccak256(message),
	)
	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		panic(fmt.Sprintf("Cannot sign test message: %v", err))
	}

	return signature
}



func (suite *PaymentChannelServiceSuite) payment() *escrow.Payment {
	payment := &escrow.Payment{
		Amount:       big.NewInt(12300),
		ChannelID:    big.NewInt(42),
		ChannelNonce: big.NewInt(3),
		MpeContractAddress: blockchain.HexToAddress("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf"),
	}
	//TO DO
	SignPayment(payment, suite.signerPrivateKey)
	return payment
}

func (suite *PaymentChannelServiceSuite) channelKey() *escrow.PaymentChannelKey {
	return &escrow.PaymentChannelKey{
		ID: big.NewInt(42),
	}
}

func (suite *PaymentChannelServiceSuite) channel() *escrow.PaymentChannelData {
	return &escrow.PaymentChannelData{
		ChannelID:        big.NewInt(42),
		Nonce:            big.NewInt(3),
		Sender:           suite.senderAddress,
		Recipient:        suite.recipientAddress,
		GroupID:          blockchain.StringToBytes32("nXzNEetD1kzU3PZqR4nHPS8erDkrUK0hN4iCBQ4vH5U"),
		FullAmount:       big.NewInt(12345),
		Expiration:       big.NewInt(100),
		Signer:           suite.signerAddress,
		AuthorizedAmount: big.NewInt(0),
		Signature:        nil,
	}
}



