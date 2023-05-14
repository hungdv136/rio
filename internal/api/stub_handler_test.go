package api

// func TestCreateStub(t *testing.T) {
// 	t.Parallel()

// 	// Assert that can submit JSON from Go client to API
// 	stub := rio.NewStub().
// 		For("GET", rio.Contains("animal/create")).
// 		WithDescription("this is description").
// 		WithHeader("X-REQUEST-ID", rio.EqualTo(uuid.NewString())).
// 		WithQuery("search_term", rio.EqualTo(uuid.NewString())).
// 		WithCookie("SESSION_ID", rio.EqualTo(uuid.NewString())).
// 		WillReturn(rio.NewResponse().WithBody(rio.MapToJSON(types.Map{"data": uuid.NewString()})))
// 	validParams := types.Map{"stubs": []*rio.Stub{stub}}

// 	// Assert that can submit with raw JSON without Go client
// 	jsonResponseParams := parseJSONFileToMap(t, "../../testdata/stubs.json")

// 	// XML or other text base should be submit as base64 due to html escaping issue
// 	xmlBody, err := os.ReadFile("../../testdata/body.xml")
// 	require.NoError(t, err)

// 	xmlResponseParams := parseJSONFileToMap(t, "../../testdata/stubs.json")
// 	xmlStubs, _ := xmlResponseParams.GetArrayMap("stubs")
// 	for _, stub := range xmlStubs {
// 		stub.ForceMap("response")["body"] = xmlBody
// 		stub.ForceMap("response")["header"] = types.Map{rio.HeaderContentType: "text/xml"}
// 	}

// 	// Assert that can submit raw html text from Go client
// 	html, err := os.ReadFile("../../testdata/html.html")
// 	require.NoError(t, err)

// 	htmlStub := rio.NewStub().
// 		For("GET", rio.Contains("animal/create_html")).
// 		WillReturn(rio.NewResponse().WithBody(rio.ContentTypeHTML, html))
// 	htmltParams := types.Map{"stubs": []*rio.Stub{htmlStub}}

// 	testCases := []*TestCase{
// 		NewTestCase("missing_required_params", http.MethodPost, "/stub/create_many", "", types.Map{}, http.StatusBadRequest, netkit.VerdictMissingParameters),
// 		NewTestCase("success", http.MethodPost, "/stub/create_many", "", validParams, http.StatusOK, netkit.VerdictSuccess),
// 		NewTestCase("success_response_json", http.MethodPost, "/stub/create_many", "", jsonResponseParams, http.StatusOK, netkit.VerdictSuccess),
// 		NewTestCase("success_response_xml", http.MethodPost, "/stub/create_many", "", xmlResponseParams, http.StatusOK, netkit.VerdictSuccess),
// 		NewTestCase("success_response_raw_html", http.MethodPost, "/stub/create_many", "", htmltParams, http.StatusOK, netkit.VerdictSuccess),
// 	}

// 	app, err := NewApp(context.Background(), config.NewConfig())
// 	require.NoError(t, err)

// 	for i := range testCases {
// 		tc := testCases[i]

// 		t.Run(tc.Name, func(t *testing.T) {
// 			t.Parallel()

// 			body := tc.Execute(t, app.kit)
// 			if tc.ExpectStatus == http.StatusOK {
// 				gotStubs, _ := body.ForceMap("data").GetArrayMap("stubs")
// 				require.NotEmpty(t, gotStubs)

// 				for i, m := range gotStubs {
// 					require.NotZero(t, m.ForceInt64("id"))
// 					require.NotEmpty(t, m)

// 					if tc.Name == "success" {
// 						actualStub := rio.Stub{}
// 						require.NoError(t, m.ToStruct(&actualStub))
// 						require.Equal(t, stub.Description, actualStub.Description)
// 					} else if tc.Name == "success_response_json" {
// 						inputStubs, _ := jsonResponseParams.GetArrayMap("stubs")
// 						require.Equal(t, inputStubs[i].ForceMap("response").ForceMap("body"), m.ForceMap("response").ForceMap("body"))
// 					} else if tc.Name == "success_response_xml" {
// 						actual, err := base64.StdEncoding.DecodeString(m.ForceMap("response").ForceString("body"))
// 						require.NoError(t, err)
// 						require.Equal(t, string(xmlBody), string(actual))
// 					} else if tc.Name == "success_response_raw_html" {
// 						actual, err := base64.StdEncoding.DecodeString(m.ForceMap("response").ForceString("body"))
// 						require.NoError(t, err)
// 						require.Equal(t, string(html), string(actual))
// 					}
// 				}
// 			}
// 		})
// 	}
// }

// func TestCreateStubWithYaml(t *testing.T) {
// 	t.Parallel()

// 	ctx := context.Background()
// 	fileData, err := os.ReadFile("../../testdata/stubs.yaml")
// 	require.NoError(t, err)
// 	require.NotEmpty(t, fileData)

// 	app, err := NewApp(ctx, config.NewConfig())
// 	require.NoError(t, err)

// 	w := ginkit.NewResponseRecorder()
// 	req, err := http.NewRequestWithContext(ctx, "POST", "/stub/create_many", bytes.NewReader(fileData))
// 	require.NoError(t, err)
// 	req.Header.Add(netkit.HeaderContentType, "application/x-yaml")
// 	app.kit.ServeHTTP(w, req)

// 	result := w.Result() //nolint:bodyclose
// 	require.Equal(t, 200, result.StatusCode)

// 	resData := types.Map{}
// 	require.NoError(t, httpkit.ParseResponse(ctx, result, &resData))
// 	require.Equal(t, netkit.VerdictSuccess, resData.ForceString("verdict"))
// }

// func TestGetStubs(t *testing.T) {
// 	t.Parallel()

// 	ctx := context.Background()
// 	cfg := config.NewConfig()

// 	app, err := NewApp(ctx, cfg)
// 	require.NoError(t, err)

// 	namespace := uuid.NewString()
// 	stub := rio.NewStub().
// 		WithNamespace(namespace).
// 		For("GET", rio.Contains("animal/create")).
// 		WithHeader("X-REQUEST-ID", rio.EqualTo(uuid.NewString())).
// 		WithQuery("search_term", rio.EqualTo(uuid.NewString())).
// 		WithCookie("SESSION_ID", rio.EqualTo(uuid.NewString())).
// 		WillReturn(rio.NewResponse().WithBody(rio.MapToJSON(types.Map{"data": uuid.NewString()})))

// 	err = app.stubStore.Create(ctx, stub)
// 	require.NoError(t, err)

// 	validParams := types.Map{"namespace": namespace}
// 	testCases := []*httpkit.TestCase{
// 		httpkit.NewTestCase("success", http.MethodGet, "/stub/list", "", validParams, http.StatusOK, netkit.VerdictSuccess),
// 	}

// 	for i := range testCases {
// 		tc := testCases[i]

// 		t.Run(tc.Name, func(t *testing.T) {
// 			t.Parallel()

// 			body := ginkit.ExecuteHTTPTestCase(t, tc, app.kit)
// 			if tc.ExpectStatus == http.StatusOK {
// 				arr, _ := body.ForceMap("data").GetArrayMap("stubs")
// 				require.Len(t, arr, 1)

// 				for _, m := range arr {
// 					stub := rio.Stub{}
// 					require.NoError(t, m.ToStruct(&stub))
// 				}
// 			}
// 		})
// 	}
// }

// func TestUploadFile(t *testing.T) {
// 	t.Parallel()

// 	ctx := context.Background()

// 	dir, err := os.MkdirTemp("", "uploaded_files")
// 	require.NoError(t, err)
// 	defer os.RemoveAll(dir)

// 	cfg := config.NewConfig()
// 	cfg.FileStorage = rio.LocalStorageConfig{StoragePath: dir}

// 	app, err := NewApp(ctx, cfg)
// 	require.NoError(t, err)

// 	w := httptest.NewRecorder()
// 	fileID := uuid.NewString()

// 	fileContents, err := os.ReadFile("stub_handler_test.go")
// 	require.NoError(t, err)

// 	req, err := rio.NewUploadRequest(ctx, "/stub/upload", fileContents, map[string]string{"file_id": fileID})
// 	require.NoError(t, err)

// 	app.kit.ServeHTTP(w, req)

// 	require.Equal(t, http.StatusOK, w.Code)
// 	resData := parseResponse(t, w.Body)

// 	returnedFileID := resData.ForceMap("data").ForceString("file_id")
// 	require.NotEmpty(t, returnedFileID)
// 	require.Equal(t, fileID, returnedFileID)

// 	downloadReader, err := app.fileStorage.DownloadFile(ctx, returnedFileID)
// 	require.NoError(t, err)
// 	downloadBytes, err := io.ReadAll(downloadReader)
// 	require.NoError(t, err)
// 	require.Equal(t, fileContents, downloadBytes)
// }

// func TestUploadProtos(t *testing.T) {
// 	t.Parallel()

// 	ctx := context.Background()

// 	dir, err := os.MkdirTemp("", "uploaded_protos")
// 	require.NoError(t, err)
// 	defer os.RemoveAll(dir)
// 	defer os.RemoveAll("cached_grpc_protos")

// 	cfg := config.NewConfig()
// 	cfg.FileStorage = rio.LocalStorageConfig{StoragePath: dir}

// 	app, err := NewApp(ctx, cfg)
// 	require.NoError(t, err)

// 	w := httptest.NewRecorder()

// 	fileContents, err := os.ReadFile("../../testdata/offer_proto")
// 	require.NoError(t, err)

// 	req, err := rio.NewUploadRequest(ctx, "/proto/upload", fileContents, map[string]string{"name": "offer proto"})
// 	require.NoError(t, err)

// 	app.kit.ServeHTTP(w, req)

// 	require.Equal(t, http.StatusOK, w.Code)
// 	resData := parseResponse(t, w.Body)

// 	returnedProto := resData.ForceMap("data").ForceMap("proto")
// 	require.NotEmpty(t, returnedProto)
// 	createdProtos, err := app.stubStore.GetProtos(ctx)
// 	require.NoError(t, err)
// 	require.NotEmpty(t, createdProtos)

// 	proto := findProtoByID(createdProtos, returnedProto.ForceInt64("id"))
// 	require.NotNil(t, proto)
// 	require.Equal(t, []string{"/offers.v1.OfferService/ValidateOffer"}, proto.Methods)

// 	downloadReader, err := app.fileStorage.DownloadFile(ctx, proto.FileID)
// 	require.NoError(t, err)
// 	downloadBytes, err := io.ReadAll(downloadReader)
// 	require.NoError(t, err)
// 	require.Equal(t, fileContents, downloadBytes)
// }

// func TestEchoHandler(t *testing.T) {
// 	t.Parallel()

// 	ctx := context.Background()
// 	expectedData := types.Map{
// 		"data":    uuid.NewString(),
// 		"verdict": netkit.VerdictSuccess,
// 	}

// 	stub := rio.NewStub().ForAny(rio.Contains("admin/animal")).WillReturn(rio.NewResponse().WithBody(rio.MapToJSON(expectedData)))
// 	stubWithNS := rio.NewStub().ForAny(rio.Contains("phone_owner/verify")).WithNamespace("dop").WillReturn(rio.NewResponse().WithBody(rio.MapToJSON(expectedData)))

// 	stubStore, err := database.NewStubDBStore(ctx, config.NewDBConfig())
// 	require.NoError(t, err)
// 	require.NoError(t, stubStore.Create(ctx, stub, stubWithNS))

// 	validParams := types.Map{"name": uuid.NewString()}
// 	testCases := []*httpkit.TestCase{
// 		httpkit.NewTestCase("success_post", http.MethodPost, "/echo/admin/animal/create", "", validParams, http.StatusOK, netkit.VerdictSuccess),
// 		httpkit.NewTestCase("success_get", http.MethodGet, "/echo/admin/animal/get", "", types.Map{}, http.StatusOK, netkit.VerdictSuccess),
// 		httpkit.NewTestCase("success_namespace", http.MethodGet, "/dop/echo/phone_owner/verify", "", types.Map{}, http.StatusOK, netkit.VerdictSuccess),
// 	}

// 	app, err := NewApp(ctx, config.NewConfig())
// 	require.NoError(t, err)

// 	for i := range testCases {
// 		tc := testCases[i]

// 		t.Run(tc.Name, func(t *testing.T) {
// 			t.Parallel()

// 			body := ginkit.ExecuteHTTPTestCase(t, tc, app.kit)
// 			require.Equal(t, expectedData["data"], body["data"])
// 			require.Equal(t, expectedData["verdict"], body["verdict"])
// 		})
// 	}
// }

// func TestEchoHandler_Reverse(t *testing.T) {
// 	t.Parallel()

// 	ctx := context.Background()
// 	expectedData := types.Map{
// 		"data":    uuid.NewString(),
// 		"verdict": netkit.VerdictSuccess,
// 	}

// 	targetServer := rio.NewLocalServerWithReporter(t)
// 	targetURL := targetServer.GetURL(ctx) + "/proxy_server"
// 	require.NoError(t, rio.NewStub().
// 		For("POST", rio.Contains("proxy_server/reverse/animal/create")).
// 		WillReturn(rio.NewResponse().WithBody(rio.MustStructToJSON(expectedData))).
// 		Send(ctx, targetServer))

// 	stubStore, err := database.NewStubDBStore(ctx, config.NewDBConfig())
// 	require.NoError(t, err)
// 	require.NoError(t, stubStore.Create(ctx, rio.NewStub().ForAny(rio.Contains("reverse/animal/create")).WithTargetURL(targetURL)))

// 	validParams := types.Map{"name": uuid.NewString()}
// 	testCases := []*httpkit.TestCase{
// 		httpkit.NewTestCase("success", http.MethodPost, "/echo/reverse/animal/create", "", validParams, http.StatusOK, netkit.VerdictSuccess),
// 	}

// 	app, err := NewApp(ctx, config.NewConfig())
// 	require.NoError(t, err)

// 	for i := range testCases {
// 		tc := testCases[i]

// 		t.Run(tc.Name, func(t *testing.T) {
// 			t.Parallel()

// 			body := ginkit.ExecuteHTTPTestCase(t, tc, app.kit)
// 			require.Equal(t, expectedData["data"], body["data"])
// 			require.Equal(t, expectedData["verdict"], body["verdict"])
// 		})
// 	}
// }

// func TestEchoHandler_Reverse_Recording(t *testing.T) {
// 	t.Parallel()

// 	ctx := context.Background()
// 	expectedData := types.Map{
// 		"data":    uuid.NewString(),
// 		"verdict": netkit.VerdictSuccess,
// 	}

// 	targetServer := rio.NewLocalServerWithReporter(t)
// 	targetURL := targetServer.GetURL(ctx) + "/proxy_server"
// 	require.NoError(t, rio.NewStub().
// 		For("POST", rio.Contains("proxy_server/reverse_recording/animal/create")).
// 		WillReturn(rio.NewResponse().WithBody(rio.MustStructToJSON(expectedData))).
// 		Send(ctx, targetServer))

// 	stubStore, err := database.NewStubDBStore(ctx, config.NewDBConfig())
// 	require.NoError(t, err)

// 	proxyStubs := rio.NewStub().
// 		ForAny(rio.Contains("reverse_recording/animal/create")).
// 		WithTargetURL(targetURL).
// 		WithEnableRecord(true)
// 	require.NoError(t, stubStore.Create(ctx, proxyStubs))

// 	validParams := types.Map{"name": uuid.NewString()}
// 	testCases := []*httpkit.TestCase{
// 		httpkit.NewTestCase("success", http.MethodPost, "/echo/reverse_recording/animal/create", "", validParams, http.StatusOK, netkit.VerdictSuccess),
// 	}

// 	app, err := NewApp(ctx, config.NewConfig())
// 	require.NoError(t, err)

// 	for i := range testCases {
// 		tc := testCases[i]

// 		t.Run(tc.Name, func(t *testing.T) {
// 			t.Parallel()

// 			body := ginkit.ExecuteHTTPTestCase(t, tc, app.kit)
// 			require.Equal(t, expectedData["data"], body["data"])
// 			require.Equal(t, expectedData["verdict"], body["verdict"])
// 		})
// 	}
// }

// func TestGetIncomingRequest(t *testing.T) {
// 	t.Parallel()

// 	ctx := context.Background()
// 	store, err := database.NewStubDBStore(ctx, config.NewDBConfig())
// 	require.NoError(t, err)

// 	requestBody, err := os.ReadFile("stub_handler_test.go")
// 	require.NoError(t, err)

// 	request := &rio.IncomingRequest{
// 		URL:    uuid.NewString(),
// 		Method: "GET",
// 		Header: types.Map{
// 			"key": uuid.NewString(),
// 		},
// 		CURL:   uuid.NewString(),
// 		Body:   requestBody,
// 		StubID: 1,
// 	}

// 	err = store.CreateIncomingRequest(ctx, request)
// 	require.NoError(t, err)

// 	validParams := types.Map{"ids": []int64{request.ID}}
// 	testCases := []*httpkit.TestCase{
// 		httpkit.NewTestCase("success", http.MethodPost, "/incoming_request/list", "", validParams, http.StatusOK, netkit.VerdictSuccess),
// 	}

// 	app, err := NewApp(ctx, config.NewConfig())
// 	require.NoError(t, err)

// 	for i := range testCases {
// 		tc := testCases[i]

// 		t.Run(tc.Name, func(t *testing.T) {
// 			t.Parallel()

// 			data := struct {
// 				Requests []*rio.IncomingRequest `json:"requests" yaml:"requests"`
// 			}{}

// 			err := ginkit.ExecuteHTTPTestCase(t, tc, app.kit).ForceMap("data").ToStruct(&data)
// 			require.NoError(t, err)
// 			require.Len(t, data.Requests, 1)
// 			require.Equal(t, request.ID, data.Requests[0].ID)
// 			require.Equal(t, requestBody, data.Requests[0].Body)
// 		})
// 	}
// }

// func findProtoByID(protos []*rio.Proto, id int64) *rio.Proto {
// 	for _, p := range protos {
// 		if p.ID == id {
// 			return p
// 		}
// 	}

// 	return nil
// }
