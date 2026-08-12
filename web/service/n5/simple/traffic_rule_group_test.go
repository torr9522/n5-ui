package simple

import (
	"testing"
	"x-ui/database"
	"x-ui/database/model"
	n5model "x-ui/database/model/n5"
	n5service "x-ui/web/service/n5"
)

func TestTrafficRuleGroupServiceCreateBuiltinAndNoMerge(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewTrafficRuleGroupService()
	group, err := svc.CreateGroup(&CreateTrafficRuleGroupRequest{
		GroupType: simpleTrafficAI,
	})
	if err != nil {
		t.Fatalf("create builtin group failed: %v", err)
	}
	if group.GroupType != simpleTrafficAI {
		t.Fatalf("unexpected group: %#v", group)
	}
	if group.RuleCount == 0 {
		t.Fatalf("expected builtin group rules: %#v", group)
	}

	fragments, err := (&n5service.XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing fragments failed: %v", err)
	}
	rules, _ := fragments["rules"].([]interface{})
	if len(rules) != 0 {
		t.Fatalf("expected no routing rules for definition-only group, got %#v", fragments)
	}
}

func TestTrafficRuleGroupServiceAddDeleteRuleAndDisable(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewTrafficRuleGroupService()
	group, err := svc.CreateGroup(&CreateTrafficRuleGroupRequest{
		GroupType: simpleTrafficCustom,
		Name:      "custom domains",
	})
	if err != nil {
		t.Fatalf("create custom group failed: %v", err)
	}
	if group.RuleCount != 0 {
		t.Fatalf("expected empty custom group: %#v", group)
	}

	rule, err := svc.AddDomainRule(&AddTrafficRuleDomainRequest{
		GroupId: group.Id,
		Domain:  "full:api64.ipify.org",
	})
	if err != nil {
		t.Fatalf("add domain rule failed: %v", err)
	}
	if rule.DisplayValue != "full:api64.ipify.org" {
		t.Fatalf("unexpected rule display value: %#v", rule)
	}

	fetched, err := svc.GetGroup(group.Id)
	if err != nil {
		t.Fatalf("get group failed: %v", err)
	}
	if len(fetched.Rules) != 1 {
		t.Fatalf("unexpected fetched rules: %#v", fetched)
	}

	disabled, err := svc.DisableGroup(group.Id)
	if err != nil {
		t.Fatalf("disable group failed: %v", err)
	}
	if disabled.Enabled {
		t.Fatalf("expected disabled group: %#v", disabled)
	}

	if err := svc.DeleteDomainRule(group.Id, rule.Id); err != nil {
		t.Fatalf("delete domain rule failed: %v", err)
	}
	fetched, err = svc.GetGroup(group.Id)
	if err != nil {
		t.Fatalf("get group after delete failed: %v", err)
	}
	if len(fetched.Rules) != 0 {
		t.Fatalf("expected no rules after delete: %#v", fetched)
	}
}

func TestTrafficRuleGroupServiceSnapshotOnlyAffectsExecPolicy(t *testing.T) {
	initSimpleTestDB(t)

	groupSvc := NewTrafficRuleGroupService()
	group, err := groupSvc.CreateGroup(&CreateTrafficRuleGroupRequest{GroupType: simpleTrafficAI})
	if err != nil {
		t.Fatalf("create group failed: %v", err)
	}

	inbound := &model.Inbound{
		UserId:         1,
		Remark:         "snapshot-inbound",
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           33101,
		Protocol:       model.Socks,
		Settings:       `{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`,
		StreamSettings: `{}`,
		Tag:            "snapshot-inbound-tag",
		Sniffing:       `{}`,
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}

	egress, err := (&n5service.EgressService{}).Create(&n5model.Egress{
		Name:         "snapshot-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: `{"protocol":"freedom","settings":{}}`,
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}

	ruleSvc := NewRuleService()
	if _, err := ruleSvc.CreateSimpleRule(&CreateSimpleRuleRequest{
		InboundId: inbound.Id,
		GroupId:   group.Id,
		EgressId:  egress.Id,
	}); err != nil {
		t.Fatalf("create snapshot exec rule failed: %v", err)
	}

	fragments, err := (&n5service.XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing fragments failed: %v", err)
	}
	assertSimpleRuleFragment(t, fragments, inbound.Tag, egress.Tag, "domain:openai.com", true)

	if err := groupSvc.DeleteGroup(group.Id); err != nil {
		t.Fatalf("delete group failed: %v", err)
	}

	fragments, err = (&n5service.XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing fragments after group delete failed: %v", err)
	}
	assertSimpleRuleFragment(t, fragments, inbound.Tag, egress.Tag, "domain:openai.com", true)
}
