package vertex

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"
)

// VeoRequestConverter Veo请求转换器
type VeoRequestConverter struct{}

// NewVeoRequestConverter 创建新的Veo请求转换器
func NewVeoRequestConverter() *VeoRequestConverter {
	return &VeoRequestConverter{}
}

// ConvertTextRequest 转换文本生成视频请求
func (c *VeoRequestConverter) ConvertTextRequest(request map[string]interface{}) (*VeoTextRequest, error) {
	veoReq := &VeoTextRequest{}

	// 处理instances
	if instances, ok := request["instances"].(map[string]interface{}); ok {
		instance := VeoTextInstance{}

		if prompt, ok := instances["prompt"].(string); ok {
			instance.Prompt = prompt
		}

		if durationSeconds, ok := instances["durationSeconds"].(float64); ok {
			instance.DurationSeconds = uint8(durationSeconds)
		} else {
			instance.DurationSeconds = 8 // 默认值
		}

		veoReq.Instances = []VeoTextInstance{instance}
	}

	// 处理parameters
	if parameters, ok := request["parameters"].(map[string]interface{}); ok {
		veoReq.Parameters = c.convertParameters(parameters)
	}

	return veoReq, nil
}

// ConvertImageRequest 转换图片生成视频请求
func (c *VeoRequestConverter) ConvertImageRequest(request map[string]interface{}) (*VeoImageRequest, error) {
	veoReq := &VeoImageRequest{}

	// 处理instances
	if instances, ok := request["instances"].(map[string]interface{}); ok {
		instance := VeoImageInstance{}

		if prompt, ok := instances["prompt"].(string); ok {
			instance.Prompt = prompt
		}

		if durationSeconds, ok := instances["durationSeconds"].(float64); ok {
			instance.DurationSeconds = uint8(durationSeconds)
		} else {
			instance.DurationSeconds = 8 // 默认值
		}

		// 处理image
		if image, ok := instances["image"].(map[string]interface{}); ok {
			veoImage, err := c.convertImage(image)
			if err != nil {
				return nil, err
			}
			instance.Image = *veoImage
		}

		// 处理lastFrame（可选）
		if lastFrame, ok := instances["lastFrame"].(map[string]interface{}); ok {
			veoLastFrame, err := c.convertImage(lastFrame)
			if err != nil {
				return nil, err
			}
			instance.LastFrame = veoLastFrame
		}

		veoReq.Instances = []VeoImageInstance{instance}
	}

	// 处理parameters
	if parameters, ok := request["parameters"].(map[string]interface{}); ok {
		veoReq.Parameters = c.convertParameters(parameters)
	}

	return veoReq, nil
}

// convertParameters 转换参数
func (c *VeoRequestConverter) convertParameters(parameters map[string]interface{}) VeoParameters {
	veoParams := VeoParameters{}

	if negativePrompt, ok := parameters["negativePrompt"].(string); ok {
		veoParams.NegativePrompt = negativePrompt
	}

	if sampleCount, ok := parameters["sampleCount"].(float64); ok {
		veoParams.SampleCount = int(sampleCount)
	}

	if aspectRatio, ok := parameters["aspectRatio"].(string); ok {
		veoParams.AspectRatio = aspectRatio
	}

	if personGeneration, ok := parameters["personGeneration"].(string); ok {
		veoParams.PersonGeneration = personGeneration
	}

	if seed, ok := parameters["seed"].(float64); ok {
		veoParams.Seed = uint32(seed)
	}

	if generateAudio, ok := parameters["generateAudio"].(bool); ok {
		veoParams.GenerateAudio = generateAudio
	}

	if resolution, ok := parameters["resolution"].(string); ok {
		veoParams.Resolution = resolution
	}

	if storageUri, ok := parameters["storageUri"].(string); ok {
		veoParams.StorageUri = storageUri
	}

	if durationSeconds, ok := parameters["durationSeconds"].(float64); ok {
		veoParams.DurationSeconds = int(durationSeconds)
	}

	if enhancePrompt, ok := parameters["enhancePrompt"].(bool); ok {
		veoParams.EnhancePrompt = enhancePrompt
	}

	return veoParams
}

// convertImage 转换图片
func (c *VeoRequestConverter) convertImage(image map[string]interface{}) (*VeoImage, error) {
	veoImage := &VeoImage{}

	if bytesBase64Encoded, ok := image["bytesBase64Encoded"].(string); ok {
		veoImage.BytesBase64Encoded = bytesBase64Encoded
	}

	if gcsUri, ok := image["gcsUri"].(string); ok {
		veoImage.GcsUri = gcsUri
	}

	if mimeType, ok := image["mimeType"].(string); ok {
		veoImage.MimeType = mimeType
	}

	return veoImage, nil
}

// ProcessImageData 处理图片数据（从URL下载并转换为Base64）
func (c *VeoRequestConverter) ProcessImageData(imageInput string) (string, string, error) {
	// 判断是URL还是Base64
	if strings.HasPrefix(imageInput, "http") {
		// 下载图片并转换为Base64
		return c.downloadAndEncodeImage(imageInput)
	} else if strings.HasPrefix(imageInput, "data:") {
		// 已经是Base64格式
		parts := strings.Split(imageInput, ",")
		if len(parts) == 2 {
			mimeType := strings.TrimPrefix(strings.Split(parts[0], ";")[0], "data:")
			return parts[1], mimeType, nil
		}
	}

	// 假设是纯Base64
	return imageInput, "image/jpeg", nil
}

// downloadAndEncodeImage 下载并编码图片
func (c *VeoRequestConverter) downloadAndEncodeImage(imageURL string) (string, string, error) {
	resp, err := http.Get(imageURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	// 检测MIME类型
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		// 根据文件扩展名推断
		if strings.HasSuffix(imageURL, ".png") {
			mimeType = "image/png"
		} else {
			mimeType = "image/jpeg"
		}
	}

	// 转换为Base64
	encoded := base64.StdEncoding.EncodeToString(imageData)
	return encoded, mimeType, nil
}
