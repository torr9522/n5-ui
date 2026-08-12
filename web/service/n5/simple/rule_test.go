package simple

import (
	"testing"
	"x-ui/database"
	"x-ui/database/model"
	n5model "x-ui/database/model/n5"
	n5service "x-ui/web/service/n5"
)

func TestSimpleRuleServiceCreateAIAndDelete(t *testing.T) {
	initSimpleTestDB(t)

	egressSvc := &n5service.EgressService{}
	egress, err := egressSvc.Create(&n5model.Egress{
		Name:         "simple-rule-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: `{"protocol":"freedom","settings":{}}`,
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}

	inbound := &model.Inbound{
		UserId:         1,
		Remark:         "simple-rule-inbound",
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           33001,
		Protocol:       model.Socks,
		Settings:       `{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`,
		StreamSettings: `{}`,
		Tag:            "simple-rule-inbound-tag",
		Sniffing:       `{}`,
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}

	svc := NewRuleService()
	created, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{
		InboundId:   inbound.Id,
		TrafficType: simpleTrafficAI,
		EgressId:    egress.Id,
	})
	if err != nil {
		t.Fatalf("create simple ai rule failed: %v", err)
	}
	if created.TrafficType != simpleTrafficAI {
		t.Fatalf("unexpected traffic type: %#v", created)
	}

	policySvc := &n5service.TrafficPolicyService{}
	policies, err := policySvc.List()
	if err != nil {
		t.Fatalf("list policies failed: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("unexpected policy count: %d", len(policies))
	}
	rules, err := policySvc.ListRules(created.PolicyId)
	if err != nil {
		t.Fatalf("list rules failed: %v", err)
	}
	if len(rules) == 0 {
		t.Fatal("expected template rules to be created")
	}
	bindings, err := policySvc.ListBindings()
	if err != nil {
		t.Fatalf("list bindings failed: %v", err)
	}
	if len(bindings) != 1 || bindings[0].InboundId != inbound.Id {
		t.Fatalf("unexpected bindings: %#v", bindings)
	}

	fragments, err := (&n5service.XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing fragments failed: %v", err)
	}
	assertSimpleRuleFragment(t, fragments, inbound.Tag, egress.Tag, "domain:openai.com", true)
	assertNoSimpleDefaultRoute(t, fragments, inbound.Tag, egress.Tag)

	if err := svc.DeleteSimpleRule(created.PolicyId); err != nil {
		t.Fatalf("delete simple rule failed: %v", err)
	}

	fragments, err = (&n5service.XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing fragments after delete failed: %v", err)
	}
	assertSimpleRuleFragment(t, fragments, inbound.Tag, egress.Tag, "domain:openai.com", false)
}

func TestSimpleRuleServiceListAndCustomDomain(t *testing.T) {
	initSimpleTestDB(t)

	egressSvc := &n5service.EgressService{}
	egress, err := egressSvc.Create(&n5model.Egress{
		Name:         "custom-domain-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: `{"protocol":"freedom","settings":{}}`,
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}

	inbound := &model.Inbound{
		UserId:         1,
		Remark:         "custom-domain-inbound",
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           33011,
		Protocol:       model.Socks,
		Settings:       `{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`,
		StreamSettings: `{}`,
		Tag:            "custom-domain-inbound-tag",
		Sniffing:       `{}`,
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}

	svc := NewRuleService()
	created, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{
		InboundId:    inbound.Id,
		TrafficType:  simpleTrafficCustomDomain,
		EgressId:     egress.Id,
		CustomDomain: "full:api.ipify.org",
	})
	if err != nil {
		t.Fatalf("create custom domain rule failed: %v", err)
	}
	if created.CustomDomain != "full:api.ipify.org" {
		t.Fatalf("unexpected custom domain: %#v", created)
	}

	list, err := svc.ListSimpleRules()
	if err != nil {
		t.Fatalf("list simple rules failed: %v", err)
	}
	if len(list.Rules) != 1 {
		t.Fatalf("unexpected simple rule count: %d", len(list.Rules))
	}
	if list.Rules[0].TrafficType != simpleTrafficCustomDomain {
		t.Fatalf("unexpected list rule: %#v", list.Rules[0])
	}
	if len(list.Inbounds) != 1 || len(list.Egresses) != 1 {
		t.Fatalf("unexpected list options: %#v", list)
	}
	trafficSet := make(map[string]bool)
	for _, item := range list.TrafficTypes {
		trafficSet[item.Value] = true
	}
	for _, name := range []string{
		simpleTrafficAll,
		simpleTrafficAI,
		simpleTrafficGame,
		simpleTrafficStreaming,
		simpleTrafficCustomDomain,
	} {
		if !trafficSet[name] {
			t.Fatalf("missing traffic type option: %s", name)
		}
	}
}

func TestParseCustomDomainRule(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantMode    string
		wantValue   string
		wantDisplay string
	}{
		{
			name:        "bare domain defaults to suffix match",
			input:       "openai.com",
			wantMode:    "suffix",
			wantValue:   "openai.com",
			wantDisplay: "domain:openai.com",
		},
		{
			name:        "explicit domain keeps suffix match",
			input:       "domain:openai.com",
			wantMode:    "suffix",
			wantValue:   "openai.com",
			wantDisplay: "domain:openai.com",
		},
		{
			name:        "explicit full keeps exact match",
			input:       "full:openai.com",
			wantMode:    "exact",
			wantValue:   "openai.com",
			wantDisplay: "full:openai.com",
		},
		{
			name:        "keyword keeps keyword match",
			input:       "keyword:openai",
			wantMode:    "keyword",
			wantValue:   "openai",
			wantDisplay: "keyword:openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMode, gotValue, gotDisplay, err := parseCustomDomainRule(tt.input)
			if err != nil {
				t.Fatalf("parseCustomDomainRule(%q) returned error: %v", tt.input, err)
			}
			if gotMode != tt.wantMode || gotValue != tt.wantValue || gotDisplay != tt.wantDisplay {
				t.Fatalf("parseCustomDomainRule(%q) = (%q, %q, %q), want (%q, %q, %q)", tt.input, gotMode, gotValue, gotDisplay, tt.wantMode, tt.wantValue, tt.wantDisplay)
			}
		})
	}
}

func assertSimpleRuleFragment(t *testing.T, fragments map[string]interface{}, inboundTag string, outboundTag string, domain string, expect bool) {
	t.Helper()
	rules, _ := fragments["rules"].([]interface{})
	found := false
	for _, item := range rules {
		rule, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		inboundTags, _ := rule["inboundTag"].([]interface{})
		if len(inboundTags) == 0 || inboundTags[0] != inboundTag {
			continue
		}
		domains, _ := rule["domain"].([]interface{})
		if len(domains) == 0 || domains[0] != domain {
			continue
		}
		if tag, _ := rule["outboundTag"].(string); tag == outboundTag {
			found = true
			break
		}
	}
	if expect && !found {
		t.Fatalf("expected fragment not found: inbound=%s outbound=%s domain=%s fragments=%#v", inboundTag, outboundTag, domain, fragments)
	}
	if !expect && found {
		t.Fatalf("unexpected fragment found after delete: inbound=%s outbound=%s domain=%s", inboundTag, outboundTag, domain)
	}
}

func assertNoSimpleDefaultRoute(t *testing.T, fragments map[string]interface{}, inboundTag string, outboundTag string) {
	t.Helper()
	rules, _ := fragments["rules"].([]interface{})
	for _, item := range rules {
		rule, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		inboundTags, _ := rule["inboundTag"].([]interface{})
		if len(inboundTags) == 0 || inboundTags[0] != inboundTag {
			continue
		}
		if _, hasDomain := rule["domain"]; hasDomain {
			continue
		}
		if tag, _ := rule["outboundTag"].(string); tag == outboundTag {
			t.Fatalf("unexpected default outbound route for inbound %s: %#v", inboundTag, rule)
		}
	}
}
