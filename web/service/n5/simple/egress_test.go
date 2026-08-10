package simple

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"x-ui/database"
	n5model "x-ui/database/model/n5"
	n5service "x-ui/web/service/n5"
)

type fakeTester struct {
	record *n5model.EgressTest
	err    error
}

func (f *fakeTester) Test(egressId int) (*n5model.EgressTest, error) {
	if f.record != nil {
		f.record.EgressId = egressId
	}
	return f.record, f.err
}

func initSimpleTestDB(t *testing.T) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "simple.db")
	if err := database.InitDB(dbPath); err != nil {
		t.Fatalf("init db failed: %v", err)
	}
}

func TestSimpleEgressServiceCreateSocksAndList(t *testing.T) {
	initSimpleTestDB(t)

	svc := &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &fakeTester{},
	}

	created, err := svc.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name:     "simple-socks",
		Protocol: "socks5",
		Address:  "example.com",
		Port:     1080,
		Username: "demo-user",
		Password: "demo-pass",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create simple egress failed: %v", err)
	}
	if created.Address != "example.com" || created.Port != 1080 {
		t.Fatalf("unexpected created egress: %#v", created)
	}

	items, err := svc.ListSimpleEgress()
	if err != nil {
		t.Fatalf("list simple egress failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected list count: %d", len(items))
	}
	if items[0].InternalTag != "n5-egress-0000000001" {
		t.Fatalf("unexpected tag: %#v", items[0])
	}
	if items[0].Protocol != "socks5" {
		t.Fatalf("unexpected protocol: %#v", items[0])
	}

	record, err := (&n5service.EgressService{}).Get(created.Id)
	if err != nil {
		t.Fatalf("get egress failed: %v", err)
	}
	if record.Protocol != "socks" {
		t.Fatalf("unexpected stored protocol: %#v", record)
	}

	outbound := make(map[string]interface{})
	if err := json.Unmarshal([]byte(record.OutboundJSON), &outbound); err != nil {
		t.Fatalf("unmarshal outbound failed: %v", err)
	}
	settings, ok := outbound["settings"].(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected outbound settings: %#v", outbound)
	}
	servers, ok := settings["servers"].([]interface{})
	if !ok || len(servers) != 1 {
		t.Fatalf("unexpected socks servers: %#v", settings)
	}
	server, ok := servers[0].(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected socks server: %#v", servers[0])
	}
	users, ok := server["users"].([]interface{})
	if !ok || len(users) != 1 {
		t.Fatalf("unexpected socks users: %#v", server)
	}
}

func TestSimpleEgressServiceDelete(t *testing.T) {
	initSimpleTestDB(t)

	svc := &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &fakeTester{},
	}

	created, err := svc.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name:     "delete-simple-egress",
		Protocol: "socks5",
		Address:  "example.org",
		Port:     2080,
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create simple egress failed: %v", err)
	}

	if err := svc.DeleteSimpleEgress(created.Id); err != nil {
		t.Fatalf("delete simple egress failed: %v", err)
	}

	items, err := svc.ListSimpleEgress()
	if err != nil {
		t.Fatalf("list simple egress failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("unexpected list count after delete: %d", len(items))
	}
}

func TestSimpleEgressServiceTest(t *testing.T) {
	initSimpleTestDB(t)

	svc := &EgressService{
		egressService: &n5service.EgressService{},
		testService: &fakeTester{
			record: &n5model.EgressTest{
				Status:   "success",
				Latency:  25,
				ExitIP:   "203.0.113.10",
				Message:  "",
				TestedAt: 1234567890,
			},
		},
	}

	created, err := svc.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name:     "test-simple-egress",
		Protocol: "socks",
		Address:  "example.net",
		Port:     3080,
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create simple egress failed: %v", err)
	}

	record, err := svc.TestSimpleEgress(created.Id)
	if err != nil {
		t.Fatalf("test simple egress failed: %v", err)
	}
	if record.Status != "success" || record.ExitIP != "203.0.113.10" {
		t.Fatalf("unexpected test result: %#v", record)
	}
}

func TestSimpleEgressServiceCreateShadowsocksConfig(t *testing.T) {
	initSimpleTestDB(t)

	svc := &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &fakeTester{},
	}

	created, err := svc.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name:     "simple-ss",
		Protocol: "ss",
		Address:  "ss.example.com",
		Port:     8388,
		Method:   "aes-256-gcm",
		Password: "ss-password",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create simple shadowsocks egress failed: %v", err)
	}
	if created.Protocol != "ss" {
		t.Fatalf("unexpected created protocol: %#v", created)
	}

	record, err := (&n5service.EgressService{}).Get(created.Id)
	if err != nil {
		t.Fatalf("get egress failed: %v", err)
	}
	if record.Protocol != "shadowsocks" {
		t.Fatalf("unexpected stored protocol: %#v", record)
	}

	outbound := make(map[string]interface{})
	if err := json.Unmarshal([]byte(record.OutboundJSON), &outbound); err != nil {
		t.Fatalf("unmarshal outbound failed: %v", err)
	}
	if outbound["protocol"] != "shadowsocks" {
		t.Fatalf("unexpected outbound protocol: %#v", outbound)
	}
	settings, ok := outbound["settings"].(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected outbound settings: %#v", outbound)
	}
	servers, ok := settings["servers"].([]interface{})
	if !ok || len(servers) != 1 {
		t.Fatalf("unexpected shadowsocks servers: %#v", settings)
	}
	server, ok := servers[0].(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected shadowsocks server: %#v", servers[0])
	}
	if server["address"] != "ss.example.com" || int(server["port"].(float64)) != 8388 {
		t.Fatalf("unexpected shadowsocks address/port: %#v", server)
	}
	if server["method"] != "aes-256-gcm" || server["password"] != "ss-password" {
		t.Fatalf("unexpected shadowsocks config: %#v", server)
	}
}

func TestSimpleEgressServiceUpdateSocksKeepsTag(t *testing.T) {
	initSimpleTestDB(t)

	svc := &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &fakeTester{},
	}

	created, err := svc.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name:     "editable-socks",
		Protocol: "socks5",
		Address:  "old.example.com",
		Port:     1080,
		Username: "old-user",
		Password: "old-pass",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create simple egress failed: %v", err)
	}
	oldTag := created.InternalTag

	updated, err := svc.UpdateSimpleEgress(created.Id, &CreateSimpleEgressRequest{
		Name:     "editable-socks-new",
		Protocol: "socks5",
		Address:  "new.example.com",
		Port:     2080,
		Username: "new-user",
		Password: "new-pass",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("update simple egress failed: %v", err)
	}
	if updated.InternalTag != oldTag {
		t.Fatalf("tag changed after update: old=%s new=%s", oldTag, updated.InternalTag)
	}
	if updated.Address != "new.example.com" || updated.Port != 2080 || updated.Username != "new-user" {
		t.Fatalf("unexpected updated socks egress: %#v", updated)
	}

	record, err := (&n5service.EgressService{}).Get(created.Id)
	if err != nil {
		t.Fatalf("get egress failed: %v", err)
	}
	if record.Tag != oldTag {
		t.Fatalf("stored tag changed after update: %#v", record)
	}

	outbound := make(map[string]interface{})
	if err := json.Unmarshal([]byte(record.OutboundJSON), &outbound); err != nil {
		t.Fatalf("unmarshal outbound failed: %v", err)
	}
	settings := outbound["settings"].(map[string]interface{})
	server := settings["servers"].([]interface{})[0].(map[string]interface{})
	if server["address"] != "new.example.com" || int(server["port"].(float64)) != 2080 {
		t.Fatalf("unexpected updated socks server: %#v", server)
	}
	users := server["users"].([]interface{})
	user := users[0].(map[string]interface{})
	if user["user"] != "new-user" || user["pass"] != "new-pass" {
		t.Fatalf("unexpected updated socks auth: %#v", user)
	}
}

func TestSimpleEgressServiceUpdateShadowsocksKeepsTag(t *testing.T) {
	initSimpleTestDB(t)

	svc := &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &fakeTester{},
	}

	created, err := svc.CreateSimpleEgress(&CreateSimpleEgressRequest{
		Name:     "editable-ss",
		Protocol: "ss",
		Address:  "old-ss.example.com",
		Port:     8388,
		Method:   "aes-128-gcm",
		Password: "old-ss-pass",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create shadowsocks egress failed: %v", err)
	}
	oldTag := created.InternalTag

	updated, err := svc.UpdateSimpleEgress(created.Id, &CreateSimpleEgressRequest{
		Name:     "editable-ss-new",
		Protocol: "ss",
		Address:  "new-ss.example.com",
		Port:     8488,
		Method:   "chacha20-poly1305",
		Password: "new-ss-pass",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("update shadowsocks egress failed: %v", err)
	}
	if updated.InternalTag != oldTag {
		t.Fatalf("tag changed after update: old=%s new=%s", oldTag, updated.InternalTag)
	}
	if updated.Method != "chacha20-poly1305" || updated.Password != "new-ss-pass" {
		t.Fatalf("unexpected updated shadowsocks egress: %#v", updated)
	}

	record, err := (&n5service.EgressService{}).Get(created.Id)
	if err != nil {
		t.Fatalf("get egress failed: %v", err)
	}
	if record.Tag != oldTag {
		t.Fatalf("stored tag changed after update: %#v", record)
	}

	outbound := make(map[string]interface{})
	if err := json.Unmarshal([]byte(record.OutboundJSON), &outbound); err != nil {
		t.Fatalf("unmarshal outbound failed: %v", err)
	}
	settings := outbound["settings"].(map[string]interface{})
	server := settings["servers"].([]interface{})[0].(map[string]interface{})
	if server["address"] != "new-ss.example.com" || int(server["port"].(float64)) != 8488 {
		t.Fatalf("unexpected updated shadowsocks server: %#v", server)
	}
	if server["method"] != "chacha20-poly1305" || server["password"] != "new-ss-pass" {
		t.Fatalf("unexpected updated shadowsocks config: %#v", server)
	}
}
