package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/router"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func initAPIOnlyResources() error {
	_ = godotenv.Load(".env")
	common.InitEnv()
	logger.SetupLogger()
	ratio_setting.InitRatioSettings()
	service.InitHttpClient()
	service.InitTokenEncoders()

	if err := model.InitDB(); err != nil {
		return fmt.Errorf("init database: %w", err)
	}
	model.CheckSetup()
	model.InitOptionMap()
	model.GetPricing()
	if err := model.InitLogDB(); err != nil {
		return fmt.Errorf("init log database: %w", err)
	}
	if err := common.InitRedisClient(); err != nil {
		return fmt.Errorf("init redis: %w", err)
	}
	perfmetrics.Init()
	if err := i18n.Init(); err != nil {
		common.SysError("failed to initialize i18n: " + err.Error())
	} else {
		common.SysLog("i18n initialized with languages: " + strings.Join(i18n.SupportedLanguages(), ", "))
	}
	i18n.SetUserLangLoader(model.GetUserLanguage)
	return nil
}

func buildAPIOnlyRouter() *gin.Engine {
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(gin.CustomRecovery(func(c *gin.Context, err any) {
		common.SysLog(fmt.Sprintf("panic detected: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": fmt.Sprintf("panic detected: %v", err),
				"type":    "token_router_panic",
			},
		})
	}))
	engine.Use(middleware.RequestId())
	engine.Use(middleware.PoweredBy())
	engine.Use(middleware.I18n())
	middleware.SetUpLogger(engine)
	store := cookie.NewStore([]byte(common.SessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   2592000,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})
	engine.Use(sessions.Sessions("session", store))

	router.SetRouter(engine)
	return engine
}

func initAPIOnlyChannelCache() {
	if common.RedisEnabled {
		common.MemoryCacheEnabled = true
	}
	if !common.MemoryCacheEnabled {
		return
	}
	common.SysLog("memory cache enabled")
	common.SysLog(fmt.Sprintf("sync frequency: %d seconds", common.SyncFrequency))
	func() {
		defer func() {
			if r := recover(); r != nil {
				common.SysLog(fmt.Sprintf("InitChannelCache panic: %v, retrying once", r))
				if _, _, err := model.FixAbility(); err != nil {
					common.FatalLog(fmt.Sprintf("InitChannelCache failed: %s", err.Error()))
				}
				model.InitChannelCache()
			}
		}()
		model.InitChannelCache()
	}()
	go model.SyncChannelCache(common.SyncFrequency)
}

func main() {
	start := time.Now()
	if err := initAPIOnlyResources(); err != nil {
		common.FatalLog("failed to initialize API-only resources: " + err.Error())
	}
	initAPIOnlyChannelCache()
	service.StartSupplyTelemetryWorker()
	service.StartUsageRecordOutboxWorker()
	defer func() {
		if err := model.CloseDB(); err != nil {
			common.FatalLog("failed to close database: " + err.Error())
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = strconv.Itoa(*common.Port)
	}
	common.SysLog(fmt.Sprintf("token-router API-only server ready in %s on port %s", time.Since(start).Round(time.Millisecond), port))
	if err := buildAPIOnlyRouter().Run(":" + port); err != nil {
		common.FatalLog("failed to start API-only server: " + err.Error())
	}
}
