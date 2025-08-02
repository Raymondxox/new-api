package vertex

import (
	"one-api/dto"
)

type VertexAIClaudeRequest struct {
	AnthropicVersion string              `json:"anthropic_version"`
	Messages         []dto.ClaudeMessage `json:"messages"`
	System           any                 `json:"system,omitempty"`
	MaxTokens        uint                `json:"max_tokens,omitempty"`
	StopSequences    []string            `json:"stop_sequences,omitempty"`
	Stream           bool                `json:"stream,omitempty"`
	Temperature      *float64            `json:"temperature,omitempty"`
	TopP             float64             `json:"top_p,omitempty"`
	TopK             int                 `json:"top_k,omitempty"`
	Tools            any                 `json:"tools,omitempty"`
	ToolChoice       any                 `json:"tool_choice,omitempty"`
	Thinking         *dto.Thinking       `json:"thinking,omitempty"`
}

func copyRequest(req *dto.ClaudeRequest, version string) *VertexAIClaudeRequest {
	return &VertexAIClaudeRequest{
		AnthropicVersion: version,
		System:           req.System,
		Messages:         req.Messages,
		MaxTokens:        req.MaxTokens,
		Stream:           req.Stream,
		Temperature:      req.Temperature,
		TopP:             req.TopP,
		TopK:             req.TopK,
		StopSequences:    req.StopSequences,
		Tools:            req.Tools,
		ToolChoice:       req.ToolChoice,
		Thinking:         req.Thinking,
	}
}

// VeoTextRequest 文本生成视频请求
type VeoTextRequest struct {
	Instances  []VeoTextInstance `json:"instances"`
	Parameters VeoParameters     `json:"parameters"`
}

// VeoImageRequest 图片生成视频请求
type VeoImageRequest struct {
	Instances  []VeoImageInstance `json:"instances"`
	Parameters VeoParameters      `json:"parameters"`
}

// VeoTextInstance 文本生成视频实例
type VeoTextInstance struct {
	Prompt          string `json:"prompt"`          // 文本描述
	DurationSeconds uint8  `json:"durationSeconds"` // 生成视频长度，仅支持5-8秒，默认为8秒
}

// VeoImageInstance 图片生成视频实例
type VeoImageInstance struct {
	Prompt          string    `json:"prompt"`              // 文本描述
	Image           VeoImage  `json:"image"`               // 第一帧的图片
	LastFrame       *VeoImage `json:"lastFrame,omitempty"` // 最后一帧的图片（可选）
	DurationSeconds uint8     `json:"durationSeconds"`     // 生成视频长度，仅支持5-8秒，默认为8秒
}

// VeoImage 图片结构
type VeoImage struct {
	BytesBase64Encoded string `json:"bytesBase64Encoded,omitempty"` // Base64编码的图片数据
	GcsUri             string `json:"gcsUri,omitempty"`             // Cloud Storage URI
	MimeType           string `json:"mimeType"`                     // MIME类型：image/jpeg 或 image/png
}

// VeoParameters 视频生成参数
type VeoParameters struct {
	AspectRatio      string `json:"aspectRatio,omitempty"`      // 生成视频的宽高比，16:9（默认，横向），9:16（纵向），Veo3仅支持16:9
	NegativePrompt   string `json:"negativePrompt,omitempty"`   // 可选。用于描述您想要阻止模型生成的内容的文本字符串。
	PersonGeneration string `json:"personGeneration,omitempty"` // 可选。用于控制是否允许人物或人脸生成的安全设置。allow_adult（默认值）：仅允许生成成年人。dont_allow：禁止在图片中包含人物或人脸。
	SampleCount      int    `json:"sampleCount,omitempty"`      // 输出视频的数量支持1-4个
	Seed             uint32 `json:"seed,omitempty"`             // 可选。在请求中添加种子编号而不更改其他参数会导致模型生成相同的视频。 接受的范围为 0-4,294,967,295。
	StorageUri       string `json:"storageUri,omitempty"`       // 可选。用于存储输出视频的 Cloud Storage 存储桶 URI，格式为 gs://BUCKET_NAME/SUBDIRECTORY。如果未提供 Cloud Storage 存储桶，则回答中会返回以 base64 编码的视频字节。
	DurationSeconds  int    `json:"durationSeconds,omitempty"`  // 生成视频长度，仅支持5-8秒，默认为8秒
	EnhancePrompt    bool   `json:"enhancePrompt,omitempty"`    // 可选。使用 Gemini 优化问题。可接受的值为 true 或 false。默认值为 true。
	GenerateAudio    bool   `json:"generateAudio,omitempty"`    // veo3的必传参数,是否为视频生成音频。veo2不支持此参数。
	Resolution       string `json:"resolution,omitempty"`       // 可选。仅限Veo3模型。生成的视频的分辨率。值：720p（默认）或 1080p。
}

// VeoResponse Veo API响应
type VeoResponse struct {
	Name     string           `json:"name"` // 在发送视频生成请求后开始的长时间运行操作的完整操作名称
	Done     bool             `json:"done"` // 布尔值，指示操作是否已完成
	Response *VeoResponseData `json:"response,omitempty"`
}

// VeoResponseData 响应数据
type VeoResponseData struct {
	Type             string               `json:"@type"`
	GeneratedSamples []VeoGeneratedSample `json:"generatedSamples"`
}

// VeoGeneratedSample 生成的视频样本
type VeoGeneratedSample struct {
	Video VeoVideo `json:"video"`
}

// VeoVideo 视频信息
type VeoVideo struct {
	Uri      string `json:"uri"`      // 生成的视频的 Cloud Storage URI
	Encoding string `json:"encoding"` // 视频编码类型
}

// VeoPollRequest 轮询请求结构体
type VeoPollRequest struct {
	OperationName string `json:"operationName"`
}
