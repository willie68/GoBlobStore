package backup

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/willie68/GoBlobStore/mocks"
)

type MockStorage struct {
	mock.Mock
}

func TestSyncForward(t *testing.T) {
	ast := assert.New(t)
	mockStg := &mocks.BlobStorageDao{}
	ast.NotNil(mockStg)
	mockStg.On("HasBlob", "100").Return(true, nil)
	mockStg.On("HasBlob", "101").Return(true, nil)
	mockStg.On("HasBlob", "102").Return(true, nil)
	mockStg.On("HasBlob", "103").Return(true, nil)
	mockStg.On("HasBlob", "104").Return(true, nil)
	mockStg.On("HasBlob", mock.Anything).Return(false, errors.New("id cannot be null or empty"))
	mockStg.On("GetBlobs", 0, 10).Return([]string{"100", "101", "102", "103", "104"}, nil)
	ok, err := mockStg.HasBlob("")
	ast.NotNil(err)
	ast.False(ok)

	ok, err = mockStg.HasBlob("100")
	ast.Nil(err)
	ast.True(ok)

	mockBck := &mocks.BlobStorageDao{}
	ast.NotNil(mockStg)

	mockBck.On("HasBlob", "100").Return(true, nil)
	mockBck.On("HasBlob", "101").Return(true, nil)
	mockBck.On("HasBlob", "102").Return(true, nil)
	mockBck.On("HasBlob", mock.Anything).Return(false, errors.New("id cannot be null or empty"))
	mockBck.On("GetBlobs", 0, 10).Return([]string{"100", "101", "102"}, nil)
    

	err = migrateBckTnt(mockStg, mockBck)


	ast.Nil(err)
}
