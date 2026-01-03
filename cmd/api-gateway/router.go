// Package main 是应用程序入口
package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/config"
	"github.com/dumeirei/smart-locker-backend/internal/common/jwt"
	authHandler "github.com/dumeirei/smart-locker-backend/internal/handler/auth"
	deviceHandler "github.com/dumeirei/smart-locker-backend/internal/handler/device"
	paymentHandler "github.com/dumeirei/smart-locker-backend/internal/handler/payment"
	rentalHandler "github.com/dumeirei/smart-locker-backend/internal/handler/rental"
	userHandler "github.com/dumeirei/smart-locker-backend/internal/handler/user"
	"github.com/dumeirei/smart-locker-backend/internal/middleware"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	authService "github.com/dumeirei/smart-locker-backend/internal/service/auth"
	deviceService "github.com/dumeirei/smart-locker-backend/internal/service/device"
	paymentService "github.com/dumeirei/smart-locker-backend/internal/service/payment"
	rentalService "github.com/dumeirei/smart-locker-backend/internal/service/rental"
	userService "github.com/dumeirei/smart-locker-backend/internal/service/user"
	"github.com/dumeirei/smart-locker-backend/pkg/sms"
	"github.com/dumeirei/smart-locker-backend/pkg/wechatpay"
)

// setupRouter 设置路由
func setupRouter(
	r *gin.Engine,
	cfg *config.Config,
	logger *zap.Logger,
	db *gorm.DB,
	redisClient *redis.Client,
) {
	// 创建 JWT 管理器
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            cfg.JWT.Secret,
		AccessExpireTime:  cfg.JWT.AccessTokenDuration(),
		RefreshExpireTime: cfg.JWT.RefreshTokenDuration(),
		Issuer:            cfg.JWT.Issuer,
	})

	// 初始化仓储
	userRepo := repository.NewUserRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	rentalRepo := repository.NewRentalRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	refundRepo := repository.NewRefundRepository(db)

	// 初始化外部服务客户端
	smsClient := sms.NewMockClient(cfg.SMS.SignName) // 开发环境使用 Mock，生产环境使用阿里云
	wechatPayClient, _ := wechatpay.NewClient(&wechatpay.Config{})

	// 初始化服务
	codeService := authService.NewCodeService(redisClient, smsClient, &authService.CodeServiceConfig{
		CodeLength: 6,
		ExpireIn:   5 * time.Minute,
	})
	authSvc := authService.NewAuthService(db, userRepo, jwtManager, codeService)
	wechatSvc := authService.NewWechatService(&authService.WechatConfig{}, db, userRepo, jwtManager)

	userSvc := userService.NewUserService(db, userRepo)
	walletSvc := userService.NewWalletService(db, userRepo)

	deviceSvc := deviceService.NewDeviceService(db, deviceRepo, venueRepo)
	venueSvc := deviceService.NewVenueService(db, venueRepo, deviceRepo)

	rentalSvc := rentalService.NewRentalService(db, rentalRepo, deviceRepo, deviceSvc, walletSvc, nil)
	paymentSvc := paymentService.NewPaymentService(db, paymentRepo, refundRepo, rentalRepo, wechatPayClient)

	// 初始化处理器
	authH := authHandler.NewHandler(authSvc, wechatSvc, codeService)
	userH := userHandler.NewHandler(userSvc, walletSvc)
	deviceH := deviceHandler.NewHandler(deviceSvc, venueSvc)
	rentalH := rentalHandler.NewHandler(rentalSvc)
	paymentH := paymentHandler.NewHandler(paymentSvc)

	// 全局中间件
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.RequestID())
	r.Use(middleware.RealIP())
	r.Use(middleware.CORS(nil))
	r.Use(middleware.AccessLog(logger))

	// 健康检查（不需要认证）
	r.GET("/health", healthHandler)
	r.GET("/ping", pingHandler)
	r.GET("/ready", readyHandler(db, redisClient))

	// Swagger 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		// 公开接口（无需认证）
		public := v1.Group("")
		{
			// 注册认证路由
			authH.RegisterRoutes(public)

			// 设备和场地公开接口
			deviceH.RegisterRoutes(public)

			// 公开信息
			public.GET("/banners", placeholderHandler("获取轮播图"))
			public.GET("/articles", placeholderHandler("获取文章列表"))
			public.GET("/articles/:id", placeholderHandler("获取文章详情"))
			public.GET("/products", placeholderHandler("获取商品列表"))
			public.GET("/products/:id", placeholderHandler("获取商品详情"))
			public.GET("/categories", placeholderHandler("获取分类列表"))
		}

		// 支付回调（需要验签，不需要认证）
		paymentH.RegisterCallbackRoutes(v1)

		// 用户端接口（需要用户认证）
		user := v1.Group("")
		user.Use(middleware.UserAuth(jwtManager))
		{
			// 认证保护路由
			authH.RegisterProtectedRoutes(user)

			// 用户路由
			userH.RegisterRoutes(user)

			// 租借路由
			rentalH.RegisterRoutes(user)

			// 支付路由
			paymentH.RegisterRoutes(user)

			// 收货地址
			user.GET("/addresses", placeholderHandler("获取地址列表"))
			user.POST("/addresses", placeholderHandler("添加地址"))
			user.PUT("/addresses/:id", placeholderHandler("更新地址"))
			user.DELETE("/addresses/:id", placeholderHandler("删除地址"))

			// 订单相关
			user.GET("/orders", placeholderHandler("获取订单列表"))
			user.POST("/orders", placeholderHandler("创建订单"))
			user.GET("/orders/:id", placeholderHandler("获取订单详情"))
			user.POST("/orders/:id/cancel", placeholderHandler("取消订单"))
			user.POST("/orders/:id/confirm", placeholderHandler("确认收货"))

			// 购物车
			user.GET("/cart", placeholderHandler("获取购物车"))
			user.POST("/cart", placeholderHandler("添加到购物车"))
			user.PUT("/cart/:id", placeholderHandler("更新购物车项"))
			user.DELETE("/cart/:id", placeholderHandler("删除购物车项"))

			// 优惠券
			user.GET("/coupons", placeholderHandler("获取可用优惠券"))
			user.GET("/user/coupons", placeholderHandler("获取我的优惠券"))
			user.POST("/coupons/:id/receive", placeholderHandler("领取优惠券"))

			// 预订相关
			user.GET("/hotels", placeholderHandler("获取酒店列表"))
			user.GET("/hotels/:id", placeholderHandler("获取酒店详情"))
			user.GET("/hotels/:id/rooms", placeholderHandler("获取房间列表"))
			user.GET("/rooms/:id/slots", placeholderHandler("获取可用时段"))
			user.POST("/bookings", placeholderHandler("创建预订"))
			user.GET("/bookings", placeholderHandler("获取预订列表"))
			user.GET("/bookings/:id", placeholderHandler("获取预订详情"))
			user.POST("/bookings/:id/cancel", placeholderHandler("取消预订"))

			// 分销相关
			user.GET("/distributor", placeholderHandler("获取分销员信息"))
			user.POST("/distributor/apply", placeholderHandler("申请成为分销员"))
			user.GET("/distributor/commissions", placeholderHandler("获取佣金记录"))
			user.POST("/distributor/withdraw", placeholderHandler("申请提现"))

			// 反馈相关
			user.POST("/feedbacks", placeholderHandler("提交反馈"))
			user.GET("/feedbacks", placeholderHandler("获取我的反馈"))
		}

		// 设备接口（需要设备认证，后续实现）
		device := v1.Group("/device")
		{
			device.POST("/heartbeat", placeholderHandler("设备心跳"))
			device.POST("/status", placeholderHandler("上报状态"))
			device.POST("/event", placeholderHandler("上报事件"))
		}
	}

	// 管理后台 API
	admin := r.Group("/api/admin")
	{
		// 管理员登录（公开）
		admin.POST("/login", placeholderHandler("管理员登录"))

		// 需要管理员认证
		adminAuth := admin.Group("")
		adminAuth.Use(middleware.AdminAuth(jwtManager))
		{
			// 管理员信息
			adminAuth.GET("/profile", placeholderHandler("获取管理员信息"))
			adminAuth.PUT("/profile", placeholderHandler("更新管理员信息"))
			adminAuth.PUT("/password", placeholderHandler("修改密码"))

			// 用户管理
			adminAuth.GET("/users", placeholderHandler("获取用户列表"))
			adminAuth.GET("/users/:id", placeholderHandler("获取用户详情"))
			adminAuth.PUT("/users/:id/status", placeholderHandler("更新用户状态"))

			// 设备管理
			adminAuth.GET("/devices", placeholderHandler("获取设备列表"))
			adminAuth.POST("/devices", placeholderHandler("添加设备"))
			adminAuth.GET("/devices/:id", placeholderHandler("获取设备详情"))
			adminAuth.PUT("/devices/:id", placeholderHandler("更新设备"))
			adminAuth.DELETE("/devices/:id", placeholderHandler("删除设备"))
			adminAuth.POST("/devices/:id/unlock", placeholderHandler("远程开锁"))

			// 场地管理
			adminAuth.GET("/venues", placeholderHandler("获取场地列表"))
			adminAuth.POST("/venues", placeholderHandler("添加场地"))
			adminAuth.PUT("/venues/:id", placeholderHandler("更新场地"))
			adminAuth.DELETE("/venues/:id", placeholderHandler("删除场地"))

			// 商户管理
			adminAuth.GET("/merchants", placeholderHandler("获取商户列表"))
			adminAuth.POST("/merchants", placeholderHandler("添加商户"))
			adminAuth.PUT("/merchants/:id", placeholderHandler("更新商户"))

			// 订单管理
			adminAuth.GET("/orders", placeholderHandler("获取订单列表"))
			adminAuth.GET("/orders/:id", placeholderHandler("获取订单详情"))
			adminAuth.POST("/orders/:id/refund", placeholderHandler("发起退款"))

			// 租借管理
			adminAuth.GET("/rentals", placeholderHandler("获取租借列表"))
			adminAuth.GET("/rentals/:id", placeholderHandler("获取租借详情"))

			// 商品管理
			adminAuth.GET("/products", placeholderHandler("获取商品列表"))
			adminAuth.POST("/products", placeholderHandler("添加商品"))
			adminAuth.PUT("/products/:id", placeholderHandler("更新商品"))
			adminAuth.DELETE("/products/:id", placeholderHandler("删除商品"))

			// 分类管理
			adminAuth.GET("/categories", placeholderHandler("获取分类列表"))
			adminAuth.POST("/categories", placeholderHandler("添加分类"))
			adminAuth.PUT("/categories/:id", placeholderHandler("更新分类"))
			adminAuth.DELETE("/categories/:id", placeholderHandler("删除分类"))

			// 酒店管理
			adminAuth.GET("/hotels", placeholderHandler("获取酒店列表"))
			adminAuth.POST("/hotels", placeholderHandler("添加酒店"))
			adminAuth.PUT("/hotels/:id", placeholderHandler("更新酒店"))
			adminAuth.DELETE("/hotels/:id", placeholderHandler("删除酒店"))

			// 房间管理
			adminAuth.GET("/hotels/:hotel_id/rooms", placeholderHandler("获取房间列表"))
			adminAuth.POST("/hotels/:hotel_id/rooms", placeholderHandler("添加房间"))
			adminAuth.PUT("/rooms/:id", placeholderHandler("更新房间"))
			adminAuth.DELETE("/rooms/:id", placeholderHandler("删除房间"))

			// 预订管理
			adminAuth.GET("/bookings", placeholderHandler("获取预订列表"))
			adminAuth.GET("/bookings/:id", placeholderHandler("获取预订详情"))

			// 营销管理
			adminAuth.GET("/coupons", placeholderHandler("获取优惠券列表"))
			adminAuth.POST("/coupons", placeholderHandler("添加优惠券"))
			adminAuth.PUT("/coupons/:id", placeholderHandler("更新优惠券"))
			adminAuth.DELETE("/coupons/:id", placeholderHandler("删除优惠券"))

			adminAuth.GET("/campaigns", placeholderHandler("获取活动列表"))
			adminAuth.POST("/campaigns", placeholderHandler("添加活动"))
			adminAuth.PUT("/campaigns/:id", placeholderHandler("更新活动"))
			adminAuth.DELETE("/campaigns/:id", placeholderHandler("删除活动"))

			// 分销管理
			adminAuth.GET("/distributors", placeholderHandler("获取分销员列表"))
			adminAuth.PUT("/distributors/:id/status", placeholderHandler("更新分销员状态"))
			adminAuth.GET("/withdrawals", placeholderHandler("获取提现列表"))
			adminAuth.PUT("/withdrawals/:id/approve", placeholderHandler("审核提现"))

			// 财务管理
			adminAuth.GET("/settlements", placeholderHandler("获取结算列表"))
			adminAuth.POST("/settlements/:id/settle", placeholderHandler("执行结算"))

			// 系统管理
			adminAuth.GET("/admins", placeholderHandler("获取管理员列表"))
			adminAuth.POST("/admins", placeholderHandler("添加管理员"))
			adminAuth.PUT("/admins/:id", placeholderHandler("更新管理员"))
			adminAuth.DELETE("/admins/:id", placeholderHandler("删除管理员"))

			adminAuth.GET("/roles", placeholderHandler("获取角色列表"))
			adminAuth.POST("/roles", placeholderHandler("添加角色"))
			adminAuth.PUT("/roles/:id", placeholderHandler("更新角色"))
			adminAuth.DELETE("/roles/:id", placeholderHandler("删除角色"))

			adminAuth.GET("/permissions", placeholderHandler("获取权限列表"))

			adminAuth.GET("/configs", placeholderHandler("获取系统配置"))
			adminAuth.PUT("/configs", placeholderHandler("更新系统配置"))

			adminAuth.GET("/banners", placeholderHandler("获取轮播图列表"))
			adminAuth.POST("/banners", placeholderHandler("添加轮播图"))
			adminAuth.PUT("/banners/:id", placeholderHandler("更新轮播图"))
			adminAuth.DELETE("/banners/:id", placeholderHandler("删除轮播图"))

			adminAuth.GET("/articles", placeholderHandler("获取文章列表"))
			adminAuth.POST("/articles", placeholderHandler("添加文章"))
			adminAuth.PUT("/articles/:id", placeholderHandler("更新文章"))
			adminAuth.DELETE("/articles/:id", placeholderHandler("删除文章"))

			adminAuth.GET("/feedbacks", placeholderHandler("获取反馈列表"))
			adminAuth.PUT("/feedbacks/:id/reply", placeholderHandler("回复反馈"))

			adminAuth.GET("/logs/operation", placeholderHandler("获取操作日志"))
			adminAuth.GET("/logs/device", placeholderHandler("获取设备日志"))
		}
	}

	// 404 处理
	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "接口不存在",
		})
	})
}

// placeholderHandler 占位处理器（待实现的接口）
func placeholderHandler(description string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(501, gin.H{
			"code":    501,
			"message": "接口待实现: " + description,
		})
	}
}
