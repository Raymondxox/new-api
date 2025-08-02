package vertex

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"one-api/common"
	"one-api/dto"
	"one-api/relay/channel"
	"one-api/relay/channel/claude"
	"one-api/relay/channel/gemini"
	"one-api/relay/channel/openai"
	relaycommon "one-api/relay/common"
	"one-api/relay/constant"
	"one-api/service"
	"one-api/setting/model_setting"
	"one-api/types"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	RequestModeClaude = 1
	RequestModeGemini = 2
	RequestModeLlama  = 3
	RequestModeVeo    = 4
)

var claudeModelMap = map[string]string{
	"claude-3-sonnet-20240229":   "claude-3-sonnet@20240229",
	"claude-3-opus-20240229":     "claude-3-opus@20240229",
	"claude-3-haiku-20240307":    "claude-3-haiku@20240307",
	"claude-3-5-sonnet-20240620": "claude-3-5-sonnet@20240620",
	"claude-3-5-sonnet-20241022": "claude-3-5-sonnet-v2@20241022",
	"claude-3-7-sonnet-20250219": "claude-3-7-sonnet@20250219",
	"claude-sonnet-4-20250514":   "claude-sonnet-4@20250514",
	"claude-opus-4-20250514":     "claude-opus-4@20250514",
}

// 添加 Veo 模型映射
var veoModelMap = map[string]string{
	"veo-2.0-generate-001":     "veo-2.0-generate-001",
	"veo-3.0-generate-preview": "veo-3.0-generate-preview",
}

const anthropicVersion = "vertex-2023-10-16"

type Adaptor struct {
	RequestMode        int
	AccountCredentials Credentials
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	if v, ok := claudeModelMap[info.UpstreamModelName]; ok {
		c.Set("request_model", v)
	} else {
		c.Set("request_model", request.Model)
	}
	vertexClaudeReq := copyRequest(request, anthropicVersion)
	return vertexClaudeReq, nil
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
	if strings.HasPrefix(info.UpstreamModelName, "claude") {
		a.RequestMode = RequestModeClaude
	} else if strings.HasPrefix(info.UpstreamModelName, "gemini") {
		a.RequestMode = RequestModeGemini
	} else if strings.Contains(info.UpstreamModelName, "llama") {
		a.RequestMode = RequestModeLlama
	} else if strings.HasPrefix(info.UpstreamModelName, "veo") {
		a.RequestMode = RequestModeVeo
	}
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	adc := &Credentials{}
	if err := json.Unmarshal([]byte(info.ApiKey), adc); err != nil {
		return "", fmt.Errorf("failed to decode credentials file: %w", err)
	}
	region := GetModelRegion(info.ApiVersion, info.OriginModelName)
	a.AccountCredentials = *adc
	suffix := ""
	if a.RequestMode == RequestModeGemini {
		if model_setting.GetGeminiSettings().ThinkingAdapterEnabled {
			// 新增逻辑：处理 -thinking-<budget> 格式
			if strings.Contains(info.UpstreamModelName, "-thinking-") {
				parts := strings.Split(info.UpstreamModelName, "-thinking-")
				info.UpstreamModelName = parts[0]
			} else if strings.HasSuffix(info.UpstreamModelName, "-thinking") { // 旧的适配
				info.UpstreamModelName = strings.TrimSuffix(info.UpstreamModelName, "-thinking")
			} else if strings.HasSuffix(info.UpstreamModelName, "-nothinking") {
				info.UpstreamModelName = strings.TrimSuffix(info.UpstreamModelName, "-nothinking")
			}
		}

		if info.IsStream {
			suffix = "streamGenerateContent?alt=sse"
		} else {
			suffix = "generateContent"
		}
		if region == "global" {
			return fmt.Sprintf(
				"https://aiplatform.googleapis.com/v1/projects/%s/locations/global/publishers/google/models/%s:%s",
				adc.ProjectID,
				info.UpstreamModelName,
				suffix,
			), nil
		} else {
			return fmt.Sprintf(
				"https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:%s",
				region,
				adc.ProjectID,
				region,
				info.UpstreamModelName,
				suffix,
			), nil
		}
	} else if a.RequestMode == RequestModeClaude {
		if info.IsStream {
			suffix = "streamRawPredict?alt=sse"
		} else {
			suffix = "rawPredict"
		}
		model := info.UpstreamModelName
		if v, ok := claudeModelMap[info.UpstreamModelName]; ok {
			model = v
		}
		if region == "global" {
			return fmt.Sprintf(
				"https://aiplatform.googleapis.com/v1/projects/%s/locations/global/publishers/anthropic/models/%s:%s",
				adc.ProjectID,
				model,
				suffix,
			), nil
		} else {
			return fmt.Sprintf(
				"https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/anthropic/models/%s:%s",
				region,
				adc.ProjectID,
				region,
				model,
				suffix,
			), nil
		}
	} else if a.RequestMode == RequestModeLlama {
		return fmt.Sprintf(
			"https://%s-aiplatform.googleapis.com/v1beta1/projects/%s/locations/%s/endpoints/openapi/chat/completions",
			region,
			adc.ProjectID,
			region,
		), nil
	} else if a.RequestMode == RequestModeVeo {
		// Veo模型使用predictLongRunning端点
		if region == "global" {
			return fmt.Sprintf(
				"https://aiplatform.googleapis.com/v1/projects/%s/locations/global/publishers/google/models/%s:predictLongRunning",
				adc.ProjectID,
				info.UpstreamModelName,
			), nil
		} else {
			return fmt.Sprintf(
				"https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:predictLongRunning",
				region,
				adc.ProjectID,
				region,
				info.UpstreamModelName,
			), nil
		}
	}
	return "", errors.New("unsupported request mode")
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	accessToken, err := getAccessToken(a, info)
	if err != nil {
		return err
	}
	req.Set("Authorization", "Bearer "+accessToken)
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	if a.RequestMode == RequestModeClaude {
		claudeReq, err := claude.RequestOpenAI2ClaudeMessage(*request)
		if err != nil {
			return nil, err
		}
		vertexClaudeReq := copyRequest(claudeReq, anthropicVersion)
		c.Set("request_model", claudeReq.Model)
		info.UpstreamModelName = claudeReq.Model
		return vertexClaudeReq, nil
	} else if a.RequestMode == RequestModeGemini {
		geminiRequest, err := gemini.CovertGemini2OpenAI(*request, info)
		if err != nil {
			return nil, err
		}
		c.Set("request_model", request.Model)
		return geminiRequest, nil
	} else if a.RequestMode == RequestModeLlama {
		return request, nil
	}
	return nil, errors.New("unsupported request mode")
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, nil
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	// TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	if info.IsStream {
		switch a.RequestMode {
		case RequestModeClaude:
			err, usage = claude.ClaudeStreamHandler(c, resp, info, claude.RequestModeMessage)
		case RequestModeGemini:
			if info.RelayMode == constant.RelayModeGemini {
				usage, err = gemini.GeminiTextGenerationStreamHandler(c, info, resp)
			} else {
				usage, err = gemini.GeminiChatStreamHandler(c, info, resp)
			}
		case RequestModeLlama:
			usage, err = openai.OaiStreamHandler(c, info, resp)
		}
	} else {
		switch a.RequestMode {
		case RequestModeClaude:
			err, usage = claude.ClaudeHandler(c, resp, claude.RequestModeMessage, info)
		case RequestModeGemini:
			if info.RelayMode == constant.RelayModeGemini {
				usage, err = gemini.GeminiTextGenerationHandler(c, info, resp)
			} else {
				usage, err = gemini.GeminiChatHandler(c, info, resp)
			}
		case RequestModeLlama:
			usage, err = openai.OpenaiHandler(c, info, resp)
		case RequestModeVeo:
			usage, err = VeoHandler(c, info, resp)
		}
	}
	return
}

func (a *Adaptor) GetModelList() []string {
	var modelList []string
	for i, s := range ModelList {
		modelList = append(modelList, s)
		ModelList[i] = s
	}
	for i, s := range claude.ModelList {
		modelList = append(modelList, s)
		claude.ModelList[i] = s
	}
	for i, s := range gemini.ModelList {
		modelList = append(modelList, s)
		gemini.ModelList[i] = s
	}
	return modelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}

// 添加Veo视频请求转换方法
func (a *Adaptor) ConvertVeoVideoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody []byte) (any, error) {
	// 解析请求体
	var request map[string]interface{}
	if err := json.Unmarshal(requestBody, &request); err != nil {
		return nil, fmt.Errorf("failed to parse request body: %w", err)
	}

	// 创建转换器
	converter := NewVeoRequestConverter()

	// 判断是文本生成视频还是图片生成视频
	if instances, ok := request["instances"].(map[string]interface{}); ok {
		if _, hasImage := instances["image"]; hasImage {
			// 图片生成视频
			return converter.ConvertImageRequest(request)
		} else {
			// 文本生成视频
			return converter.ConvertTextRequest(request)
		}
	}

	return nil, errors.New("invalid request format")
}

// 添加轮询方法
func (a *Adaptor) PollVeoOperation(c *gin.Context, operationName string, info *relaycommon.RelayInfo) (any, *types.NewAPIError) {
	// 构建轮询请求
	pollRequest := VeoPollRequest{
		OperationName: operationName,
	}

	requestBody, err := json.Marshal(pollRequest)
	if err != nil {
		return nil, &types.NewAPIError{
			Err:        err,
			StatusCode: http.StatusInternalServerError,
		}
	}

	// 构建轮询URL
	pollURL := strings.Replace(info.BaseUrl, ":predictLongRunning", ":fetchPredictOperation", 1)

	req, err := http.NewRequest("POST", pollURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, &types.NewAPIError{
			Err:        err,
			StatusCode: http.StatusInternalServerError,
		}
	}

	// 设置请求头
	if err := a.SetupRequestHeader(c, &req.Header, info); err != nil {
		return nil, &types.NewAPIError{
			Err:        err,
			StatusCode: http.StatusInternalServerError,
		}
	}

	// 发送请求
	client := service.GetHttpClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, &types.NewAPIError{
			Err:        err,
			StatusCode: http.StatusInternalServerError,
		}
	}
	defer resp.Body.Close()

	// 处理响应
	return a.handleVeoResponse(c, resp, info)
}

// VeoHandler 处理Veo API响应
func VeoHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	a := &Adaptor{}
	a.Init(info)
	return a.handleVeoResponse(c, resp, info)
}

// handleVeoResponse 处理Veo响应
func (a *Adaptor) handleVeoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*dto.Usage, *types.NewAPIError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &types.NewAPIError{
			Err:        err,
			StatusCode: http.StatusInternalServerError,
		}
	}
	common.CloseResponseBodyGracefully(resp)

	var veoResp VeoResponse
	if err := json.Unmarshal(responseBody, &veoResp); err != nil {
		return nil, &types.NewAPIError{
			Err:        err,
			StatusCode: http.StatusInternalServerError,
		}
	}

	// 如果操作还在进行中，返回操作名称供轮询
	if !veoResp.Done {
		c.JSON(http.StatusOK, gin.H{
			"operation_name": veoResp.Name,
			"status":         "processing",
			"message":        "Video generation in progress",
		})
		return nil, nil
	}

	// 操作完成，返回视频结果
	if veoResp.Response != nil && len(veoResp.Response.GeneratedSamples) > 0 {
		videos := make([]map[string]interface{}, 0)
		for _, sample := range veoResp.Response.GeneratedSamples {
			videos = append(videos, map[string]interface{}{
				"url":      sample.Video.Uri,
				"encoding": sample.Video.Encoding,
			})
		}

		// 构建标准化的响应
		response := map[string]interface{}{
			"task_id": veoResp.Name,
			"status":  "succeeded",
			"url":     videos[0]["url"], // 第一个视频作为主要URL
			"format":  "mp4",
			"videos":  videos,
		}

		c.JSON(http.StatusOK, response)
		return nil, nil
	}

	return nil, &types.NewAPIError{
		Err:        err,
		StatusCode: http.StatusInternalServerError,
	}
}
