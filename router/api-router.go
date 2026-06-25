package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	// Import oauth package to register providers via init()
	_ "github.com/QuantumNous/new-api/oauth"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetApiRouter(router *gin.Engine) {
	apiRouter := router.Group("/api")
	apiRouter.Use(middleware.RouteTag("api"))
	apiRouter.Use(gzip.Gzip(gzip.DefaultCompression))
	apiRouter.Use(middleware.BodyStorageCleanup()) // 清理请求体存储
	apiRouter.Use(middleware.GlobalAPIRateLimit())
	anonymousRequestBodyLimit := middleware.AnonymousRequestBodyLimit()
	{
		apiRouter.GET("/setup", controller.GetSetup)
		apiRouter.POST("/setup", anonymousRequestBodyLimit, controller.PostSetup)
		apiRouter.GET("/status", controller.GetStatus)
		apiRouter.GET("/uptime/status", controller.GetUptimeKumaStatus)
		apiRouter.GET("/models", middleware.UserAuth(), controller.DashboardListModels)
		apiRouter.GET("/status/test", middleware.AdminAuth(), controller.TestStatus)
		apiRouter.GET("/notice", controller.GetNotice)
		apiRouter.GET("/user-agreement", controller.GetUserAgreement)
		apiRouter.GET("/privacy-policy", controller.GetPrivacyPolicy)
		apiRouter.GET("/about", controller.GetAbout)
		//apiRouter.GET("/midjourney", controller.GetMidjourney)
		apiRouter.GET("/home_page_content", controller.GetHomePageContent)
		apiRouter.GET("/pricing", middleware.HeaderNavModuleAuth("pricing"), controller.GetPricing)
		perfMetricsRoute := apiRouter.Group("/perf-metrics")
		perfMetricsRoute.Use(middleware.HeaderNavModulePublicOrUserAuth("pricing"))
		{
			perfMetricsRoute.GET("/summary", controller.GetPerfMetricsSummary)
			perfMetricsRoute.GET("", controller.GetPerfMetrics)
		}
		apiRouter.GET("/rankings", middleware.HeaderNavModuleAuth("rankings"), controller.GetRankings)
		apiRouter.GET("/verification", middleware.EmailVerificationRateLimit(), middleware.TurnstileCheck(), controller.SendEmailVerification)
		apiRouter.GET("/reset_password", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.SendPasswordResetEmail)
		apiRouter.POST("/user/reset", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.ResetPassword)
		// OAuth routes - specific routes must come before :provider wildcard
		apiRouter.GET("/oauth/state", middleware.CriticalRateLimit(), controller.GenerateOAuthCode)
		apiRouter.POST("/oauth/email/bind", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.EmailBind)
		// Non-standard OAuth (WeChat, Telegram) - keep original routes
		apiRouter.GET("/oauth/wechat", middleware.CriticalRateLimit(), controller.WeChatAuth)
		apiRouter.POST("/oauth/wechat/bind", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.WeChatBind)
		apiRouter.GET("/oauth/telegram/login", middleware.CriticalRateLimit(), controller.TelegramLogin)
		apiRouter.GET("/oauth/telegram/bind", middleware.CriticalRateLimit(), controller.TelegramBind)
		// Standard OAuth providers (GitHub, Discord, OIDC, LinuxDO) - unified route
		apiRouter.GET("/oauth/:provider", middleware.CriticalRateLimit(), controller.HandleOAuth)
		apiRouter.GET("/ratio_config", middleware.CriticalRateLimit(), controller.GetRatioConfig)

		apiRouter.POST("/stripe/webhook", anonymousRequestBodyLimit, controller.StripeWebhook)
		apiRouter.POST("/creem/webhook", anonymousRequestBodyLimit, controller.CreemWebhook)
		apiRouter.POST("/waffo/webhook", anonymousRequestBodyLimit, controller.WaffoWebhook)
		// :env separates test vs prod URLs so the operator can register each
		// in Pancake's matching webhook slot; handler enforces env match.
		apiRouter.POST("/waffo-pancake/webhook/:env", anonymousRequestBodyLimit, controller.WaffoPancakeWebhook)

		// Universal secure verification routes
		apiRouter.POST("/verify", middleware.UserAuth(), middleware.CriticalRateLimit(), controller.UniversalVerify)

		userRoute := apiRouter.Group("/user")
		{
			userRoute.POST("/register", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, middleware.TurnstileCheck(), controller.Register)
			userRoute.POST("/login", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, middleware.TurnstileCheck(), controller.Login)
			userRoute.POST("/login/2fa", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.Verify2FALogin)
			userRoute.POST("/passkey/login/begin", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.PasskeyLoginBegin)
			userRoute.POST("/passkey/login/finish", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.PasskeyLoginFinish)
			//userRoute.POST("/tokenlog", middleware.CriticalRateLimit(), controller.TokenLog)
			userRoute.GET("/logout", controller.Logout)
			userRoute.POST("/epay/notify", anonymousRequestBodyLimit, controller.EpayNotify)
			userRoute.GET("/epay/notify", controller.EpayNotify)
			userRoute.GET("/groups", controller.GetUserGroups)

			selfRoute := userRoute.Group("/")
			selfRoute.Use(middleware.UserAuth())
			{
				selfRoute.GET("/self/groups", controller.GetUserGroups)
				selfRoute.GET("/self", controller.GetSelf)
				selfRoute.GET("/models", controller.GetUserModels)
				selfRoute.PUT("/self", controller.UpdateSelf)
				selfRoute.DELETE("/self", controller.DeleteSelf)
				selfRoute.GET("/token", controller.GenerateAccessToken)
				selfRoute.GET("/passkey", controller.PasskeyStatus)
				selfRoute.POST("/passkey/register/begin", controller.PasskeyRegisterBegin)
				selfRoute.POST("/passkey/register/finish", controller.PasskeyRegisterFinish)
				selfRoute.POST("/passkey/verify/begin", controller.PasskeyVerifyBegin)
				selfRoute.POST("/passkey/verify/finish", controller.PasskeyVerifyFinish)
				selfRoute.DELETE("/passkey", controller.PasskeyDelete)
				selfRoute.GET("/aff", controller.GetAffCode)
				selfRoute.GET("/topup/info", controller.GetTopUpInfo)
				selfRoute.GET("/topup/self", controller.GetUserTopUps)
				selfRoute.POST("/topup", middleware.CriticalRateLimit(), controller.TopUp)
				selfRoute.POST("/pay", middleware.CriticalRateLimit(), controller.RequestEpay)
				selfRoute.POST("/amount", controller.RequestAmount)
				selfRoute.POST("/stripe/pay", middleware.CriticalRateLimit(), controller.RequestStripePay)
				selfRoute.POST("/stripe/amount", controller.RequestStripeAmount)
				selfRoute.POST("/creem/pay", middleware.CriticalRateLimit(), controller.RequestCreemPay)
				selfRoute.POST("/waffo/amount", controller.RequestWaffoAmount)
				selfRoute.POST("/waffo/pay", middleware.CriticalRateLimit(), controller.RequestWaffoPay)
				selfRoute.POST("/waffo-pancake/amount", controller.RequestWaffoPancakeAmount)
				selfRoute.POST("/waffo-pancake/pay", middleware.CriticalRateLimit(), controller.RequestWaffoPancakePay)
				selfRoute.POST("/aff_transfer", controller.TransferAffQuota)
				selfRoute.PUT("/setting", controller.UpdateUserSetting)

				// 2FA routes
				selfRoute.GET("/2fa/status", controller.Get2FAStatus)
				selfRoute.POST("/2fa/setup", controller.Setup2FA)
				selfRoute.POST("/2fa/enable", controller.Enable2FA)
				selfRoute.POST("/2fa/disable", controller.Disable2FA)
				selfRoute.POST("/2fa/backup_codes", controller.RegenerateBackupCodes)

				// Check-in routes
				selfRoute.GET("/checkin", controller.GetCheckinStatus)
				selfRoute.POST("/checkin", middleware.TurnstileCheck(), controller.DoCheckin)

				// Custom OAuth bindings
				selfRoute.GET("/oauth/bindings", controller.GetUserOAuthBindings)
				selfRoute.DELETE("/oauth/bindings/:provider_id", controller.UnbindCustomOAuth)
			}

			adminRoute := userRoute.Group("/")
			adminRoute.Use(middleware.AdminAuth())
			{
				adminRoute.GET("/", controller.GetAllUsers)
				adminRoute.GET("/topup", controller.GetAllTopUps)
				adminRoute.POST("/topup/complete", controller.AdminCompleteTopUp)
				adminRoute.GET("/search", controller.SearchUsers)
				adminRoute.GET("/:id/oauth/bindings", controller.GetUserOAuthBindingsByAdmin)
				adminRoute.DELETE("/:id/oauth/bindings/:provider_id", controller.UnbindCustomOAuthByAdmin)
				adminRoute.DELETE("/:id/bindings/:binding_type", controller.AdminClearUserBinding)
				adminRoute.GET("/:id", controller.GetUser)
				adminRoute.POST("/", controller.CreateUser)
				adminRoute.POST("/manage", controller.ManageUser)
				adminRoute.PUT("/", controller.UpdateUser)
				adminRoute.DELETE("/:id", controller.DeleteUser)
				adminRoute.DELETE("/:id/reset_passkey", controller.AdminResetPasskey)

				// Admin 2FA routes
				adminRoute.GET("/2fa/stats", controller.Admin2FAStats)
				adminRoute.DELETE("/:id/2fa", controller.AdminDisable2FA)
			}
		}

		// Subscription billing (plans, purchase, admin management)
		subscriptionRoute := apiRouter.Group("/subscription")
		subscriptionRoute.Use(middleware.UserAuth())
		{
			subscriptionRoute.GET("/plans", controller.GetSubscriptionPlans)
			subscriptionRoute.GET("/self", controller.GetSubscriptionSelf)
			subscriptionRoute.PUT("/self/preference", controller.UpdateSubscriptionPreference)
			subscriptionRoute.POST("/balance/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestBalancePay)
			subscriptionRoute.POST("/epay/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestEpay)
			subscriptionRoute.POST("/stripe/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestStripePay)
			subscriptionRoute.POST("/creem/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestCreemPay)
			subscriptionRoute.POST("/waffo-pancake/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestWaffoPancakePay)
		}
		subscriptionAdminRoute := apiRouter.Group("/subscription/admin")
		subscriptionAdminRoute.Use(middleware.AdminAuth())
		{
			subscriptionAdminRoute.GET("/plans", controller.AdminListSubscriptionPlans)
			subscriptionAdminRoute.POST("/plans", controller.AdminCreateSubscriptionPlan)
			subscriptionAdminRoute.PUT("/plans/:id", controller.AdminUpdateSubscriptionPlan)
			subscriptionAdminRoute.PATCH("/plans/:id", controller.AdminUpdateSubscriptionPlanStatus)
			subscriptionAdminRoute.POST("/bind", controller.AdminBindSubscription)

			// User subscription management (admin)
			subscriptionAdminRoute.GET("/users/:id/subscriptions", controller.AdminListUserSubscriptions)
			subscriptionAdminRoute.POST("/users/:id/subscriptions", controller.AdminCreateUserSubscription)
			subscriptionAdminRoute.POST("/user_subscriptions/:id/invalidate", controller.AdminInvalidateUserSubscription)
			subscriptionAdminRoute.DELETE("/user_subscriptions/:id", controller.AdminDeleteUserSubscription)
		}

		// Subscription payment callbacks (no auth)
		apiRouter.POST("/subscription/epay/notify", anonymousRequestBodyLimit, controller.SubscriptionEpayNotify)
		apiRouter.GET("/subscription/epay/notify", controller.SubscriptionEpayNotify)
		apiRouter.GET("/subscription/epay/return", controller.SubscriptionEpayReturn)
		apiRouter.POST("/subscription/epay/return", anonymousRequestBodyLimit, controller.SubscriptionEpayReturn)
		optionRoute := apiRouter.Group("/option")
		optionRoute.Use(middleware.RootAuth())
		{
			optionRoute.GET("/", controller.GetOptions)
			optionRoute.PUT("/", controller.UpdateOption)
			optionRoute.POST("/payment_compliance", controller.ConfirmPaymentCompliance)
			optionRoute.GET("/channel_affinity_cache", controller.GetChannelAffinityCacheStats)
			optionRoute.DELETE("/channel_affinity_cache", controller.ClearChannelAffinityCache)
			optionRoute.POST("/rest_model_ratio", controller.ResetModelRatio)
			optionRoute.POST("/migrate_console_setting", controller.MigrateConsoleSetting) // 用于迁移检测的旧键，下个版本会删除
			optionRoute.GET("/waffo-pancake/catalog", controller.ListWaffoPancakeCatalog)
			optionRoute.POST("/waffo-pancake/pair", controller.CreateWaffoPancakePair)
			optionRoute.POST("/waffo-pancake/save", controller.SaveWaffoPancake)
			optionRoute.POST("/waffo-pancake/subscription-product", controller.CreateWaffoPancakeSubscriptionProduct)
			optionRoute.GET("/waffo-pancake/subscription-product-options", controller.ListWaffoPancakeSubscriptionProductOptions)
		}

		// Custom OAuth provider management (root only)
		customOAuthRoute := apiRouter.Group("/custom-oauth-provider")
		customOAuthRoute.Use(middleware.RootAuth())
		{
			customOAuthRoute.POST("/discovery", controller.FetchCustomOAuthDiscovery)
			customOAuthRoute.GET("/", controller.GetCustomOAuthProviders)
			customOAuthRoute.GET("/:id", controller.GetCustomOAuthProvider)
			customOAuthRoute.POST("/", controller.CreateCustomOAuthProvider)
			customOAuthRoute.PUT("/:id", controller.UpdateCustomOAuthProvider)
			customOAuthRoute.DELETE("/:id", controller.DeleteCustomOAuthProvider)
		}
		performanceRoute := apiRouter.Group("/performance")
		performanceRoute.Use(middleware.RootAuth())
		{
			performanceRoute.GET("/stats", controller.GetPerformanceStats)
			performanceRoute.DELETE("/disk_cache", controller.ClearDiskCache)
			performanceRoute.POST("/reset_stats", controller.ResetPerformanceStats)
			performanceRoute.POST("/gc", controller.ForceGC)
			performanceRoute.GET("/logs", controller.GetLogFiles)
			performanceRoute.DELETE("/logs", controller.CleanupLogFiles)
		}
		ratioSyncRoute := apiRouter.Group("/ratio_sync")
		ratioSyncRoute.Use(middleware.RootAuth())
		{
			ratioSyncRoute.GET("/channels", controller.GetSyncableChannels)
			ratioSyncRoute.POST("/fetch", controller.FetchUpstreamRatios)
		}
		channelRoute := apiRouter.Group("/channel")
		channelRoute.Use(middleware.AdminAuth())
		{
			channelRoute.GET("/", controller.GetAllChannels)
			channelRoute.GET("/search", controller.SearchChannels)
			channelRoute.GET("/models", controller.ChannelListModels)
			channelRoute.GET("/models_enabled", controller.EnabledListModels)
			channelRoute.GET("/ops", controller.GetChannelOps)
			channelRoute.GET("/:id", controller.GetChannel)
			channelRoute.POST("/:id/key", middleware.RootAuth(), middleware.CriticalRateLimit(), middleware.DisableCache(), middleware.SecureVerificationRequired(), controller.GetChannelKey)
			channelRoute.GET("/test", controller.TestAllChannels)
			channelRoute.GET("/test/:id", controller.TestChannel)
			channelRoute.GET("/test_cache/:id", controller.ProbeChannelCache)
			channelRoute.GET("/update_balance", controller.UpdateAllChannelsBalance)
			channelRoute.GET("/update_balance/:id", controller.UpdateChannelBalance)
			channelRoute.POST("/", controller.AddChannel)
			channelRoute.PUT("/", controller.UpdateChannel)
			channelRoute.DELETE("/disabled", controller.DeleteDisabledChannel)
			channelRoute.POST("/tag/disabled", controller.DisableTagChannels)
			channelRoute.POST("/tag/enabled", controller.EnableTagChannels)
			channelRoute.PUT("/tag", controller.EditTagChannels)
			channelRoute.DELETE("/:id", controller.DeleteChannel)
			channelRoute.POST("/batch", controller.DeleteChannelBatch)
			channelRoute.POST("/fix", controller.FixChannelsAbilities)
			channelRoute.GET("/fetch_models/:id", controller.FetchUpstreamModels)
			channelRoute.POST("/fetch_models", middleware.RootAuth(), controller.FetchModels)
			channelRoute.POST("/:id/codex/refresh", controller.RefreshCodexChannelCredential)
			channelRoute.GET("/:id/codex/usage", controller.GetCodexChannelUsage)
			channelRoute.GET("/:id/codex/usage/reset-credits", controller.GetCodexChannelRateLimitResetCredits)
			channelRoute.POST("/:id/codex/usage/reset", controller.ResetCodexChannelUsage)
			channelRoute.POST("/ollama/pull", controller.OllamaPullModel)
			channelRoute.POST("/ollama/pull/stream", controller.OllamaPullModelStream)
			channelRoute.DELETE("/ollama/delete", controller.OllamaDeleteModel)
			channelRoute.GET("/ollama/version/:id", controller.OllamaVersion)
			channelRoute.POST("/batch/tag", controller.BatchSetChannelTag)
			channelRoute.GET("/tag/models", controller.GetTagModels)
			channelRoute.POST("/copy/:id", controller.CopyChannel)
			channelRoute.POST("/multi_key/manage", controller.ManageMultiKeys)
			channelRoute.POST("/upstream_updates/apply", controller.ApplyChannelUpstreamModelUpdates)
			channelRoute.POST("/upstream_updates/apply_all", controller.ApplyAllChannelUpstreamModelUpdates)
			channelRoute.POST("/upstream_updates/detect", controller.DetectChannelUpstreamModelUpdates)
			channelRoute.POST("/upstream_updates/detect_all", controller.DetectAllChannelUpstreamModelUpdates)
		}
		tokenRoute := apiRouter.Group("/token")
		tokenRoute.Use(middleware.UserAuth())
		{
			tokenRoute.GET("/", controller.GetAllTokens)
			tokenRoute.GET("/search", middleware.SearchRateLimit(), controller.SearchTokens)
			tokenRoute.GET("/:id", controller.GetToken)
			tokenRoute.POST("/:id/key", middleware.CriticalRateLimit(), middleware.DisableCache(), controller.GetTokenKey)
			tokenRoute.POST("/", controller.AddToken)
			tokenRoute.PUT("/", controller.UpdateToken)
			tokenRoute.DELETE("/:id", controller.DeleteToken)
			tokenRoute.POST("/batch", controller.DeleteTokenBatch)
			tokenRoute.POST("/batch/keys", middleware.CriticalRateLimit(), middleware.DisableCache(), controller.GetTokenKeysBatch)
		}

		usageRoute := apiRouter.Group("/usage")
		usageRoute.Use(middleware.CORS(), middleware.CriticalRateLimit())
		{
			tokenUsageRoute := usageRoute.Group("/token")
			tokenUsageRoute.Use(middleware.TokenAuthReadOnly())
			{
				tokenUsageRoute.GET("/", controller.GetTokenUsage)
			}
		}

		redemptionRoute := apiRouter.Group("/redemption")
		redemptionRoute.Use(middleware.AdminAuth())
		{
			redemptionRoute.GET("/", controller.GetAllRedemptions)
			redemptionRoute.GET("/search", controller.SearchRedemptions)
			redemptionRoute.GET("/:id", controller.GetRedemption)
			redemptionRoute.POST("/", controller.AddRedemption)
			redemptionRoute.PUT("/", controller.UpdateRedemption)
			redemptionRoute.DELETE("/invalid", controller.DeleteInvalidRedemption)
			redemptionRoute.DELETE("/:id", controller.DeleteRedemption)
		}
		logRoute := apiRouter.Group("/log")
		logRoute.GET("/", middleware.AdminAuth(), controller.GetAllLogs)
		logRoute.DELETE("/", middleware.AdminAuth(), controller.DeleteHistoryLogs)
		logRoute.GET("/stat", middleware.AdminAuth(), controller.GetLogsStat)
		logRoute.GET("/self/stat", middleware.UserAuth(), controller.GetLogsSelfStat)
		logRoute.GET("/channel_affinity_usage_cache", middleware.AdminAuth(), controller.GetChannelAffinityUsageCacheStats)
		logRoute.GET("/search", middleware.AdminAuth(), controller.SearchAllLogs)
		logRoute.GET("/self", middleware.UserAuth(), controller.GetUserLogs)
		logRoute.GET("/self/search", middleware.UserAuth(), middleware.SearchRateLimit(), controller.SearchUserLogs)

		dataRoute := apiRouter.Group("/data")
		dataRoute.GET("/", middleware.AdminAuth(), controller.GetAllQuotaDates)
		dataRoute.GET("/users", middleware.AdminAuth(), controller.GetQuotaDatesByUser)
		dataRoute.GET("/self", middleware.UserAuth(), controller.GetUserQuotaDates)
		dataRoute.GET("/flow", middleware.AdminAuth(), controller.GetAllFlowQuotaDates)
		dataRoute.GET("/flow/self", middleware.UserAuth(), controller.GetUserFlowQuotaDates)

		logRoute.Use(middleware.CORS(), middleware.CriticalRateLimit())
		{
			logRoute.GET("/token", middleware.TokenAuthReadOnly(), controller.GetLogByKey)
		}
		groupRoute := apiRouter.Group("/group")
		groupRoute.Use(middleware.AdminAuth())
		{
			groupRoute.GET("/", controller.GetGroups)
		}

		prefillGroupRoute := apiRouter.Group("/prefill_group")
		prefillGroupRoute.Use(middleware.AdminAuth())
		{
			prefillGroupRoute.GET("/", controller.GetPrefillGroups)
			prefillGroupRoute.POST("/", controller.CreatePrefillGroup)
			prefillGroupRoute.PUT("/", controller.UpdatePrefillGroup)
			prefillGroupRoute.DELETE("/:id", controller.DeletePrefillGroup)
		}

		mjRoute := apiRouter.Group("/mj")
		mjRoute.GET("/self", middleware.UserAuth(), controller.GetUserMidjourney)
		mjRoute.GET("/", middleware.AdminAuth(), controller.GetAllMidjourney)

		taskRoute := apiRouter.Group("/task")
		{
			taskRoute.GET("/self", middleware.UserAuth(), controller.GetUserTask)
			taskRoute.GET("/", middleware.AdminAuth(), controller.GetAllTask)
		}

		vendorRoute := apiRouter.Group("/vendors")
		vendorRoute.Use(middleware.AdminAuth())
		{
			vendorRoute.GET("/", controller.GetAllVendors)
			vendorRoute.GET("/search", controller.SearchVendors)
			vendorRoute.GET("/:id", controller.GetVendorMeta)
			vendorRoute.POST("/", controller.CreateVendorMeta)
			vendorRoute.PUT("/", controller.UpdateVendorMeta)
			vendorRoute.DELETE("/:id", controller.DeleteVendorMeta)
		}

		supplierRoute := apiRouter.Group("/suppliers")
		supplierRoute.Use(middleware.AdminAuth())
		{
			supplierRoute.GET("/", controller.GetSuppliers)
			supplierRoute.GET("/:id", controller.GetSupplier)
			supplierRoute.POST("/", controller.CreateSupplier)
			supplierRoute.PUT("/", controller.UpdateSupplier)
			supplierRoute.DELETE("/:id", controller.DeleteSupplier)
		}

		supplierAgreementRoute := apiRouter.Group("/supplier_agreements")
		supplierAgreementRoute.Use(middleware.AdminAuth())
		{
			supplierAgreementRoute.GET("/", controller.GetSupplierAgreements)
			supplierAgreementRoute.GET("/:id", controller.GetSupplierAgreement)
			supplierAgreementRoute.POST("/", controller.CreateSupplierAgreement)
			supplierAgreementRoute.PUT("/", controller.UpdateSupplierAgreement)
			supplierAgreementRoute.DELETE("/:id", controller.DeleteSupplierAgreement)
		}

		supplyCapacityRoute := apiRouter.Group("/supply_capacities")
		supplyCapacityRoute.Use(middleware.AdminAuth())
		{
			supplyCapacityRoute.GET("/", controller.GetSupplyCapacities)
			supplyCapacityRoute.GET("/:id", controller.GetSupplyCapacity)
			supplyCapacityRoute.POST("/refresh_usage", controller.RefreshSupplyCapacityUsage)
			supplyCapacityRoute.POST("/", controller.CreateSupplyCapacity)
			supplyCapacityRoute.PUT("/", controller.UpdateSupplyCapacity)
			supplyCapacityRoute.DELETE("/:id", controller.DeleteSupplyCapacity)
		}

		supplyCapacityTelemetryRoute := apiRouter.Group("/supply_capacity_telemetries")
		supplyCapacityTelemetryRoute.Use(middleware.AdminAuth())
		{
			supplyCapacityTelemetryRoute.GET("/", controller.GetSupplyCapacityTelemetries)
			supplyCapacityTelemetryRoute.POST("/collect", controller.CollectSupplyCapacityTelemetry)
			supplyCapacityTelemetryRoute.POST("/record", controller.RecordSupplyCapacityTelemetry)
			supplyCapacityTelemetryRoute.POST("/sweep", controller.SweepSupplyCapacityTelemetry)
		}

		supplyTelemetryAgentRoute := apiRouter.Group("/supply_telemetry_agents")
		supplyTelemetryAgentRoute.Use(middleware.AdminAuth())
		{
			supplyTelemetryAgentRoute.GET("/", controller.GetSupplyTelemetryAgents)
			supplyTelemetryAgentRoute.POST("/heartbeat", controller.RecordSupplyTelemetryAgentHeartbeat)
			supplyTelemetryAgentRoute.POST("/sweep_result", controller.RecordSupplyTelemetryAgentSweepResult)
		}

		supplyCostProfileRoute := apiRouter.Group("/supply_cost_profiles")
		supplyCostProfileRoute.Use(middleware.AdminAuth())
		{
			supplyCostProfileRoute.GET("/", controller.GetSupplyCostProfiles)
			supplyCostProfileRoute.POST("/record", controller.RecordSupplyCostProfile)
		}

		supplyPrepaidLotRoute := apiRouter.Group("/supply_prepaid_lots")
		supplyPrepaidLotRoute.Use(middleware.AdminAuth())
		{
			supplyPrepaidLotRoute.GET("/", controller.GetSupplyPrepaidLots)
			supplyPrepaidLotRoute.POST("/record", controller.RecordSupplyPrepaidLot)
			supplyPrepaidLotRoute.POST("/refresh_usage", controller.RefreshSupplyPrepaidLotUsage)
		}

		usageLedgerRoute := apiRouter.Group("/usage_ledgers")
		usageLedgerRoute.Use(middleware.AdminAuth())
		{
			usageLedgerRoute.GET("/", controller.GetUsageLedgers)
			usageLedgerRoute.GET("/request/:request_id", controller.GetUsageLedgerByRequestID)
		}

		reportRoute := apiRouter.Group("/reports")
		reportRoute.Use(middleware.AdminAuth())
		{
			reportRoute.GET("/margin_summary", controller.GetMarginSummary)
			reportRoute.GET("/quality_summary", controller.GetQualitySummary)
		}

		supplierScorecardRoute := apiRouter.Group("/supplier_scorecards")
		supplierScorecardRoute.Use(middleware.AdminAuth())
		{
			supplierScorecardRoute.GET("/", controller.GetSupplierScorecards)
			supplierScorecardRoute.POST("/generate", controller.GenerateSupplierScorecards)
		}

		supplierEvaluationRoute := apiRouter.Group("/supplier_evaluations")
		supplierEvaluationRoute.Use(middleware.AdminAuth())
		{
			supplierEvaluationRoute.GET("/", controller.GetSupplierEvaluations)
			supplierEvaluationRoute.POST("/generate", controller.GenerateSupplierEvaluations)
			supplierEvaluationRoute.POST("/:id/approve", controller.ApproveSupplierEvaluation)
			supplierEvaluationRoute.POST("/:id/reject", controller.RejectSupplierEvaluation)
			supplierEvaluationRoute.POST("/:id/apply", controller.ApplySupplierEvaluation)
		}

		supplierPostureRecommendationRoute := apiRouter.Group("/supplier_posture_recommendations")
		supplierPostureRecommendationRoute.Use(middleware.AdminAuth())
		{
			supplierPostureRecommendationRoute.GET("/", controller.GetSupplierPostureRecommendations)
			supplierPostureRecommendationRoute.POST("/generate", controller.GenerateSupplierPostureRecommendations)
			supplierPostureRecommendationRoute.POST("/:id/approve", controller.ApproveSupplierPostureRecommendation)
			supplierPostureRecommendationRoute.POST("/:id/reject", controller.RejectSupplierPostureRecommendation)
			supplierPostureRecommendationRoute.POST("/:id/apply", controller.ApplySupplierPostureRecommendation)
		}

		supplierRoutePreferenceRoute := apiRouter.Group("/supplier_route_preferences")
		supplierRoutePreferenceRoute.Use(middleware.AdminAuth())
		{
			supplierRoutePreferenceRoute.GET("/", controller.GetSupplierRoutePreferences)
			supplierRoutePreferenceRoute.POST("/activate", controller.ActivateSupplierRoutePreference)
			supplierRoutePreferenceRoute.POST("/:supplier_id/disable", controller.DisableSupplierRoutePreference)
		}

		slaContractRoute := apiRouter.Group("/sla_contracts")
		slaContractRoute.Use(middleware.AdminAuth())
		{
			slaContractRoute.GET("/", controller.GetSlaContracts)
			slaContractRoute.GET("/:id", controller.GetSlaContract)
			slaContractRoute.POST("/import", controller.ImportSlaContract)
		}

		slaProbePlanRoute := apiRouter.Group("/sla_probe_plans")
		slaProbePlanRoute.Use(middleware.AdminAuth())
		{
			slaProbePlanRoute.GET("/", controller.GetSlaProbePlans)
			slaProbePlanRoute.GET("/:id", controller.GetSlaProbePlan)
			slaProbePlanRoute.POST("/generate", controller.GenerateSlaProbePlan)
		}

		slaProbeRunRoute := apiRouter.Group("/sla_probe_runs")
		slaProbeRunRoute.Use(middleware.AdminAuth())
		{
			slaProbeRunRoute.GET("/", controller.GetSlaProbeRuns)
			slaProbeRunRoute.GET("/:id", controller.GetSlaProbeRun)
			slaProbeRunRoute.POST("/record", controller.RecordSlaProbeRun)
		}

		trafficProfileRoute := apiRouter.Group("/traffic_profiles")
		trafficProfileRoute.Use(middleware.AdminAuth())
		{
			trafficProfileRoute.GET("/", controller.GetTrafficProfiles)
			trafficProfileRoute.POST("/generate", controller.GenerateTrafficProfiles)
		}

		trafficForecastRoute := apiRouter.Group("/traffic_forecasts")
		trafficForecastRoute.Use(middleware.AdminAuth())
		{
			trafficForecastRoute.GET("/", controller.GetTrafficForecasts)
			trafficForecastRoute.POST("/generate", controller.GenerateTrafficForecasts)
		}

		supplyDecisionRoute := apiRouter.Group("/supply_decisions")
		supplyDecisionRoute.Use(middleware.AdminAuth())
		{
			supplyDecisionRoute.GET("/", controller.GetSupplyDecisions)
			supplyDecisionRoute.POST("/generate", controller.GenerateSupplyDecisions)
			supplyDecisionRoute.POST("/:id/approve", controller.ApproveSupplyDecision)
			supplyDecisionRoute.POST("/:id/reject", controller.RejectSupplyDecision)
		}

		supplyExpansionOpportunityRoute := apiRouter.Group("/supply_expansion_opportunities")
		supplyExpansionOpportunityRoute.Use(middleware.AdminAuth())
		{
			supplyExpansionOpportunityRoute.GET("/", controller.GetSupplyExpansionOpportunities)
			supplyExpansionOpportunityRoute.POST("/generate", controller.GenerateSupplyExpansionOpportunities)
		}

		pricingRecommendationRoute := apiRouter.Group("/pricing_recommendations")
		pricingRecommendationRoute.Use(middleware.AdminAuth())
		{
			pricingRecommendationRoute.GET("/", controller.GetPricingRecommendations)
			pricingRecommendationRoute.POST("/generate", controller.GeneratePricingRecommendations)
			pricingRecommendationRoute.POST("/:id/approve", controller.ApprovePricingRecommendation)
			pricingRecommendationRoute.POST("/:id/reject", controller.RejectPricingRecommendation)
		}

		operatingInsightRoute := apiRouter.Group("/operating_insights")
		operatingInsightRoute.Use(middleware.AdminAuth())
		{
			operatingInsightRoute.GET("/", controller.GetOperatingInsights)
			operatingInsightRoute.POST("/generate", controller.GenerateOperatingInsights)
			operatingInsightRoute.POST("/:id/acknowledge", controller.AcknowledgeOperatingInsight)
			operatingInsightRoute.POST("/:id/dismiss", controller.DismissOperatingInsight)
		}

		supplyActionPlanRoute := apiRouter.Group("/supply_action_plans")
		supplyActionPlanRoute.Use(middleware.AdminAuth())
		{
			supplyActionPlanRoute.GET("/", controller.GetSupplyActionPlans)
			supplyActionPlanRoute.POST("/generate", controller.GenerateSupplyActionPlans)
			supplyActionPlanRoute.POST("/:id/status", controller.UpdateSupplyActionPlanStatus)
		}

		supplyActionExecutionRoute := apiRouter.Group("/supply_action_executions")
		supplyActionExecutionRoute.Use(middleware.AdminAuth())
		{
			supplyActionExecutionRoute.GET("/", controller.GetSupplyActionExecutions)
			supplyActionExecutionRoute.POST("/record", controller.RecordSupplyActionExecution)
			supplyActionExecutionRoute.POST("/refresh_usage", controller.RefreshSupplyActionExecutionUsage)
		}

		supplyRoutingPolicyRoute := apiRouter.Group("/supply_routing_policies")
		supplyRoutingPolicyRoute.Use(middleware.AdminAuth())
		{
			supplyRoutingPolicyRoute.GET("/", controller.GetSupplyRoutingPolicies)
			supplyRoutingPolicyRoute.POST("/activate", controller.ActivateSupplyRoutingPolicy)
			supplyRoutingPolicyRoute.POST("/:id/disable", controller.DisableSupplyRoutingPolicy)
		}

		settlementStatementRoute := apiRouter.Group("/settlement_statements")
		settlementStatementRoute.Use(middleware.AdminAuth())
		{
			settlementStatementRoute.GET("/", controller.GetSettlementStatements)
			settlementStatementRoute.POST("/generate", controller.GenerateSettlementStatement)
			settlementStatementRoute.GET("/:id/items", controller.GetSettlementStatementItems)
			settlementStatementRoute.GET("/:id/items.csv", controller.ExportSettlementStatementItemsCSV)
			settlementStatementRoute.GET("/:id", controller.GetSettlementStatement)
		}

		modelsRoute := apiRouter.Group("/models")
		modelsRoute.Use(middleware.AdminAuth())
		{
			modelsRoute.GET("/sync_upstream/preview", controller.SyncUpstreamPreview)
			modelsRoute.POST("/sync_upstream", controller.SyncUpstreamModels)
			modelsRoute.GET("/missing", controller.GetMissingModels)
			modelsRoute.GET("/", controller.GetAllModelsMeta)
			modelsRoute.GET("/search", controller.SearchModelsMeta)
			modelsRoute.GET("/:id", controller.GetModelMeta)
			modelsRoute.POST("/", controller.CreateModelMeta)
			modelsRoute.PUT("/", controller.UpdateModelMeta)
			modelsRoute.DELETE("/:id", controller.DeleteModelMeta)
		}

		// Deployments (model deployment management)
		deploymentsRoute := apiRouter.Group("/deployments")
		deploymentsRoute.Use(middleware.AdminAuth())
		{
			deploymentsRoute.GET("/settings", controller.GetModelDeploymentSettings)
			deploymentsRoute.POST("/settings/test-connection", controller.TestIoNetConnection)
			deploymentsRoute.GET("/", controller.GetAllDeployments)
			deploymentsRoute.GET("/search", controller.SearchDeployments)
			deploymentsRoute.POST("/test-connection", controller.TestIoNetConnection)
			deploymentsRoute.GET("/hardware-types", controller.GetHardwareTypes)
			deploymentsRoute.GET("/locations", controller.GetLocations)
			deploymentsRoute.GET("/available-replicas", controller.GetAvailableReplicas)
			deploymentsRoute.POST("/price-estimation", controller.GetPriceEstimation)
			deploymentsRoute.GET("/check-name", controller.CheckClusterNameAvailability)
			deploymentsRoute.POST("/", controller.CreateDeployment)

			deploymentsRoute.GET("/:id", controller.GetDeployment)
			deploymentsRoute.GET("/:id/logs", controller.GetDeploymentLogs)
			deploymentsRoute.GET("/:id/containers", controller.ListDeploymentContainers)
			deploymentsRoute.GET("/:id/containers/:container_id", controller.GetContainerDetails)
			deploymentsRoute.PUT("/:id", controller.UpdateDeployment)
			deploymentsRoute.PUT("/:id/name", controller.UpdateDeploymentName)
			deploymentsRoute.POST("/:id/extend", controller.ExtendDeployment)
			deploymentsRoute.DELETE("/:id", controller.DeleteDeployment)
		}
	}
}
