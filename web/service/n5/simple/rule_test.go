package simple

import (
	"encoding/base64"
	"sort"
	"strconv"
	"strings"
	"testing"
	"x-ui/database"
	"x-ui/database/model"
	n5model "x-ui/database/model/n5"
	n5service "x-ui/web/service/n5"
)

func TestSimpleRuleServiceAllAndAICanCoexist(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, err := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	if err != nil {
		t.Fatalf("get ai group failed: %v", err)
	}
	sg := createTestRuleEgress(t, "sg-egress")
	us := createTestRuleEgress(t, "us-egress")
	inbound := createTestRuleInbound(t, 33001, "coexist-inbound")

	allRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{
		InboundId:   inbound.Id,
		TrafficType: simpleTrafficAll,
		EgressId:    sg.Id,
	})
	if err != nil {
		t.Fatalf("create all rule failed: %v", err)
	}
	aiRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{
		InboundId: inbound.Id,
		GroupId:   aiGroup.Id,
		EgressId:  us.Id,
	})
	if err != nil {
		t.Fatalf("create ai rule failed: %v", err)
	}

	list := mustListSimpleRules(t, svc)
	if len(list.Rules) != 2 {
		t.Fatalf("unexpected rule count: %d", len(list.Rules))
	}
	if findAllRule(list.Rules, inbound.Id) == nil || findGroupRule(list.Rules, inbound.Id, simpleTrafficAI) == nil {
		t.Fatalf("expected all + ai rows, got %#v", list.Rules)
	}

	ctx := mustLoadSimplePolicyContext(t, svc, inbound.Id)
	if ctx.Policy.DefaultTargetId != sg.Id {
		t.Fatalf("unexpected default target: %#v", ctx.Policy)
	}
	if ctx.ExecRemark == nil || len(ctx.ExecRemark.Items) != 1 {
		t.Fatalf("unexpected execution items: %#v", ctx.ExecRemark)
	}
	item := ctx.ExecRemark.Items[0]
	if item.GroupId != aiGroup.Id || item.GroupType != simpleTrafficAI {
		t.Fatalf("unexpected exec item: %#v", item)
	}
	if len(item.RuleIDs) != aiGroup.RuleCount {
		t.Fatalf("unexpected ai snapshot size: %d", len(item.RuleIDs))
	}

	fragments, err := (&n5service.XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("generate routing fragments failed: %v", err)
	}
	assertSimpleRuleFragment(t, fragments, inbound.Tag, us.Tag, "domain:openai.com", true)
	assertDefaultRoute(t, fragments, inbound.Tag, sg.Tag, true)
	last := lastInboundRule(t, fragments, inbound.Tag)
	if last["outboundTag"] != sg.Tag {
		t.Fatalf("expected default route last, got %#v", last)
	}
	if allRule.RuleId == aiRule.RuleId {
		t.Fatalf("rule ids should be unique: %#v %#v", allRule, aiRule)
	}
}

func TestSimpleRuleServiceAllAIAndGameCanCoexist(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	gameGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficGame)
	sg := createTestRuleEgress(t, "sg-egress")
	us := createTestRuleEgress(t, "us-egress")
	jp := createTestRuleEgress(t, "jp-egress")
	inbound := createTestRuleInbound(t, 33002, "multi-group-inbound")

	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: sg.Id}); err != nil {
		t.Fatalf("create all failed: %v", err)
	}
	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: us.Id}); err != nil {
		t.Fatalf("create ai failed: %v", err)
	}
	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: gameGroup.Id, EgressId: jp.Id}); err != nil {
		t.Fatalf("create game failed: %v", err)
	}

	list := mustListSimpleRules(t, svc)
	if len(list.Rules) != 3 {
		t.Fatalf("unexpected rule count: %d", len(list.Rules))
	}
	if findAllRule(list.Rules, inbound.Id) == nil || findGroupRule(list.Rules, inbound.Id, simpleTrafficAI) == nil || findGroupRule(list.Rules, inbound.Id, simpleTrafficGame) == nil {
		t.Fatalf("expected all + ai + game rows, got %#v", list.Rules)
	}

	ctx := mustLoadSimplePolicyContext(t, svc, inbound.Id)
	if len(ctx.ExecRemark.Items) != 2 {
		t.Fatalf("unexpected exec item count: %#v", ctx.ExecRemark)
	}
	if len(ctx.Rules) != aiGroup.RuleCount+gameGroup.RuleCount {
		t.Fatalf("unexpected merged rule count: %d", len(ctx.Rules))
	}
}

func TestSimpleRuleServiceRejectsDuplicateAll(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	sg := createTestRuleEgress(t, "sg-egress")
	hk := createTestRuleEgress(t, "hk-egress")
	inbound := createTestRuleInbound(t, 33003, "duplicate-all-inbound")

	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: sg.Id}); err != nil {
		t.Fatalf("create all failed: %v", err)
	}
	_, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: hk.Id})
	if err == nil || !strings.Contains(err.Error(), "已存在默认出口") {
		t.Fatalf("unexpected duplicate all error: %v", err)
	}
}

func TestSimpleRuleServiceRejectsDuplicateAI(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	us := createTestRuleEgress(t, "us-egress")
	sg := createTestRuleEgress(t, "sg-egress")
	inbound := createTestRuleInbound(t, 33004, "duplicate-ai-inbound")

	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: us.Id}); err != nil {
		t.Fatalf("create ai failed: %v", err)
	}
	_, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: sg.Id})
	if err == nil || !strings.Contains(err.Error(), "已存在该分流规则") {
		t.Fatalf("unexpected duplicate ai error: %v", err)
	}
}

func TestSimpleRuleServiceDeleteAIKeepsAll(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	sg := createTestRuleEgress(t, "sg-egress")
	us := createTestRuleEgress(t, "us-egress")
	inbound := createTestRuleInbound(t, 33005, "delete-ai-inbound")

	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: sg.Id}); err != nil {
		t.Fatalf("create all failed: %v", err)
	}
	aiRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: us.Id})
	if err != nil {
		t.Fatalf("create ai failed: %v", err)
	}
	if err := svc.DeleteSimpleRule(aiRule.RuleId); err != nil {
		t.Fatalf("delete ai failed: %v", err)
	}

	list := mustListSimpleRules(t, svc)
	if len(list.Rules) != 1 || findAllRule(list.Rules, inbound.Id) == nil {
		t.Fatalf("unexpected rules after delete ai: %#v", list.Rules)
	}
	ctx := mustLoadSimplePolicyContext(t, svc, inbound.Id)
	if ctx.Policy.DefaultTargetId != sg.Id || len(ctx.Rules) != 0 || len(ctx.ExecRemark.Items) != 0 {
		t.Fatalf("unexpected context after delete ai: %#v %#v", ctx.Policy, ctx.ExecRemark)
	}
}

func TestSimpleRuleServiceDeleteAllKeepsAI(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	sg := createTestRuleEgress(t, "sg-egress")
	us := createTestRuleEgress(t, "us-egress")
	inbound := createTestRuleInbound(t, 33006, "delete-all-inbound")

	allRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: sg.Id})
	if err != nil {
		t.Fatalf("create all failed: %v", err)
	}
	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: us.Id}); err != nil {
		t.Fatalf("create ai failed: %v", err)
	}
	if err := svc.DeleteSimpleRule(allRule.RuleId); err != nil {
		t.Fatalf("delete all failed: %v", err)
	}

	list := mustListSimpleRules(t, svc)
	if len(list.Rules) != 1 || findGroupRule(list.Rules, inbound.Id, simpleTrafficAI) == nil {
		t.Fatalf("unexpected rules after delete all: %#v", list.Rules)
	}
	ctx := mustLoadSimplePolicyContext(t, svc, inbound.Id)
	if ctx.Policy.DefaultTargetId != 0 || len(ctx.Rules) != aiGroup.RuleCount || len(ctx.ExecRemark.Items) != 1 {
		t.Fatalf("unexpected context after delete all: %#v %#v", ctx.Policy, ctx.ExecRemark)
	}
}

func TestSimpleRuleServiceUpdateAllKeepsAISnapshot(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	sg := createTestRuleEgress(t, "sg-egress")
	us := createTestRuleEgress(t, "us-egress")
	hk := createTestRuleEgress(t, "hk-egress")
	inbound := createTestRuleInbound(t, 33007, "update-all-inbound")

	allRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, TrafficType: simpleTrafficAll, EgressId: sg.Id})
	if err != nil {
		t.Fatalf("create all failed: %v", err)
	}
	aiRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: us.Id})
	if err != nil {
		t.Fatalf("create ai failed: %v", err)
	}
	before := mustExecutionItemSnapshot(t, svc, inbound.Id, simpleTrafficAI)

	if _, err := svc.UpdateSimpleRule(allRule.RuleId, &CreateSimpleRuleRequest{
		InboundId:   inbound.Id,
		TrafficType: simpleTrafficAll,
		EgressId:    hk.Id,
	}); err != nil {
		t.Fatalf("update all failed: %v", err)
	}

	after := mustExecutionItemSnapshot(t, svc, inbound.Id, simpleTrafficAI)
	if strings.Join(before, ",") != strings.Join(after, ",") {
		t.Fatalf("ai snapshot changed after all update: before=%v after=%v", before, after)
	}
	list := mustListSimpleRules(t, svc)
	allRow := findAllRule(list.Rules, inbound.Id)
	aiRow := findRuleByRuleID(list.Rules, aiRule.RuleId)
	if allRow == nil || allRow.EgressId != hk.Id {
		t.Fatalf("unexpected all row after update: %#v", allRow)
	}
	if aiRow == nil || aiRow.EgressId != us.Id {
		t.Fatalf("unexpected ai row after update: %#v", aiRow)
	}
}

func TestSimpleRuleServiceUpdateAITargetKeepsSnapshot(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	us := createTestRuleEgress(t, "us-egress")
	jp := createTestRuleEgress(t, "jp-egress")
	inbound := createTestRuleInbound(t, 33008, "update-ai-inbound")

	aiRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: us.Id})
	if err != nil {
		t.Fatalf("create ai failed: %v", err)
	}
	before := mustExecutionItemSnapshot(t, svc, inbound.Id, simpleTrafficAI)

	if _, err := svc.UpdateSimpleRule(aiRule.RuleId, &CreateSimpleRuleRequest{
		InboundId:   inbound.Id,
		TrafficType: simpleTrafficGroup,
		GroupId:     aiGroup.Id,
		EgressId:    jp.Id,
	}); err != nil {
		t.Fatalf("update ai failed: %v", err)
	}

	after := mustExecutionItemSnapshot(t, svc, inbound.Id, simpleTrafficAI)
	if strings.Join(before, ",") != strings.Join(after, ",") {
		t.Fatalf("ai snapshot changed after target update: before=%v after=%v", before, after)
	}
	ctx := mustLoadSimplePolicyContext(t, svc, inbound.Id)
	for _, id := range ctx.ExecRemark.Items[0].RuleIDs {
		rule := ctx.RuleMap[id]
		if rule.TargetId != jp.Id {
			t.Fatalf("unexpected rule target after ai update: %#v", rule)
		}
	}
}

func TestSimpleRuleServiceSourceGroupChangeDoesNotAlterHistoricalSnapshot(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	us := createTestRuleEgress(t, "us-egress")
	inboundA := createTestRuleInbound(t, 33009, "snapshot-a")
	inboundB := createTestRuleInbound(t, 33010, "snapshot-b")

	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inboundA.Id, GroupId: aiGroup.Id, EgressId: us.Id}); err != nil {
		t.Fatalf("create inboundA ai failed: %v", err)
	}
	before := mustExecutionItemSnapshot(t, svc, inboundA.Id, simpleTrafficAI)
	if len(before) != aiGroup.RuleCount {
		t.Fatalf("unexpected initial snapshot count: %d", len(before))
	}

	if _, err := groupSvc.AddDomainRule(&AddTrafficRuleDomainRequest{
		GroupId: aiGroup.Id,
		Domain:  "full:snapshot-test.example",
	}); err != nil {
		t.Fatalf("add source domain failed: %v", err)
	}
	updatedGroup, err := groupSvc.GetGroup(aiGroup.Id)
	if err != nil {
		t.Fatalf("reload ai group failed: %v", err)
	}
	if updatedGroup.RuleCount != aiGroup.RuleCount+1 {
		t.Fatalf("unexpected updated group count: %d", updatedGroup.RuleCount)
	}

	afterOld := mustExecutionItemSnapshot(t, svc, inboundA.Id, simpleTrafficAI)
	if strings.Join(before, ",") != strings.Join(afterOld, ",") {
		t.Fatalf("historical snapshot changed: before=%v after=%v", before, afterOld)
	}
	if containsSnapshotValue(afterOld, "snapshot-test.example") {
		t.Fatalf("historical snapshot should not contain new domain: %v", afterOld)
	}

	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inboundB.Id, GroupId: aiGroup.Id, EgressId: us.Id}); err != nil {
		t.Fatalf("create inboundB ai failed: %v", err)
	}
	afterNew := mustExecutionItemSnapshot(t, svc, inboundB.Id, simpleTrafficAI)
	if len(afterNew) != updatedGroup.RuleCount {
		t.Fatalf("unexpected new snapshot count: %d", len(afterNew))
	}
	if !containsSnapshotValue(afterNew, "snapshot-test.example") {
		t.Fatalf("new snapshot should contain latest source domain: %v", afterNew)
	}
}

func TestSimpleRuleServiceDeleteLastItemCleansPolicyAndBinding(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	us := createTestRuleEgress(t, "us-egress")
	inbound := createTestRuleInbound(t, 33011, "cleanup-inbound")

	aiRule, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inbound.Id, GroupId: aiGroup.Id, EgressId: us.Id})
	if err != nil {
		t.Fatalf("create ai failed: %v", err)
	}
	if err := svc.DeleteSimpleRule(aiRule.RuleId); err != nil {
		t.Fatalf("delete ai failed: %v", err)
	}

	ctx, err := svc.loadSimplePolicyContextByInbound(inbound.Id)
	if err != nil {
		t.Fatalf("load context after cleanup failed: %v", err)
	}
	if ctx != nil {
		t.Fatalf("expected context to be cleaned up, got %#v", ctx)
	}
}

func TestSimpleRuleServiceLegacySingleRuleRemainsCompatible(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	us := createTestRuleEgress(t, "us-egress")
	sg := createTestRuleEgress(t, "sg-egress")
	jp := createTestRuleEgress(t, "jp-egress")
	inbound := createTestRuleInbound(t, 33012, "legacy-inbound")

	legacyPolicyID := createLegacySimpleAI(t, inbound, us.Id)
	list := mustListSimpleRules(t, svc)
	if len(list.Rules) != 1 {
		t.Fatalf("unexpected legacy list count: %d", len(list.Rules))
	}
	legacyRow := list.Rules[0]
	if !strings.HasPrefix(legacyRow.RuleId, legacySimpleRuleIDPrefix) {
		t.Fatalf("expected legacy rule id, got %#v", legacyRow)
	}

	if _, err := svc.UpdateSimpleRule(legacyRow.RuleId, &CreateSimpleRuleRequest{
		InboundId:   inbound.Id,
		TrafficType: legacyRow.TrafficType,
		GroupId:     aiGroup.Id,
		EgressId:    jp.Id,
	}); err != nil {
		t.Fatalf("update legacy ai failed: %v", err)
	}
	ctx := mustLoadSimplePolicyContext(t, svc, inbound.Id)
	if ctx.Policy.Id != legacyPolicyID || ctx.LegacyMeta == nil {
		t.Fatalf("expected legacy policy to remain before conversion: %#v", ctx)
	}
	for _, rule := range ctx.Rules {
		if rule.TargetId != jp.Id {
			t.Fatalf("unexpected legacy target after update: %#v", rule)
		}
	}

	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{
		InboundId:   inbound.Id,
		TrafficType: simpleTrafficAll,
		EgressId:    sg.Id,
	}); err != nil {
		t.Fatalf("add all to legacy policy failed: %v", err)
	}
	ctx = mustLoadSimplePolicyContext(t, svc, inbound.Id)
	if ctx.Policy.Id != legacyPolicyID || ctx.ExecRemark == nil || ctx.LegacyMeta != nil {
		t.Fatalf("expected legacy policy conversion in-place, got %#v", ctx)
	}
	if ctx.Policy.DefaultTargetId != sg.Id || len(ctx.ExecRemark.Items) != 1 {
		t.Fatalf("unexpected converted context: %#v %#v", ctx.Policy, ctx.ExecRemark)
	}
	list = mustListSimpleRules(t, svc)
	if len(list.Rules) != 2 {
		t.Fatalf("unexpected rule count after legacy conversion: %d", len(list.Rules))
	}
}

func TestSimpleRemarkKindsAreStrictlySeparated(t *testing.T) {
	validExec, _, err := buildSimpleExecutionRemark(&simpleExecutionRemark{
		Version: simpleExecutionRemarkVersion,
		Items: []*simpleExecutionItem{
			{TrafficType: simpleTrafficGroup, GroupId: 1, GroupType: simpleTrafficAI, RuleIDs: []int{1}},
		},
	})
	if err != nil {
		t.Fatalf("build exec remark failed: %v", err)
	}

	cases := []struct {
		name       string
		remark     string
		wantNew    bool
		wantLegacy bool
		wantOrd    bool
	}{
		{name: "new exec", remark: validExec, wantNew: true},
		{name: "legacy", remark: "n5-simple|type=group|groupId=1|groupName=AI%E5%88%86%E6%B5%81|groupType=ai", wantLegacy: true},
		{name: "ordinary", remark: "my custom policy", wantOrd: true},
		{name: "simple lookalike", remark: "n5-simple-test", wantOrd: true},
		{name: "prefixed exec text", remark: "foo n5-simple-exec|xxx", wantOrd: true},
		{name: "empty", remark: "", wantOrd: true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNewSimpleExecutionRemark(tt.remark); got != tt.wantNew {
				t.Fatalf("isNewSimpleExecutionRemark(%q)=%v want %v", tt.remark, got, tt.wantNew)
			}
			if got := isLegacySimpleRemark(tt.remark); got != tt.wantLegacy {
				t.Fatalf("isLegacySimpleRemark(%q)=%v want %v", tt.remark, got, tt.wantLegacy)
			}
			if got := isOrdinaryPolicyRemark(tt.remark); got != tt.wantOrd {
				t.Fatalf("isOrdinaryPolicyRemark(%q)=%v want %v", tt.remark, got, tt.wantOrd)
			}
		})
	}
}

func TestDecodeSimpleExecutionRemarkFailuresAreSafe(t *testing.T) {
	cases := []struct {
		name   string
		remark string
	}{
		{name: "empty payload", remark: "n5-simple-exec|"},
		{name: "invalid base64", remark: "n5-simple-exec|@@@@"},
		{name: "invalid json", remark: "n5-simple-exec|" + base64.RawURLEncoding.EncodeToString([]byte(`{"version":1,"items":[`))},
		{name: "unsupported version", remark: "n5-simple-exec|" + base64.RawURLEncoding.EncodeToString([]byte(`{"version":2,"items":[]}`))},
		{name: "duplicate item", remark: "n5-simple-exec|" + base64.RawURLEncoding.EncodeToString([]byte(`{"version":1,"items":[{"trafficType":"group","groupId":1,"groupType":"ai","ruleIds":[1]},{"trafficType":"group","groupId":1,"groupType":"ai","ruleIds":[2]}]}`))},
		{name: "invalid rule id", remark: "n5-simple-exec|" + base64.RawURLEncoding.EncodeToString([]byte(`{"version":1,"items":[{"trafficType":"group","groupId":1,"groupType":"ai","ruleIds":[0]}]}`))},
		{name: "wrong rule id type", remark: "n5-simple-exec|" + base64.RawURLEncoding.EncodeToString([]byte(`{"version":1,"items":[{"trafficType":"group","groupId":1,"groupType":"ai","ruleIds":["x"]}]}`))},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := decodeSimpleExecutionRemark(tt.remark); err == nil {
				t.Fatalf("expected decode error for %s", tt.name)
			}
			if _, ok := parseSimpleExecutionRemark(tt.remark); ok {
				t.Fatalf("parseSimpleExecutionRemark should fail for %s", tt.name)
			}
		})
	}
}

func TestSimpleExecutionRemarkEncodeDecodeRoundTrip(t *testing.T) {
	remark := &simpleExecutionRemark{
		Version: simpleExecutionRemarkVersion,
		Items: []*simpleExecutionItem{
			{TrafficType: simpleTrafficGroup, GroupId: 1, GroupName: "AI分流", GroupType: simpleTrafficAI, RuleIDs: []int{1, 2, 3}},
			{TrafficType: simpleTrafficCustomDomain, CustomDomain: "full:api.ipify.org", RuleIDs: []int{4}},
		},
	}

	encoded, stats, err := buildSimpleExecutionRemark(remark)
	if err != nil {
		t.Fatalf("buildSimpleExecutionRemark failed: %v", err)
	}
	if stats.TotalRemarkBytes != len(encoded) {
		t.Fatalf("unexpected total bytes: %#v", stats)
	}

	decoded, err := decodeSimpleExecutionRemark(encoded)
	if err != nil {
		t.Fatalf("decodeSimpleExecutionRemark failed: %v", err)
	}
	if decoded.Version != simpleExecutionRemarkVersion || len(decoded.Items) != 2 {
		t.Fatalf("unexpected decoded remark: %#v", decoded)
	}
	if decoded.Items[0].GroupId != 1 || decoded.Items[1].CustomDomain != "full:api.ipify.org" {
		t.Fatalf("unexpected decoded items: %#v", decoded.Items)
	}
}

func TestSimpleExecutionRemarkSizeProfiles(t *testing.T) {
	cases := []struct {
		name           string
		itemCount      int
		ruleIDsPerItem int
	}{
		{name: "normal", itemCount: 3, ruleIDsPerItem: 12},
		{name: "10x100", itemCount: 10, ruleIDsPerItem: 100},
		{name: "20x500", itemCount: 20, ruleIDsPerItem: 500},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			encoded, stats, err := buildSimpleExecutionRemark(buildSyntheticExecutionRemark(tt.itemCount, tt.ruleIDsPerItem))
			if err != nil {
				t.Fatalf("buildSimpleExecutionRemark failed: %v", err)
			}
			if encoded == "" || stats.RawJSONBytes <= 0 || stats.Base64Bytes <= 0 || stats.TotalRemarkBytes <= 0 {
				t.Fatalf("unexpected size stats: %#v", stats)
			}
			t.Logf("%s raw=%d base64=%d total=%d", tt.name, stats.RawJSONBytes, stats.Base64Bytes, stats.TotalRemarkBytes)
		})
	}
}

func TestSimpleExecutionRemarkOversizeIsRejected(t *testing.T) {
	_, _, err := buildSimpleExecutionRemark(buildSyntheticExecutionRemark(60, 2000))
	if err == nil || !strings.Contains(err.Error(), "元数据过大") {
		t.Fatalf("unexpected oversize error: %v", err)
	}
}

func TestListSimpleRulesSkipsCorruptedExecMetadataSafely(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	inbound := createTestRuleInbound(t, 33013, "corrupted-list-inbound")
	policySvc := &n5service.TrafficPolicyService{}
	policy, err := policySvc.Create(&n5model.TrafficPolicy{
		Name:    "corrupted-list-policy",
		Remark:  "n5-simple-exec|broken",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("create policy failed: %v", err)
	}
	if _, err := policySvc.BindInboundPolicy(inbound.Id, policy.Id); err != nil {
		t.Fatalf("bind policy failed: %v", err)
	}

	list, err := svc.ListSimpleRules()
	if err != nil {
		t.Fatalf("ListSimpleRules should not fail on corrupted metadata: %v", err)
	}
	if len(list.Rules) != 0 {
		t.Fatalf("expected corrupted metadata to be skipped, got %#v", list.Rules)
	}
}

func TestDeleteSimpleRuleRejectsCrossPolicyRuleIDTamper(t *testing.T) {
	initSimpleTestDB(t)

	svc := NewRuleService()
	groupSvc := NewTrafficRuleGroupService()
	aiGroup, _ := mustGetBuiltinGroup(groupSvc, simpleTrafficAI)
	us := createTestRuleEgress(t, "us-egress")
	jp := createTestRuleEgress(t, "jp-egress")
	inboundA := createTestRuleInbound(t, 33014, "tamper-a")
	inboundB := createTestRuleInbound(t, 33015, "tamper-b")

	aiRuleA, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inboundA.Id, GroupId: aiGroup.Id, EgressId: us.Id})
	if err != nil {
		t.Fatalf("create ai A failed: %v", err)
	}
	if _, err := svc.CreateSimpleRule(&CreateSimpleRuleRequest{InboundId: inboundB.Id, GroupId: aiGroup.Id, EgressId: jp.Id}); err != nil {
		t.Fatalf("create ai B failed: %v", err)
	}

	ctxA := mustLoadSimplePolicyContext(t, svc, inboundA.Id)
	ctxB := mustLoadSimplePolicyContext(t, svc, inboundB.Id)
	ctxA.ExecRemark.Items[0].RuleIDs = []int{ctxB.ExecRemark.Items[0].RuleIDs[0]}
	remarkValue, _, err := buildSimpleExecutionRemark(ctxA.ExecRemark)
	if err != nil {
		t.Fatalf("build tampered remark failed: %v", err)
	}
	if _, err := (&n5service.TrafficPolicyService{}).UpdatePolicy(&n5model.TrafficPolicy{
		Id:                ctxA.Policy.Id,
		Name:              ctxA.Policy.Name,
		Remark:            remarkValue,
		Enabled:           ctxA.Policy.Enabled,
		DefaultTargetType: ctxA.Policy.DefaultTargetType,
		DefaultTargetId:   ctxA.Policy.DefaultTargetId,
	}); err != nil {
		t.Fatalf("persist tampered remark failed: %v", err)
	}

	if err := svc.DeleteSimpleRule(aiRuleA.RuleId); err == nil {
		t.Fatal("expected delete to reject cross-policy rule id tamper")
	}
	var countA, countB int64
	if err := database.GetDB().Model(&n5model.TrafficPolicyRule{}).Where("policy_id = ?", ctxA.Policy.Id).Count(&countA).Error; err != nil {
		t.Fatalf("count policy A rules failed: %v", err)
	}
	if err := database.GetDB().Model(&n5model.TrafficPolicyRule{}).Where("policy_id = ?", ctxB.Policy.Id).Count(&countB).Error; err != nil {
		t.Fatalf("count policy B rules failed: %v", err)
	}
	if countA == 0 || countB == 0 {
		t.Fatalf("tampered delete should not remove rules: countA=%d countB=%d", countA, countB)
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
		{name: "bare domain defaults to suffix match", input: "openai.com", wantMode: "suffix", wantValue: "openai.com", wantDisplay: "domain:openai.com"},
		{name: "explicit domain keeps suffix match", input: "domain:openai.com", wantMode: "suffix", wantValue: "openai.com", wantDisplay: "domain:openai.com"},
		{name: "explicit full keeps exact match", input: "full:openai.com", wantMode: "exact", wantValue: "openai.com", wantDisplay: "full:openai.com"},
		{name: "keyword keeps keyword match", input: "keyword:openai", wantMode: "keyword", wantValue: "openai", wantDisplay: "keyword:openai"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMode, gotValue, gotDisplay, err := parseCustomDomainRule(tt.input)
			if err != nil {
				t.Fatalf("parseCustomDomainRule(%q) returned error: %v", tt.input, err)
			}
			if gotMode != tt.wantMode || gotValue != tt.wantValue || gotDisplay != tt.wantDisplay {
				t.Fatalf("parseCustomDomainRule(%q) = (%q, %q, %q), want (%q, %q, %q)",
					tt.input, gotMode, gotValue, gotDisplay, tt.wantMode, tt.wantValue, tt.wantDisplay)
			}
		})
	}
}

func createTestRuleEgress(t *testing.T, name string) *n5model.Egress {
	t.Helper()
	egress, err := (&n5service.EgressService{}).Create(&n5model.Egress{
		Name:         name,
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: `{"protocol":"freedom","settings":{}}`,
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}
	return egress
}

func createTestRuleInbound(t *testing.T, port int, remark string) *model.Inbound {
	t.Helper()
	inbound := &model.Inbound{
		UserId:         1,
		Remark:         remark,
		Enable:         true,
		Listen:         "0.0.0.0",
		Port:           port,
		Protocol:       model.Socks,
		Settings:       `{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`,
		StreamSettings: `{}`,
		Tag:            "tag-" + strconv.Itoa(port),
		Sniffing:       `{}`,
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}
	return inbound
}

func createLegacySimpleAI(t *testing.T, inbound *model.Inbound, egressID int) int {
	t.Helper()
	result, err := (&n5service.TrafficTemplateService{}).Create(&n5service.TrafficTemplateCreateRequest{
		TemplateName: simpleTrafficAI,
		PolicyName:   simplePolicyName(inbound, simpleTrafficAI),
		InboundId:    inbound.Id,
		TargetType:   "egress",
		TargetId:     egressID,
	})
	if err != nil {
		t.Fatalf("create legacy ai template failed: %v", err)
	}
	if err := database.GetDB().Model(&n5model.TrafficPolicy{}).Where("id = ?", result.Policy.Id).Updates(map[string]interface{}{
		"remark":              buildSimpleRuleRemark(simpleTrafficAI, ""),
		"default_target_type": "",
		"default_target_id":   0,
	}).Error; err != nil {
		t.Fatalf("mark legacy ai policy failed: %v", err)
	}
	return result.Policy.Id
}

func mustListSimpleRules(t *testing.T, svc *RuleService) *SimpleRuleListResult {
	t.Helper()
	list, err := svc.ListSimpleRules()
	if err != nil {
		t.Fatalf("list simple rules failed: %v", err)
	}
	return list
}

func mustLoadSimplePolicyContext(t *testing.T, svc *RuleService, inboundId int) *simplePolicyContext {
	t.Helper()
	ctx, err := svc.loadSimplePolicyContextByInbound(inboundId)
	if err != nil {
		t.Fatalf("load simple policy context failed: %v", err)
	}
	if ctx == nil {
		t.Fatal("expected simple policy context")
	}
	return ctx
}

func mustExecutionItemSnapshot(t *testing.T, svc *RuleService, inboundId int, groupType string) []string {
	t.Helper()
	ctx := mustLoadSimplePolicyContext(t, svc, inboundId)
	for _, item := range ctx.ExecRemark.Items {
		if normalizeSimpleGroupType(item.GroupType) != groupType {
			continue
		}
		values := make([]string, 0, len(item.RuleIDs))
		for _, ruleID := range item.RuleIDs {
			rule := ctx.RuleMap[ruleID]
			if rule != nil {
				values = append(values, rule.MatchValue)
			}
		}
		sort.Strings(values)
		return values
	}
	t.Fatalf("group item not found: %s", groupType)
	return nil
}

func findAllRule(rules []*SimpleRule, inboundId int) *SimpleRule {
	for _, rule := range rules {
		if rule.InboundId == inboundId && rule.TrafficType == simpleTrafficAll {
			return rule
		}
	}
	return nil
}

func findGroupRule(rules []*SimpleRule, inboundId int, groupType string) *SimpleRule {
	for _, rule := range rules {
		if rule.InboundId == inboundId && normalizeSimpleGroupType(rule.GroupType) == groupType {
			return rule
		}
	}
	return nil
}

func findRuleByRuleID(rules []*SimpleRule, ruleID string) *SimpleRule {
	for _, rule := range rules {
		if rule.RuleId == ruleID {
			return rule
		}
	}
	return nil
}

func containsSnapshotValue(values []string, target string) bool {
	for _, value := range values {
		if strings.Contains(value, target) {
			return true
		}
	}
	return false
}

func buildSyntheticExecutionRemark(itemCount int, ruleIDsPerItem int) *simpleExecutionRemark {
	items := make([]*simpleExecutionItem, 0, itemCount)
	for i := 0; i < itemCount; i++ {
		ruleIDs := make([]int, 0, ruleIDsPerItem)
		for j := 1; j <= ruleIDsPerItem; j++ {
			ruleIDs = append(ruleIDs, j)
		}
		items = append(items, &simpleExecutionItem{
			TrafficType: simpleTrafficGroup,
			GroupId:     i + 1,
			GroupName:   "group-" + strconv.Itoa(i+1),
			RuleIDs:     ruleIDs,
		})
	}
	return &simpleExecutionRemark{
		Version: simpleExecutionRemarkVersion,
		Items:   items,
	}
}

func inboundRoutingRules(t *testing.T, fragments map[string]interface{}, inboundTag string) []map[string]interface{} {
	t.Helper()
	rules, _ := fragments["rules"].([]interface{})
	result := make([]map[string]interface{}, 0)
	for _, item := range rules {
		rule, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		inboundTags, _ := rule["inboundTag"].([]interface{})
		if len(inboundTags) == 0 || inboundTags[0] != inboundTag {
			continue
		}
		result = append(result, rule)
	}
	return result
}

func lastInboundRule(t *testing.T, fragments map[string]interface{}, inboundTag string) map[string]interface{} {
	t.Helper()
	rules := inboundRoutingRules(t, fragments, inboundTag)
	if len(rules) == 0 {
		t.Fatalf("no inbound rules found for %s", inboundTag)
	}
	return rules[len(rules)-1]
}

func assertSimpleRuleFragment(t *testing.T, fragments map[string]interface{}, inboundTag string, outboundTag string, domain string, expect bool) {
	t.Helper()
	found := false
	for _, rule := range inboundRoutingRules(t, fragments, inboundTag) {
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
		t.Fatalf("unexpected fragment found: inbound=%s outbound=%s domain=%s", inboundTag, outboundTag, domain)
	}
}

func assertDefaultRoute(t *testing.T, fragments map[string]interface{}, inboundTag string, outboundTag string, expect bool) {
	t.Helper()
	found := false
	for _, rule := range inboundRoutingRules(t, fragments, inboundTag) {
		if _, hasDomain := rule["domain"]; hasDomain {
			continue
		}
		if _, hasIP := rule["ip"]; hasIP {
			continue
		}
		if tag, _ := rule["outboundTag"].(string); tag == outboundTag {
			found = true
			break
		}
	}
	if expect && !found {
		t.Fatalf("expected default route not found: inbound=%s outbound=%s", inboundTag, outboundTag)
	}
	if !expect && found {
		t.Fatalf("unexpected default route found: inbound=%s outbound=%s", inboundTag, outboundTag)
	}
}
