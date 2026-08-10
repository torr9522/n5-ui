package n5

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"x-ui/database"
	"x-ui/database/model"
	n5model "x-ui/database/model/n5"
	n5service "x-ui/web/service/n5"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func testFiles(t *testing.T) []string {
	t.Helper()
	files := make([]string, 0)
	root := filepath.Join("..", "..", "html")
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".html" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func newTestEngine(t *testing.T) *gin.Engine {
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
	engine.LoadHTMLFiles(testFiles(t)...)
	g := engine.Group("/")
	NewEgressController(g)
	NewEgressLabelController(g)
	NewPoolController(g)
	NewTrafficPolicyController(g)
	NewTrafficTemplateController(g)
	NewXrayController(g)
	return engine
}

func initControllerTestDB(t *testing.T) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "controller.db")
	if err := database.InitDB(dbPath); err != nil {
		t.Fatalf("init db failed: %v", err)
	}
}

func TestN5PageRoutesRender(t *testing.T) {
	initControllerTestDB(t)
	engine := newTestEngine(t)

	for _, path := range []string{"/n5/egress", "/n5/egress-detail?id=1", "/n5/pools", "/n5/traffic-policy", "/n5/traffic-policy-detail?id=1", "/n5/xray-status", "/n5/config-history", "/n5/egress-test"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, path, nil)
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("unexpected status for %s: %d", path, w.Code)
		}
	}
}

func TestN5APIChain(t *testing.T) {
	initControllerTestDB(t)
	engine := newTestEngine(t)

	egressSvc := &n5service.EgressService{}
	labelSvc := &n5service.EgressLabelService{}
	egress, err := egressSvc.Create(&n5model.Egress{
		Name:         "controller-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: `{"protocol":"freedom","settings":{}}`,
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}
	label, err := labelSvc.Create(&n5model.EgressLabel{
		Name: "controller-label",
		Type: "usage",
	})
	if err != nil {
		t.Fatalf("create label failed: %v", err)
	}
	if _, err := labelSvc.Bind(egress.Id, label.Id); err != nil {
		t.Fatalf("bind label failed: %v", err)
	}

	poolSvc := &n5service.EgressPoolService{}
	pool, err := poolSvc.Create(&n5model.EgressPool{
		Name:     "controller-pool",
		Strategy: "random",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create pool failed: %v", err)
	}
	if _, err := poolSvc.AddMember(pool.Id, egress.Id, 1, 1); err != nil {
		t.Fatalf("add member failed: %v", err)
	}

	db := database.GetDB()
	inbound := &model.Inbound{
		UserId:         1,
		Remark:         "controller-inbound",
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           32001,
		Protocol:       model.Socks,
		Settings:       `{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`,
		StreamSettings: `{}`,
		Tag:            "controller-inbound-tag",
		Sniffing:       `{}`,
	}
	if err := db.Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}

	policySvc := &n5service.TrafficPolicyService{}
	policy, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:              "controller-policy",
		Enabled:           true,
		DefaultTargetType: "pool",
		DefaultTargetId:   pool.Id,
	})
	if err != nil {
		t.Fatalf("create policy failed: %v", err)
	}
	if _, err := policySvc.BindInboundPolicy(inbound.Id, policy.Id); err != nil {
		t.Fatalf("bind policy failed: %v", err)
	}

	for _, reqPath := range []string{
		"/n5/api/egress/detail/1",
		"/n5/api/egress-label/list",
		"/n5/api/egress/list",
		"/n5/api/egress/test",
		"/n5/api/pool/list",
		"/n5/api/pool/member/list/1",
		"/n5/api/traffic-policy/list",
		"/n5/api/traffic-policy/binding/list",
		"/n5/api/traffic-policy/fragments",
		"/n5/api/traffic-template/list",
		"/n5/api/traffic-template/preview/ai",
		"/n5/api/xray/status",
		"/n5/api/xray/history/list",
		"/n5/api/xray/egress-test/entry",
	} {
		w := httptest.NewRecorder()
		method := http.MethodPost
		if reqPath == "/n5/api/egress/detail/1" || reqPath == "/n5/api/egress-label/list" || reqPath == "/n5/api/traffic-template/list" || reqPath == "/n5/api/traffic-template/preview/ai" {
			method = http.MethodGet
		}
		req, _ := http.NewRequest(method, reqPath, nil)
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("unexpected status for %s: %d", reqPath, w.Code)
		}
		if len(w.Body.Bytes()) == 0 {
			t.Fatalf("empty response for %s", reqPath)
		}
	}
}

func TestTrafficTemplateAPIResponses(t *testing.T) {
	initControllerTestDB(t)
	engine := newTestEngine(t)

	egressSvc := &n5service.EgressService{}
	egress, err := egressSvc.Create(&n5model.Egress{
		Name:         "template-api-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: `{"protocol":"freedom","settings":{}}`,
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}

	inbound := &model.Inbound{
		UserId:         1,
		Remark:         "template-api-inbound",
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           32011,
		Protocol:       model.Socks,
		Settings:       `{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`,
		StreamSettings: `{}`,
		Tag:            "template-api-inbound-tag",
		Sniffing:       `{}`,
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}

	listResp := httptest.NewRecorder()
	listReq, _ := http.NewRequest(http.MethodGet, "/n5/api/traffic-template/list", nil)
	engine.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected template list status: %d", listResp.Code)
	}
	if !bytes.Contains(listResp.Body.Bytes(), []byte(`"name":"ai"`)) {
		t.Fatalf("template list missing ai: %s", listResp.Body.String())
	}

	previewResp := httptest.NewRecorder()
	previewReq, _ := http.NewRequest(http.MethodGet, "/n5/api/traffic-template/preview/streaming", nil)
	engine.ServeHTTP(previewResp, previewReq)
	if previewResp.Code != http.StatusOK {
		t.Fatalf("unexpected template preview status: %d", previewResp.Code)
	}
	if !bytes.Contains(previewResp.Body.Bytes(), []byte(`"name":"streaming"`)) {
		t.Fatalf("template preview missing streaming: %s", previewResp.Body.String())
	}

	createBody := bytes.NewBufferString(
		`{"templateName":"ai","policyName":"AI API Policy","inboundId":` +
			strconv.Itoa(inbound.Id) +
			`,"targetType":"egress","targetId":` + strconv.Itoa(egress.Id) + `}`,
	)
	createReq, _ := http.NewRequest(http.MethodPost, "/n5/api/traffic-template/create", createBody)
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	engine.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusOK {
		t.Fatalf("unexpected template create status: %d", createResp.Code)
	}
	if !bytes.Contains(createResp.Body.Bytes(), []byte(`"templateName":"ai"`)) {
		t.Fatalf("template create response missing template name: %s", createResp.Body.String())
	}
	if !bytes.Contains(createResp.Body.Bytes(), []byte(`"policyName":"AI API Policy"`)) &&
		!bytes.Contains(createResp.Body.Bytes(), []byte(`"name":"AI API Policy"`)) {
		t.Fatalf("template create response missing policy name: %s", createResp.Body.String())
	}
}

func TestEgressLabelAndDetailAPIResponses(t *testing.T) {
	initControllerTestDB(t)
	engine := newTestEngine(t)

	egressSvc := &n5service.EgressService{}
	poolSvc := &n5service.EgressPoolService{}
	policySvc := &n5service.TrafficPolicyService{}

	egress, err := egressSvc.Create(&n5model.Egress{
		Name:         "detail-api-egress",
		Protocol:     "socks",
		Enabled:      true,
		OutboundJSON: `{"protocol":"socks","settings":{"servers":[{"address":"8.8.8.8","port":1080}]}}`,
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}

	pool, err := poolSvc.Create(&n5model.EgressPool{
		Name:     "detail-api-pool",
		Strategy: "random",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create pool failed: %v", err)
	}
	if _, err := poolSvc.AddMember(pool.Id, egress.Id, 1, 1); err != nil {
		t.Fatalf("add pool member failed: %v", err)
	}

	if _, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:              "detail-api-policy",
		Enabled:           true,
		DefaultTargetType: "pool",
		DefaultTargetId:   pool.Id,
	}); err != nil {
		t.Fatalf("create policy failed: %v", err)
	}

	addBody := bytes.NewBufferString(`{"name":"Singapore","type":"region"}`)
	addReq, _ := http.NewRequest(http.MethodPost, "/n5/api/egress-label/add", addBody)
	addReq.Header.Set("Content-Type", "application/json")
	addResp := httptest.NewRecorder()
	engine.ServeHTTP(addResp, addReq)
	if addResp.Code != http.StatusOK {
		t.Fatalf("unexpected add label status: %d", addResp.Code)
	}

	var addMsg struct {
		Success bool `json:"success"`
		Obj     struct {
			Id int `json:"id"`
		} `json:"obj"`
	}
	if err := json.Unmarshal(addResp.Body.Bytes(), &addMsg); err != nil {
		t.Fatalf("unmarshal add label response failed: %v", err)
	}
	if !addMsg.Success || addMsg.Obj.Id <= 0 {
		t.Fatalf("unexpected add label response: %s", addResp.Body.String())
	}

	bindBody := bytes.NewBufferString(
		`{"egressId":` + strconv.Itoa(egress.Id) + `,"labelId":` + strconv.Itoa(addMsg.Obj.Id) + `}`,
	)
	bindReq, _ := http.NewRequest(http.MethodPost, "/n5/api/egress-label/bind", bindBody)
	bindReq.Header.Set("Content-Type", "application/json")
	bindResp := httptest.NewRecorder()
	engine.ServeHTTP(bindResp, bindReq)
	if bindResp.Code != http.StatusOK {
		t.Fatalf("unexpected bind label status: %d", bindResp.Code)
	}

	listResp := httptest.NewRecorder()
	listReq, _ := http.NewRequest(http.MethodGet, "/n5/api/egress-label/list", nil)
	engine.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected label list status: %d", listResp.Code)
	}
	if !bytes.Contains(listResp.Body.Bytes(), []byte("Singapore")) {
		t.Fatalf("label list response missing label: %s", listResp.Body.String())
	}

	detailResp := httptest.NewRecorder()
	detailReq, _ := http.NewRequest(http.MethodGet, "/n5/api/egress/detail/"+strconv.Itoa(egress.Id), nil)
	engine.ServeHTTP(detailResp, detailReq)
	if detailResp.Code != http.StatusOK {
		t.Fatalf("unexpected detail status: %d", detailResp.Code)
	}
	if !bytes.Contains(detailResp.Body.Bytes(), []byte(`"address":"8.8.8.8"`)) {
		t.Fatalf("detail response missing address: %s", detailResp.Body.String())
	}
	if !bytes.Contains(detailResp.Body.Bytes(), []byte(`"name":"detail-api-pool"`)) {
		t.Fatalf("detail response missing pool: %s", detailResp.Body.String())
	}
	if !bytes.Contains(detailResp.Body.Bytes(), []byte(`"name":"detail-api-policy"`)) {
		t.Fatalf("detail response missing policy: %s", detailResp.Body.String())
	}
}

func TestTrafficPolicyManagementAPIResponses(t *testing.T) {
	initControllerTestDB(t)
	engine := newTestEngine(t)

	egressSvc := &n5service.EgressService{}
	policySvc := &n5service.TrafficPolicyService{}

	egressA, err := egressSvc.Create(&n5model.Egress{
		Name:         "policy-api-egress-a",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: `{"protocol":"freedom","settings":{}}`,
	})
	if err != nil {
		t.Fatalf("create egress a failed: %v", err)
	}
	egressB, err := egressSvc.Create(&n5model.Egress{
		Name:         "policy-api-egress-b",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: `{"protocol":"freedom","settings":{}}`,
	})
	if err != nil {
		t.Fatalf("create egress b failed: %v", err)
	}

	inbound := &model.Inbound{
		UserId:         1,
		Remark:         "policy-api-inbound",
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           32021,
		Protocol:       model.Socks,
		Settings:       `{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`,
		StreamSettings: `{}`,
		Tag:            "policy-api-inbound-tag",
		Sniffing:       `{}`,
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}

	policy, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:              "policy-api",
		Enabled:           true,
		DefaultTargetType: "egress",
		DefaultTargetId:   egressA.Id,
	})
	if err != nil {
		t.Fatalf("create policy failed: %v", err)
	}
	ruleA, err := policySvc.AddRule(&n5model.TrafficPolicyRule{
		PolicyId:   policy.Id,
		RuleType:   "domain",
		MatchMode:  "exact",
		MatchValue: "a.example.com",
		TargetType: "egress",
		TargetId:   egressA.Id,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("create rule a failed: %v", err)
	}
	ruleB, err := policySvc.AddRule(&n5model.TrafficPolicyRule{
		PolicyId:   policy.Id,
		RuleType:   "domain",
		MatchMode:  "suffix",
		MatchValue: "example.org",
		TargetType: "egress",
		TargetId:   egressB.Id,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("create rule b failed: %v", err)
	}
	if _, err := policySvc.BindInboundPolicy(inbound.Id, policy.Id); err != nil {
		t.Fatalf("bind policy failed: %v", err)
	}

	getResp := httptest.NewRecorder()
	getReq, _ := http.NewRequest(http.MethodGet, "/n5/api/traffic-policy/get/"+strconv.Itoa(policy.Id), nil)
	engine.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("unexpected get policy status: %d", getResp.Code)
	}
	if !bytes.Contains(getResp.Body.Bytes(), []byte(`"name":"policy-api"`)) {
		t.Fatalf("get policy response missing policy: %s", getResp.Body.String())
	}
	if !bytes.Contains(getResp.Body.Bytes(), []byte(`"inboundTag":"policy-api-inbound-tag"`)) {
		t.Fatalf("get policy response missing inbound tag: %s", getResp.Body.String())
	}

	updateBody := bytes.NewBufferString(
		`{"name":"policy-api-updated","remark":"phase35c","enabled":true,"defaultTargetType":"egress","defaultTargetId":` +
			strconv.Itoa(egressB.Id) + `}`,
	)
	updateReq, _ := http.NewRequest(http.MethodPost, "/n5/api/traffic-policy/update/"+strconv.Itoa(policy.Id), updateBody)
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := httptest.NewRecorder()
	engine.ServeHTTP(updateResp, updateReq)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("unexpected update policy status: %d", updateResp.Code)
	}
	if !bytes.Contains(updateResp.Body.Bytes(), []byte(`"name":"policy-api-updated"`)) {
		t.Fatalf("update policy response missing name: %s", updateResp.Body.String())
	}

	disableResp := httptest.NewRecorder()
	disableReq, _ := http.NewRequest(http.MethodPost, "/n5/api/traffic-policy/disable/"+strconv.Itoa(policy.Id), nil)
	engine.ServeHTTP(disableResp, disableReq)
	if disableResp.Code != http.StatusOK || !bytes.Contains(disableResp.Body.Bytes(), []byte(`"enabled":false`)) {
		t.Fatalf("unexpected disable policy response: %s", disableResp.Body.String())
	}

	enableResp := httptest.NewRecorder()
	enableReq, _ := http.NewRequest(http.MethodPost, "/n5/api/traffic-policy/enable/"+strconv.Itoa(policy.Id), nil)
	engine.ServeHTTP(enableResp, enableReq)
	if enableResp.Code != http.StatusOK || !bytes.Contains(enableResp.Body.Bytes(), []byte(`"enabled":true`)) {
		t.Fatalf("unexpected enable policy response: %s", enableResp.Body.String())
	}

	updateRuleBody := bytes.NewBufferString(
		`{"ruleType":"domain","matchMode":"keyword","matchValue":"updated-keyword","targetType":"egress","targetId":` +
			strconv.Itoa(egressB.Id) + `,"sortOrder":2}`,
	)
	updateRuleReq, _ := http.NewRequest(http.MethodPost, "/n5/api/traffic-policy/rule/update/"+strconv.Itoa(ruleA.Id), updateRuleBody)
	updateRuleReq.Header.Set("Content-Type", "application/json")
	updateRuleResp := httptest.NewRecorder()
	engine.ServeHTTP(updateRuleResp, updateRuleReq)
	if updateRuleResp.Code != http.StatusOK {
		t.Fatalf("unexpected update rule status: %d", updateRuleResp.Code)
	}
	if !bytes.Contains(updateRuleResp.Body.Bytes(), []byte(`"matchValue":"updated-keyword"`)) {
		t.Fatalf("update rule response missing match value: %s", updateRuleResp.Body.String())
	}

	ruleDisableResp := httptest.NewRecorder()
	ruleDisableReq, _ := http.NewRequest(http.MethodPost, "/n5/api/traffic-policy/rule/disable/"+strconv.Itoa(ruleB.Id), nil)
	engine.ServeHTTP(ruleDisableResp, ruleDisableReq)
	if ruleDisableResp.Code != http.StatusOK || !bytes.Contains(ruleDisableResp.Body.Bytes(), []byte(`"enabled":false`)) {
		t.Fatalf("unexpected disable rule response: %s", ruleDisableResp.Body.String())
	}

	ruleEnableResp := httptest.NewRecorder()
	ruleEnableReq, _ := http.NewRequest(http.MethodPost, "/n5/api/traffic-policy/rule/enable/"+strconv.Itoa(ruleB.Id), nil)
	engine.ServeHTTP(ruleEnableResp, ruleEnableReq)
	if ruleEnableResp.Code != http.StatusOK || !bytes.Contains(ruleEnableResp.Body.Bytes(), []byte(`"enabled":true`)) {
		t.Fatalf("unexpected enable rule response: %s", ruleEnableResp.Body.String())
	}

	reorderBody := bytes.NewBufferString(
		`{"policyId":` + strconv.Itoa(policy.Id) + `,"ruleIds":[` + strconv.Itoa(ruleB.Id) + `,` + strconv.Itoa(ruleA.Id) + `]}`,
	)
	reorderReq, _ := http.NewRequest(http.MethodPost, "/n5/api/traffic-policy/rule/reorder", reorderBody)
	reorderReq.Header.Set("Content-Type", "application/json")
	reorderResp := httptest.NewRecorder()
	engine.ServeHTTP(reorderResp, reorderReq)
	if reorderResp.Code != http.StatusOK {
		t.Fatalf("unexpected reorder rule status: %d", reorderResp.Code)
	}

	unbindBody := bytes.NewBufferString(`{"inboundId":` + strconv.Itoa(inbound.Id) + `}`)
	unbindReq, _ := http.NewRequest(http.MethodPost, "/n5/api/traffic-policy/unbind", unbindBody)
	unbindReq.Header.Set("Content-Type", "application/json")
	unbindResp := httptest.NewRecorder()
	engine.ServeHTTP(unbindResp, unbindReq)
	if unbindResp.Code != http.StatusOK {
		t.Fatalf("unexpected unbind status: %d", unbindResp.Code)
	}

	rebindBody := bytes.NewBufferString(
		`{"inboundId":` + strconv.Itoa(inbound.Id) + `,"policyId":` + strconv.Itoa(policy.Id) + `}`,
	)
	rebindReq, _ := http.NewRequest(http.MethodPost, "/n5/api/traffic-policy/rebind", rebindBody)
	rebindReq.Header.Set("Content-Type", "application/json")
	rebindResp := httptest.NewRecorder()
	engine.ServeHTTP(rebindResp, rebindReq)
	if rebindResp.Code != http.StatusOK {
		t.Fatalf("unexpected rebind status: %d", rebindResp.Code)
	}
	if !bytes.Contains(rebindResp.Body.Bytes(), []byte(`"inboundId":`+strconv.Itoa(inbound.Id))) {
		t.Fatalf("rebind response missing inbound id: %s", rebindResp.Body.String())
	}

	deleteResp := httptest.NewRecorder()
	deleteReq, _ := http.NewRequest(http.MethodPost, "/n5/api/traffic-policy/del/"+strconv.Itoa(policy.Id), nil)
	engine.ServeHTTP(deleteResp, deleteReq)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("unexpected delete policy status: %d", deleteResp.Code)
	}

	var policyCount int64
	if err := database.GetDB().Model(&n5model.TrafficPolicy{}).Where("id = ?", policy.Id).Count(&policyCount).Error; err != nil {
		t.Fatalf("count policy failed: %v", err)
	}
	if policyCount != 0 {
		t.Fatalf("expected policy deleted, count=%d", policyCount)
	}
}
