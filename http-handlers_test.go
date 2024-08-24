package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/jhtohru/go-album-catalog/internal/random"
)

func TestRequest(t *testing.T) {
	problemsWant := map[string]string{
		"title":  "is empty",
		"artist": "is empty",
		"price":  "is not greater than zero",
	}
	var req request
	assert.Equal(t, problemsWant, req.Valid())
}

func TestCreateAlbumHandler(t *testing.T) {
	type testCase struct {
		requestBody      string
		validateProblems map[string]string
		newID            uuid.UUID
		now              time.Time
		insertErr        error
		statusCodeWant   int
		responseBodyWant string
		logSubstrsWant   []string
	}
	tests := map[string]testCase{
		"malformed request body": {
			requestBody: "", // malformed request body

			statusCodeWant:   http.StatusBadRequest,
			responseBodyWant: `{"message": "malformed request body"}`,
		},
		"invalid request body": {
			requestBody: "{}",
			validateProblems: map[string]string{
				"title":  "is empty",
				"artist": "is empty",
				"price":  "is not greater than zero",
			},

			statusCodeWant: http.StatusBadRequest,
			responseBodyWant: `
				{
					"message": "invalid request body",
					"problems": {
						"title":  "is empty",
						"artist": "is empty",
						"price":  "is not greater than zero"
					}
				}`,
		},
		"unexpected insert error": {
			requestBody: "{}",
			insertErr:   fmt.Errorf("unexpected insert error"),

			statusCodeWant:   http.StatusInternalServerError,
			responseBodyWant: `{"message": "internal error"}`,
			logSubstrsWant: []string{
				`level=ERROR`,
				`msg="inserting album into the storage"`,
				`error="unexpected insert error"`,
			},
		},
		"happy path": func() testCase {
			newID := uuid.New()
			now := random.Time()
			return testCase{
				requestBody: `
					{
						"title":  "Anathema",
						"artist": "Judgement",
						"price":  1234
					}`,
				newID: newID,
				now:   now,

				statusCodeWant: http.StatusCreated,
				responseBodyWant: `
					{
						"id":         "` + newID.String() + `",
						"title":      "Anathema",
						"artist":     "Judgement",
						"price":      1234,
						"created_at": "` + now.Format(time.RFC3339Nano) + `",
						"updated_at": "` + now.Format(time.RFC3339Nano) + `"
					}`,
			}
		}(),
	}
	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			storage := &storageSpy{}
			storage.insert = func(ctx context.Context, alb Album) error {
				return test.insertErr
			}
			logsBuf := bytes.NewBuffer(nil)
			logger := slog.New(slog.NewTextHandler(logsBuf, nil))
			validate := func(Validator) map[string]string {
				return test.validateProblems
			}
			newID := func() uuid.UUID {
				return test.newID
			}
			timeNow := func() time.Time {
				return test.now
			}
			handler := createAlbumHandler(
				storage,
				logger,
				validate,
				newID,
				timeNow,
			)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("", "/", strings.NewReader(test.requestBody))

			handler.ServeHTTP(rec, req)

			assert.Equal(t, test.statusCodeWant, rec.Result().StatusCode)
			assert.Equal(t, rec.Header().Get("Content-Type"), "application/json; charset=utf-8")
			assert.JSONEq(t, test.responseBodyWant, rec.Body.String())

			logs := logsBuf.String()

			for _, substr := range test.logSubstrsWant {
				assert.Contains(t, logs, substr)
			}
		})
	}
}

func TestListAlbumsHandler(t *testing.T) {
	type testCase struct {
		urlValues        url.Values
		offsetWant       int
		limitWant        int
		findAllAlbs      []Album
		findAllErr       error
		statusCodeWant   int
		responseBodyWant string
		logSubstrsWant   []string
	}
	tests := map[string]testCase{
		"missing page_size": {
			urlValues: url.Values{
				// missing "page_size"
				"page_number": []string{"1"},
			},
			statusCodeWant:   http.StatusBadRequest,
			responseBodyWant: `{"message": "query parameter page_size is missing"}`,
		},
		"malformed page size": {
			urlValues: url.Values{
				"page_size":   []string{""}, // malformed page size
				"page_number": []string{"1"},
			},
			statusCodeWant:   http.StatusBadRequest,
			responseBodyWant: `{"message": "page size is not a valid number"}`,
		},
		"missing page_number": {
			urlValues: url.Values{
				"page_size": []string{"1"},
				// missing "page_number"
			},

			statusCodeWant:   http.StatusBadRequest,
			responseBodyWant: `{"message": "query parameter page_number is missing"}`,
		},
		"malformed page number": {
			urlValues: url.Values{
				"page_size":   []string{"1"},
				"page_number": []string{""}, // malformed page number
			},

			statusCodeWant:   http.StatusBadRequest,
			responseBodyWant: `{"message": "page number is not a valid number"}`,
		},
		"page size is too small": {
			urlValues: url.Values{
				"page_size":   []string{"0"}, // invalid page size
				"page_number": []string{"1"},
			},

			statusCodeWant:   http.StatusBadRequest,
			responseBodyWant: `{"message": "page size is less than 1"}`,
		},
		"page size is too big": {
			urlValues: url.Values{
				"page_size":   []string{"100"}, // invalid page size
				"page_number": []string{"1"},
			},

			statusCodeWant:   http.StatusBadRequest,
			responseBodyWant: `{"message": "page size is greater than 50"}`,
		},
		"page number is too small": {
			urlValues: url.Values{
				"page_size":   []string{"1"},
				"page_number": []string{"0"}, // invalid page number
			},

			statusCodeWant:   http.StatusBadRequest,
			responseBodyWant: `{"message": "page number is less than 1"}`,
		},
		"unexpected find error": {
			urlValues: url.Values{
				"page_size":   []string{"10"},
				"page_number": []string{"3"},
			},
			offsetWant: 20,
			limitWant:  10,
			findAllErr: fmt.Errorf("unexpected find error"),

			statusCodeWant:   http.StatusInternalServerError,
			responseBodyWant: `{"message": "internal error"}`,
			logSubstrsWant: []string{
				"level=ERROR",
				`msg="finding albums in the storage"`,
				`error="unexpected find error"`,
			},
		},
		"no results": {
			urlValues: url.Values{
				"page_size":   []string{"10"},
				"page_number": []string{"3"},
			},
			offsetWant: 20,
			limitWant:  10,

			findAllErr:       ErrAlbumNotFound,
			statusCodeWant:   http.StatusOK,
			responseBodyWant: "[]",
		},
		"happy path": func() testCase {
			albs := randomAlbums(10)
			bodyWantBytes, _ := json.Marshal(albs)
			return testCase{
				urlValues: url.Values{
					"page_size":   []string{"10"},
					"page_number": []string{"3"},
				},
				offsetWant:  20,
				limitWant:   10,
				findAllAlbs: albs,

				statusCodeWant:   http.StatusOK,
				responseBodyWant: string(bodyWantBytes),
			}
		}(),
	}
	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			storageSpy := &storageSpy{}
			storageSpy.findAll = func(ctx context.Context, offset, limit int) ([]Album, error) {
				assert.Equal(t, test.offsetWant, offset)
				assert.Equal(t, test.limitWant, limit)
				return test.findAllAlbs, test.findAllErr
			}
			logsBuf := bytes.NewBuffer(nil)
			logger := slog.New(slog.NewTextHandler(logsBuf, nil))
			handler := listAlbumsHandler(storageSpy, logger)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("", "/?"+test.urlValues.Encode(), nil)

			handler.ServeHTTP(rec, req)

			assert.Equal(t, test.statusCodeWant, rec.Result().StatusCode)
			assert.Equal(t, rec.Header().Get("Content-Type"), "application/json; charset=utf-8")
			assert.JSONEq(t, test.responseBodyWant, rec.Body.String())

			logs := logsBuf.String()

			for _, substr := range test.logSubstrsWant {
				assert.Contains(t, logs, substr)
			}
		})
	}
}

func TestGetAlbumHandler(t *testing.T) {
	type testCase struct {
		albumID          string
		findOneAlb       Album
		findOneErr       error
		statusCodeWant   int
		responseBodyWant string
		logSubstrsWant   []string
	}
	tests := map[string]testCase{
		"malformed album id": {
			albumID: "", // malformed album id

			statusCodeWant:   http.StatusBadRequest,
			responseBodyWant: `{"message": "malformed album id"}`,
		},
		"album not found": {
			albumID:    "00000000-0000-0000-0000-000000000000",
			findOneErr: ErrAlbumNotFound,

			statusCodeWant:   http.StatusNotFound,
			responseBodyWant: `{"message": "album not found"}`,
		},
		"unexpected find error": {
			albumID:    "00000000-0000-0000-0000-000000000000",
			findOneErr: fmt.Errorf("unexpected find error"),

			statusCodeWant:   http.StatusInternalServerError,
			responseBodyWant: `{"message": "internal error"}`,
			logSubstrsWant: []string{
				`level=ERROR`,
				`msg="finding one album in the storage"`,
				`error="unexpected find error"`,
			},
		},
		"happy path": func() testCase {
			alb := randomAlbum()
			bodyWantBytes, _ := json.Marshal(alb)
			return testCase{
				albumID:    "00000000-0000-0000-0000-000000000000",
				findOneAlb: alb,

				statusCodeWant:   http.StatusOK,
				responseBodyWant: string(bodyWantBytes),
			}
		}(),
	}
	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			storage := &storageSpy{}
			storage.findOne = func(ctx context.Context, id uuid.UUID) (Album, error) {
				return test.findOneAlb, test.findOneErr
			}
			logsBuf := bytes.NewBuffer(nil)
			logger := slog.New(slog.NewTextHandler(logsBuf, nil))
			handler := getAlbumHandler(storage, logger)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("", "/", nil)
			req.SetPathValue("album_id", test.albumID)

			handler.ServeHTTP(rec, req)

			assert.Equal(t, test.statusCodeWant, rec.Result().StatusCode)
			assert.Equal(t, rec.Header().Get("Content-Type"), "application/json; charset=utf-8")
			assert.JSONEq(t, test.responseBodyWant, rec.Body.String())

			logs := logsBuf.String()

			for _, substr := range test.logSubstrsWant {
				assert.Contains(t, logs, substr)
			}
		})
	}
}

func TestUpdateAlbumHandler(t *testing.T) {
	type testCase struct {
		albumID          string
		requestBody      string
		validateProblems map[string]string
		now              time.Time
		findOneAlb       Album
		findOneErr       error
		updateErr        error
		statusCodeWant   int
		responseBodyWant string
		logSubstrsWant   []string
	}
	tests := map[string]testCase{
		"malformed album id": {
			albumID:     "", // malformed album id
			requestBody: "00000000-0000-0000-0000-000000000000",

			statusCodeWant:   http.StatusBadRequest,
			responseBodyWant: `{"message": "malformed album id"}`,
		},
		"malformed request body": {
			albumID:     "00000000-0000-0000-0000-000000000000",
			requestBody: "", // malformed request body

			statusCodeWant:   http.StatusBadRequest,
			responseBodyWant: `{"message": "malformed request body"}`,
		},
		"invalid request body": {
			albumID:     "00000000-0000-0000-0000-000000000000",
			requestBody: "{}",
			validateProblems: map[string]string{
				"title":  "is empty",
				"artist": "is empty",
				"price":  "is not greater than zero",
			},

			statusCodeWant: http.StatusBadRequest,
			responseBodyWant: `{
				"message": "invalid request body",
				"problems": {
					"title":  "is empty",
					"artist": "is empty",
					"price":  "is not greater than zero"
				}
			}`,
		},
		"album not found": {
			albumID:     "00000000-0000-0000-0000-000000000000",
			requestBody: "{}",
			findOneErr:  ErrAlbumNotFound,

			statusCodeWant:   http.StatusNotFound,
			responseBodyWant: `{"message": "album not found"}`,
		},
		"unexpected find error": {
			albumID:     "00000000-0000-0000-0000-000000000000",
			requestBody: "{}",
			findOneErr:  fmt.Errorf("unexpected find error"),

			statusCodeWant:   http.StatusInternalServerError,
			responseBodyWant: `{"message": "internal error"}`,
			logSubstrsWant: []string{
				`level=ERROR`,
				`msg="finding one album in the storage"`,
				`error="unexpected find error"`,
			},
		},
		"album not found on update": {
			albumID:     "00000000-0000-0000-0000-000000000000",
			requestBody: "{}",
			updateErr:   ErrAlbumNotFound,

			statusCodeWant:   http.StatusNotFound,
			responseBodyWant: `{"message": "album not found"}`,
			logSubstrsWant:   nil,
		},
		"unexpected update error": {
			albumID:     "00000000-0000-0000-0000-000000000000",
			requestBody: "{}",
			updateErr:   fmt.Errorf("unexpected update error"),

			statusCodeWant:   http.StatusInternalServerError,
			responseBodyWant: `{"message": "internal error"}`,
			logSubstrsWant: []string{
				`level=ERROR`,
				`msg="updating album in the storage"`,
				`error="unexpected update error"`,
			},
		},
		"happy path": func() testCase {
			now := random.Time()
			alb := randomAlbum()
			return testCase{
				albumID: "00000000-0000-0000-0000-000000000000",
				requestBody: `
					{
						"title":  "Babylon By Gus Vol.1 - O Ano do Macaco",
						"artist": "Black Alien",
						"price":  12345
					}`,
				now:        now,
				findOneAlb: alb,

				statusCodeWant: http.StatusOK,
				responseBodyWant: `
					{
						"id":         "` + alb.ID.String() + `",
						"title":      "Babylon By Gus Vol.1 - O Ano do Macaco",
						"artist":     "Black Alien",
						"price":      12345,
						"created_at": "` + alb.CreatedAt.Format(time.RFC3339Nano) + `",
						"updated_at": "` + now.Format(time.RFC3339Nano) + `"
					}`,
			}
		}(),
	}
	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			storage := &storageSpy{}
			storage.findOne = func(ctx context.Context, id uuid.UUID) (Album, error) {
				return test.findOneAlb, test.findOneErr
			}
			storage.update = func(context.Context, Album) error {
				return test.updateErr
			}
			logsBuf := bytes.NewBuffer(nil)
			logger := slog.New(slog.NewTextHandler(logsBuf, nil))
			validate := func(Validator) map[string]string {
				return test.validateProblems
			}
			timeNow := func() time.Time {
				return test.now
			}
			handler := updateAlbumHandler(
				storage,
				logger,
				validate,
				timeNow,
			)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("", "/", strings.NewReader(test.requestBody))
			req.SetPathValue("album_id", test.albumID)

			handler.ServeHTTP(rec, req)

			assert.Equal(t, test.statusCodeWant, rec.Result().StatusCode)
			assert.Equal(t, rec.Header().Get("Content-Type"), "application/json; charset=utf-8")
			assert.JSONEq(t, test.responseBodyWant, rec.Body.String())

			logs := logsBuf.String()

			for _, substr := range test.logSubstrsWant {
				assert.Contains(t, logs, substr)
			}
		})
	}
}

func TestDeleteAlbumHandler(t *testing.T) {
	type testCase struct {
		albumID          string
		findOneAlb       Album
		findOneErr       error
		removeErr        error
		statusCodeWant   int
		responseBodyWant string
		logSubstrsWant   []string
	}
	tests := map[string]testCase{
		"malformed album id": {
			albumID: "", // malformed album id

			statusCodeWant:   http.StatusBadRequest,
			responseBodyWant: `{"message":"malformed album id"}`,
		},
		"album not found": {
			albumID:    "00000000-0000-0000-0000-000000000000",
			findOneErr: ErrAlbumNotFound,

			statusCodeWant:   http.StatusNotFound,
			responseBodyWant: `{"message":"album not found"}`,
		},
		"unexpected find error": {
			albumID:        "00000000-0000-0000-0000-000000000000",
			findOneErr:     fmt.Errorf("unexpected find error"),
			statusCodeWant: http.StatusInternalServerError,

			responseBodyWant: `{"message":"internal error"}`,
			logSubstrsWant: []string{
				"level=ERROR",
				`msg="finding one album in the storage"`,
				`error="unexpected find error"`,
			},
		},
		"album not found on remove": {
			albumID:   "00000000-0000-0000-0000-000000000000",
			removeErr: ErrAlbumNotFound,

			statusCodeWant:   http.StatusNotFound,
			responseBodyWant: `{"message":"album not found"}`,
		},
		"unexpected remove error": {
			albumID:   "00000000-0000-0000-0000-000000000000",
			removeErr: fmt.Errorf("unexpected remove error"),

			statusCodeWant:   http.StatusInternalServerError,
			responseBodyWant: `{"message":"internal error"}`,
			logSubstrsWant: []string{
				"level=ERROR",
				`msg="removing album from the storage"`,
				`error="unexpected remove error"`,
			},
		},
		"happy path": func() testCase {
			alb := randomAlbum()
			bodyWantBytes, _ := json.Marshal(alb)
			return testCase{
				albumID:    "00000000-0000-0000-0000-000000000000",
				findOneAlb: alb,

				statusCodeWant:   http.StatusOK,
				responseBodyWant: string(bodyWantBytes),
			}
		}(),
	}
	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			storage := &storageSpy{}
			storage.findOne = func(ctx context.Context, id uuid.UUID) (Album, error) {
				return test.findOneAlb, test.findOneErr
			}
			storage.remove = func(ctx context.Context, id uuid.UUID) error {
				return test.removeErr
			}
			logsBuf := bytes.NewBuffer(nil)
			logger := slog.New(slog.NewTextHandler(logsBuf, nil))
			handler := deleteAlbumHandler(
				storage,
				logger,
			)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("", "/", nil)
			req.SetPathValue("album_id", test.albumID)

			handler.ServeHTTP(rec, req)

			assert.Equal(t, test.statusCodeWant, rec.Result().StatusCode)
			assert.Equal(t, rec.Header().Get("Content-Type"), "application/json; charset=utf-8")
			assert.JSONEq(t, test.responseBodyWant, rec.Body.String())

			logs := logsBuf.String()

			for _, substr := range test.logSubstrsWant {
				assert.Contains(t, logs, substr)
			}
		})
	}
}

type storageSpy struct {
	insert  func(ctx context.Context, alb Album) error
	findAll func(ctx context.Context, offset, limit int) ([]Album, error)
	findOne func(ctx context.Context, id uuid.UUID) (Album, error)
	update  func(ctx context.Context, alb Album) error
	remove  func(ctx context.Context, id uuid.UUID) error
}

func (spy *storageSpy) Insert(ctx context.Context, alb Album) error {
	return spy.insert(ctx, alb)
}

func (spy *storageSpy) FindAll(ctx context.Context, offset, limit int) ([]Album, error) {
	return spy.findAll(ctx, offset, limit)
}

func (spy *storageSpy) FindOne(ctx context.Context, id uuid.UUID) (Album, error) {
	return spy.findOne(ctx, id)
}

func (spy *storageSpy) Update(ctx context.Context, alb Album) error {
	return spy.update(ctx, alb)
}

func (spy *storageSpy) Remove(ctx context.Context, id uuid.UUID) error {
	return spy.remove(ctx, id)
}

// randomAlbum returns a randomly generated Album.
func randomAlbum() Album {
	return Album{
		ID:        uuid.New(),
		Title:     random.String(20 + rand.IntN(20)),
		Artist:    random.String(20 + rand.IntN(20)),
		Price:     rand.IntN(100000),
		CreatedAt: random.Time(),
		UpdatedAt: random.Time(),
	}
}

// randomAlbums returns a slice containing n randomly generated Albums.
func randomAlbums(n int) []Album {
	albs := make([]Album, n)
	for i := range albs {
		albs[i] = randomAlbum()
	}
	return albs
}
