package simple

import (
	"bytes"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
	"x-ui/database/model"
	simpleservice "x-ui/web/service/n5/simple"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

type fakeRuleService struct {
	listResult *simpleservice.SimpleRuleListResult
	created    *simpleservice.SimpleRule
	deletedID  int
}

func (f *fakeRuleService) ListSimpleRules() (*simpleservice.SimpleRuleListResult, error) {
	return f.listResult, nil
}

func (f *fakeRuleService) CreateSimpleRule(req *simpleservice.CreateSimpleRuleRequest) (*simpleservice.SimpleRule, error) {
	return &simpleservice.SimpleRule{
		Id:           21,
		PolicyId:     21,
		InboundId:    req.InboundId,
		TrafficType:  req.TrafficType,
		EgressId:     req.EgressId,
		CustomDomain: req.CustomDomain,
		Status:       "enabled",
		Enabled:      true,
	}, nil
}

func (f *fakeRuleService) DeleteSimpleRule(policyId int) error {
	f.deletedID = policyId
	return nil
}

func newRuleTestEngine(t *testing.T, svc ruleAPI) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.SetFuncMap(template.FuncMap{
		"i18n": func(key string, params ...string) (string, error) {
			return key, nil
		},
	})
	store := cookie.NewStore([]byte("test-secret"))
	engine.Use(sessions.Sessions("session", store))
	engine.Use(func(c *gin.Context) {
		c.Set("base_path", "/")
		s := sessions.Default(c)
		s.Set("LOGIN_USER", model.User{Id: 1, Username: "admin"})
		_ = s.Save()
		c.Next()
	})
	engine.LoadHTMLFiles(testFiles()...)
	g := engine.Group("/")
	controller := &RuleController{service: svc}
	controller.initRouter(g)
	return engine
}

func TestSimpleRulePageRouteRender(t *testing.T) {
	engine := newRuleTestEngine(t, &fakeRuleService{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/n5/simple/rules", nil)
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
}

func TestSimpleRuleAPIResponses(t *testing.T) {
	svc := &fakeRuleService{
		listResult: &simpleservice.SimpleRuleListResult{
			Rules: []*simpleservice.SimpleRule{
				{
					Id:          1,
					PolicyId:    1,
					InboundId:   7,
					InboundName: "simple-inbound",
					TrafficType: "ai",
					EgressId:    9,
					EgressName:  "simple-egress",
					Status:      "enabled",
					Enabled:     true,
				},
			},
			Inbounds: []*simpleservice.SimpleInboundOption{
				{Id: 7, Name: "simple-inbound", Tag: "simple-inbound-tag", Enabled: true},
			},
			Egresses: []*simpleservice.SimpleRuleEgressOption{
				{Id: 9, Name: "simple-egress", Tag: "n5-egress-0000000009", Enabled: true},
			},
			TrafficTypes: []*simpleservice.SimpleTrafficOption{
				{Value: "ai", Label: "AI 分流"},
			},
		},
	}
	engine := newRuleTestEngine(t, svc)

	listResp := httptest.NewRecorder()
	listReq, _ := http.NewRequest(http.MethodGet, "/n5/api/simple/rule/list", nil)
	engine.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d", listResp.Code)
	}
	if !bytes.Contains(listResp.Body.Bytes(), []byte(`"trafficType":"ai"`)) {
		t.Fatalf("unexpected list body: %s", listResp.Body.String())
	}

	addBody := bytes.NewBufferString(`{"inboundId":7,"trafficType":"ai","egressId":9}`)
	addReq, _ := http.NewRequest(http.MethodPost, "/n5/api/simple/rule/add", addBody)
	addReq.Header.Set("Content-Type", "application/json")
	addResp := httptest.NewRecorder()
	engine.ServeHTTP(addResp, addReq)
	if addResp.Code != http.StatusOK {
		t.Fatalf("unexpected add status: %d", addResp.Code)
	}
	if !bytes.Contains(addResp.Body.Bytes(), []byte(`"policyId":21`)) {
		t.Fatalf("unexpected add body: %s", addResp.Body.String())
	}

	delBody := bytes.NewBufferString(`{"id":21}`)
	delReq, _ := http.NewRequest(http.MethodPost, "/n5/api/simple/rule/delete", delBody)
	delReq.Header.Set("Content-Type", "application/json")
	delResp := httptest.NewRecorder()
	engine.ServeHTTP(delResp, delReq)
	if delResp.Code != http.StatusOK {
		t.Fatalf("unexpected delete status: %d", delResp.Code)
	}
	if svc.deletedID != 21 {
		t.Fatalf("unexpected deleted id: %d", svc.deletedID)
	}
}
