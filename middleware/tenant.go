package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func TenantScopeAuth(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantId := tenantIdFromRequest(c)
		if tenantId <= 0 {
			common.ApiErrorMsg(c, "tenant_id is required")
			c.Abort()
			return
		}
		userId := c.GetInt("id")
		if userId <= 0 {
			common.ApiErrorMsg(c, "user is not authenticated")
			c.Abort()
			return
		}
		if c.GetInt("role") >= common.RoleRootUser {
			common.SetContextKey(c, constant.ContextKeyScopeType, model.ScopeTenant)
			common.SetContextKey(c, constant.ContextKeyTenantId, tenantId)
			common.SetContextKey(c, constant.ContextKeyTenantMemberRole, model.TenantRoleOwner)
			c.Next()
			return
		}
		member, err := model.GetTenantMember(tenantId, userId)
		if err != nil || member.Status != model.TenantMemberStatusActive {
			common.ApiErrorMsg(c, "没有权限访问该租户")
			c.Abort()
			return
		}
		if !model.TenantRoleAllowed(member.Role, allowedRoles) {
			common.ApiErrorMsg(c, "没有该租户操作权限")
			c.Abort()
			return
		}
		common.SetContextKey(c, constant.ContextKeyScopeType, model.ScopeTenant)
		common.SetContextKey(c, constant.ContextKeyTenantId, tenantId)
		common.SetContextKey(c, constant.ContextKeyTenantMemberRole, member.Role)
		c.Next()
	}
}

func tenantIdFromRequest(c *gin.Context) int {
	for _, value := range []string{
		c.Param("tenant_id"),
		c.Param("tenantId"),
		c.Query("tenant_id"),
		c.GetHeader("X-Tenant-Id"),
		c.GetHeader("X-Tenant-ID"),
	} {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		id, err := strconv.Atoi(value)
		if err == nil {
			return id
		}
	}
	if id := c.GetInt(string(constant.ContextKeyTenantId)); id > 0 {
		return id
	}
	return 0
}

func TenantReadAuth() gin.HandlerFunc {
	return TenantScopeAuth()
}

func TenantWriteAuth() gin.HandlerFunc {
	return TenantScopeAuth(model.TenantRoleOwner, model.TenantRoleAdmin, model.TenantRoleOps, model.TenantRoleDeveloper)
}

func TenantFinanceAuth() gin.HandlerFunc {
	return TenantScopeAuth(model.TenantRoleOwner, model.TenantRoleAdmin, model.TenantRoleFinance)
}

func TenantOwnerAdminAuth() gin.HandlerFunc {
	return TenantScopeAuth(model.TenantRoleOwner, model.TenantRoleAdmin)
}

func abortTenantAuth(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, gin.H{"success": false, "message": message})
	c.Abort()
}
