package training

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoModelSupportTrainingService(t *testing.T) {
	s := NoModelSupportTrainingService{}
	ctx := context.Background()

	_, err := s.CreateModel(ctx, &NewModel{})
	assert.Error(t, err)

	_, err = s.ValidateModelPrice(ctx, &ValidateRequest{})
	assert.Error(t, err)

	err = s.UploadAndValidate(nil)
	assert.Error(t, err)

	_, err = s.ValidateModel(ctx, &ValidateRequest{})
	assert.Error(t, err)

	_, err = s.TrainModelPrice(ctx, &ModelID{})
	assert.Error(t, err)

	_, err = s.TrainModel(ctx, &ModelID{})
	assert.Error(t, err)

	_, err = s.DeleteModel(ctx, &ModelID{})
	assert.Error(t, err)

	_, err = s.GetModelStatus(ctx, &ModelID{})
	assert.Error(t, err)

	assert.Panics(t, func() { s.mustEmbedUnimplementedModelServer() })
}

func TestNoTrainingDaemonServerUploadAndValidate(t *testing.T) {
	s := NoTrainingDaemonServer{}

	err := s.UploadAndValidate(nil)
	assert.Error(t, err)

	assert.Panics(t, func() { s.mustEmbedUnimplementedDaemonServer() })
}

func TestDaemonServiceMustEmbed(t *testing.T) {
	ds := &DaemonService{}
	assert.Panics(t, func() { ds.mustEmbedUnimplementedDaemonServer() })
}

func TestTestTrainServerMethods(t *testing.T) {
	s := &TestTrainServer{}
	ctx := context.Background()

	price, err := s.ValidateModelPrice(ctx, &ValidateRequest{})
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), price.Price)

	resp, err := s.ValidateModel(ctx, &ValidateRequest{})
	assert.NoError(t, err)
	assert.Equal(t, Status_VALIDATING, resp.Status)

	price, err = s.TrainModelPrice(ctx, &ModelID{})
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), price.Price)

	resp, err = s.TrainModel(ctx, &ModelID{})
	assert.NoError(t, err)
	assert.Equal(t, Status_TRAINING, resp.Status)

	resp, err = s.DeleteModel(ctx, &ModelID{})
	assert.NoError(t, err)
	assert.Equal(t, Status_DELETED, resp.Status)

	assert.Panics(t, func() { _ = s.UploadAndValidate(nil) })
	assert.Panics(t, func() { s.mustEmbedUnimplementedModelServer() })
}
