package apiv1

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/willie68/GoBlobStore/internal/api"
	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/dao"
	"github.com/willie68/GoBlobStore/internal/serror"
	"github.com/willie68/GoBlobStore/internal/utils/httputils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const Baseurl = "/api/v1"

const BlobsSubpath = "/blobs"

//APIKey the apikey of this service
var APIKey string

//SystemID the systemid of this service
var SystemID string

// BlobStore the blobstorage implementation to use
var BlobStore dao.BlobStorageDao

func BlobRoutes() *chi.Mux {
	router := chi.NewRouter()
	router.Post("/", PostBlobEndpoint)
	router.Get("/", GetBlobsEndpoint)
	router.Get("/{id}", GetBlobEndpoint)
	router.Get("/{id}/info", GetBlobInfoEndpoint)
	router.Delete("/{id}", DeleteBlobEndpoint)
	router.Get("/{id}/resetretention", GetBlobResetRetentionEndpoint)
	return router
}

func getBlobLocation(blobid string) string {
	return fmt.Sprintf(Baseurl+BlobsSubpath+"/%s", blobid)
}

/*
GetBlobEndpoint getting one blob file for a tenant from the storage
path parameter
id: the id of the blob file
*/
func GetBlobEndpoint(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	idStr := chi.URLParam(request, "id")
	storage, err := dao.GetStorageDao(tenant)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	b, err := storage.GetBlobDescription(idStr)
	if err != nil {
		if os.IsNotExist(err) {
			httputils.Err(response, request, serror.NotFound("blob", idStr, nil))
			return
		}
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	if b == nil {
		httputils.Err(response, request, serror.NotFound("blob", idStr, nil))
		return
	}

	for k, v := range b.Properties {
		response.Header().Set(k, v.(string))
	}

	response.Header().Add("Location", idStr)
	RetentionHeader, ok := config.Get().HeaderMapping[api.RetentionHeaderKey]
	if ok {
		response.Header().Add(RetentionHeader, strconv.FormatInt(int64(b.Retention), 10))
	}
	response.Header().Set("Content-Type", b.ContentType)
	response.WriteHeader(http.StatusOK)

	err = storage.RetrieveBlob(idStr, response)

	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
}

/*
GetBlobInfoEndpoint getting the info of a blob file from the storage
path param:
id: the id of the blob file
*/
func GetBlobInfoEndpoint(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	idStr := chi.URLParam(request, "id")

	storage, err := dao.GetStorageDao(tenant)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	b, err := storage.GetBlobDescription(idStr)
	if err != nil {
		if os.IsNotExist(err) {
			httputils.Err(response, request, serror.NotFound("blob", idStr, nil))
			return
		}
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	if b == nil {
		httputils.Err(response, request, serror.NotFound("blob", idStr, nil))
		return
	}
	b.BlobURL = getBlobLocation(b.BlobID)

	render.JSON(response, request, b)
}

/*
GetBlobResetRetentionEndpoint restting the retention time of a blob to the new value
path param:
id: the id of the lob file
*/
//TODO missing implementation
func GetBlobResetRetentionEndpoint(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	storage, err := dao.GetStorageDao(tenant)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	idStr := chi.URLParam(request, "id")
	found, err := storage.HasBlob(idStr)
	if err != nil {
		if os.IsNotExist(err) {
			httputils.Err(response, request, serror.NotFound("blob", idStr, nil))
			return
		}
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	render.JSON(response, request, found)
}

/*
GetBlobsEndpoint query all blobs from the storage for a tenant
query params
offset: the offset to start from
limit: max count of blobs
*/
func GetBlobsEndpoint(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	url := request.URL
	values := url.Query()

	storage, err := dao.GetStorageDao(tenant)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	offset := 0
	if values["offset"] != nil {
		offset, _ = strconv.Atoi(values["offset"][0])
	}
	limit := 1000
	if values["limit"] != nil {
		limit, _ = strconv.Atoi(values["limit"][0])
	}
	blobs, err := storage.GetBlobs(offset, limit)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	render.JSON(response, request, blobs)
}

/*
PostBlobsEndpoint creating a new blob in the storage for the tenant.
*/
func PostBlobEndpoint(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	mimeType := request.Header.Get("Content-Type")
	var cntLength int64
	var filename string
	var f io.Reader
	if strings.HasPrefix(mimeType, "multipart/form-data") {
		err := request.ParseMultipartForm(1024 * 1024 * 1024)
		if err != nil {
			httputils.Err(response, request, serror.InternalServerError(err))
			return
		}
		mpf, fileHeader, err := request.FormFile("file")
		if err != nil {
			httputils.Err(response, request, serror.InternalServerError(err))
			return
		}

		mimeType = fileHeader.Header.Get("Content-type")
		cntLength = fileHeader.Size
		filename = fileHeader.Filename
		f = mpf
		defer mpf.Close()
	} else {
		mpf := request.Body
		defer mpf.Close()
		if err != nil {
			httputils.Err(response, request, serror.InternalServerError(err))
			return
		}
		cntLength = -1
		filename = "data.bin"

		FilenameHeader, ok := config.Get().HeaderMapping[api.FilenameKey]
		if ok {
			filename = request.Header.Get(FilenameHeader)
		}
		f = mpf
	}

	var retentionTime int64 = 0
	RetentionHeader, ok := config.Get().HeaderMapping[api.RetentionHeaderKey]
	if ok {
		retention := request.Header.Get(RetentionHeader)
		retentionTime, _ = strconv.ParseInt(retention, 10, 64)
	}

	metadata := make(map[string]interface{})
	headerPrefix, ok := config.Get().HeaderMapping[api.HeaderPrefixKey]
	if ok {
		headerPrefix = strings.ToLower(headerPrefix)
		for key := range request.Header {
			if strings.HasPrefix(strings.ToLower(key), headerPrefix) {
				metadata[key] = request.Header.Get(key)
			}
		}
	}

	b := model.BlobDescription{
		StoreID:       tenant,
		TenantID:      tenant,
		ContentLength: cntLength,
		ContentType:   mimeType,
		Retention:     retentionTime,
		Filename:      filename,
		Properties:    metadata,
		CreationDate:  int(time.Now().UnixNano() / 1000000),
	}
	storage, err := dao.GetStorageDao(tenant)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	_, err = storage.StoreBlob(&b, f)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	location := getBlobLocation(b.BlobID)
	b.BlobURL = location
	response.Header().Add("Location", location)
	response.Header().Add(RetentionHeader, strconv.FormatInt(retentionTime, 10))
	render.Status(request, http.StatusCreated)
	render.JSON(response, request, b)
}

/*
DeleteBlobEndpoint delete a dedicated blob from the storage for the tenant
path param
id: the id of the blob to remove
*/
func DeleteBlobEndpoint(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	storage, err := dao.GetStorageDao(tenant)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	idStr := chi.URLParam(request, "id")

	b, err := storage.GetBlobDescription(idStr)
	if err != nil {
		if os.IsNotExist(err) {
			httputils.Err(response, request, serror.NotFound("blob", idStr, nil))
			return
		}
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	if b == nil {
		httputils.Err(response, request, serror.NotFound("blob", idStr, err))
		return
	}
	err = storage.DeleteBlob(idStr)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	render.JSON(response, request, idStr)
}
