package apiv1

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/willie68/GoBlobStore/internal/api"
	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/dao"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/serror"
	"github.com/willie68/GoBlobStore/internal/utils/httputils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const blobsSubpath = "/blobs"
const searchSubpath = "/search"

// BlobStore the blobstorage implementation to use
var BlobStore interfaces.BlobStorageDao

func BlobRoutes() (string, *chi.Mux) {
	router := chi.NewRouter()
	router.Post("/", PostBlob)
	router.Get("/", GetBlobs)
	router.Get("/{id}", GetBlob)
	router.Get("/{id}/info", GetBlobInfo)
	router.Put("/{id}/info", PutBlobInfo)
	router.Delete("/{id}", DeleteBlob)
	router.Get("/{id}/resetretention", GetBlobResetRetention)
	router.Get("/{id}/check", GetBlobCheck)
	router.Post("/{id}/check", PostBlobCheck)
	return BaseURL + blobsSubpath, router
}

func SearchRoutes() (string, *chi.Mux) {
	router := chi.NewRouter()
	router.Get("/", SearchBlobs)
	return BaseURL + searchSubpath, router
}

func getBlobLocation(blobid string) string {
	return fmt.Sprintf(BaseURL+blobsSubpath+"/%s", blobid)
}

/*
GetBlob getting one blob file for a tenant from the storage
path parameter
id: the id of the blob file
*/
func GetBlob(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	idStr := chi.URLParam(request, "id")

	stgf, err := dao.GetStorageFactory()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	storage, err := stgf.GetStorageDao(tenant)
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

	for k, i := range b.Properties {
		switch v := i.(type) {
		case int:
			response.Header().Set(k, strconv.Itoa(v))
		case int64:
			response.Header().Set(k, strconv.FormatInt(v, 10))
		case string:
			response.Header().Set(k, v)
		default:
		}
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
GetBlobInfo getting the info of a blob file from the storage
path param:
id: the id of the blob file
*/
func GetBlobInfo(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	idStr := chi.URLParam(request, "id")

	stgf, err := dao.GetStorageFactory()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	storage, err := stgf.GetStorageDao(tenant)
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
PutBlobInfo getting the info of a blob file from the storage
path param:
id: the id of the blob file
*/
func PutBlobInfo(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	id := chi.URLParam(request, "id")

	stgf, err := dao.GetStorageFactory()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	storage, err := stgf.GetStorageDao(tenant)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	b, err := storage.GetBlobDescription(id)
	if err != nil {
		if os.IsNotExist(err) {
			httputils.Err(response, request, serror.NotFound("blob", id, nil))
			return
		}
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	if b == nil {
		httputils.Err(response, request, serror.NotFound("blob", id, nil))
		return
	}
	var bd model.BlobDescription
	err = json.NewDecoder(request.Body).Decode(&bd)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	for k, v := range bd.Properties {
		if v == nil {
			delete(b.Properties, k)
		} else {
			b.Properties[k] = v
		}
	}

	err = storage.UpdateBlobDescription(id, b)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	b.BlobURL = getBlobLocation(b.BlobID)

	render.JSON(response, request, b)
}

/*
GetBlobResetRetention restting the retention time of a blob to the new value
path param:
id: the id of the lob file
*/
func GetBlobResetRetention(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	stgf, err := dao.GetStorageFactory()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	storage, err := stgf.GetStorageDao(tenant)
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
	if found {
		err = storage.ResetRetention(idStr)
		if err != nil {
			httputils.Err(response, request, serror.InternalServerError(err))
		}
	}
	render.JSON(response, request, found)
}

/*
GetBlobs query all blobs from the storage for a tenant
query params
offset: the offset to start from
limit: max count of blobs
*/
func GetBlobs(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := fmt.Sprintf("tenant missing: %v", err)
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	url := request.URL
	values := url.Query()

	stgf, err := dao.GetStorageFactory()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	storage, err := stgf.GetStorageDao(tenant)
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
	blobs := make([]string, 0)
	index := 0
	err = storage.GetBlobs(func(id string) bool {
		if (index >= offset) && (index-offset < limit) {
			blobs = append(blobs, id)
		}
		if index-offset > limit {
			return false
		}
		index++
		return true
	})
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	render.JSON(response, request, blobs)
}

/*
PostBlobsEndpoint creating a new blob in the storage for the tenant.
*/
func PostBlob(response http.ResponseWriter, request *http.Request) {
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

	stgf, err := dao.GetStorageFactory()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	storage, err := stgf.GetStorageDao(tenant)
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
DeleteBlob delete a dedicated blob from the storage for the tenant
path param
id: the id of the blob to remove
*/
func DeleteBlob(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}

	stgf, err := dao.GetStorageFactory()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	storage, err := stgf.GetStorageDao(tenant)
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

/*
SearchBlobs search for blobs meeting the criteria
query params
offset: the offset to start from
limit: max count of blobs
q: query to use
*/
func SearchBlobs(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := fmt.Sprintf("tenant missing: %v", err)
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	url := request.URL
	values := url.Query()

	stgf, err := dao.GetStorageFactory()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	storage, err := stgf.GetStorageDao(tenant)
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
	var query string
	if values["q"] != nil {
		query = values["q"][0]
	}
	if query != "" {
		log.Logger.Debugf("search for blobs with: %s", query)
	}
	blobs := make([]string, 0)
	index := 0
	err = storage.SearchBlobs(query, func(id string) bool {
		if (index >= offset) && (index-offset < limit) {
			blobs = append(blobs, id)
		}
		if index-offset > limit {
			return false
		}
		index++
		return true
	})
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	render.JSON(response, request, blobs)
}

/*
GetBlobCheck getting the latest check info of a blob file from the storage
path param:
id: the id of the blob file
*/
func GetBlobCheck(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	idStr := chi.URLParam(request, "id")

	stgf, err := dao.GetStorageFactory()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	storage, err := stgf.GetStorageDao(tenant)
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
	check := b.Check
	if check == nil {
		check = &model.Check{
			Healthy: false,
			Message: "not checked",
		}
	}
	render.JSON(response, request, check)
}

// PostBlobCheck starting a new check for this tenant
// @Summary starting a new check for this tenant
// @Tags configs
// @Accept  json
// @Produce  json
// @Security api_key
// @Param tenant header string true "Tenant"
// @Success 200 {array} CreateResponse "response with the id of the tenant as json"
// @Failure 400 {object} serror.Serr "client error information as json"
// @Failure 500 {object} serror.Serr "server error information as json"
// @Router /api/v1/admin/check [post]
func PostBlobCheck(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	idStr := chi.URLParam(request, "id")

	log.Logger.Infof("do check for tenant %s on blob %s", tenant, idStr)
	stgf, err := dao.GetStorageFactory()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	storage, err := stgf.GetStorageDao(tenant)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	res, err := storage.CheckBlob(idStr)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	render.Status(request, http.StatusCreated)
	render.JSON(response, request, res)
}
