package simple

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
	"x-ui/database/model"
	n5model "x-ui/database/model/n5"
	"x-ui/util/common"
	coreservice "x-ui/web/service"
	n5service "x-ui/web/service/n5"
)

const (
	simpleRuleRemarkPrefix = "n5-simple|"

	simpleTrafficAll          = "all"
	simpleTrafficAI           = "ai"
	simpleTrafficGame         = "game"
	simpleTrafficStreaming    = "streaming"
	simpleTrafficCustomDomain = "custom-domain"
)

type inboundManager interface {
	GetInbound(id int) (*model.Inbound, error)
	GetAllInbounds() ([]*model.Inbound, error)
}

type trafficPolicyManager interface {
	Create(policy *n5model.TrafficPolicy) (*n5model.TrafficPolicy, error)
	GetPolicy(id int) (*n5model.TrafficPolicy, error)
	List() ([]*n5model.TrafficPolicy, error)
	UpdatePolicy(policy *n5model.TrafficPolicy) (*n5model.TrafficPolicy, error)
	DeletePolicy(id int) error
	AddRule(rule *n5model.TrafficPolicyRule) (*n5model.TrafficPolicyRule, error)
	ListRules(policyId int) ([]*n5model.TrafficPolicyRule, error)
	ListBindings() ([]*n5model.TrafficPolicyBinding, error)
	BindInboundPolicy(inboundId int, policyId int) (*n5model.TrafficPolicyBinding, error)
}

type trafficTemplateManager interface {
	List() ([]*n5service.TrafficTemplateSummary, error)
	Create(req *n5service.TrafficTemplateCreateRequest) (*n5service.TrafficTemplateCreateResult, error)
}

type SimpleRule struct {
	Id              int    `json:"id"`
	PolicyId        int    `json:"policyId"`
	InboundId       int    `json:"inboundId"`
	InboundName     string `json:"inboundName"`
	InboundTag      string `json:"inboundTag"`
	TrafficType     string `json:"trafficType"`
	TrafficLabel    string `json:"trafficLabel"`
	CustomDomain    string `json:"customDomain"`
	EgressId        int    `json:"egressId"`
	EgressName      string `json:"egressName"`
	EgressTag       string `json:"egressTag"`
	Status          string `json:"status"`
	Enabled         bool   `json:"enabled"`
	RuleCount       int    `json:"ruleCount"`
	CreatedAt       int64  `json:"createdAt"`
	UpdatedAt       int64  `json:"updatedAt"`
	PolicyName      string `json:"policyName"`
	PolicyRemark    string `json:"policyRemark"`
	DefaultTargetId int    `json:"defaultTargetId"`
}

type SimpleRuleListResult struct {
	Rules        []*SimpleRule             `json:"rules"`
	Inbounds     []*SimpleInboundOption    `json:"inbounds"`
	Egresses     []*SimpleRuleEgressOption `json:"egresses"`
	TrafficTypes []*SimpleTrafficOption    `json:"trafficTypes"`
}

type SimpleInboundOption struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Enabled bool   `json:"enabled"`
}

type SimpleRuleEgressOption struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Address string `json:"address"`
	Enabled bool   `json:"enabled"`
}

type SimpleTrafficOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type CreateSimpleRuleRequest struct {
	InboundId    int    `json:"inboundId" form:"inboundId"`
	TrafficType  string `json:"trafficType" form:"trafficType"`
	EgressId     int    `json:"egressId" form:"egressId"`
	CustomDomain string `json:"customDomain" form:"customDomain"`
}

type RuleService struct {
	inboundService  inboundManager
	egressService   egressManager
	policyService   trafficPolicyManager
	templateService trafficTemplateManager
}

func NewRuleService() *RuleService {
	return &RuleService{
		inboundService:  &coreservice.InboundService{},
		egressService:   &n5service.EgressService{},
		policyService:   &n5service.TrafficPolicyService{},
		templateService: &n5service.TrafficTemplateService{},
	}
}

func (s *RuleService) getInboundService() inboundManager {
	if s.inboundService != nil {
		return s.inboundService
	}
	return &coreservice.InboundService{}
}

func (s *RuleService) getEgressService() egressManager {
	if s.egressService != nil {
		return s.egressService
	}
	return &n5service.EgressService{}
}

func (s *RuleService) getPolicyService() trafficPolicyManager {
	if s.policyService != nil {
		return s.policyService
	}
	return &n5service.TrafficPolicyService{}
}

func (s *RuleService) getTemplateService() trafficTemplateManager {
	if s.templateService != nil {
		return s.templateService
	}
	return &n5service.TrafficTemplateService{}
}

func (s *RuleService) ListSimpleRules() (*SimpleRuleListResult, error) {
	policies, err := s.getPolicyService().List()
	if err != nil {
		return nil, err
	}
	bindings, err := s.getPolicyService().ListBindings()
	if err != nil {
		return nil, err
	}
	inbounds, err := s.getInboundService().GetAllInbounds()
	if err != nil {
		return nil, err
	}
	egresses, err := s.getEgressService().List()
	if err != nil {
		return nil, err
	}

	bindingByPolicy := make(map[int]*n5model.TrafficPolicyBinding, len(bindings))
	for _, binding := range bindings {
		bindingByPolicy[binding.PolicyId] = binding
	}
	inboundByID := make(map[int]*model.Inbound, len(inbounds))
	inboundOptions := make([]*SimpleInboundOption, 0, len(inbounds))
	for _, inbound := range inbounds {
		inboundByID[inbound.Id] = inbound
		inboundOptions = append(inboundOptions, &SimpleInboundOption{
			Id:      inbound.Id,
			Name:    simpleInboundName(inbound),
			Tag:     inbound.Tag,
			Enabled: inbound.Enable,
		})
	}
	egressByID := make(map[int]*n5model.Egress, len(egresses))
	egressOptions := make([]*SimpleRuleEgressOption, 0, len(egresses))
	for _, egress := range egresses {
		egressByID[egress.Id] = egress
		address, port := extractAddressPort(egress.OutboundJSON)
		label := address
		if address != "" && port > 0 {
			label = address + ":" + strconv.Itoa(port)
		}
		egressOptions = append(egressOptions, &SimpleRuleEgressOption{
			Id:      egress.Id,
			Name:    egress.Name,
			Tag:     egress.Tag,
			Address: label,
			Enabled: egress.Enabled,
		})
	}

	items := make([]*SimpleRule, 0)
	for _, policy := range policies {
		meta, ok := parseSimpleRuleRemark(policy.Remark)
		if !ok {
			continue
		}
		binding := bindingByPolicy[policy.Id]
		if binding == nil {
			continue
		}
		inbound := inboundByID[binding.InboundId]
		if inbound == nil {
			continue
		}
		rules, err := s.getPolicyService().ListRules(policy.Id)
		if err != nil {
			return nil, err
		}

		egressID := 0
		if len(rules) > 0 {
			egressID = rules[0].TargetId
		} else if strings.EqualFold(policy.DefaultTargetType, "egress") {
			egressID = policy.DefaultTargetId
		}
		egress := egressByID[egressID]
		if egress == nil {
			continue
		}

		enabled := policy.Enabled && binding.Enabled
		status := "disabled"
		if enabled {
			status = "enabled"
		}
		items = append(items, &SimpleRule{
			Id:              policy.Id,
			PolicyId:        policy.Id,
			InboundId:       inbound.Id,
			InboundName:     simpleInboundName(inbound),
			InboundTag:      inbound.Tag,
			TrafficType:     meta.TrafficType,
			TrafficLabel:    simpleTrafficLabel(meta.TrafficType),
			CustomDomain:    meta.CustomDomain,
			EgressId:        egress.Id,
			EgressName:      egress.Name,
			EgressTag:       egress.Tag,
			Status:          status,
			Enabled:         enabled,
			RuleCount:       len(rules),
			CreatedAt:       policy.CreatedAt,
			UpdatedAt:       policy.UpdatedAt,
			PolicyName:      policy.Name,
			PolicyRemark:    policy.Remark,
			DefaultTargetId: policy.DefaultTargetId,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].InboundId == items[j].InboundId {
			return items[i].PolicyId < items[j].PolicyId
		}
		return items[i].InboundId < items[j].InboundId
	})

	return &SimpleRuleListResult{
		Rules:        items,
		Inbounds:     inboundOptions,
		Egresses:     egressOptions,
		TrafficTypes: s.buildTrafficTypes(),
	}, nil
}

func (s *RuleService) CreateSimpleRule(req *CreateSimpleRuleRequest) (*SimpleRule, error) {
	if req == nil {
		return nil, common.NewError("simple rule request is nil")
	}
	trafficType := normalizeSimpleTrafficType(req.TrafficType)
	if !isSupportedSimpleTrafficType(trafficType) {
		return nil, common.NewError("unsupported traffic type")
	}
	inbound, err := s.getInboundService().GetInbound(req.InboundId)
	if err != nil {
		return nil, err
	}
	egress, err := s.getEgressService().Get(req.EgressId)
	if err != nil {
		return nil, err
	}

	if err := s.ensureInboundAvailableForSimpleRule(inbound.Id); err != nil {
		return nil, err
	}

	switch trafficType {
	case simpleTrafficAll:
		return s.createAllRule(inbound, egress)
	case simpleTrafficAI, simpleTrafficGame, simpleTrafficStreaming:
		return s.createTemplateRule(inbound, egress, trafficType)
	case simpleTrafficCustomDomain:
		return s.createCustomDomainRule(inbound, egress, req.CustomDomain)
	default:
		return nil, common.NewError("unsupported traffic type")
	}
}

func (s *RuleService) DeleteSimpleRule(policyId int) error {
	if policyId <= 0 {
		return common.NewError("invalid simple rule id")
	}
	policy, err := s.getPolicyService().GetPolicy(policyId)
	if err != nil {
		return err
	}
	if _, ok := parseSimpleRuleRemark(policy.Remark); !ok {
		return common.NewError("simple rule not found")
	}
	return s.getPolicyService().DeletePolicy(policyId)
}

func (s *RuleService) createAllRule(inbound *model.Inbound, egress *n5model.Egress) (*SimpleRule, error) {
	policy, err := s.getPolicyService().Create(&n5model.TrafficPolicy{
		Name:              simplePolicyName(inbound, simpleTrafficAll),
		Remark:            buildSimpleRuleRemark(simpleTrafficAll, ""),
		Enabled:           true,
		DefaultTargetType: "egress",
		DefaultTargetId:   egress.Id,
	})
	if err != nil {
		return nil, err
	}
	binding, err := s.getPolicyService().BindInboundPolicy(inbound.Id, policy.Id)
	if err != nil {
		_ = s.getPolicyService().DeletePolicy(policy.Id)
		return nil, err
	}
	return buildSimpleRule(policy, nil, binding, inbound, egress, simpleTrafficAll, ""), nil
}

func (s *RuleService) createTemplateRule(inbound *model.Inbound, egress *n5model.Egress, trafficType string) (*SimpleRule, error) {
	result, err := s.getTemplateService().Create(&n5service.TrafficTemplateCreateRequest{
		TemplateName: trafficType,
		PolicyName:   simplePolicyName(inbound, trafficType),
		InboundId:    inbound.Id,
		TargetType:   "egress",
		TargetId:     egress.Id,
	})
	if err != nil {
		return nil, err
	}
	policy, err := s.getPolicyService().UpdatePolicy(&n5model.TrafficPolicy{
		Id:                result.Policy.Id,
		Name:              result.Policy.Name,
		Remark:            buildSimpleRuleRemark(trafficType, ""),
		Enabled:           result.Policy.Enabled,
		DefaultTargetType: "",
		DefaultTargetId:   0,
	})
	if err != nil {
		_ = s.getPolicyService().DeletePolicy(result.Policy.Id)
		return nil, err
	}
	return buildSimpleRule(policy, result.Rules, result.Binding, inbound, egress, trafficType, ""), nil
}

func (s *RuleService) createCustomDomainRule(inbound *model.Inbound, egress *n5model.Egress, customDomain string) (*SimpleRule, error) {
	matchMode, matchValue, displayValue, err := parseCustomDomainRule(customDomain)
	if err != nil {
		return nil, err
	}

	policy, err := s.getPolicyService().Create(&n5model.TrafficPolicy{
		Name:    simplePolicyName(inbound, simpleTrafficCustomDomain),
		Remark:  buildSimpleRuleRemark(simpleTrafficCustomDomain, displayValue),
		Enabled: true,
	})
	if err != nil {
		return nil, err
	}

	rule, err := s.getPolicyService().AddRule(&n5model.TrafficPolicyRule{
		PolicyId:   policy.Id,
		RuleType:   "domain",
		MatchMode:  matchMode,
		MatchValue: matchValue,
		TargetType: "egress",
		TargetId:   egress.Id,
		Enabled:    true,
	})
	if err != nil {
		_ = s.getPolicyService().DeletePolicy(policy.Id)
		return nil, err
	}
	binding, err := s.getPolicyService().BindInboundPolicy(inbound.Id, policy.Id)
	if err != nil {
		_ = s.getPolicyService().DeletePolicy(policy.Id)
		return nil, err
	}
	return buildSimpleRule(policy, []*n5model.TrafficPolicyRule{rule}, binding, inbound, egress, simpleTrafficCustomDomain, displayValue), nil
}

func (s *RuleService) ensureInboundAvailableForSimpleRule(inboundId int) error {
	bindings, err := s.getPolicyService().ListBindings()
	if err != nil {
		return err
	}
	for _, binding := range bindings {
		if binding.InboundId != inboundId {
			continue
		}
		policy, err := s.getPolicyService().GetPolicy(binding.PolicyId)
		if err != nil {
			return err
		}
		if _, ok := parseSimpleRuleRemark(policy.Remark); ok {
			return s.getPolicyService().DeletePolicy(policy.Id)
		}
		return common.NewError("inbound already bound to advanced traffic policy")
	}
	return nil
}

func (s *RuleService) buildTrafficTypes() []*SimpleTrafficOption {
	items := []*SimpleTrafficOption{
		{
			Value:       simpleTrafficAll,
			Label:       simpleTrafficLabel(simpleTrafficAll),
			Description: "该入口的全部流量走指定出口。",
		},
		{
			Value:       simpleTrafficCustomDomain,
			Label:       simpleTrafficLabel(simpleTrafficCustomDomain),
			Description: "为单个域名生成简单分流规则。",
		},
	}

	templates, err := s.getTemplateService().List()
	if err != nil {
		return items
	}
	for _, template := range templates {
		items = append(items, &SimpleTrafficOption{
			Value:       normalizeSimpleTrafficType(template.Name),
			Label:       template.DisplayName,
			Description: template.Description,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		order := map[string]int{
			simpleTrafficAll:          1,
			simpleTrafficAI:           2,
			simpleTrafficGame:         3,
			simpleTrafficStreaming:    4,
			simpleTrafficCustomDomain: 5,
		}
		return order[items[i].Value] < order[items[j].Value]
	})
	return items
}

type simpleRuleRemark struct {
	TrafficType  string
	CustomDomain string
}

func buildSimpleRule(policy *n5model.TrafficPolicy, rules []*n5model.TrafficPolicyRule, binding *n5model.TrafficPolicyBinding, inbound *model.Inbound, egress *n5model.Egress, trafficType string, customDomain string) *SimpleRule {
	enabled := policy != nil && policy.Enabled && binding != nil && binding.Enabled
	status := "disabled"
	if enabled {
		status = "enabled"
	}
	ruleCount := len(rules)
	if trafficType == simpleTrafficAll {
		ruleCount = 0
	}
	return &SimpleRule{
		Id:              policy.Id,
		PolicyId:        policy.Id,
		InboundId:       inbound.Id,
		InboundName:     simpleInboundName(inbound),
		InboundTag:      inbound.Tag,
		TrafficType:     trafficType,
		TrafficLabel:    simpleTrafficLabel(trafficType),
		CustomDomain:    customDomain,
		EgressId:        egress.Id,
		EgressName:      egress.Name,
		EgressTag:       egress.Tag,
		Status:          status,
		Enabled:         enabled,
		RuleCount:       ruleCount,
		CreatedAt:       policy.CreatedAt,
		UpdatedAt:       policy.UpdatedAt,
		PolicyName:      policy.Name,
		PolicyRemark:    policy.Remark,
		DefaultTargetId: policy.DefaultTargetId,
	}
}

func buildSimpleRuleRemark(trafficType string, customDomain string) string {
	values := []string{
		simpleRuleRemarkPrefix + "type=" + normalizeSimpleTrafficType(trafficType),
	}
	if strings.TrimSpace(customDomain) != "" {
		values = append(values, "value="+url.QueryEscape(strings.TrimSpace(customDomain)))
	}
	return strings.Join(values, "|")
}

func parseSimpleRuleRemark(remark string) (*simpleRuleRemark, bool) {
	remark = strings.TrimSpace(remark)
	if !strings.HasPrefix(remark, simpleRuleRemarkPrefix) {
		return nil, false
	}
	meta := &simpleRuleRemark{}
	parts := strings.Split(remark, "|")
	for _, part := range parts {
		if strings.HasPrefix(part, simpleRuleRemarkPrefix) {
			part = strings.TrimPrefix(part, simpleRuleRemarkPrefix)
		}
		if strings.HasPrefix(part, "type=") {
			meta.TrafficType = normalizeSimpleTrafficType(strings.TrimPrefix(part, "type="))
		}
		if strings.HasPrefix(part, "value=") {
			value, err := url.QueryUnescape(strings.TrimPrefix(part, "value="))
			if err == nil {
				meta.CustomDomain = value
			}
		}
	}
	if !isSupportedSimpleTrafficType(meta.TrafficType) {
		return nil, false
	}
	return meta, true
}

func simplePolicyName(inbound *model.Inbound, trafficType string) string {
	name := simpleInboundName(inbound)
	if name == "" {
		name = "inbound-" + strconv.Itoa(inbound.Id)
	}
	return "Simple " + simpleTrafficLabel(trafficType) + " - " + name
}

func simpleInboundName(inbound *model.Inbound) string {
	if inbound == nil {
		return ""
	}
	if value := strings.TrimSpace(inbound.Remark); value != "" {
		return value
	}
	if value := strings.TrimSpace(inbound.Tag); value != "" {
		return value
	}
	return "inbound-" + strconv.Itoa(inbound.Id)
}

func simpleTrafficLabel(trafficType string) string {
	switch normalizeSimpleTrafficType(trafficType) {
	case simpleTrafficAll:
		return "全部流量"
	case simpleTrafficAI:
		return "AI 分流"
	case simpleTrafficGame:
		return "游戏分流"
	case simpleTrafficStreaming:
		return "流媒体分流"
	case simpleTrafficCustomDomain:
		return "自定义域名"
	default:
		return trafficType
	}
}

func normalizeSimpleTrafficType(trafficType string) string {
	return strings.TrimSpace(strings.ToLower(trafficType))
}

func isSupportedSimpleTrafficType(trafficType string) bool {
	switch normalizeSimpleTrafficType(trafficType) {
	case simpleTrafficAll, simpleTrafficAI, simpleTrafficGame, simpleTrafficStreaming, simpleTrafficCustomDomain:
		return true
	default:
		return false
	}
}

func parseCustomDomainRule(raw string) (string, string, string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", "", "", common.NewError("custom domain is required")
	}
	switch {
	case strings.HasPrefix(value, "full:"):
		match := strings.TrimSpace(strings.TrimPrefix(value, "full:"))
		if match == "" {
			return "", "", "", common.NewError("custom domain is required")
		}
		return "exact", match, "full:" + match, nil
	case strings.HasPrefix(value, "domain:"):
		match := strings.TrimSpace(strings.TrimPrefix(value, "domain:"))
		if match == "" {
			return "", "", "", common.NewError("custom domain is required")
		}
		return "suffix", match, "domain:" + match, nil
	case strings.HasPrefix(value, "keyword:"):
		match := strings.TrimSpace(strings.TrimPrefix(value, "keyword:"))
		if match == "" {
			return "", "", "", common.NewError("custom domain is required")
		}
		return "keyword", match, "keyword:" + match, nil
	default:
		return "suffix", value, "domain:" + value, nil
	}
}