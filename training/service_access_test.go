package training

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrainingAccessGrantRevokeAndDelete(t *testing.T) {
	ds := trainingFixture(t)
	startCoverageProvider(t, ds, false)
	const user = "0x0000000000000000000000000000000000000001"
	key := ds.userStorage.buildModelUserKey(user)
	require.NoError(t, ds.userStorage.Put(key, &ModelUserData{ModelIds: []string{"another-model"}}))
	updateAccess := func(addresses []string) {
		t.Helper()
		_, err := ds.UpdateModel(trainingContext(t, "update_model"), &UpdateModelRequest{
			Authorization: trainingAuth("update_model"), ModelId: "owned", AddressList: addresses,
		})
		require.NoError(t, err)
	}
	assertIndex := func(want []string) {
		t.Helper()
		data, ok, err := ds.userStorage.Get(key)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, want, data.ModelIds)
	}

	// Repeated addresses and repeated requests must not duplicate the model ID.
	updateAccess([]string{testUserAddress, user, user})
	assertIndex([]string{"another-model", "owned"})
	updateAccess([]string{testUserAddress, user})
	assertIndex([]string{"another-model", "owned"})
	updateAccess([]string{testUserAddress})
	assertIndex([]string{"another-model"})
	require.Error(t, ds.verifySignerHasAccessToTheModel("owned", user))
	updateAccess([]string{testUserAddress, user})
	assertIndex([]string{"another-model", "owned"})
	_, err := ds.DeleteModel(trainingContext(t, "delete_model"), &CommonRequest{
		Authorization: trainingAuth("delete_model"), ModelId: "owned",
	})
	require.NoError(t, err)
	assertIndex([]string{"another-model"})
	require.Error(t, ds.verifySignerHasAccessToTheModel("owned", user))
}

func TestTrainingRemovesLegacyDuplicateModelIDs(t *testing.T) {
	for _, operation := range []string{"revoke access", "delete model"} {
		t.Run(operation, func(t *testing.T) {
			ds := trainingFixture(t)
			startCoverageProvider(t, ds, false)
			const user = "0x0000000000000000000000000000000000000001"
			model, err := ds.storage.GetModel("owned")
			require.NoError(t, err)
			model.AuthorizedAddresses = []string{testUserAddress, user}
			require.NoError(t, ds.storage.Put(ds.storage.buildModelKey("owned"), model))
			key := ds.userStorage.buildModelUserKey(user)
			require.NoError(t, ds.userStorage.Put(key, &ModelUserData{
				ModelIds: []string{"owned", "keep-first", "owned", "keep-second", "owned"},
			}))
			if operation == "revoke access" {
				_, err = ds.UpdateModel(trainingContext(t, "update_model"), &UpdateModelRequest{
					Authorization: trainingAuth("update_model"), ModelId: "owned", AddressList: []string{testUserAddress},
				})
			} else {
				_, err = ds.DeleteModel(trainingContext(t, "delete_model"), &CommonRequest{
					Authorization: trainingAuth("delete_model"), ModelId: "owned",
				})
			}
			require.NoError(t, err)
			data, ok, err := ds.userStorage.Get(key)
			require.NoError(t, err)
			require.True(t, ok)
			require.Equal(t, []string{"keep-first", "keep-second"}, data.ModelIds)
			require.Error(t, ds.verifySignerHasAccessToTheModel("owned", user))
		})
	}
}

func TestTrainingRenamePreservesAccessIndex(t *testing.T) {
	ds := trainingFixture(t)
	name := "renamed"
	_, err := ds.UpdateModel(trainingContext(t, "update_model"), &UpdateModelRequest{
		Authorization: trainingAuth("update_model"), ModelId: "owned", ModelName: &name,
	})
	require.NoError(t, err)
	data, ok, err := ds.userStorage.Get(ds.userStorage.buildModelUserKey(testUserAddress))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []string{"owned"}, data.ModelIds)
	require.NoError(t, ds.verifySignerHasAccessToTheModel("owned", testUserAddress))
}

func TestTrainingCreateModelDoesNotDuplicateUserIndex(t *testing.T) {
	ds := trainingFixture(t)
	_, err := ds.createModelDetails(&NewModelRequest{
		Authorization: trainingAuth("create_model"),
		Model:         &NewModel{Name: "new", AddressList: []string{testUserAddress, testUserAddress}},
	}, &ModelID{ModelId: "new-model"})
	require.NoError(t, err)
	data, ok, err := ds.userStorage.Get(ds.userStorage.buildModelUserKey(testUserAddress))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []string{"owned", "new-model"}, data.ModelIds)
}
