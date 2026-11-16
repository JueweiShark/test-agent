package customCodec

import (
    "net/http"

    kratosHttp "github.com/go-kratos/kratos/v2/transport/http"
)

// CentralDecodeRequest is the central decoder that selects the appropriate decoder based on Content-Type
func CentralDecodeRequest(req *http.Request, request interface{}) error {
    //contentType := req.Header.Get("Content-Type")

    //switch {
    //case strings.HasPrefix(contentType, "multipart/form-data"):
    //    return decodeMultipartRequest(req, request)
    //default:
    //    return kratosHttp.DefaultRequestDecoder(req, request)
    //}
    return kratosHttp.DefaultRequestDecoder(req, request)
}
