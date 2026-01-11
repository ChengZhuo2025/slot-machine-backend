// Package response 统一响应格式单元测试
package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTest 创建测试用的 Gin 上下文
func setupTest() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

// parseResponse 解析响应为 Response 结构
func parseResponse(t *testing.T, w *httptest.ResponseRecorder) Response {
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	return resp
}

// ==================== Success 测试 ====================

func TestSuccess(t *testing.T) {
	c, w := setupTest()

	data := map[string]interface{}{
		"id":   123,
		"name": "test",
	}

	Success(c, data)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "success", resp.Message)
	assert.NotNil(t, resp.Data)
}

func TestSuccess_WithNilData(t *testing.T) {
	c, w := setupTest()

	Success(c, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "success", resp.Message)
}

// ==================== SuccessWithMessage 测试 ====================

func TestSuccessWithMessage(t *testing.T) {
	c, w := setupTest()

	message := "操作成功"
	data := map[string]string{"status": "ok"}

	SuccessWithMessage(c, message, data)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, message, resp.Message)
	assert.NotNil(t, resp.Data)
}

// ==================== SuccessPage 测试 ====================

func TestSuccessPage(t *testing.T) {
	c, w := setupTest()

	list := []map[string]interface{}{
		{"id": 1, "name": "item1"},
		{"id": 2, "name": "item2"},
	}
	total := int64(100)
	page := 2
	pageSize := 20

	SuccessPage(c, list, total, page, pageSize)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "success", resp.Message)

	// 验证分页数据
	pageData, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(100), pageData["total"])
	assert.Equal(t, float64(2), pageData["page"])
	assert.Equal(t, float64(20), pageData["page_size"])
	assert.NotNil(t, pageData["list"])
}

func TestSuccessPage_EmptyList(t *testing.T) {
	c, w := setupTest()

	SuccessPage(c, []interface{}{}, 0, 1, 10)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)

	pageData, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(0), pageData["total"])
}

// ==================== SuccessWithPage 测试 ====================

func TestSuccessWithPage(t *testing.T) {
	c, w := setupTest()

	list := []string{"a", "b", "c"}
	SuccessWithPage(c, list, 3, 1, 10)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "success", resp.Message)
}

// ==================== SuccessList 测试 ====================

func TestSuccessList_WithPagination(t *testing.T) {
	c, w := setupTest()

	list := []int{1, 2, 3}
	total := int64(50)
	page := 3
	pageSize := 15

	SuccessList(c, list, total, page, pageSize)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)

	listData, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(50), listData["total"])
	assert.Equal(t, float64(3), listData["page"])
	assert.Equal(t, float64(15), listData["page_size"])
}

func TestSuccessList_WithoutPagination(t *testing.T) {
	c, w := setupTest()

	list := []string{"x", "y"}
	total := int64(2)

	// 不提供分页参数，应该使用默认值
	SuccessList(c, list, total)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)

	listData, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(2), listData["total"])
	assert.Equal(t, float64(1), listData["page"])    // 默认值
	assert.Equal(t, float64(20), listData["page_size"]) // 默认值
}

func TestSuccessList_PartialPagination(t *testing.T) {
	c, w := setupTest()

	// 只提供一个参数时，应该使用默认值
	SuccessList(c, []int{}, 0, 5) // 只有page，没有pageSize

	resp := parseResponse(t, w)
	listData, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	// 由于只有一个参数，不满足 len >= 2，使用默认值
	assert.Equal(t, float64(1), listData["page"])
	assert.Equal(t, float64(20), listData["page_size"])
}

// ==================== Error 测试 ====================

func TestError(t *testing.T) {
	c, w := setupTest()

	Error(c, 1001, "参数错误")

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 1001, resp.Code)
	assert.Equal(t, "参数错误", resp.Message)
	assert.Nil(t, resp.Data)
}

func TestError_WithDifferentCodes(t *testing.T) {
	tests := []struct {
		name    string
		code    int
		message string
	}{
		{"Generic error", 1000, "未知错误"},
		{"Business error", 5001, "订单状态异常"},
		{"Auth error", 2000, "未登录"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupTest()

			Error(c, tt.code, tt.message)

			resp := parseResponse(t, w)
			assert.Equal(t, tt.code, resp.Code)
			assert.Equal(t, tt.message, resp.Message)
		})
	}
}

// ==================== ErrorWithData 测试 ====================

func TestErrorWithData(t *testing.T) {
	c, w := setupTest()

	data := map[string]interface{}{
		"field": "username",
		"error": "用户名已存在",
	}

	ErrorWithData(c, 1001, "验证失败", data)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 1001, resp.Code)
	assert.Equal(t, "验证失败", resp.Message)
	assert.NotNil(t, resp.Data)
}

// ==================== BadRequest 测试 ====================

func TestBadRequest(t *testing.T) {
	c, w := setupTest()

	BadRequest(c, "无效的请求参数")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "无效的请求参数", resp.Message)
}

// ==================== Unauthorized 测试 ====================

func TestUnauthorized(t *testing.T) {
	t.Run("With custom message", func(t *testing.T) {
		c, w := setupTest()

		Unauthorized(c, "登录已过期")

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 401, resp.Code)
		assert.Equal(t, "登录已过期", resp.Message)
	})

	t.Run("With empty message", func(t *testing.T) {
		c, w := setupTest()

		Unauthorized(c, "")

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 401, resp.Code)
		assert.Equal(t, "unauthorized", resp.Message)
	})
}

// ==================== Forbidden 测试 ====================

func TestForbidden(t *testing.T) {
	t.Run("With custom message", func(t *testing.T) {
		c, w := setupTest()

		Forbidden(c, "权限不足")

		assert.Equal(t, http.StatusForbidden, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 403, resp.Code)
		assert.Equal(t, "权限不足", resp.Message)
	})

	t.Run("With empty message", func(t *testing.T) {
		c, w := setupTest()

		Forbidden(c, "")

		assert.Equal(t, http.StatusForbidden, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 403, resp.Code)
		assert.Equal(t, "forbidden", resp.Message)
	})
}

// ==================== NotFound 测试 ====================

func TestNotFound(t *testing.T) {
	t.Run("With custom message", func(t *testing.T) {
		c, w := setupTest()

		NotFound(c, "用户不存在")

		assert.Equal(t, http.StatusNotFound, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 404, resp.Code)
		assert.Equal(t, "用户不存在", resp.Message)
	})

	t.Run("With empty message", func(t *testing.T) {
		c, w := setupTest()

		NotFound(c, "")

		assert.Equal(t, http.StatusNotFound, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 404, resp.Code)
		assert.Equal(t, "not found", resp.Message)
	})
}

// ==================== InternalError 测试 ====================

func TestInternalError(t *testing.T) {
	t.Run("With custom message", func(t *testing.T) {
		c, w := setupTest()

		InternalError(c, "数据库连接失败")

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 500, resp.Code)
		assert.Equal(t, "数据库连接失败", resp.Message)
	})

	t.Run("With empty message", func(t *testing.T) {
		c, w := setupTest()

		InternalError(c, "")

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 500, resp.Code)
		assert.Equal(t, "internal server error", resp.Message)
	})
}

// ==================== TooManyRequests 测试 ====================

func TestTooManyRequests(t *testing.T) {
	t.Run("With custom message", func(t *testing.T) {
		c, w := setupTest()

		TooManyRequests(c, "请求次数超过限制")

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 429, resp.Code)
		assert.Equal(t, "请求次数超过限制", resp.Message)
	})

	t.Run("With empty message", func(t *testing.T) {
		c, w := setupTest()

		TooManyRequests(c, "")

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 429, resp.Code)
		assert.Equal(t, "too many requests", resp.Message)
	})
}

// ==================== 数据结构测试 ====================

func TestResponse_JSONMarshaling(t *testing.T) {
	resp := Response{
		Code:    0,
		Message: "success",
		Data:    map[string]string{"key": "value"},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.Contains(t, string(data), "\"code\":0")
	assert.Contains(t, string(data), "\"message\":\"success\"")
	assert.Contains(t, string(data), "\"data\"")
}

func TestPageData_Structure(t *testing.T) {
	pageData := PageData{
		List:     []int{1, 2, 3},
		Total:    100,
		Page:     2,
		PageSize: 20,
	}

	data, err := json.Marshal(pageData)
	require.NoError(t, err)
	assert.Contains(t, string(data), "\"total\":100")
	assert.Contains(t, string(data), "\"page\":2")
	assert.Contains(t, string(data), "\"page_size\":20")
}

func TestListData_Structure(t *testing.T) {
	listData := ListData{
		List:     []string{"a", "b"},
		Total:    2,
		Page:     1,
		PageSize: 10,
	}

	data, err := json.Marshal(listData)
	require.NoError(t, err)
	assert.Contains(t, string(data), "\"total\":2")
	assert.Contains(t, string(data), "\"page\":1")
}

// ==================== 边界条件测试 ====================

func TestSuccess_WithLargeData(t *testing.T) {
	c, w := setupTest()

	// 大数据量
	largeList := make([]int, 10000)
	for i := range largeList {
		largeList[i] = i
	}

	Success(c, largeList)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
}

func TestError_WithZeroCode(t *testing.T) {
	c, w := setupTest()

	Error(c, 0, "成功")

	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "成功", resp.Message)
}

func TestSuccessPage_WithZeroTotal(t *testing.T) {
	c, w := setupTest()

	SuccessPage(c, []interface{}{}, 0, 1, 10)

	resp := parseResponse(t, w)
	pageData, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(0), pageData["total"])
}

// ==================== 组合使用测试 ====================

func TestMultipleResponses(t *testing.T) {
	// 在同一个测试中调用多个响应函数应该不会有问题
	c1, w1 := setupTest()
	Success(c1, "data1")
	assert.Equal(t, http.StatusOK, w1.Code)

	c2, w2 := setupTest()
	Error(c2, 1001, "error")
	assert.Equal(t, http.StatusOK, w2.Code)

	c3, w3 := setupTest()
	NotFound(c3, "not found")
	assert.Equal(t, http.StatusNotFound, w3.Code)
}
