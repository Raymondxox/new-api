package router

import (
	"one-api/controller"
	"one-api/middleware"
	"one-api/relay/channel/vertex"

	"github.com/gin-gonic/gin"
)

func SetVideoRouter(router *gin.Engine) {
	videoV1Router := router.Group("/v1")
	videoV1Router.Use(middleware.TokenAuth(), middleware.Distribute())
	{
		videoV1Router.POST("/video/generations", controller.RelayTask)
		videoV1Router.GET("/video/generations/:task_id", controller.RelayTask)
	}

	klingV1Router := router.Group("/kling/v1")
	klingV1Router.Use(middleware.KlingRequestConvert(), middleware.TokenAuth(), middleware.Distribute())
	{
		klingV1Router.POST("/videos/text2video", controller.RelayTask)
		klingV1Router.POST("/videos/image2video", controller.RelayTask)
	}

	// SetVeoRouter 设置Veo路由
	veoV1Router := router.Group("/veo/v1")
	veoV1Router.Use(middleware.TokenAuth(), middleware.Distribute())
	{
		veoV1Router.POST("/videos/generations", vertex.RelayVeoVideo)                   // 提交视频生成请求
		veoV1Router.POST("/videos/operations/:operation_name", vertex.PollVeoOperation) // 轮询操作状态
	}
}
