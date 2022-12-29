module github.com/singnet/snet-daemon

go 1.15

require (
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/OneOfOne/go-utils v0.0.0-20180319162427-6019ff89a94e
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412
	github.com/allegro/bigcache v1.2.1 // indirect
	github.com/aristanetworks/goarista v0.0.0-20180607101720-59944ff78bc1
	github.com/beorn7/perks v0.0.0-20180321164747-3a771d992973
	github.com/btcsuite/btcd v0.0.0-20190213025234-306aecffea32
	github.com/btcsuite/btcutil v0.0.0-20190207003914-4c204d697803
	github.com/census-instrumentation/opencensus-proto v0.2.1 // indirect
	github.com/coreos/bbolt v1.3.1-etcd.8
	github.com/coreos/etcd v3.3.10+incompatible
	github.com/coreos/go-semver v0.3.0
	github.com/coreos/go-systemd v0.0.0-20181031085051-9002847aa142
	github.com/coreos/pkg v0.0.0-20180108230652-97fdf19511ea
	github.com/davecgh/go-spew v1.1.1
	github.com/deckarep/golang-set v1.7.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/edsrzf/mmap-go v0.0.0-20170320065105-0bce6a688712
	github.com/emicklei/proto v1.11.1 // indirect
	github.com/envoyproxy/go-control-plane v0.7.1 // indirect
	github.com/envoyproxy/protoc-gen-validate v0.1.0 // indirect
	github.com/ethereum/go-ethereum v1.8.27
	github.com/fsnotify/fsnotify v1.4.7
	github.com/ghodss/yaml v1.0.0
	github.com/go-stack/stack v1.7.0
	github.com/gogo/protobuf v1.2.1
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.2.0
	github.com/golang/snappy v0.0.0-20180518054509-2e65f85255db
	github.com/google/btree v1.0.0
	github.com/google/go-cmp v0.5.0 // indirect
	github.com/gorilla/handlers v1.3.0
	github.com/gorilla/rpc v1.1.0
	github.com/gorilla/websocket v1.2.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.5.1
	github.com/gxed/hashland v0.0.0-20180221191214-d9f6b97f8db2
	github.com/hashicorp/golang-lru v0.5.0
	github.com/hashicorp/hcl v0.0.0-20180404174102-ef8a98b0bbce
	github.com/improbable-eng/grpc-web v0.0.0-20181031170435-f683dbb3b587
	github.com/inconshreveable/mousetrap v1.0.0
	github.com/ipfs/go-ipfs-api v0.0.2-0.20190404072909-740521c74b61
	github.com/ipfs/go-ipfs-files v0.0.2
	github.com/jonboulle/clockwork v0.1.0
	github.com/konsorten/go-windows-terminal-sequences v0.0.0-20180402223658-b729f2633dfe
	github.com/lestrrat-go/file-rotatelogs v2.2.0+incompatible
	github.com/lestrrat-go/strftime v0.0.0-20180821113735-8b31f9c59b0f
	github.com/libp2p/go-flow-metrics v0.0.3 // indirect
	github.com/libp2p/go-libp2p-crypto v0.1.0 // indirect
	github.com/libp2p/go-libp2p-metrics v0.1.0 // indirect
	github.com/libp2p/go-libp2p-peer v0.2.0 // indirect
	github.com/libp2p/go-libp2p-protocol v0.1.0 // indirect
	github.com/magiconair/properties v1.8.0
	github.com/mattn/goveralls v0.0.7 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/minio/sha256-simd v0.1.1-0.20190913151208-6de447530771
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v0.0.0-20180511142126-bb74f1db0675
	github.com/mr-tron/base58 v1.1.3
	github.com/multiformats/go-multiaddr v0.3.1 // indirect
	github.com/multiformats/go-multiaddr-net v0.2.0 // indirect
	github.com/multiformats/go-multihash v0.0.14 // indirect
	github.com/pborman/uuid v0.0.0-20170612153648-e790cca94e6c
	github.com/pelletier/go-toml v1.2.0
	github.com/petar/GoLLRB v0.0.0-20130427215148-53be0d36a84c
	github.com/pkg/errors v0.8.0
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/client_golang v0.9.1
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/prometheus/common v0.0.0-20181120120127-aeab699e26f4
	github.com/prometheus/procfs v0.0.0-20181005140218-185b4288413d
	github.com/rjeczalik/notify v0.0.0-20180312213058-d152f3ce359a
	github.com/rs/cors v1.4.0
	github.com/rs/xid v1.2.1
	github.com/sirupsen/logrus v1.1.0
	github.com/soheilhy/cmux v0.1.4
	github.com/spaolacci/murmur3 v1.1.0
	github.com/spf13/afero v1.1.1
	github.com/spf13/cast v1.2.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/jwalterweatherman v0.0.0-20180109140146-7c0cea34c8ec
	github.com/spf13/pflag v1.0.1
	github.com/spf13/viper v1.0.2
	github.com/stretchr/testify v1.2.2
	github.com/syndtr/goleveldb v0.0.0-20180609010929-e2150783cd35
	github.com/tmc/grpc-websocket-proxy v0.0.0-20171017195756-830351dc03c6
	github.com/tyler-smith/go-bip39 v0.0.0-20160629163856-8e7a99b3e716
	github.com/ugorji/go v1.1.1
	github.com/whyrusleeping/tar-utils v0.0.0-20180509141711-8c6c8ba81d5c
	github.com/xiang90/probing v0.0.0-20160813154853-07dd2e8dfe18
	github.com/zbindenren/logrus_mail v0.0.0-20170904205430-14351100bf70
	go.etcd.io/etcd v3.3.10+incompatible
	go.uber.org/atomic v1.3.2
	go.uber.org/multierr v1.1.0
	go.uber.org/zap v1.9.1
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/exp v0.0.0-20190121172915-509febef88a4 // indirect
	golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5 // indirect
	golang.org/x/net v0.0.0-20201209123823-ac852fbbde11
	golang.org/x/sys v0.0.0-20201211090839-8ad439b19e0f
	golang.org/x/text v0.3.4
	golang.org/x/time v0.0.0-20181108054448-85acf8d2951c
	golang.org/x/tools v0.0.0-20201211185031-d93e913c1a58
	google.golang.org/appengine v1.4.0 // indirect
	google.golang.org/genproto v0.0.0-20180817151627-c66870c02cf8
	google.golang.org/grpc v1.16.0
	gopkg.in/fatih/set.v0 v0.1.0
	gopkg.in/karalabe/cookiejar.v2 v2.0.0-20150724131613-8dcd6a7f4951
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce
	gopkg.in/yaml.v2 v2.2.1
	honnef.co/go/tools v0.0.0-20190523083050-ea95bdfd59fc // indirect
)
