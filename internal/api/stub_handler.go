package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hungdv136/rio"
	"github.com/hungdv136/rio/internal/grpc"
	"github.com/hungdv136/rio/internal/log"
	"github.com/hungdv136/rio/internal/types"
	"github.com/hungdv136/rio/internal/util"
)

// StubRequestParam stub request
type StubRequestParam struct {
	Stubs []*rio.Stub `json:"stubs" yaml:"stubs"`

	// Go SDK expect the response should be encoded data
	ReturnEncoded bool `json:"return_encoded" yaml:"return_encoded"`
}

// handleCreate handles create a list of stubs
// CreateStubs godoc
// @Summary     Create stubs
// @Description Create a list of stubs
// @ID          create-stubs
// @Tags        Stubs
// @Param       request body StubRequestParam true "request body"
// @Success     200 {object}types.Map{stubs=[]rio.Stub}
// @Failure     404 {object}types.Map{message=string}
// @Failure     500 {object}types.Map{message=string}
// @Router      /stub/create_many [post]
func (app *App) handleCreate(ctx *gin.Context) {
	params := StubRequestParam{}
	if err := ctx.ShouldBind(&params); err != nil {
		log.Error(ctx, err)
		SendError(ctx, err)
		return
	}

	if len(params.Stubs) == 0 {
		SendJSON(ctx, http.StatusBadRequest, VedictMissingParameters, "missing parameters", types.Map{})
		return
	}

	for i, stub := range params.Stubs {
		if err := stub.Validate(ctx); err != nil {
			msg := fmt.Sprintf("invalid stub at index %d - %s", i, err.Error())
			SendJSON(ctx, http.StatusBadRequest, VerdictInvalidParameters, msg, types.Map{})
			return
		}

		if stub.Request != nil && stub.Request.Method == rio.MethodGrpc {
			stub.Protocol = rio.ProtocolGrpc
		} else {
			stub.Protocol = rio.ProtocolHTTP
		}
	}

	if err := app.stubStore.Create(ctx, params.Stubs...); err != nil {
		SendError(ctx, err)
		return
	}

	data, err := buildResponseStubs(ctx, !params.ReturnEncoded, params.Stubs)
	if err != nil {
		SendError(ctx, err)
		return
	}

	SendSuccess(ctx, "add stubs successfully", data)
}

// handleGetStubs handles get a list of stubs
// GetStubs godoc
// @Summary     Get stubs
// @Description Get a list of stubs
// @ID          get-stubs
// @Tags        Stubs
// @Param       namespace query string false "Namespace"
// @Success     200 {object}types.Map{stubs=[]rio.Stub}
// @Failure     404 {object}types.Map{message=string}
// @Failure     500 {object}types.Map{message=string}
// @Router      /stub/list [get]
func (app *App) handleGetStubs(ctx *gin.Context) {
	namespace := ctx.Query("namespace")
	stubs, err := app.stubStore.GetAll(ctx, namespace)
	if err != nil {
		SendError(ctx, err)
		return
	}

	data, err := buildResponseStubs(ctx, !ctx.GetBool("return_encoded"), stubs)
	if err != nil {
		SendError(ctx, err)
		return
	}

	SendSuccess(ctx, "list stubs successfully", data)
}

// handleUpload
// Upload godoc
// @Summary Upload file API
// @Description Upload file to storage
// @ID update-file
// @Tags UploadFile
// @Success 200 {object}types.Map{file_id=string}
// @Failure 400 {object}types.Map{message=string}
// @Router /stub/upload [post]
func (app *App) handleUpload(ctx *gin.Context) {
	if err := ctx.Request.ParseMultipartForm(1024 * 1024 * 20 << 20); err != nil {
		log.Error(ctx, err)
		SendError(ctx, err)
		return
	}

	fileID := ctx.Request.FormValue("file_id")
	if len(fileID) == 0 {
		fileID = uuid.NewString()
	}

	file, _, err := ctx.Request.FormFile("file")
	if err != nil {
		log.Error(ctx, err)
		SendError(ctx, err)
		return
	}
	defer util.CloseSilently(ctx, file.Close)

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(file); err != nil {
		log.Error(ctx, err)
		SendError(ctx, err)
		return
	}

	if _, err := app.fileStorage.UploadFile(ctx, fileID, buf); err != nil {
		log.Error(ctx, err)
		SendError(ctx, err)
		return
	}

	SendSuccess(ctx, "upload file successfully", types.Map{"file_id": fileID})
}

// handleUploadProto
// UploadProto godoc
// @Summary Upload proto API
// @Description Upload proto to storage
// @ID update-proto
// @Tags UploadProto
// @Success 200 {object}types.Map{proto=int}
// @Failure 400 {object}types.Map{message=string}
// @Router /proto/upload [post]
func (app *App) handleUploadProto(ctx *gin.Context) {
	if err := ctx.Request.ParseMultipartForm(1024 * 1024 * 20 << 20); err != nil {
		log.Error(ctx, err)
		SendError(ctx, err)
		return
	}

	name := ctx.Request.FormValue("name")
	file, _, err := ctx.Request.FormFile("file")
	if err != nil {
		log.Error(ctx, err)
		SendError(ctx, err)
		return
	}
	defer util.CloseSilently(ctx, file.Close)

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(file); err != nil {
		log.Error(ctx, err)
		SendError(ctx, err)
		return
	}

	fileID := uuid.NewString()
	if _, err = app.fileStorage.UploadFile(ctx, fileID, buf); err != nil {
		log.Error(ctx, err)
		SendError(ctx, err)
		return
	}

	desc, err := grpc.NewServiceDescriptor(app.fileStorage).GetDescriptor(ctx, fileID)
	if err != nil {
		_ = app.fileStorage.DeleteFile(ctx, fileID)
		SendError(ctx, err)
		return
	}

	proto := &rio.Proto{
		Name:    name,
		FileID:  fileID,
		Methods: desc.GetAllMethods(),
		Types:   desc.GetAllMessages(),
	}

	if len(proto.Methods) == 0 {
		_ = app.fileStorage.DeleteFile(ctx, fileID)
		SendJSON(ctx, http.StatusBadRequest, VerdictInvalidParameters, "invalid proto", types.Map{})
		return
	}

	if err := app.stubStore.CreateProto(ctx, proto); err != nil {
		SendError(ctx, err)
		return
	}

	SendSuccess(ctx, "upload proto successfully", types.Map{"proto": proto})
}

// handleGetIncomingRequest handles get a list of requests
// GetRequests godoc
// @Summary     Get requests
// @Description Get a list of requests
// @ID          get-requests
// @Tags        Requests
// @Success     200 {object}types.Map{requests=[]rio.IncomingRequest}
// @Failure     404 {object}types.Map{message=string}
// @Failure     500 {object}types.Map{message=string}
// @Router      /incoming_request/list [get]
func (app *App) handleGetIncomingRequest(ctx *gin.Context) {
	params := rio.IncomingQueryOption{}
	if err := ctx.ShouldBind(&params); err != nil {
		log.Error(ctx, err)
		SendError(ctx, err)
		return
	}

	if params.Limit == 0 {
		params.Limit = 10
	}

	requests, err := app.stubStore.GetIncomingRequests(ctx, &params)
	if err != nil {
		SendError(ctx, err)
		return
	}

	SendSuccess(ctx, "get incoming request successfully", types.Map{"requests": requests})
}

// handleReset handles reset stubs by a namespace. If the namespace is "reset_all", then reset all stubs
// Reset godoc
// @Summary     Reset stubs
// @Description Reset stubs by a namespace
// @ID          reset
// @Tags        Reset
// @Param       namespace query string true "Namespace"
// @Success     200 {object}types.Map{message=string}
// @Failure     404 {object}types.Map{message=string}
// @Failure     500 {object}types.Map{message=string}
// @Router      /reset [delete]
func (app *App) handleReset(ctx *gin.Context) {
	option := &rio.ResetQueryOption{
		Namespace: ctx.Query("namespace"),
		Tag:       ctx.Query("tag"),
	}

	if err := app.stubStore.Reset(ctx, option); err != nil {
		SendError(ctx, err)
		return
	}

	SendSuccess(ctx, fmt.Sprintf("reset data in '%s' and tag '%s' successfully", option.Namespace, option.Tag), types.Map{})
}

func (app *App) handlePing(ctx *gin.Context) {
	SendSuccess(ctx, "pong", types.Map{})
}

func buildResponseStubs(ctx context.Context, shouldDecode bool, stubs []*rio.Stub) (types.Map, error) {
	if !shouldDecode {
		return types.Map{"stubs": stubs}, nil
	}

	stubsMap := make([]types.Map, len(stubs))
	for i, stub := range stubs {
		m, err := types.CreateMapFromStruct(stub)
		if err != nil {
			log.Error(ctx, err)
			return nil, err
		}

		stubsMap[i] = m
		res := stub.Response

		if res == nil || res.Body == nil || res.Header == nil {
			log.Info(ctx, "body is nil")
			continue
		}

		contentType := res.Header[rio.HeaderContentType]
		if strings.Contains(contentType, rio.ContentTypeJSON) {
			bodyMap, err := types.CreateMapFromReader(bytes.NewReader(res.Body))
			if err != nil {
				log.Error(ctx, err, string(res.Body))
				return nil, err
			}

			m.ForceMap("response")["body"] = bodyMap
			log.Info(ctx, "decoded body from base64 to raw json", contentType, m.ForceJSON())
			continue
		}
	}

	return types.Map{"stubs": stubsMap}, nil
}
