package n5

import (
	"strings"
	"testing"
	"x-ui/database"
	legacyModel "x-ui/database/model"
	n5model "x-ui/database/model/n5"
)

func createTrafficTestInbound(t *testing.T, port int, tag string) *legacyModel.Inbound {
	t.Helper()

	inbound := &legacyModel.Inbound{
		UserId:         1,
		Remark:         tag,
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           port,
		Protocol:       legacyModel.Socks,
		Settings:       `{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`,
		StreamSettings: `{}`,
		Tag:            tag,
		Sniffing:       `{}`,
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}
	return inbound
}

func createTrafficTestEgress(t *testing.T, svc *EgressService, name string) *n5model.Egress {
	t.Helper()

	egress, err := svc.Create(&n5model.Egress{
		Name:         name,
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: freedomOutboundJSON(),
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}
	return egress
}

func TestTrafficPolicyServiceManagePolicyRuleAndBinding(t *testing.T) {
	initTestDB(t)

	egressSvc := &EgressService{}
	policySvc := &TrafficPolicyService{}

	egressA := createTrafficTestEgress(t, egressSvc, "manage-egress-a")
	egressB := createTrafficTestEgress(t, egressSvc, "manage-egress-b")
	inbound := createTrafficTestInbound(t, 34101, "manage-inbound")

	policy, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:              "manage-policy",
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   egressA.Id,
	})
	if err != nil {
		t.Fatalf("create policy failed: %v", err)
	}
	ruleA, err := policySvc.AddRule(&n5model.TrafficPolicyRule{
		PolicyId:   policy.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeExact,
		MatchValue: "a.example.com",
		TargetType: targetTypeEgress,
		TargetId:   egressA.Id,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("create rule a failed: %v", err)
	}
	ruleB, err := policySvc.AddRule(&n5model.TrafficPolicyRule{
		PolicyId:   policy.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeSuffix,
		MatchValue: "example.org",
		TargetType: targetTypeEgress,
		TargetId:   egressB.Id,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("create rule b failed: %v", err)
	}

	updatedPolicy, err := policySvc.UpdatePolicy(&n5model.TrafficPolicy{
		Id:                policy.Id,
		Name:              "manage-policy-updated",
		Remark:            "phase35c",
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   egressB.Id,
	})
	if err != nil {
		t.Fatalf("update policy failed: %v", err)
	}
	if updatedPolicy.Name != "manage-policy-updated" || updatedPolicy.DefaultTargetId != egressB.Id {
		t.Fatalf("unexpected updated policy: %#v", updatedPolicy)
	}

	disabledPolicy, err := policySvc.DisablePolicy(policy.Id)
	if err != nil {
		t.Fatalf("disable policy failed: %v", err)
	}
	if disabledPolicy.Enabled {
		t.Fatalf("expected policy disabled: %#v", disabledPolicy)
	}
	enabledPolicy, err := policySvc.EnablePolicy(policy.Id)
	if err != nil {
		t.Fatalf("enable policy failed: %v", err)
	}
	if !enabledPolicy.Enabled {
		t.Fatalf("expected policy enabled: %#v", enabledPolicy)
	}

	updatedRule, err := policySvc.UpdateRule(&n5model.TrafficPolicyRule{
		Id:         ruleA.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeKeyword,
		MatchValue: "updated",
		TargetType: targetTypeEgress,
		TargetId:   egressB.Id,
		SortOrder:  2,
	})
	if err != nil {
		t.Fatalf("update rule failed: %v", err)
	}
	if updatedRule.MatchMode != domainModeKeyword || updatedRule.MatchValue != "updated" || updatedRule.TargetId != egressB.Id {
		t.Fatalf("unexpected updated rule: %#v", updatedRule)
	}

	disabledRule, err := policySvc.DisableRule(ruleB.Id)
	if err != nil {
		t.Fatalf("disable rule failed: %v", err)
	}
	if disabledRule.Enabled {
		t.Fatalf("expected rule disabled: %#v", disabledRule)
	}
	enabledRule, err := policySvc.EnableRule(ruleB.Id)
	if err != nil {
		t.Fatalf("enable rule failed: %v", err)
	}
	if !enabledRule.Enabled {
		t.Fatalf("expected rule enabled: %#v", enabledRule)
	}

	if err := policySvc.ReorderRules(policy.Id, []int{ruleB.Id, ruleA.Id}); err != nil {
		t.Fatalf("reorder rules failed: %v", err)
	}
	rules, err := policySvc.ListRules(policy.Id)
	if err != nil {
		t.Fatalf("list rules failed: %v", err)
	}
	if len(rules) != 2 || rules[0].Id != ruleB.Id || rules[1].Id != ruleA.Id {
		t.Fatalf("unexpected rule order: %#v", rules)
	}

	binding, err := policySvc.BindInboundPolicy(inbound.Id, policy.Id)
	if err != nil {
		t.Fatalf("bind policy failed: %v", err)
	}
	if binding.InboundId != inbound.Id || binding.PolicyId != policy.Id {
		t.Fatalf("unexpected binding: %#v", binding)
	}
	if err := policySvc.UnbindInboundPolicy(inbound.Id); err != nil {
		t.Fatalf("unbind policy failed: %v", err)
	}
	bindings, err := policySvc.ListBindings()
	if err != nil {
		t.Fatalf("list bindings failed: %v", err)
	}
	if len(bindings) != 0 {
		t.Fatalf("expected no bindings after unbind: %#v", bindings)
	}

	policyB, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:              "manage-policy-b",
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   egressA.Id,
	})
	if err != nil {
		t.Fatalf("create second policy failed: %v", err)
	}
	if _, err := policySvc.RebindInboundPolicy(inbound.Id, policy.Id); err != nil {
		t.Fatalf("rebind to policy a failed: %v", err)
	}
	if _, err := policySvc.RebindInboundPolicy(inbound.Id, policyB.Id); err != nil {
		t.Fatalf("rebind to policy b failed: %v", err)
	}
	bindings, err = policySvc.ListBindings()
	if err != nil {
		t.Fatalf("list bindings after rebind failed: %v", err)
	}
	if len(bindings) != 1 || bindings[0].InboundId != inbound.Id || bindings[0].PolicyId != policyB.Id {
		t.Fatalf("expected one effective binding after rebind: %#v", bindings)
	}

	if err := policySvc.DeletePolicy(policy.Id); err != nil {
		t.Fatalf("delete policy failed: %v", err)
	}
	var policyCount int64
	if err := database.GetDB().Model(&n5model.TrafficPolicy{}).Where("id = ?", policy.Id).Count(&policyCount).Error; err != nil {
		t.Fatalf("count deleted policy failed: %v", err)
	}
	if policyCount != 0 {
		t.Fatalf("expected deleted policy count 0, got %d", policyCount)
	}
	var inboundCount int64
	if err := database.GetDB().Model(&legacyModel.Inbound{}).Where("id = ?", inbound.Id).Count(&inboundCount).Error; err != nil {
		t.Fatalf("count legacy inbound failed: %v", err)
	}
	if inboundCount != 1 {
		t.Fatalf("expected legacy inbound unchanged, got %d", inboundCount)
	}
}

func TestTrafficPolicyDisableExcludesRulesFromXrayFragments(t *testing.T) {
	initTestDB(t)

	egressSvc := &EgressService{}
	policySvc := &TrafficPolicyService{}
	extSvc := &XrayExtService{}

	egress := createTrafficTestEgress(t, egressSvc, "fragment-egress")
	inbound := createTrafficTestInbound(t, 34111, "fragment-inbound")

	policy, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:              "fragment-policy",
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   egress.Id,
	})
	if err != nil {
		t.Fatalf("create policy failed: %v", err)
	}
	rule, err := policySvc.AddRule(&n5model.TrafficPolicyRule{
		PolicyId:   policy.Id,
		RuleType:   ruleTypeDomain,
		MatchMode:  domainModeExact,
		MatchValue: "fragment.example.com",
		TargetType: targetTypeEgress,
		TargetId:   egress.Id,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("create rule failed: %v", err)
	}
	if _, err := policySvc.BindInboundPolicy(inbound.Id, policy.Id); err != nil {
		t.Fatalf("bind policy failed: %v", err)
	}

	routing, err := extSvc.GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing failed: %v", err)
	}
	rules := routing["rules"].([]interface{})
	if len(rules) != 2 {
		t.Fatalf("expected 2 routing rules before disable, got %d", len(rules))
	}

	if _, err := policySvc.DisableRule(rule.Id); err != nil {
		t.Fatalf("disable rule failed: %v", err)
	}
	routing, err = extSvc.GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing after rule disable failed: %v", err)
	}
	rules = routing["rules"].([]interface{})
	if len(rules) != 1 {
		t.Fatalf("expected only default routing rule after rule disable, got %d", len(rules))
	}

	if _, err := policySvc.EnableRule(rule.Id); err != nil {
		t.Fatalf("enable rule failed: %v", err)
	}
	if _, err := policySvc.DisablePolicy(policy.Id); err != nil {
		t.Fatalf("disable policy failed: %v", err)
	}
	routing, err = extSvc.GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing after policy disable failed: %v", err)
	}
	rules = routing["rules"].([]interface{})
	if len(rules) != 0 {
		t.Fatalf("expected no routing rules after policy disable, got %d", len(rules))
	}
}

func TestTrafficTemplateCreatedPolicyCanBeEdited(t *testing.T) {
	initTestDB(t)

	egressSvc := &EgressService{}
	policySvc := &TrafficPolicyService{}
	templateSvc := &TrafficTemplateService{}

	egress := createTrafficTestEgress(t, egressSvc, "template-edit-egress")
	inbound := createTrafficTestInbound(t, 34121, "template-edit-inbound")

	result, err := templateSvc.Create(&TrafficTemplateCreateRequest{
		TemplateName: "ai",
		PolicyName:   "Template Edit Policy",
		InboundId:    inbound.Id,
		TargetType:   targetTypeEgress,
		TargetId:     egress.Id,
	})
	if err != nil {
		t.Fatalf("create template policy failed: %v", err)
	}

	updatedPolicy, err := policySvc.UpdatePolicy(&n5model.TrafficPolicy{
		Id:                result.Policy.Id,
		Name:              "Template Edit Policy Updated",
		Remark:            "editable",
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   egress.Id,
	})
	if err != nil {
		t.Fatalf("update template policy failed: %v", err)
	}
	if updatedPolicy.Name != "Template Edit Policy Updated" {
		t.Fatalf("unexpected updated template policy: %#v", updatedPolicy)
	}

	firstRule := result.Rules[0]
	updatedRule, err := policySvc.UpdateRule(&n5model.TrafficPolicyRule{
		Id:         firstRule.Id,
		RuleType:   firstRule.RuleType,
		MatchMode:  domainModeKeyword,
		MatchValue: "openai",
		TargetType: firstRule.TargetType,
		TargetId:   firstRule.TargetId,
		SortOrder:  firstRule.SortOrder,
	})
	if err != nil {
		t.Fatalf("update template rule failed: %v", err)
	}
	if updatedRule.MatchMode != domainModeKeyword || updatedRule.MatchValue != "openai" {
		t.Fatalf("unexpected updated template rule: %#v", updatedRule)
	}
}

func TestTrafficPolicyServiceRejectsRemarkTransitionForSimpleManagedPolicy(t *testing.T) {
	initTestDB(t)

	policySvc := &TrafficPolicyService{}
	policy, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:    "simple-managed",
		Remark:  "n5-simple-exec|eyJ2ZXJzaW9uIjoxLCJpdGVtcyI6W119",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("create policy failed: %v", err)
	}

	_, err = policySvc.UpdatePolicy(&n5model.TrafficPolicy{
		Id:      policy.Id,
		Name:    policy.Name,
		Remark:  "ordinary-remark",
		Enabled: true,
	})
	if err == nil || !strings.Contains(err.Error(), "Simple 出口规则管理") {
		t.Fatalf("unexpected update error: %v", err)
	}
}
