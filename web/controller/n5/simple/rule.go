package simple

import (
	"net/http"
	"x-ui/web/entity"
	coreservice "x-ui/web/service"
	simpleservice "x-ui/web/service/n5/simple"

	"github.com/gin-gonic/gin"
)

type ruleAPI interface {
	ListSimpleRules() (*simpleservice.SimpleRuleListResult, error)
	CreateSimpleRule(req *simpleservice.CreateSimpleRuleRequest) (*simpleservice.SimpleRule, error)
	DeleteSimpleRule(policyId int) error
}

type ruleRestartTrigger interface {
	SetToNeedRestart()
}

type ruleSettingAPI interface {
	GetAllSetting() (*entity.AllSetting, error)
	GetN5XrayExtensionEnable() (bool, error)
	UpdateAllSetting(allSetting *entity.AllSetting) error
}

type RuleController struct {
	service        ruleAPI
	xrayService    ruleRestartTrigger
	settingService ruleSettingAPI
}

func NewRuleController(g *gin.RouterGroup) *RuleController {
	a := &RuleController{
		service:        simpleservice.NewRuleService(),
		xrayService:    &coreservice.XrayService{},
		settingService: &coreservice.SettingService{},
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
	apiGroup.GET("/n5-status", a.n5Status)
	apiGroup.POST("/n5-status", a.updateN5Status)
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
	a.getXrayService().SetToNeedRestart()
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
	err := a.service.DeleteSimpleRule(payload.Id)
	if err == nil {
		a.getXrayService().SetToNeedRestart()
	}
	jsonMsg(c, "delete simple rule", err)
}

func (a *RuleController) n5Status(c *gin.Context) {
	enabled, err := a.getSettingService().GetN5XrayExtensionEnable()
	if err != nil {
		jsonMsg(c, "get n5 simple status", err)
		return
	}
	jsonObj(c, gin.H{"enabled": enabled}, nil)
}

func (a *RuleController) updateN5Status(c *gin.Context) {
	payload := struct {
		Enabled bool `json:"enabled" form:"enabled"`
	}{}
	if err := c.ShouldBind(&payload); err != nil {
		jsonMsg(c, "update n5 simple status", err)
		return
	}
	allSetting, err := a.getSettingService().GetAllSetting()
	if err != nil {
		jsonMsg(c, "update n5 simple status", err)
		return
	}
	allSetting.N5XrayExtensionEnable = payload.Enabled
	if err := a.getSettingService().UpdateAllSetting(allSetting); err != nil {
		jsonMsg(c, "update n5 simple status", err)
		return
	}
	a.getXrayService().SetToNeedRestart()
	jsonObj(c, gin.H{"enabled": payload.Enabled}, nil)
}

func (a *RuleController) getXrayService() ruleRestartTrigger {
	if a.xrayService != nil {
		return a.xrayService
	}
	return &coreservice.XrayService{}
}

func (a *RuleController) getSettingService() ruleSettingAPI {
	if a.settingService != nil {
		return a.settingService
	}
	return &coreservice.SettingService{}
}
