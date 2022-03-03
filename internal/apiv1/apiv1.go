package apiv1

import "fmt"

const ApiVersion = "1"

var BaseURL = fmt.Sprintf("/api/v%s", ApiVersion)

//APIKey the apikey of this service
var APIKey string

const adminSubpath = "/admin"
const storesSubpath = "/stores"
const configSubpath = "/config"
const blobsSubpath = "/blobs"
const searchSubpath = "/search"
