package metrics

import (
	"encoding/json"
	"github.com/rs/xid"
	"google.golang.org/grpc/metadata"
)

//Get the value of the first Pair
func GetValue(md metadata.MD, key string) string {
	array := md.Get(key)
	if len(array) == 0 {
		return ""
	}
	return array[0]
}

//Generate a unique global Id
func GenXid() string {
	id := xid.New()
	return id.String()
}

//convert the given struct to its corresponding json.
func ConvertObjectToJSON(structbody interface{}) (string, error) {
	b, err := json.Marshal(&structbody)
	if err != nil {
		return "", err
	}
	return string(b[:]), nil
}
