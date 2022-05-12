package training

import (
	"github.com/singnet/snet-daemon/storage"
	"reflect"
	"testing"
)

func TestModelStorage_CompareAndSwap(t *testing.T) {
	type fields struct {
		delegate storage.TypedAtomicStorage
	}
	type args struct {
		key       *ModelUserKey
		prevState *ModelUserData
		newState  *ModelUserData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantOk  bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &ModelStorage{
				delegate: tt.fields.delegate,
			}
			gotOk, err := storage.CompareAndSwap(tt.args.key, tt.args.prevState, tt.args.newState)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompareAndSwap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotOk != tt.wantOk {
				t.Errorf("CompareAndSwap() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestModelStorage_Get(t *testing.T) {
	type fields struct {
		delegate storage.TypedAtomicStorage
	}
	type args struct {
		key *ModelUserKey
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantState *ModelUserData
		wantOk    bool
		wantErr   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &ModelStorage{
				delegate: tt.fields.delegate,
			}
			gotState, gotOk, err := storage.Get(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotState, tt.wantState) {
				t.Errorf("Get() gotState = %v, want %v", gotState, tt.wantState)
			}
			if gotOk != tt.wantOk {
				t.Errorf("Get() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestModelStorage_GetAll(t *testing.T) {
	type fields struct {
		delegate storage.TypedAtomicStorage
	}
	tests := []struct {
		name       string
		fields     fields
		wantStates []*ModelUserData
		wantErr    bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &ModelStorage{
				delegate: tt.fields.delegate,
			}
			gotStates, err := storage.GetAll()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotStates, tt.wantStates) {
				t.Errorf("GetAll() gotStates = %v, want %v", gotStates, tt.wantStates)
			}
		})
	}
}

func TestModelStorage_Put(t *testing.T) {
	type fields struct {
		delegate storage.TypedAtomicStorage
	}
	type args struct {
		key   *ModelUserKey
		state *ModelUserData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &ModelStorage{
				delegate: tt.fields.delegate,
			}
			if err := storage.Put(tt.args.key, tt.args.state); (err != nil) != tt.wantErr {
				t.Errorf("Put() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestModelStorage_PutIfAbsent(t *testing.T) {
	type fields struct {
		delegate storage.TypedAtomicStorage
	}
	type args struct {
		key   *ModelUserKey
		state *ModelUserData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantOk  bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &ModelStorage{
				delegate: tt.fields.delegate,
			}
			gotOk, err := storage.PutIfAbsent(tt.args.key, tt.args.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("PutIfAbsent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotOk != tt.wantOk {
				t.Errorf("PutIfAbsent() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestModelUserKey_String(t *testing.T) {
	type fields struct {
		OrganizationId string
		ServiceId      string
		GroupID        string
		MethodName     string
		ModelId        string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &ModelUserKey{
				OrganizationId: tt.fields.OrganizationId,
				ServiceId:      tt.fields.ServiceId,
				GroupID:        tt.fields.GroupID,
				MethodName:     tt.fields.MethodName,
				ModelId:        tt.fields.ModelId,
			}
			if got := key.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewUserModelStorage(t *testing.T) {
	type args struct {
		atomicStorage storage.AtomicStorage
	}
	tests := []struct {
		name string
		args args
		want *ModelStorage
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewUserModelStorage(tt.args.atomicStorage); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUserModelStorage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_serializeModelKey(t *testing.T) {
	type args struct {
		key interface{}
	}
	tests := []struct {
		name           string
		args           args
		wantSerialized string
		wantErr        bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSerialized, err := serializeModelKey(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("serializeModelKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotSerialized != tt.wantSerialized {
				t.Errorf("serializeModelKey() gotSerialized = %v, want %v", gotSerialized, tt.wantSerialized)
			}
		})
	}
}
