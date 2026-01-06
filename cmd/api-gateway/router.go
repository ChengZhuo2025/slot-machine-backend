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
	"github.com/dumeirei/smart-locker-backend/internal/common/crypto"
	"github.com/dumeirei/smart-locker-backend/internal/common/jwt"
	"github.com/dumeirei/smart-locker-backend/internal/common/middleware"
	adminHandler "github.com/dumeirei/smart-locker-backend/internal/handler/admin"
	authHandler "github.com/dumeirei/smart-locker-backend/internal/handler/auth"
	deviceHandler "github.com/dumeirei/smart-locker-backend/internal/handler/device"
	distributionHandler "github.com/dumeirei/smart-locker-backend/internal/handler/distribution"
	hotelHandler "github.com/dumeirei/smart-locker-backend/internal/handler/hotel"
	mallHandler "github.com/dumeirei/smart-locker-backend/internal/handler/mall"
	marketingHandler "github.com/dumeirei/smart-locker-backend/internal/handler/marketing"
	orderHandler "github.com/dumeirei/smart-locker-backend/internal/handler/order"
	paymentHandler "github.com/dumeirei/smart-locker-backend/internal/handler/payment"
	rentalHandler "github.com/dumeirei/smart-locker-backend/internal/handler/rental"
	userHandler "github.com/dumeirei/smart-locker-backend/internal/handler/user"
	userMiddleware "github.com/dumeirei/smart-locker-backend/internal/middleware"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
	authService "github.com/dumeirei/smart-locker-backend/internal/service/auth"
	deviceService "github.com/dumeirei/smart-locker-backend/internal/service/device"
	distributionService "github.com/dumeirei/smart-locker-backend/internal/service/distribution"
	financeService "github.com/dumeirei/smart-locker-backend/internal/service/finance"
	hotelService "github.com/dumeirei/smart-locker-backend/internal/service/hotel"
	mallService "github.com/dumeirei/smart-locker-backend/internal/service/mall"
	marketingService "github.com/dumeirei/smart-locker-backend/internal/service/marketing"
	orderService "github.com/dumeirei/smart-locker-backend/internal/service/order"
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
	categoryRepo := repository.NewCategoryRepository(db)
	productRepo := repository.NewProductRepository(db)
	productSkuRepo := repository.NewProductSkuRepository(db)
	cartRepo := repository.NewCartRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	reviewRepo := repository.NewReviewRepository(db)

	// 酒店相关仓储
	hotelRepo := repository.NewHotelRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	roomTimeSlotRepo := repository.NewRoomTimeSlotRepository(db)
	bookingRepo := repository.NewBookingRepository(db)

	// 分销相关仓储
	distributorRepo := repository.NewDistributorRepository(db)
	commissionRepo := repository.NewCommissionRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)

	// 营销相关仓储
	couponRepo := repository.NewCouponRepository(db)
	userCouponRepo := repository.NewUserCouponRepository(db)
	campaignRepo := repository.NewCampaignRepository(db)

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

	// 商城服务
	productSvc := mallService.NewProductService(db, productRepo, categoryRepo, productSkuRepo)
	cartSvc := mallService.NewCartService(db, cartRepo, productRepo, productSkuRepo)
	mallOrderSvc := mallService.NewMallOrderService(db, orderRepo, cartRepo, productRepo, productSkuRepo, productSvc)
	reviewSvc := mallService.NewReviewService(db, reviewRepo, orderRepo)
	searchSvc := mallService.NewSearchService(db, productRepo)

	// 退款服务
	refundSvc := orderService.NewRefundService(db, refundRepo, orderRepo, paymentRepo)

	// 酒店服务
	hotelCodeSvc := hotelService.NewCodeService()
	hotelSvc := hotelService.NewHotelService(db, hotelRepo, roomRepo, roomTimeSlotRepo)
	bookingSvc := hotelService.NewBookingService(db, bookingRepo, roomRepo, hotelRepo, orderRepo, roomTimeSlotRepo, hotelCodeSvc, deviceSvc, nil)

	// 分销服务
	distributorSvc := distributionService.NewDistributorService(distributorRepo, userRepo, db)
	commissionSvc := distributionService.NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
	inviteSvc := distributionService.NewInviteService(distributorRepo, "") // BaseURL 在 InviteService 中有默认值
	withdrawSvc := distributionService.NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)

	// 营销服务
	couponSvc := marketingService.NewCouponService(db, couponRepo, userCouponRepo)
	userCouponSvc := marketingService.NewUserCouponService(db, couponRepo, userCouponRepo)
	campaignSvc := marketingService.NewCampaignService(campaignRepo)

	// 初始化处理器
	authH := authHandler.NewHandler(authSvc, wechatSvc, codeService)
	userH := userHandler.NewHandler(userSvc, walletSvc)
	deviceH := deviceHandler.NewHandler(deviceSvc, venueSvc)
	rentalH := rentalHandler.NewHandler(rentalSvc)
	paymentH := paymentHandler.NewHandler(paymentSvc)

	// 商城处理器
	mallProductH := mallHandler.NewProductHandler(productSvc, searchSvc)
	cartH := mallHandler.NewCartHandler(cartSvc)
	mallOrderH := mallHandler.NewOrderHandler(mallOrderSvc)
	reviewH := mallHandler.NewReviewHandler(reviewSvc)

	// 退款处理器
	refundH := orderHandler.NewRefundHandler(refundSvc)

	// 酒店处理器
	hotelH := hotelHandler.NewHandler(hotelSvc)
	bookingH := hotelHandler.NewBookingHandler(bookingSvc)

	// 分销处理器
	distributionH := distributionHandler.NewHandler(distributorSvc, commissionSvc, inviteSvc, withdrawSvc)

	// 营销处理器
	couponH := marketingHandler.NewCouponHandler(couponSvc, userCouponSvc)
	_ = campaignSvc // 活动服务将来用于扩展

	// 全局中间件
	r.Use(userMiddleware.Recovery(logger))
	r.Use(userMiddleware.RequestID())
	r.Use(userMiddleware.RealIP())
	r.Use(userMiddleware.CORS(nil))
	r.Use(userMiddleware.AccessLog(logger))

	// 健康检查（不需要认证）
	r.GET("/health", healthHandler)
	r.GET("/ping", pingHandler)
	r.GET("/ready", readyHandler(db, redisClient))

	// Swagger 文档
	// Swagger UI 实际读取的是 /swagger/doc.json。
	// Gin 不允许同时注册 /swagger/index.html 与 /swagger/*any（会冲突），所以这里用单一路由
	// /swagger/*any，在请求 index.html 时返回自定义页面，其余静态资源仍交给 gin-swagger。
	swaggerHandler := ginSwagger.WrapHandler(swaggerFiles.Handler)
	r.GET("/swagger/*any", func(c *gin.Context) {
		any := c.Param("any")
		if any == "" || any == "/" || any == "/index.html" {
			renderSwaggerIndex(c)
			return
		}
		swaggerHandler(c)
	})

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

			// 商城公开接口
			public.GET("/categories", mallProductH.GetCategories)
			public.GET("/products", mallProductH.GetProducts)
			public.GET("/products/:id", mallProductH.GetProductDetail)
			public.GET("/products/search", mallProductH.SearchProducts)
			public.GET("/search/hot-keywords", mallProductH.GetHotKeywords)
			public.GET("/search/suggestions", mallProductH.GetSearchSuggestions)
			public.GET("/products/:id/reviews", reviewH.GetProductReviews)
			public.GET("/products/:id/review-stats", reviewH.GetProductReviewStats)
		}

		// 支付回调（需要验签，不需要认证）
		paymentH.RegisterCallbackRoutes(v1)

		// 用户端接口（需要用户认证）
		user := v1.Group("")
		user.Use(userMiddleware.UserAuth(jwtManager))
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

			// 购物车
			user.GET("/cart", cartH.GetCart)
			user.POST("/cart", cartH.AddItem)
			user.PUT("/cart/:id", cartH.UpdateItem)
			user.DELETE("/cart/:id", cartH.RemoveItem)
			user.DELETE("/cart", cartH.ClearCart)
			user.PUT("/cart/select-all", cartH.SelectAll)
			user.GET("/cart/count", cartH.GetCartCount)

			// 商城订单
			user.GET("/orders", mallOrderH.GetOrders)
			user.POST("/orders", mallOrderH.CreateOrder)
			user.POST("/orders/from-cart", mallOrderH.CreateOrderFromCart)
			user.GET("/orders/:id", mallOrderH.GetOrderDetail)
			user.POST("/orders/:id/cancel", mallOrderH.CancelOrder)
			user.POST("/orders/:id/confirm", mallOrderH.ConfirmReceive)

			// 退款
			user.GET("/refunds", refundH.GetRefunds)
			user.POST("/refunds", refundH.CreateRefund)
			user.GET("/refunds/:id", refundH.GetRefundDetail)
			user.POST("/refunds/:id/cancel", refundH.CancelRefund)

			// 商品评价
			user.POST("/reviews", reviewH.CreateReview)
			user.GET("/user/reviews", reviewH.GetUserReviews)
			user.DELETE("/reviews/:id", reviewH.DeleteReview)

			// 优惠券
			marketing := user.Group("/marketing")
			{
				// 可领取的优惠券
				marketing.GET("/coupons", couponH.GetCouponList)
				marketing.GET("/coupons/:id", couponH.GetCouponDetail)
				marketing.POST("/coupons/:id/receive", couponH.ReceiveCoupon)

				// 用户优惠券
				marketing.GET("/user-coupons", couponH.GetUserCoupons)
				marketing.GET("/user-coupons/available", couponH.GetAvailableCoupons)
				marketing.GET("/user-coupons/for-order", couponH.GetAvailableCouponsForOrder)
				marketing.GET("/user-coupons/count", couponH.GetCouponCountByStatus)
				marketing.GET("/user-coupons/:id", couponH.GetUserCouponDetail)
			}

			// 酒店/预订相关
			user.GET("/hotels", hotelH.GetHotelList)
			user.GET("/hotels/cities", hotelH.GetCities)
			user.GET("/hotels/:id", hotelH.GetHotelDetail)
			user.GET("/hotels/:id/rooms", hotelH.GetRoomList)
			user.GET("/rooms/:id", hotelH.GetRoomDetail)
			user.GET("/rooms/:id/availability", hotelH.CheckRoomAvailability)
			user.GET("/rooms/:id/time-slots", hotelH.GetRoomTimeSlots)
			user.POST("/bookings", bookingH.CreateBooking)
			user.GET("/bookings", bookingH.GetMyBookings)
			user.GET("/bookings/:id", bookingH.GetBookingDetail)
			user.GET("/bookings/no/:booking_no", bookingH.GetBookingByNo)
			user.POST("/bookings/:id/cancel", bookingH.CancelBooking)
			user.POST("/bookings/unlock", bookingH.UnlockByCode)

			// 分销相关
			distribution := user.Group("/distribution")
			{
				distribution.GET("/check", distributionH.CheckStatus)
				distribution.POST("/apply", distributionH.Apply)
				distribution.GET("/info", distributionH.GetInfo)
				distribution.GET("/dashboard", distributionH.GetDashboard)
				distribution.GET("/team/stats", distributionH.GetTeamStats)
				distribution.GET("/team/members", distributionH.GetTeamMembers)
				distribution.GET("/invite", distributionH.GetInviteInfo)
				distribution.GET("/invite/validate", distributionH.ValidateInviteCode)
				distribution.GET("/share", distributionH.GetShareContent)
				distribution.GET("/commissions", distributionH.GetCommissions)
				distribution.GET("/commissions/stats", distributionH.GetCommissionStats)
				distribution.POST("/withdraw", distributionH.ApplyWithdraw)
				distribution.GET("/withdrawals", distributionH.GetWithdrawals)
				distribution.GET("/withdraw/config", distributionH.GetWithdrawConfig)
				distribution.GET("/ranking", distributionH.GetRanking)
			}

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
		// 初始化管理员相关仓储
		adminRepo := repository.NewAdminRepository(db)
		roleRepo := repository.NewRoleRepository(db)
		permissionRepo := repository.NewPermissionRepository(db)
		merchantRepo := repository.NewMerchantRepository(db)
		deviceLogRepo := repository.NewDeviceLogRepository(db)
		deviceMaintenanceRepo := repository.NewDeviceMaintenanceRepository(db)
		deviceAlertRepo := repository.NewDeviceAlertRepository(db)
		operationLogRepo := repository.NewOperationLogRepository(db)

		// 初始化 AES 加密器（用于敏感数据加密）
		aesEncryptor, _ := crypto.NewAES(cfg.Crypto.AESKey)

		// 初始化管理员服务
		adminAuthSvc := adminService.NewAdminAuthService(adminRepo, jwtManager)
		_ = adminService.NewPermissionService(roleRepo, permissionRepo, adminRepo)
		deviceAdminSvc := adminService.NewDeviceAdminService(deviceRepo, deviceLogRepo, deviceMaintenanceRepo, venueRepo, nil)
		venueAdminSvc := adminService.NewVenueAdminService(venueRepo, merchantRepo, deviceRepo)
		merchantAdminSvc := adminService.NewMerchantAdminService(merchantRepo, aesEncryptor)
		_ = adminService.NewDeviceAlertService(deviceRepo, deviceLogRepo, deviceAlertRepo) // 告警服务（后续集成使用）
		productAdminSvc := adminService.NewProductAdminService(db, categoryRepo, productRepo, productSkuRepo)
		hotelAdminSvc := adminService.NewHotelAdminService(db, hotelRepo, roomRepo, bookingRepo, roomTimeSlotRepo)
		distributionAdminSvc := adminService.NewDistributionAdminService(distributorRepo, commissionRepo, withdrawalRepo, db)
		marketingAdminSvc := adminService.NewMarketingAdminService(db, couponRepo, campaignRepo)

		// 初始化管理员处理器
		adminAuthH := adminHandler.NewAuthHandler(adminAuthSvc)
		deviceAdminH := adminHandler.NewDeviceHandler(deviceAdminSvc)
		venueAdminH := adminHandler.NewVenueHandler(venueAdminSvc)
		merchantAdminH := adminHandler.NewMerchantHandler(merchantAdminSvc)
		productAdminH := adminHandler.NewProductHandler(productAdminSvc)
		hotelAdminH := adminHandler.NewHotelHandler(hotelAdminSvc)
		bookingVerifyH := adminHandler.NewBookingVerifyHandler(bookingSvc)
		distributionAdminH := adminHandler.NewDistributionHandler(distributionAdminSvc)
		marketingAdminH := adminHandler.NewMarketingHandler(marketingAdminSvc)

		// 财务相关仓储和服务
		settlementRepo := repository.NewSettlementRepository(db)
		transactionRepo := repository.NewTransactionRepository(db)

		settlementSvc := financeService.NewSettlementService(db, settlementRepo, orderRepo, merchantRepo, commissionRepo, distributorRepo)
		statisticsSvc := financeService.NewStatisticsService(db, settlementRepo, transactionRepo, orderRepo, paymentRepo, commissionRepo, withdrawalRepo)
		withdrawalAuditSvc := financeService.NewWithdrawalAuditService(db, withdrawalRepo, distributorRepo)
		exportSvc := financeService.NewExportService(db, settlementRepo, transactionRepo, orderRepo, withdrawalRepo)

		financeAdminH := adminHandler.NewFinanceHandler(settlementSvc, statisticsSvc, withdrawalAuditSvc, exportSvc)

		// 操作日志中间件
		operationLogger := middleware.NewOperationLogger(operationLogRepo)

		// 管理员认证路由（公开）
		adminAuthH.RegisterRoutes(admin)

		// 需要管理员认证
		adminAuth := admin.Group("")
		adminAuth.Use(userMiddleware.AdminAuth(jwtManager))
		adminAuth.Use(operationLogger.Log())
		{
			// 认证保护路由
			adminAuthH.RegisterProtectedRoutes(adminAuth)

			// 设备管理
			deviceAdminH.RegisterRoutes(adminAuth)

			// 场地管理
			venueAdminH.RegisterRoutes(adminAuth)

			// 商户管理
			merchantAdminH.RegisterRoutes(adminAuth)

			// 以下为尚未实现的接口占位

			// 用户管理
			adminAuth.GET("/users", placeholderHandler("获取用户列表"))
			adminAuth.GET("/users/:id", placeholderHandler("获取用户详情"))
			adminAuth.PUT("/users/:id/status", placeholderHandler("更新用户状态"))

			// 订单管理
			adminAuth.GET("/orders", placeholderHandler("获取订单列表"))
			adminAuth.GET("/orders/:id", placeholderHandler("获取订单详情"))
			adminAuth.POST("/orders/:id/refund", placeholderHandler("发起退款"))

			// 租借管理
			adminAuth.GET("/rentals", placeholderHandler("获取租借列表"))
			adminAuth.GET("/rentals/:id", placeholderHandler("获取租借详情"))

			// 商品管理
			adminAuth.GET("/products", productAdminH.GetProducts)
			adminAuth.POST("/products", productAdminH.CreateProduct)
			adminAuth.GET("/products/:id", productAdminH.GetProductDetail)
			adminAuth.PUT("/products/:id", productAdminH.UpdateProduct)
			adminAuth.DELETE("/products/:id", productAdminH.DeleteProduct)
			adminAuth.PUT("/products/:id/status", productAdminH.UpdateProductStatus)

			// 分类管理
			adminAuth.GET("/categories", productAdminH.GetCategories)
			adminAuth.POST("/categories", productAdminH.CreateCategory)
			adminAuth.PUT("/categories/:id", productAdminH.UpdateCategory)
			adminAuth.DELETE("/categories/:id", productAdminH.DeleteCategory)

			// 酒店管理
			hotelAdminH.RegisterRoutes(adminAuth)

			// 预订核销
			bookingVerifyH.RegisterRoutes(adminAuth)

			// 营销管理
			marketingAdmin := adminAuth.Group("/marketing")
			{
				// 优惠券管理
				marketingAdmin.GET("/coupons", marketingAdminH.GetCouponList)
				marketingAdmin.POST("/coupons", marketingAdminH.CreateCoupon)
				marketingAdmin.GET("/coupons/:id", marketingAdminH.GetCouponDetail)
				marketingAdmin.PUT("/coupons/:id", marketingAdminH.UpdateCoupon)
				marketingAdmin.PUT("/coupons/:id/status", marketingAdminH.UpdateCouponStatus)
				marketingAdmin.DELETE("/coupons/:id", marketingAdminH.DeleteCoupon)

				// 活动管理
				marketingAdmin.GET("/campaigns", marketingAdminH.GetCampaignList)
				marketingAdmin.POST("/campaigns", marketingAdminH.CreateCampaign)
				marketingAdmin.GET("/campaigns/:id", marketingAdminH.GetCampaignDetail)
				marketingAdmin.PUT("/campaigns/:id", marketingAdminH.UpdateCampaign)
				marketingAdmin.PUT("/campaigns/:id/status", marketingAdminH.UpdateCampaignStatus)
				marketingAdmin.DELETE("/campaigns/:id", marketingAdminH.DeleteCampaign)
			}

			// 分销管理
			distAdmin := adminAuth.Group("/distribution")
			{
				distAdmin.GET("/stats", distributionAdminH.GetStats)
				distAdmin.GET("/distributors", distributionAdminH.ListDistributors)
				distAdmin.GET("/distributors/pending", distributionAdminH.GetPendingDistributors)
				distAdmin.GET("/distributors/:id", distributionAdminH.GetDistributor)
				distAdmin.POST("/distributors/:id/approve", distributionAdminH.ApproveDistributor)
				distAdmin.GET("/commissions", distributionAdminH.ListCommissions)
				distAdmin.GET("/withdrawals", distributionAdminH.ListWithdrawals)
				distAdmin.GET("/withdrawals/pending", distributionAdminH.GetPendingWithdrawals)
				distAdmin.GET("/withdrawals/:id", distributionAdminH.GetWithdrawal)
				distAdmin.POST("/withdrawals/:id/handle", distributionAdminH.HandleWithdrawal)
			}

			// 财务管理
			finance := adminAuth.Group("/finance")
			{
				// 概览和统计
				finance.GET("/overview", financeAdminH.GetOverview)
				finance.GET("/revenue/statistics", financeAdminH.GetRevenueStatistics)
				finance.GET("/revenue/daily", financeAdminH.GetDailyRevenueReport)
				finance.GET("/revenue/by-type", financeAdminH.GetOrderRevenueByType)
				finance.GET("/transactions/statistics", financeAdminH.GetTransactionStatistics)

				// 结算管理
				finance.GET("/settlements", financeAdminH.ListSettlements)
				finance.POST("/settlements", financeAdminH.CreateSettlement)
				finance.GET("/settlements/summary", financeAdminH.GetSettlementSummary)
				finance.POST("/settlements/generate", financeAdminH.GenerateSettlements)
				finance.GET("/settlements/:id", financeAdminH.GetSettlement)
				finance.POST("/settlements/:id/process", financeAdminH.ProcessSettlement)

				// 提现管理
				finance.GET("/withdrawals", financeAdminH.ListWithdrawals)
				finance.GET("/withdrawals/summary", financeAdminH.GetWithdrawalSummary)
				finance.POST("/withdrawals/batch", financeAdminH.BatchHandleWithdrawals)
				finance.GET("/withdrawals/:id", financeAdminH.GetWithdrawal)
				finance.POST("/withdrawals/:id/handle", financeAdminH.HandleWithdrawal)

				// 报表
				finance.GET("/reports/merchant-settlement", financeAdminH.GetMerchantSettlementReport)

				// 导出
				finance.GET("/export/settlements", financeAdminH.ExportSettlements)
				finance.GET("/export/withdrawals", financeAdminH.ExportWithdrawals)
				finance.GET("/export/daily-revenue", financeAdminH.ExportDailyRevenue)
				finance.GET("/export/merchant-settlement", financeAdminH.ExportMerchantSettlement)
				finance.GET("/export/transactions", financeAdminH.ExportTransactions)
			}

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
func renderSwaggerIndex(c *gin.Context) {
	const html = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width,initial-scale=1" />
  <title>Swagger UI</title>
  <link rel="stylesheet" type="text/css" href="./swagger-ui.css" />
  <link rel="stylesheet" type="text/css" href="./index.css" />
  <link rel="icon" type="image/png" href="./favicon-32x32.png" sizes="32x32" />
  <link rel="icon" type="image/png" href="./favicon-16x16.png" sizes="16x16" />
  <style>
    .api-stats {
      margin-left: auto;
      color: #fff;
      font-size: 14px;
      font-weight: 600;
      opacity: .95;
      white-space: nowrap;
    }
    .api-stats small { font-weight: 400; opacity: .85; }
  </style>
</head>

<body>
  <div class="swagger-ui">
    <div class="topbar">
      <div class="wrapper">
        <div class="topbar-wrapper">
          <a class="link" href="#">
            <span>Swagger</span>
          </a>
          <div id="api-stats" class="api-stats">API 接口总数：加载中…</div>
        </div>
      </div>
    </div>
  </div>

  <div id="swagger-ui"></div>

  <script src="./swagger-ui-bundle.js" charset="UTF-8"></script>
  <script src="./swagger-ui-standalone-preset.js" charset="UTF-8"></script>
  <script>
    async function updateApiStats() {
      try {
        // Swagger UI 默认读取 doc.json（相对路径：/swagger/doc.json）
        const res = await fetch('doc.json', { cache: 'no-store' });
        const spec = await res.json();

        const methods = ['get','post','put','delete','patch','options','head'];
        let ops = 0;
        let pathCount = 0;

        const paths = spec && spec.paths ? spec.paths : {};
        for (const p in paths) {
          if (!Object.prototype.hasOwnProperty.call(paths, p)) continue;
          pathCount++;
          const item = paths[p] || {};
          for (const m of methods) {
            if (item[m]) ops++;
          }
        }

        const el = document.getElementById('api-stats');
        if (el) {
          el.innerHTML = 'API 接口总数：' + ops + ' <small>(paths：' + pathCount + ')</small>';
        }
      } catch (e) {
        const el = document.getElementById('api-stats');
        if (el) el.textContent = 'API 接口总数：统计失败';
      }
    }

    window.onload = async function() {
      await updateApiStats();

      const ui = SwaggerUIBundle({
        url: 'doc.json',
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset
        ],
        layout: 'StandaloneLayout'
      });

      window.ui = ui;
    };
  </script>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(200, html)
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
