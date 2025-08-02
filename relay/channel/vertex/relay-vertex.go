package vertex

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"one-api/common"
	"one-api/model"
	relaycommon "one-api/relay/common"
	"one-api/types"
)

func GetModelRegion(other string, localModelName string) string {
	// if other is json string
	if common.IsJsonObject(other) {
		m, err := common.StrToMap(other)
		if err != nil {
			return other // return original if parsing fails
		}
		if m[localModelName] != nil {
			return m[localModelName].(string)
		} else {
			return m["default"].(string)
		}
	}
	return other
}

func RelayVeoVideo(c *gin.Context) {
	// 读取请求体
	requestBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// 验证请求格式
	var request map[string]interface{}
	if err := json.Unmarshal(requestBody, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	// 验证必要字段
	if instances, ok := request["instances"].(map[string]interface{}); ok {
		if prompt, ok := instances["prompt"].(string); ok && prompt == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Prompt is required"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instances field is required"})
		return
	}

	// 获取渠道信息
	channel, err := getChannelFromContext(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取模型名称
	modelName := c.Query("model")
	if modelName == "" {
		modelName = "veo-2.0-generate-001" // 默认模型
	}

	// 处理 BaseURL（*string 转 string，避免空指针）
	var baseURL string
	if channel.BaseURL != nil {
		baseURL = *channel.BaseURL // 指针非空时取实际值
	} else {
		baseURL = "" // 指针为空时用默认空字符串
	}

	// 构建RelayInfo
	info := &relaycommon.RelayInfo{
		ChannelId:         channel.Id,
		ChannelType:       channel.Type,
		BaseUrl:           baseURL,
		ApiKey:            channel.Key,
		ApiVersion:        channel.Other,
		UpstreamModelName: modelName,
		OriginModelName:   modelName,
		IsStream:          false,
	}

	// 创建适配器
	adaptor := &Adaptor{}
	adaptor.Init(info)

	// 转换请求
	veoRequest, err := adaptor.ConvertVeoVideoRequest(c, info, requestBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 序列化请求
	veoRequestBody, err := json.Marshal(veoRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal request"})
		return
	}

	// 发送请求
	resp, err := adaptor.DoRequest(c, info, bytes.NewReader(veoRequestBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 处理响应
	httpResp := resp.(*http.Response)
	usage, apiErr := adaptor.DoResponse(c, httpResp, info)
	if apiErr != nil {
		c.JSON(apiErr.StatusCode, gin.H{"error": apiErr.Err})
		return
	}

	// 记录使用量
	if usage != nil {
		common.LogInfo(c, "Veo video generation completed")
	}
}

// PollVeoOperation 轮询Veo操作状态
func PollVeoOperation(c *gin.Context) {
	operationName := c.Param("operation_name")
	if operationName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Operation name is required"})
		return
	}

	// 获取渠道信息
	channel, err := getChannelFromContext(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取模型名称
	modelName := c.Query("model")
	if modelName == "" {
		modelName = "veo-2.0-generate-001" // 默认模型
	}
	// 处理 BaseURL（*string 转 string，避免空指针）
	var baseURL string
	if channel.BaseURL != nil {
		baseURL = *channel.BaseURL // 指针非空时取实际值
	} else {
		baseURL = "" // 指针为空时用默认空字符串
	}

	// 构建RelayInfo
	info := &relaycommon.RelayInfo{
		ChannelId:         channel.Id,
		ChannelType:       channel.Type,
		BaseUrl:           baseURL,
		ApiKey:            channel.Key,
		ApiVersion:        channel.Other,
		UpstreamModelName: modelName,
		OriginModelName:   modelName,
		IsStream:          false,
	}

	// 创建适配器
	adaptor := &Adaptor{}
	adaptor.Init(info)

	// 轮询操作
	result, apiErr := adaptor.PollVeoOperation(c, operationName, info)
	if apiErr != nil {
		c.JSON(apiErr.StatusCode, gin.H{"error": apiErr.Err})
		return
	}

	// 返回结果
	if result != nil {
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusOK, gin.H{"status": "processing"})
	}
}

// 辅助函数：从上下文获取渠道信息
func getChannelFromContext(c *gin.Context) (*model.Channel, error) {
	channelId := c.GetInt("channel_id")
	if channelId == 0 {
		return nil, types.NewError(errors.New("channel not found"), types.ErrorCodeChannelInvalidKey)
	}

	channel, err := model.GetChannelById(channelId, false)
	if err != nil {
		return nil, err
	}

	return channel, nil
}
