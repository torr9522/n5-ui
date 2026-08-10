package n5

import (
	"github.com/gin-gonic/gin"
	"strconv"
	n5model "x-ui/database/model/n5"
	n5service "x-ui/web/service/n5"
)

type TrafficPolicyController struct {
	policyService n5service.TrafficPolicyService
	detailService n5service.TrafficPolicyDetailService
	xrayExt       n5service.XrayExtService
}

func NewTrafficPolicyController(g *gin.RouterGroup) *TrafficPolicyController {
	a := &TrafficPolicyController{}
	a.initRouter(g)
	return a
}

func (a *TrafficPolicyController) initRouter(g *gin.RouterGroup) {
	pageGroup := g.Group("/n5")
	pageGroup.Use(checkLogin)
	pageGroup.GET("/traffic-policy", a.page)
	pageGroup.GET("/traffic-policy-detail", a.detailPage)

	apiGroup := g.Group("/n5/api/traffic-policy")
	apiGroup.Use(checkLogin)
	apiGroup.POST("/list", a.list)
	apiGroup.POST("/add", a.add)
	apiGroup.GET("/get/:id", a.get)
	apiGroup.POST("/update/:id", a.update)
	apiGroup.POST("/del/:id", a.del)
	apiGroup.POST("/enable/:id", a.enable)
	apiGroup.POST("/disable/:id", a.disable)
	apiGroup.POST("/rule/list/:id", a.listRules)
	apiGroup.POST("/rule/add", a.addRule)
	apiGroup.POST("/rule/update/:id", a.updateRule)
	apiGroup.POST("/rule/del/:id", a.delRule)
	apiGroup.POST("/rule/enable/:id", a.enableRule)
	apiGroup.POST("/rule/disable/:id", a.disableRule)
	apiGroup.POST("/rule/reorder", a.reorderRules)
	apiGroup.POST("/binding/list", a.listBindings)
	apiGroup.POST("/bind", a.bind)
	apiGroup.POST("/unbind", a.unbind)
	apiGroup.POST("/rebind", a.rebind)
	apiGroup.POST("/fragments", a.fragments)
}

func (a *TrafficPolicyController) page(c *gin.Context) {
	html(c, "traffic_policy.html", "流量分流", nil)
}

func (a *TrafficPolicyController) detailPage(c *gin.Context) {
	html(c, "traffic_policy_detail.html", "策略详情", nil)
}

func (a *TrafficPolicyController) list(c *gin.Context) {
	records, err := a.policyService.List()
	if err != nil {
		jsonMsg(c, "list traffic policy", err)
		return
	}
	jsonObj(c, records, nil)
}

func (a *TrafficPolicyController) add(c *gin.Context) {
	record := &n5model.TrafficPolicy{}
	if err := c.ShouldBind(record); err != nil {
		jsonMsg(c, "add traffic policy", err)
		return
	}
	created, err := a.policyService.Create(record)
	if err != nil {
		jsonMsg(c, "add traffic policy", err)
		return
	}
	jsonObj(c, created, nil)
}

func (a *TrafficPolicyController) get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "get traffic policy", err)
		return
	}
	record, err := a.detailService.Get(id)
	if err != nil {
		jsonMsg(c, "get traffic policy", err)
		return
	}
	jsonObj(c, record, nil)
}

func (a *TrafficPolicyController) update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "update traffic policy", err)
		return
	}
	record := &n5model.TrafficPolicy{Id: id}
	if err := c.ShouldBind(record); err != nil {
		jsonMsg(c, "update traffic policy", err)
		return
	}
	updated, err := a.policyService.UpdatePolicy(record)
	if err != nil {
		jsonMsg(c, "update traffic policy", err)
		return
	}
	jsonObj(c, updated, nil)
}

func (a *TrafficPolicyController) del(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "delete traffic policy", err)
		return
	}
	jsonMsg(c, "delete traffic policy", a.policyService.DeletePolicy(id))
}

func (a *TrafficPolicyController) enable(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "enable traffic policy", err)
		return
	}
	record, err := a.policyService.EnablePolicy(id)
	if err != nil {
		jsonMsg(c, "enable traffic policy", err)
		return
	}
	jsonObj(c, record, nil)
}

func (a *TrafficPolicyController) disable(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "disable traffic policy", err)
		return
	}
	record, err := a.policyService.DisablePolicy(id)
	if err != nil {
		jsonMsg(c, "disable traffic policy", err)
		return
	}
	jsonObj(c, record, nil)
}

func (a *TrafficPolicyController) listRules(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "list traffic rules", err)
		return
	}
	records, err := a.policyService.ListRules(id)
	if err != nil {
		jsonMsg(c, "list traffic rules", err)
		return
	}
	jsonObj(c, records, nil)
}

func (a *TrafficPolicyController) addRule(c *gin.Context) {
	record := &n5model.TrafficPolicyRule{}
	if err := c.ShouldBind(record); err != nil {
		jsonMsg(c, "add traffic rule", err)
		return
	}
	created, err := a.policyService.AddRule(record)
	if err != nil {
		jsonMsg(c, "add traffic rule", err)
		return
	}
	jsonObj(c, created, nil)
}

func (a *TrafficPolicyController) updateRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "update traffic rule", err)
		return
	}
	record := &n5model.TrafficPolicyRule{Id: id}
	if err := c.ShouldBind(record); err != nil {
		jsonMsg(c, "update traffic rule", err)
		return
	}
	updated, err := a.policyService.UpdateRule(record)
	if err != nil {
		jsonMsg(c, "update traffic rule", err)
		return
	}
	jsonObj(c, updated, nil)
}

func (a *TrafficPolicyController) delRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "delete traffic rule", err)
		return
	}
	err = a.policyService.DeleteRule(id)
	jsonMsg(c, "delete traffic rule", err)
}

func (a *TrafficPolicyController) enableRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "enable traffic rule", err)
		return
	}
	record, err := a.policyService.EnableRule(id)
	if err != nil {
		jsonMsg(c, "enable traffic rule", err)
		return
	}
	jsonObj(c, record, nil)
}

func (a *TrafficPolicyController) disableRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "disable traffic rule", err)
		return
	}
	record, err := a.policyService.DisableRule(id)
	if err != nil {
		jsonMsg(c, "disable traffic rule", err)
		return
	}
	jsonObj(c, record, nil)
}

func (a *TrafficPolicyController) reorderRules(c *gin.Context) {
	payload := struct {
		PolicyId int   `json:"policyId" form:"policyId"`
		RuleIds  []int `json:"ruleIds" form:"ruleIds"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "reorder traffic rules", err)
		return
	}
	jsonMsg(c, "reorder traffic rules", a.policyService.ReorderRules(payload.PolicyId, payload.RuleIds))
}

func (a *TrafficPolicyController) listBindings(c *gin.Context) {
	records, err := a.policyService.ListBindings()
	if err != nil {
		jsonMsg(c, "list traffic bindings", err)
		return
	}
	jsonObj(c, records, nil)
}

func (a *TrafficPolicyController) bind(c *gin.Context) {
	payload := struct {
		InboundId int `json:"inboundId" form:"inboundId"`
		PolicyId  int `json:"policyId" form:"policyId"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "bind traffic policy", err)
		return
	}
	record, err := a.policyService.BindInboundPolicy(payload.InboundId, payload.PolicyId)
	if err != nil {
		jsonMsg(c, "bind traffic policy", err)
		return
	}
	jsonObj(c, record, nil)
}

func (a *TrafficPolicyController) unbind(c *gin.Context) {
	payload := struct {
		InboundId int `json:"inboundId" form:"inboundId"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "unbind traffic policy", err)
		return
	}
	jsonMsg(c, "unbind traffic policy", a.policyService.UnbindInboundPolicy(payload.InboundId))
}

func (a *TrafficPolicyController) rebind(c *gin.Context) {
	payload := struct {
		InboundId int `json:"inboundId" form:"inboundId"`
		PolicyId  int `json:"policyId" form:"policyId"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "rebind traffic policy", err)
		return
	}
	record, err := a.policyService.RebindInboundPolicy(payload.InboundId, payload.PolicyId)
	if err != nil {
		jsonMsg(c, "rebind traffic policy", err)
		return
	}
	jsonObj(c, record, nil)
}

func (a *TrafficPolicyController) fragments(c *gin.Context) {
	outbounds, err := a.xrayExt.GenerateOutboundFragments()
	if err != nil {
		jsonMsg(c, "generate fragments", err)
		return
	}
	routing, err := a.xrayExt.GenerateRoutingFragments()
	if err != nil {
		jsonMsg(c, "generate fragments", err)
		return
	}
	jsonObj(c, gin.H{
		"outbounds": outbounds,
		"routing":   routing,
	}, nil)
}
