package simple

import (
	"encoding/json"
	"strconv"
	"strings"
	n5model "x-ui/database/model/n5"
	"x-ui/util/common"
	n5service "x-ui/web/service/n5"
)

type egressManager interface {
	List() ([]*n5model.Egress, error)
	Create(egress *n5model.Egress) (*n5model.Egress, error)
	Update(egress *n5model.Egress) (*n5model.Egress, error)
	Delete(id int) error
	Get(id int) (*n5model.Egress, error)
}

type egressTester interface {
	Test(egressId int) (*n5model.EgressTest, error)
}

type SimpleEgress struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	Protocol     string `json:"protocol"`
	Address      string `json:"address"`
	Port         int    `json:"port"`
	Username     string `json:"username"`
	Method       string `json:"method"`
	Password     string `json:"password"`
	Status       string `json:"status"`
	ExitIP       string `json:"exitIp"`
	TestTime     int64  `json:"testTime"`
	LatencyMs    int    `json:"latencyMs"`
	LastError    string `json:"lastError"`
	Enabled      bool   `json:"enabled"`
	InternalTag  string `json:"internalTag"`
	DisplayLabel string `json:"displayLabel"`
}

type CreateSimpleEgressRequest struct {
	Name     string `json:"name" form:"name"`
	Protocol string `json:"protocol" form:"protocol"`
	Address  string `json:"address" form:"address"`
	Port     int    `json:"port" form:"port"`
	Username string `json:"username" form:"username"`
	Method   string `json:"method" form:"method"`
	Password string `json:"password" form:"password"`
	Enabled  bool   `json:"enabled" form:"enabled"`
}

type SimpleEgressTestResult struct {
	Id       int    `json:"id"`
	Status   string `json:"status"`
	Latency  int    `json:"latency"`
	ExitIP   string `json:"exitIp"`
	Message  string `json:"message"`
	TestedAt int64  `json:"testedAt"`
}

type EgressService struct {
	egressService egressManager
	testService   egressTester
}

func NewEgressService() *EgressService {
	return &EgressService{
		egressService: &n5service.EgressService{},
		testService:   &n5service.EgressTestService{},
	}
}

func (s *EgressService) getEgressService() egressManager {
	if s.egressService != nil {
		return s.egressService
	}
	return &n5service.EgressService{}
}

func (s *EgressService) getTestService() egressTester {
	if s.testService != nil {
		return s.testService
	}
	return &n5service.EgressTestService{}
}

func (s *EgressService) ListSimpleEgress() ([]*SimpleEgress, error) {
	records, err := s.getEgressService().List()
	if err != nil {
		return nil, err
	}

	items := make([]*SimpleEgress, 0, len(records))
	for _, record := range records {
		items = append(items, toSimpleEgress(record))
	}
	return items, nil
}

func (s *EgressService) GetSimpleEgress(id int) (*SimpleEgress, error) {
	record, err := s.getEgressService().Get(id)
	if err != nil {
		return nil, err
	}
	return toSimpleEgress(record), nil
}

func (s *EgressService) CreateSimpleEgress(req *CreateSimpleEgressRequest) (*SimpleEgress, error) {
	if req == nil {
		return nil, common.NewError("simple egress request is nil")
	}

	record, err := s.saveSimpleEgress(0, req)
	if err != nil {
		return nil, err
	}
	return toSimpleEgress(record), nil
}

func (s *EgressService) UpdateSimpleEgress(id int, req *CreateSimpleEgressRequest) (*SimpleEgress, error) {
	if id <= 0 {
		return nil, common.NewError("invalid simple egress id")
	}
	if req == nil {
		return nil, common.NewError("simple egress request is nil")
	}

	record, err := s.saveSimpleEgress(id, req)
	if err != nil {
		return nil, err
	}
	return toSimpleEgress(record), nil
}

func (s *EgressService) saveSimpleEgress(id int, req *CreateSimpleEgressRequest) (*n5model.Egress, error) {
	protocol := normalizeSimpleProtocol(req.Protocol)
	address := strings.TrimSpace(req.Address)
	if address == "" {
		return nil, common.NewError("address is required")
	}
	if req.Port <= 0 || req.Port > 65535 {
		return nil, common.NewError("port is invalid")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, common.NewError("name is required")
	}

	outbound, err := buildSimpleOutbound(protocol, req, address)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(outbound)
	if err != nil {
		return nil, err
	}

	if id > 0 {
		return s.getEgressService().Update(&n5model.Egress{
			Id:           id,
			Name:         name,
			Protocol:     protocol,
			Enabled:      req.Enabled,
			OutboundJSON: string(data),
		})
	}

	return s.getEgressService().Create(&n5model.Egress{
		Name:         name,
		Protocol:     protocol,
		Enabled:      req.Enabled,
		OutboundJSON: string(data),
	})
}

func (s *EgressService) DeleteSimpleEgress(id int) error {
	return s.getEgressService().Delete(id)
}

func (s *EgressService) TestSimpleEgress(id int) (*SimpleEgressTestResult, error) {
	record, err := s.getTestService().Test(id)
	if err != nil {
		return nil, err
	}
	return &SimpleEgressTestResult{
		Id:       record.Id,
		Status:   record.Status,
		Latency:  record.Latency,
		ExitIP:   record.ExitIP,
		Message:  record.Message,
		TestedAt: record.TestedAt,
	}, nil
}

func normalizeSimpleProtocol(protocol string) string {
	protocol = strings.TrimSpace(strings.ToLower(protocol))
	switch protocol {
	case "", "socks", "socks5":
		return "socks"
	case "ss", "shadowsocks":
		return "shadowsocks"
	default:
		return protocol
	}
}

func toSimpleEgress(record *n5model.Egress) *SimpleEgress {
	if record == nil {
		return nil
	}
	address, port, username, method, password := extractSimpleConfig(record.Protocol, record.OutboundJSON)
	item := &SimpleEgress{
		Id:          record.Id,
		Name:        record.Name,
		Protocol:    toSimpleDisplayProtocol(record.Protocol),
		Address:     address,
		Port:        port,
		Username:    username,
		Method:      method,
		Password:    password,
		Status:      record.LastStatus,
		ExitIP:      record.LastExitIP,
		TestTime:    record.LastTestTime,
		LatencyMs:   record.LastTestLatencyMs,
		LastError:   record.LastTestError,
		Enabled:     record.Enabled,
		InternalTag: record.Tag,
	}
	if item.Address != "" && item.Port > 0 {
		item.DisplayLabel = item.Address + ":" + strconv.Itoa(item.Port)
	} else if item.Address != "" {
		item.DisplayLabel = item.Address
	} else {
		item.DisplayLabel = "-"
	}
	return item
}

func extractAddressPort(outboundJSON string) (string, int) {
	address, port, _, _, _ := extractSimpleConfig("", outboundJSON)
	return address, port
}

func extractSimpleConfig(protocol string, outboundJSON string) (string, int, string, string, string) {
	obj := make(map[string]interface{})
	if err := json.Unmarshal([]byte(outboundJSON), &obj); err != nil {
		return "", 0, "", "", ""
	}
	settings, ok := obj["settings"].(map[string]interface{})
	if !ok {
		return "", 0, "", "", ""
	}
	servers, ok := settings["servers"].([]interface{})
	if !ok || len(servers) == 0 {
		return "", 0, "", "", ""
	}
	server, ok := servers[0].(map[string]interface{})
	if !ok {
		return "", 0, "", "", ""
	}
	address, _ := server["address"].(string)
	port := 0
	switch value := server["port"].(type) {
	case float64:
		port = int(value)
	case int:
		port = value
	}
	username := ""
	method := ""
	password := ""
	switch normalizeSimpleProtocol(protocol) {
	case "socks":
		users, ok := server["users"].([]interface{})
		if ok && len(users) > 0 {
			user, ok := users[0].(map[string]interface{})
			if ok {
				username, _ = user["user"].(string)
				password, _ = user["pass"].(string)
			}
		}
	case "shadowsocks":
		method, _ = server["method"].(string)
		password, _ = server["password"].(string)
	}
	return strings.TrimSpace(address), port, strings.TrimSpace(username), strings.TrimSpace(method), strings.TrimSpace(password)
}

func buildSimpleOutbound(protocol string, req *CreateSimpleEgressRequest, address string) (map[string]interface{}, error) {
	switch protocol {
	case "socks":
		server := map[string]interface{}{
			"address": address,
			"port":    req.Port,
		}
		if strings.TrimSpace(req.Username) != "" || strings.TrimSpace(req.Password) != "" {
			server["users"] = []map[string]string{
				{
					"user": strings.TrimSpace(req.Username),
					"pass": strings.TrimSpace(req.Password),
				},
			}
		}
		return map[string]interface{}{
			"protocol": protocol,
			"settings": map[string]interface{}{
				"servers": []map[string]interface{}{server},
			},
		}, nil
	case "shadowsocks":
		method := strings.TrimSpace(strings.ToLower(req.Method))
		if method == "" {
			return nil, common.NewError("method is required")
		}
		password := strings.TrimSpace(req.Password)
		if password == "" {
			return nil, common.NewError("password is required")
		}
		return map[string]interface{}{
			"protocol": protocol,
			"settings": map[string]interface{}{
				"servers": []map[string]interface{}{
					{
						"address":  address,
						"port":     req.Port,
						"method":   method,
						"password": password,
					},
				},
			},
		}, nil
	default:
		return nil, common.NewError("simple mode only supports socks5 and shadowsocks")
	}
}

func toSimpleDisplayProtocol(protocol string) string {
	switch normalizeSimpleProtocol(protocol) {
	case "socks":
		return "socks5"
	case "shadowsocks":
		return "ss"
	default:
		return strings.TrimSpace(strings.ToLower(protocol))
	}
}
