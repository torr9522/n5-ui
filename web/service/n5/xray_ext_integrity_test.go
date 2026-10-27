package n5

import (
	"errors"
	"fmt"
	"testing"

	"x-ui/database"
	legacyModel "x-ui/database/model"
	n5model "x-ui/database/model/n5"

	"gorm.io/gorm"
)

func createIntegrityRoutingFixture(t *testing.T, inboundCount int) ([]*legacyModel.Inbound, *n5model.TrafficPolicy) {
	t.Helper()
	initTestDB(t)

	egress, err := (&EgressService{}).Create(&n5model.Egress{
		Name:         "integrity-egress",
		Protocol:     "freedom",
		Enabled:      true,
		OutboundJSON: freedomOutboundJSON(),
	})
	if err != nil {
		t.Fatalf("create egress failed: %v", err)
	}
	policy, err := (&TrafficPolicyService{}).Create(&n5model.TrafficPolicy{
		Name:              "integrity-policy",
		Enabled:           true,
		DefaultTargetType: targetTypeEgress,
		DefaultTargetId:   egress.Id,
	})
	if err != nil {
		t.Fatalf("create policy failed: %v", err)
	}

	inbounds := make([]*legacyModel.Inbound, 0, inboundCount)
	for index := 0; index < inboundCount; index++ {
		inbound := &legacyModel.Inbound{
			UserId: 1, Remark: fmt.Sprintf("integrity-%d", index), Enable: true,
			Listen: "127.0.0.1", Port: 36000 + index, Protocol: legacyModel.Socks,
			Settings:       `{"auth":"noauth","udp":false,"ip":"127.0.0.1"}`,
			StreamSettings: `{}`, Tag: fmt.Sprintf("integrity-inbound-%d", index), Sniffing: `{}`,
		}
		if err := database.GetDB().Create(inbound).Error; err != nil {
			t.Fatalf("create inbound failed: %v", err)
		}
		if err := database.GetDB().Create(&n5model.TrafficPolicyBinding{
			InboundId: inbound.Id, PolicyId: policy.Id, Enabled: true,
		}).Error; err != nil {
			t.Fatalf("create binding failed: %v", err)
		}
		inbounds = append(inbounds, inbound)
	}
	return inbounds, policy
}

func addDanglingBinding(t *testing.T, inboundID int, policyID int) {
	t.Helper()
	if err := database.GetDB().Create(&n5model.TrafficPolicyBinding{
		InboundId: inboundID, PolicyId: policyID, Enabled: true,
	}).Error; err != nil {
		t.Fatalf("create dangling binding failed: %v", err)
	}
}

func routingInboundTags(t *testing.T, routing map[string]interface{}) map[string]int {
	t.Helper()
	tags := make(map[string]int)
	for _, rawRule := range routing["rules"].([]interface{}) {
		rule := rawRule.(map[string]interface{})
		switch inboundTags := rule["inboundTag"].(type) {
		case []string:
			for _, tag := range inboundTags {
				tags[tag]++
			}
		case []interface{}:
			for _, rawTag := range inboundTags {
				tags[rawTag.(string)]++
			}
		default:
			t.Fatalf("unexpected inboundTag type %T", rule["inboundTag"])
		}
	}
	return tags
}

func TestGenerateConfigSkipsDanglingTrafficPolicyBinding(t *testing.T) {
	_, policy := createIntegrityRoutingFixture(t, 0)
	addDanglingBinding(t, 999999, policy.Id)

	routing, err := (&XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("dangling binding must not fail generation: %v", err)
	}
	if len(routing["rules"].([]interface{})) != 0 {
		t.Fatalf("dangling binding generated routing rules: %#v", routing)
	}
}

func TestGenerateConfigPreservesValidBindingsWhenOneIsDangling(t *testing.T) {
	inbounds, policy := createIntegrityRoutingFixture(t, 2)
	addDanglingBinding(t, 999999, policy.Id)

	routing, err := (&XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("mixed binding generation failed: %v", err)
	}
	tags := routingInboundTags(t, routing)
	for _, inbound := range inbounds {
		if tags[inbound.Tag] != 1 {
			t.Fatalf("valid inbound routing was not preserved for %s: %#v", inbound.Tag, routing)
		}
	}
	if len(tags) != 2 {
		t.Fatalf("dangling binding leaked into routing: %#v", routing)
	}
}

func TestGenerateConfigWithAllDanglingBindingsProducesBaseFragment(t *testing.T) {
	_, policy := createIntegrityRoutingFixture(t, 0)
	for _, inboundID := range []int{4, 6, 7, 8} {
		addDanglingBinding(t, inboundID, policy.Id)
	}
	routing, err := (&XrayExtService{}).GenerateRoutingFragments()
	if err != nil {
		t.Fatalf("all-dangling generation failed: %v", err)
	}
	if len(routing["rules"].([]interface{})) != 0 {
		t.Fatalf("all-dangling bindings generated rules: %#v", routing)
	}
}

func TestGenerateConfigReturnsNonNotFoundDatabaseErrors(t *testing.T) {
	createIntegrityRoutingFixture(t, 1)
	db := database.GetDB()
	wantErr := errors.New("injected inbound query failure")
	callbackName := "n5:test_inbound_query_error"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "inbounds" {
			tx.AddError(wantErr)
		}
	}); err != nil {
		t.Fatalf("register query callback failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove(callbackName) })

	_, err := (&XrayExtService{}).GenerateRoutingFragments()
	if !errors.Is(err, wantErr) {
		t.Fatalf("non-not-found error was not preserved: got %v want %v", err, wantErr)
	}
}
