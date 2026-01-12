package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	"github.com/dumeirei/smart-locker-backend/internal/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// 辅助函数：创建测试上下文
func createTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

// 辅助函数：创建带路径参数的测试上下文
func createTestContextWithParam(paramName, paramValue string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: paramName, Value: paramValue}}
	return c, w
}

// 辅助函数：创建带查询参数的测试上下文
func createTestContextWithQuery(query string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?"+query, nil)
	return c, w
}

// 辅助函数：创建已登录的测试上下文
func createAuthenticatedContext(userID int64) (*gin.Context, *httptest.ResponseRecorder) {
	c, w := createTestContext()
	c.Set(middleware.ContextKeyUserID, userID)
	return c, w
}

// 辅助函数：解析响应
func parseResponse(w *httptest.ResponseRecorder) response.Response {
	var resp response.Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp
}

// ============================================================================
// Phase 1: 错误处理测试
// ============================================================================

func TestHandleError_NilError(t *testing.T) {
	c, _ := createTestContext()

	handled := HandleError(c, nil)

	assert.False(t, handled, "nil error should not be handled")
}

func TestHandleError_AppError(t *testing.T) {
	c, w := createTestContext()
	appErr := errors.New(1001, "参数错误")

	handled := HandleError(c, appErr)

	assert.True(t, handled, "AppError should be handled")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseResponse(w)
	assert.Equal(t, 1001, resp.Code)
	assert.Equal(t, "参数错误", resp.Message)
}

func TestHandleError_GenericError(t *testing.T) {
	c, w := createTestContext()
	err := assert.AnError

	handled := HandleError(c, err)

	assert.True(t, handled, "generic error should be handled")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleErrorWithMessage_CustomMessage(t *testing.T) {
	c, w := createTestContext()
	err := assert.AnError

	handled := HandleErrorWithMessage(c, err, "操作失败")

	assert.True(t, handled)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	resp := parseResponse(w)
	assert.Equal(t, "操作失败", resp.Message)
}

func TestMustSucceed_Success(t *testing.T) {
	c, w := createTestContext()
	data := map[string]string{"key": "value"}

	MustSucceed(c, nil, data)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(w)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "success", resp.Message)
}

func TestMustSucceed_Error(t *testing.T) {
	c, w := createTestContext()
	appErr := errors.ErrNotFound

	MustSucceed(c, appErr, nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
	resp := parseResponse(w)
	assert.Equal(t, appErr.Code, resp.Code)
}

func TestMustSucceedPage_Success(t *testing.T) {
	c, w := createTestContext()
	list := []string{"item1", "item2"}

	MustSucceedPage(c, nil, list, 100, 1, 10)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(w)
	assert.Equal(t, 0, resp.Code)

	// 验证分页数据
	dataMap := resp.Data.(map[string]interface{})
	assert.Equal(t, float64(100), dataMap["total"])
	assert.Equal(t, float64(1), dataMap["page"])
	assert.Equal(t, float64(10), dataMap["page_size"])
}

// ============================================================================
// Phase 2: 认证检查测试
// ============================================================================

func TestRequireUserID_Authenticated(t *testing.T) {
	c, _ := createAuthenticatedContext(12345)

	userID, ok := RequireUserID(c)

	assert.True(t, ok)
	assert.Equal(t, int64(12345), userID)
}

func TestRequireUserID_NotAuthenticated(t *testing.T) {
	c, w := createTestContext()

	userID, ok := RequireUserID(c)

	assert.False(t, ok)
	assert.Equal(t, int64(0), userID)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	resp := parseResponse(w)
	assert.Equal(t, "请先登录", resp.Message)
}

func TestRequireAdminID_Authenticated(t *testing.T) {
	c, _ := createAuthenticatedContext(99999)

	adminID, ok := RequireAdminID(c)

	assert.True(t, ok)
	assert.Equal(t, int64(99999), adminID)
}

func TestGetOptionalUserID_Authenticated(t *testing.T) {
	c, _ := createAuthenticatedContext(12345)

	userID := GetOptionalUserID(c)

	assert.Equal(t, int64(12345), userID)
}

func TestGetOptionalUserID_NotAuthenticated(t *testing.T) {
	c, _ := createTestContext()

	userID := GetOptionalUserID(c)

	assert.Equal(t, int64(0), userID)
}

// ============================================================================
// Phase 3: ID 解析测试
// ============================================================================

func TestParseID_Valid(t *testing.T) {
	c, _ := createTestContextWithParam("id", "12345")

	id, ok := ParseID(c, "订单")

	assert.True(t, ok)
	assert.Equal(t, int64(12345), id)
}

func TestParseID_Invalid(t *testing.T) {
	c, w := createTestContextWithParam("id", "invalid")

	id, ok := ParseID(c, "订单")

	assert.False(t, ok)
	assert.Equal(t, int64(0), id)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseResponse(w)
	assert.Equal(t, "无效的订单ID", resp.Message)
}

func TestParseParamID_Valid(t *testing.T) {
	c, _ := createTestContextWithParam("hotel_id", "999")

	id, ok := ParseParamID(c, "hotel_id", "酒店")

	assert.True(t, ok)
	assert.Equal(t, int64(999), id)
}

func TestParseQueryID_Empty(t *testing.T) {
	c, _ := createTestContextWithQuery("")

	id, ok := ParseQueryID(c, "venue_id", "场地")

	assert.True(t, ok)
	assert.Nil(t, id)
}

func TestParseQueryID_Valid(t *testing.T) {
	c, _ := createTestContextWithQuery("venue_id=123")

	id, ok := ParseQueryID(c, "venue_id", "场地")

	assert.True(t, ok)
	require.NotNil(t, id)
	assert.Equal(t, int64(123), *id)
}

func TestParseQueryID_Invalid(t *testing.T) {
	c, w := createTestContextWithQuery("venue_id=abc")

	id, ok := ParseQueryID(c, "venue_id", "场地")

	assert.False(t, ok)
	assert.Nil(t, id)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParseRequiredQueryID_Empty(t *testing.T) {
	c, w := createTestContextWithQuery("")

	id, ok := ParseRequiredQueryID(c, "device_id", "设备")

	assert.False(t, ok)
	assert.Equal(t, int64(0), id)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseResponse(w)
	assert.Equal(t, "请提供设备ID", resp.Message)
}

func TestParseRequiredQueryID_Valid(t *testing.T) {
	c, _ := createTestContextWithQuery("device_id=456")

	id, ok := ParseRequiredQueryID(c, "device_id", "设备")

	assert.True(t, ok)
	assert.Equal(t, int64(456), id)
}

// ============================================================================
// Phase 4: 时间解析测试
// ============================================================================

func TestParseDate_Valid(t *testing.T) {
	date, err := ParseDate("2024-01-15")

	assert.NoError(t, err)
	assert.Equal(t, 2024, date.Year())
	assert.Equal(t, time.January, date.Month())
	assert.Equal(t, 15, date.Day())
}

func TestParseDate_Invalid(t *testing.T) {
	_, err := ParseDate("invalid")

	assert.Error(t, err)
}

func TestParseDateTime_MultipleFormats(t *testing.T) {
	testCases := []struct {
		input string
		valid bool
	}{
		{"2024-01-15T10:30:00+08:00", true},
		{"2024-01-15 10:30:00", true},
		{"2024-01-15T10:30:00", true},
		{"2024-01-15 10:30", true},
		{"invalid", false},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			_, err := ParseDateTime(tc.input)
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestParseQueryDate_Empty(t *testing.T) {
	c, _ := createTestContextWithQuery("")

	date, ok := ParseQueryDate(c, "date", "无效的日期")

	assert.True(t, ok)
	assert.Nil(t, date)
}

func TestParseQueryDate_Valid(t *testing.T) {
	c, _ := createTestContextWithQuery("date=2024-01-15")

	date, ok := ParseQueryDate(c, "date", "无效的日期")

	assert.True(t, ok)
	require.NotNil(t, date)
	assert.Equal(t, 2024, date.Year())
}

func TestParseQueryDate_Invalid(t *testing.T) {
	c, w := createTestContextWithQuery("date=invalid")

	date, ok := ParseQueryDate(c, "date", "无效的日期")

	assert.False(t, ok)
	assert.Nil(t, date)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParseQueryDateRange_BothEmpty(t *testing.T) {
	c, _ := createTestContextWithQuery("")

	start, end, ok := ParseQueryDateRange(c)

	assert.True(t, ok)
	assert.Nil(t, start)
	assert.Nil(t, end)
}

func TestParseQueryDateRange_Valid(t *testing.T) {
	c, _ := createTestContextWithQuery("start_date=2024-01-01&end_date=2024-01-31")

	start, end, ok := ParseQueryDateRange(c)

	assert.True(t, ok)
	require.NotNil(t, start)
	require.NotNil(t, end)
	assert.Equal(t, 2024, start.Year())
	assert.Equal(t, time.January, start.Month())
	assert.Equal(t, 1, start.Day())
	// 结束日期应该是当天结束时间
	assert.Equal(t, 23, end.Hour())
	assert.Equal(t, 59, end.Minute())
}

func TestParseQueryDateRange_InvalidStart(t *testing.T) {
	c, w := createTestContextWithQuery("start_date=invalid&end_date=2024-01-31")

	_, _, ok := ParseQueryDateRange(c)

	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParseRequiredQueryDateRange_BothEmpty(t *testing.T) {
	c, w := createTestContextWithQuery("")

	_, _, ok := ParseRequiredQueryDateRange(c)

	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseResponse(w)
	assert.Equal(t, "请指定开始和结束日期", resp.Message)
}

func TestParseRequiredQueryDateRange_Valid(t *testing.T) {
	c, _ := createTestContextWithQuery("start_date=2024-01-01&end_date=2024-01-31")

	start, end, ok := ParseRequiredQueryDateRange(c)

	assert.True(t, ok)
	assert.Equal(t, 2024, start.Year())
	assert.Equal(t, 2024, end.Year())
}

// ============================================================================
// Phase 5: 分页测试
// ============================================================================

func TestBindPagination_Defaults(t *testing.T) {
	c, _ := createTestContextWithQuery("")

	p := BindPagination(c)

	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 10, p.PageSize)
}

func TestBindPagination_CustomValues(t *testing.T) {
	c, _ := createTestContextWithQuery("page=3&page_size=20")

	p := BindPagination(c)

	assert.Equal(t, 3, p.Page)
	assert.Equal(t, 20, p.PageSize)
}

func TestBindPagination_Normalize(t *testing.T) {
	c, _ := createTestContextWithQuery("page=-1&page_size=200")

	p := BindPagination(c)

	assert.Equal(t, 1, p.Page)       // 规范化为1
	assert.Equal(t, 100, p.PageSize) // 最大100
}

func TestBindPaginationWithDefaults(t *testing.T) {
	c, _ := createTestContextWithQuery("")

	p := BindPaginationWithDefaults(c, 1, 20)

	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 20, p.PageSize)
}

func TestBindPagination_GetOffsetAndLimit(t *testing.T) {
	c, _ := createTestContextWithQuery("page=3&page_size=10")

	p := BindPagination(c)

	assert.Equal(t, 20, p.GetOffset()) // (3-1)*10
	assert.Equal(t, 10, p.GetLimit())
}

// ============================================================================
// 组合函数测试
// ============================================================================

func TestRequireUserAndParseID_Success(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: "123"}}
	c.Set(middleware.ContextKeyUserID, int64(456))

	userID, resourceID, ok := RequireUserAndParseID(c, "订单")

	assert.True(t, ok)
	assert.Equal(t, int64(456), userID)
	assert.Equal(t, int64(123), resourceID)
}

func TestRequireUserAndParseID_NotAuthenticated(t *testing.T) {
	c, w := createTestContextWithParam("id", "123")

	userID, resourceID, ok := RequireUserAndParseID(c, "订单")

	assert.False(t, ok)
	assert.Equal(t, int64(0), userID)
	assert.Equal(t, int64(0), resourceID)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireUserAndParseID_InvalidID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	c.Set(middleware.ContextKeyUserID, int64(456))

	userID, resourceID, ok := RequireUserAndParseID(c, "订单")

	assert.False(t, ok)
	assert.Equal(t, int64(0), userID)
	assert.Equal(t, int64(0), resourceID)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRequireAdminAndParseID_Success(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: "789"}}
	c.Set(middleware.ContextKeyUserID, int64(111))

	adminID, resourceID, ok := RequireAdminAndParseID(c, "设备")

	assert.True(t, ok)
	assert.Equal(t, int64(111), adminID)
	assert.Equal(t, int64(789), resourceID)
}
