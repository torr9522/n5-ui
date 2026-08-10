package simple

import (
	"net/http"
	simpleservice "x-ui/web/service/n5/simple"

	"github.com/gin-gonic/gin"
)

type ruleAPI interface {
	ListSimpleRules() (*simpleservice.SimpleRuleListResult, error)
	CreateSimpleRule(req *simpleservice.CreateSimpleRuleRequest) (*simpleservice.SimpleRule, error)
	DeleteSimpleRule(policyId int) error
}

type RuleController struct {
	service ruleAPI
}

func NewRuleController(g *gin.RouterGroup) *RuleController {
	a := &RuleController{
		service: simpleservice.NewRuleService(),
	}
	a.initRouter(g)
	return a
}

func (a *RuleController) initRouter(g *gin.RouterGroup) {
	pageGroup := g.Group("/n5")
	pageGroup.Use(checkLogin)
	pageGroup.GET("/simple/rules", a.page)

	apiGroup := g.Group("/n5/api/simple/rule")
	apiGroup.Use(checkLogin)
	apiGroup.GET("/list", a.list)
	apiGroup.POST("/add", a.add)
	apiGroup.POST("/delete", a.del)
}

func (a *RuleController) page(c *gin.Context) {
	html(c, "simple_rules.html", "出口规则", nil)
}

func (a *RuleController) list(c *gin.Context) {
	record, err := a.service.ListSimpleRules()
	if err != nil {
		jsonMsg(c, "list simple rule", err)
		return
	}
	jsonObj(c, record, nil)
}

func (a *RuleController) add(c *gin.Context) {
	record := &simpleservice.CreateSimpleRuleRequest{}
	if err := c.ShouldBind(record); err != nil {
		jsonMsg(c, "add simple rule", err)
		return
	}
	created, err := a.service.CreateSimpleRule(record)
	if err != nil {
		jsonMsg(c, "add simple rule", err)
		return
	}
	jsonObj(c, created, nil)
}

func (a *RuleController) del(c *gin.Context) {
	payload := struct {
		Id int `json:"id" form:"id"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "delete simple rule", err)
		return
	}
	if payload.Id <= 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "delete simple rule失败: invalid simple rule id",
		})
		return
	}
	jsonMsg(c, "delete simple rule", a.service.DeleteSimpleRule(payload.Id))
}
