package httpapi

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	backenddocs "didcontract/backend/docs"
	"didcontract/backend/internal/config"
	"didcontract/backend/internal/domain"
	"didcontract/backend/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	actorKey      = "actorAlias"
	loginActorKey = "loginActorAlias"
	adminKey      = "sessionAdmin"
	requestKey    = "requestID"
)

type principal struct {
	alias string
	admin bool
}

type API struct {
	service *service.Service
	tokens  map[string]principal
	actors  map[string]struct{}
	origins map[string]struct{}
}

func New(cfg config.Config, application *service.Service) *API {
	tokens := make(map[string]principal, len(cfg.Actors))
	actors := make(map[string]struct{}, len(cfg.Actors))
	for _, actor := range cfg.Actors {
		tokens[actor.TokenSHA256] = principal{alias: actor.Alias, admin: actor.Admin}
		actors[actor.Alias] = struct{}{}
	}
	origins := make(map[string]struct{}, len(cfg.Server.AllowedOrigins))
	for _, origin := range cfg.Server.AllowedOrigins {
		origins[origin] = struct{}{}
	}
	return &API{service: application, tokens: tokens, actors: actors, origins: origins}
}

func (a *API) Router() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery(), a.requestID(), a.cors(), a.bodyLimit(2<<20))
	router.GET("/swagger", func(c *gin.Context) { c.Redirect(http.StatusTemporaryRedirect, "/swagger/") })
	router.GET("/swagger/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(backenddocs.SwaggerHTML))
	})
	router.GET("/swagger/openapi.yaml", func(c *gin.Context) { c.Data(http.StatusOK, "application/yaml; charset=utf-8", backenddocs.OpenAPI) })
	router.GET("/api/health", func(c *gin.Context) {
		success(c, http.StatusOK, service.Result{Data: map[string]string{"status": "up"}})
	})

	api := router.Group("/api")
	api.Use(a.authenticate(), a.audit())
	api.GET("/ready", a.ready)
	api.GET("/system/session", a.sessionInfo)
	api.GET("/system/status", a.systemStatus)
	api.POST("/system/governance", a.setGovernance)
	api.PUT("/system/roles", a.updateRoles)
	api.GET("/system/roles", a.getRoles)
	api.GET("/system/audit", a.audits)

	did := api.Group("/did")
	did.POST("/generateDID", a.generateDID)
	did.POST("/registerDID", a.registerDID)
	did.GET("/query", a.queryDID)
	did.GET("/list", a.listDIDs)
	did.PUT("/update", a.rotateDID)
	did.DELETE("/deactivate", a.deactivateDID)

	policy := api.Group("/policy")
	policy.POST("/register", a.registerPolicy)
	policy.GET("/query", a.queryPolicy)
	policy.POST("/deactivate", a.deactivatePolicy)

	vc := api.Group("/vc")
	vc.POST("/issue", a.issueVC)
	vc.GET("/query", a.queryVC)
	vc.POST("/verify", a.verifyVC)

	vp := api.Group("/vp")
	vp.POST("/nonce", a.issueNonce)
	vp.POST("/generate", a.generateVP)
	vp.POST("/verify", a.verifyVP)
	vp.POST("/verifyPolicy", a.verifyPolicy)

	privacy := api.Group("/privacy")
	privacy.POST("/group/create", a.createGroup)
	privacy.POST("/group/member", a.addGroupMember)
	privacy.GET("/group", a.queryGroup)
	privacy.PUT("/group/status", a.setGroupStatus)
	privacy.POST("/vp/generate", a.generatePrivacyVP)
	privacy.POST("/vp/verify", a.verifyPrivacyVP)
	privacy.POST("/keyimage", a.computeKeyImage)
	privacy.GET("/keyimage", a.checkKeyImage)

	revocation := api.Group("/revocation")
	revocation.POST("/issuer/register", a.registerEventIssuer)
	revocation.DELETE("/issuer", a.removeEventIssuer)
	revocation.GET("/issuer", a.getEventIssuer)
	revocation.POST("/committee/create", a.createCommittee)
	revocation.POST("/committee/member", a.addCommitteeMember)
	revocation.DELETE("/committee/member", a.removeCommitteeMember)
	revocation.GET("/committee", a.queryCommittee)
	revocation.GET("/committee/rotations", a.rotationLogs)
	revocation.POST("/threshold/update", a.updateThreshold)
	revocation.POST("/requests", a.createRevocationDraft)
	revocation.GET("/requests/:id", a.getRevocationDraft)
	revocation.POST("/requests/:id/approvals", a.approveRevocation)
	revocation.POST("/execute", a.executeRevocation)
	revocation.GET("/logs", a.revocationLogs)
	revocation.GET("/consumed", a.credentialConsumed)
	return router
}

func (a *API) authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
			failure(c, domain.NewError("UNAUTHORIZED", http.StatusUnauthorized, "缺少 Bearer token"))
			c.Abort()
			return
		}
		token := strings.TrimSpace(header[len("Bearer "):])
		hash := sha256.Sum256([]byte(token))
		hashText := hex.EncodeToString(hash[:])
		current := principal{}
		for expected, candidate := range a.tokens {
			if len(expected) == len(hashText) && subtle.ConstantTimeCompare([]byte(expected), []byte(hashText)) == 1 {
				current = candidate
			}
		}
		if current.alias == "" {
			failure(c, domain.NewError("UNAUTHORIZED", http.StatusUnauthorized, "Bearer token 无效"))
			c.Abort()
			return
		}
		executionAlias := strings.TrimSpace(c.GetHeader("X-Actor-Alias"))
		if executionAlias == "" {
			executionAlias = current.alias
		}
		if executionAlias != current.alias && !current.admin {
			failure(c, domain.NewError("ACTOR_SWITCH_FORBIDDEN", http.StatusForbidden, "当前登录主体不能切换链上执行身份"))
			c.Abort()
			return
		}
		if _, exists := a.actors[executionAlias]; !exists {
			failure(c, domain.NewError("ACTOR_NOT_FOUND", http.StatusBadRequest, "指定的链上执行身份不存在"))
			c.Abort()
			return
		}
		c.Set(actorKey, executionAlias)
		c.Set(loginActorKey, current.alias)
		c.Set(adminKey, current.admin)
		c.Next()
	}
}

func (a *API) requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if requestID == "" {
			requestID = time.Now().UTC().Format("20060102T150405.000000000")
		}
		c.Set(requestKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func (a *API) audit() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		alias := actorAlias(c)
		actor, _ := a.service.Actor(alias)
		a.service.AppendAudit(domain.AuditEntry{Time: time.Now().Unix(), RequestID: requestID(c), Actor: alias, LoginActor: loginActorAlias(c), DID: actor.DID, Action: c.Request.Method + " " + c.FullPath(), Success: c.Writer.Status() < 400})
	}
}

func (a *API) cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := a.origins[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, X-Actor-Alias")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (a *API) bodyLimit(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) { c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit); c.Next() }
}

func actorAlias(c *gin.Context) string {
	value, _ := c.Get(actorKey)
	alias, _ := value.(string)
	return alias
}
func loginActorAlias(c *gin.Context) string {
	value, _ := c.Get(loginActorKey)
	alias, _ := value.(string)
	return alias
}
func sessionAdmin(c *gin.Context) bool {
	value, _ := c.Get(adminKey)
	admin, _ := value.(bool)
	return admin
}
func requestID(c *gin.Context) string {
	value, _ := c.Get(requestKey)
	id, _ := value.(string)
	return id
}

func success(c *gin.Context, status int, result service.Result) {
	response := gin.H{"code": "OK", "message": "success", "data": result.Data, "requestId": requestID(c)}
	if result.Tx != nil {
		response["tx"] = result.Tx
	}
	c.JSON(status, response)
}

func failure(c *gin.Context, err error) {
	appError, ok := err.(*domain.Error)
	if !ok {
		appError = domain.WrapError("INTERNAL_ERROR", http.StatusInternalServerError, "服务器内部错误", err)
	}
	c.JSON(appError.Status, gin.H{"code": appError.Code, "message": appError.Message, "requestId": requestID(c)})
}

func bind(c *gin.Context, input interface{}) bool {
	if err := c.ShouldBindJSON(input); err != nil {
		failure(c, domain.WrapError("INVALID_JSON", http.StatusBadRequest, "请求 JSON 无效", err))
		return false
	}
	return true
}
