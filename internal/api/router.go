package api

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/phoenix-panel/phoenix/internal/audit"
	"github.com/phoenix-panel/phoenix/internal/config"
	"github.com/phoenix-panel/phoenix/internal/middleware"
	"github.com/phoenix-panel/phoenix/internal/security"
	"github.com/phoenix-panel/phoenix/internal/service"
)

// Version is stamped into health responses; overridden at build time via ldflags.
var Version = "dev"

// Dependencies bundles everything the router needs to construct handlers.
type Dependencies struct {
	Cfg    *config.Config
	DB     *gorm.DB
	JWT    *security.JWTManager
	Audit  *audit.Logger
}

// NewRouter builds the fully-wired Gin engine.
func NewRouter(d Dependencies) *gin.Engine {
	if d.Cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Services.
	userSvc := service.NewUserService(d.DB)
	authSvc := service.NewAuthService(d.DB, d.JWT)
	nodeSvc := service.NewNodeService(d.DB)
	subSvc := service.NewSubscriptionService(d.DB, userSvc)

	// Handlers.
	authH := NewAuthHandler(authSvc, d.Audit)
	userH := NewUserHandler(userSvc, d.Audit, d.Cfg)
	nodeH := NewNodeHandler(nodeSvc, d.Audit)
	subH := NewSubscriptionHandler(subSvc)
	healthH := NewHealthHandler(d.DB, Version)

	r := gin.New()
	// Trust no proxy headers by default; operators set this explicitly if
	// behind a known reverse proxy. Prevents client IP spoofing for rate limits.
	_ = r.SetTrustedProxies(nil)

	// Global middleware chain.
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(corsMiddleware(d.Cfg))

	// Health checks (no rate limit, no auth).
	r.GET("/healthz", healthH.Live)
	r.GET("/readyz", healthH.Ready)

	// Web UI — serve the single-page dashboard from ./web/
	r.Static("/web", "./web")
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/web/")
	})
	r.GET("/web", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/web/")
	})

	// Public subscription endpoint — token-authenticated, lightly rate limited.
	r.GET("/sub/:token",
		middleware.RateLimit(d.Cfg.Rate.RPS, d.Cfg.Rate.Burst),
		subH.Get)

	// API group with general rate limiting.
	apiGroup := r.Group("/api")
	apiGroup.Use(middleware.RateLimit(d.Cfg.Rate.RPS, d.Cfg.Rate.Burst))

	admin := apiGroup.Group("/admin")

	// Login: stricter, dedicated rate limit to slow brute force (OWASP A07).
	admin.POST("/login",
		middleware.RateLimit(d.Cfg.Rate.LoginRPS, d.Cfg.Rate.LoginBurst),
		authH.Login)

	// Authenticated admin routes.
	auth := admin.Group("")
	auth.Use(middleware.Auth(d.JWT))
	{
		auth.GET("/me", authH.Me)
		auth.POST("/change-password", authH.ChangePassword)

		// Users.
		auth.GET("/users", userH.List)
		auth.POST("/users", userH.Create)
		auth.GET("/users/:id", userH.Get)
		auth.PATCH("/users/:id", userH.Update)
		auth.DELETE("/users/:id", userH.Delete)
		auth.POST("/users/:id/reset", userH.ResetTraffic)
		auth.POST("/users/:id/regenerate-sub", userH.RegenerateSub)

		// Nodes & inbounds (sudo-only mutations).
		auth.GET("/nodes", nodeH.ListNodes)
		auth.GET("/nodes/:id", nodeH.GetNode)
		auth.POST("/nodes", middleware.RequireSudo(), nodeH.CreateNode)
		auth.DELETE("/nodes/:id", middleware.RequireSudo(), nodeH.DeleteNode)
		auth.POST("/inbounds", middleware.RequireSudo(), nodeH.CreateInbound)
		auth.DELETE("/inbounds/:id", middleware.RequireSudo(), nodeH.DeleteInbound)
	}

	// 404 fallback in JSON.
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
	})

	return r
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	conf := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID", "Subscription-Userinfo"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	// "*" with credentials is invalid; gin-cors handles AllowAllOrigins safely
	// only without credentials, so switch behavior based on config.
	if len(cfg.CORS.Origins) == 1 && cfg.CORS.Origins[0] == "*" {
		conf.AllowAllOrigins = true
		conf.AllowCredentials = false
	} else {
		conf.AllowOrigins = cfg.CORS.Origins
	}
	return cors.New(conf)
}
