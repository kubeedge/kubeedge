package handlerfactory

import (
    "bytes"
    "encoding/json"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apiserver/pkg/endpoints/request"

    "github.com/kubeedge/api/apis/devices/v1beta1"
    beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
)

func TestFactory_Get(t *testing.T) {
    f := NewFactory()

    tests := []struct {
        name           string
        setupHandler   bool
        expectedCalls  int
    }{
        {
            name:          "Get existing handler",
            setupHandler:  true,
            expectedCalls: 1,
        },
        {
            name:          "Get new handler",
            setupHandler:  false,
            expectedCalls: 1,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.setupHandler {
                f.Get() // Setup handler first
            }

            handler := f.Get()
            assert.NotNil(t, handler)
        })
    }
}

func TestFactory_List(t *testing.T) {
    f := NewFactory()

    tests := []struct {
        name           string
        setupHandler   bool
        expectedCalls  int
    }{
        {
            name:          "List existing handler",
            setupHandler:  true,
            expectedCalls: 1,
        },
        {
            name:          "List new handler",
            setupHandler:  false,
            expectedCalls: 1,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.setupHandler {
                f.List() // Setup handler first
            }

            handler := f.List()
            assert.NotNil(t, handler)
        })
    }
}

func TestFactory_Create(t *testing.T) {
    f := NewFactory()
    req := &request.RequestInfo{
        APIGroup:   "test",
        APIVersion: "v1",
        Resource:   "tests",
    }

    handler := f.Create(req)
    assert.NotNil(t, handler)
}

func TestFactory_Delete(t *testing.T) {
    f := NewFactory()

    tests := []struct {
        name           string
        setupHandler   bool
        expectedCalls  int
    }{
        {
            name:          "Delete existing handler",
            setupHandler:  true,
            expectedCalls: 1,
        },
        {
            name:          "Delete new handler",
            setupHandler:  false,
            expectedCalls: 1,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.setupHandler {
                f.Delete() // Setup handler first
            }

            handler := f.Delete()
            assert.NotNil(t, handler)
        })
    }
}

func TestFactory_Update(t *testing.T) {
    f := NewFactory()

    tests := []struct {
        name      string
        reqInfo   *request.RequestInfo
        isDevice  bool
    }{
        {
            name: "Update normal resource",
            reqInfo: &request.RequestInfo{
                APIGroup:   "test",
                APIVersion: "v1",
                Resource:   "tests",
            },
            isDevice: false,
        },
        {
            name: "Update device resource",
            reqInfo: &request.RequestInfo{
                Resource: "devices",
            },
            isDevice: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            handler := f.Update(tt.reqInfo)
            assert.NotNil(t, handler)

            if tt.isDevice {
                // Test device update handler
                device := &v1beta1.Device{
                    ObjectMeta: metav1.ObjectMeta{
                        Name:      "test-device",
                        Namespace: "default",
                    },
                }
                deviceJSON, _ := json.Marshal(device)
                
                req := httptest.NewRequest("PUT", "/api/v1/devices/test-device", bytes.NewReader(deviceJSON))
                w := httptest.NewRecorder()

                beehiveContext.InitContext([]string{"channel"})
                handler.ServeHTTP(w, req)
            }
        })
    }
}

func TestFactory_Patch(t *testing.T) {
    f := NewFactory()
    reqInfo := &request.RequestInfo{
        APIGroup:   "test",
        APIVersion: "v1",
        Resource:   "tests",
    }

    handler := f.Patch(reqInfo)
    assert.NotNil(t, handler)

    tests := []struct {
        name        string
        contentType string
        body        []byte
        timeout     string
    }{
        {
            name:        "JSON Patch",
            contentType: "application/json-patch+json",
            body:        []byte(`[{"op": "replace", "path": "/spec/replicas", "value": 2}]`),
            timeout:     "30s",
        },
        {
            name:        "Strategic Merge Patch",
            contentType: "application/strategic-merge-patch+json",
            body:        []byte(`{"spec":{"replicas":2}}`),
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            url := "/api/v1/tests/test"
            if tt.timeout != "" {
                url += "?timeout=" + tt.timeout
            }
            
            req := httptest.NewRequest("PATCH", url, bytes.NewReader(tt.body))
            req.Header.Set("Content-Type", tt.contentType)
            w := httptest.NewRecorder()

            handler.ServeHTTP(w, req)
        })
    }
}

func Test_parseTimeout(t *testing.T) {
    tests := []struct {
        name     string
        timeout  string
        expected time.Duration
    }{
        {
            name:     "Valid timeout",
            timeout:  "60s",
            expected: 60 * time.Second,
        },
        {
            name:     "Invalid timeout",
            timeout:  "invalid",
            expected: 34 * time.Second,
        },
        {
            name:     "Empty timeout",
            timeout:  "",
            expected: 34 * time.Second,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := parseTimeout(tt.timeout)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func Test_limitedReadBody(t *testing.T) {
    tests := []struct {
        name      string
        body      string
        limit     int64
        wantError bool
    }{
        {
            name:      "Under limit",
            body:      "test data",
            limit:     100,
            wantError: false,
        },
        {
            name:      "Over limit",
            body:      "test data",
            limit:     4,
            wantError: true,
        },
        {
            name:      "No limit",
            body:      "test data",
            limit:     0,
            wantError: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest("POST", "/", bytes.NewBufferString(tt.body))
            data, err := limitedReadBody(req, tt.limit)
            
            if tt.wantError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.body, string(data))
            }
        })
    }
}