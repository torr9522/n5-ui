package n5

import (
	"strings"
	"x-ui/database"
	n5model "x-ui/database/model/n5"
	"x-ui/util/common"

	"gorm.io/gorm"
)

type TrafficPolicyService struct {
}

func (s *TrafficPolicyService) Create(policy *n5model.TrafficPolicy) (*n5model.TrafficPolicy, error) {
	if policy == nil {
		return nil, common.NewError("traffic policy is nil")
	}

	record := &n5model.TrafficPolicy{
		Name:              normalizeName(policy.Name),
		Remark:            strings.TrimSpace(policy.Remark),
		Enabled:           policy.Enabled,
		DefaultTargetType: normalizeTargetType(policy.DefaultTargetType),
		DefaultTargetId:   policy.DefaultTargetId,
	}
	if !policy.Enabled {
		record.Enabled = false
	} else {
		record.Enabled = true
	}
	if record.Name == "" {
		return nil, common.NewError("policy name is required")
	}
	if record.DefaultTargetType != "" || record.DefaultTargetId > 0 {
		if err := validateTarget(record.DefaultTargetType, record.DefaultTargetId); err != nil {
			return nil, err
		}
	}

	if err := database.GetDB().Create(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *TrafficPolicyService) Get(id int) (*n5model.TrafficPolicy, error) {
	return s.GetPolicy(id)
}

func (s *TrafficPolicyService) GetPolicy(id int) (*n5model.TrafficPolicy, error) {
	if id <= 0 {
		return nil, common.NewError("invalid policy id")
	}
	record := &n5model.TrafficPolicy{}
	if err := database.GetDB().Model(&n5model.TrafficPolicy{}).Where("id = ?", id).First(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *TrafficPolicyService) List() ([]*n5model.TrafficPolicy, error) {
	records := make([]*n5model.TrafficPolicy, 0)
	err := database.GetDB().Model(&n5model.TrafficPolicy{}).Order("id asc").Find(&records).Error
	return records, err
}

func (s *TrafficPolicyService) UpdatePolicy(policy *n5model.TrafficPolicy) (*n5model.TrafficPolicy, error) {
	if policy == nil || policy.Id <= 0 {
		return nil, common.NewError("invalid traffic policy")
	}

	record, err := s.GetPolicy(policy.Id)
	if err != nil {
		return nil, err
	}

	name := normalizeName(policy.Name)
	if name == "" {
		return nil, common.NewError("policy name is required")
	}
	targetType := normalizeTargetType(policy.DefaultTargetType)
	if targetType != "" || policy.DefaultTargetId > 0 {
		if err := validateTarget(targetType, policy.DefaultTargetId); err != nil {
			return nil, err
		}
	}

	record.Name = name
	record.Remark = strings.TrimSpace(policy.Remark)
	record.DefaultTargetType = targetType
	record.DefaultTargetId = policy.DefaultTargetId
	record.Enabled = policy.Enabled
	if err := database.GetDB().Save(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *TrafficPolicyService) DeletePolicy(id int) error {
	if id <= 0 {
		return common.NewError("invalid policy id")
	}

	return database.GetDB().Transaction(func(tx *gorm.DB) error {
		record := &n5model.TrafficPolicy{}
		if err := tx.Model(&n5model.TrafficPolicy{}).Where("id = ?", id).First(record).Error; err != nil {
			return err
		}
		if err := tx.Where("policy_id = ?", id).Delete(&n5model.TrafficPolicyRule{}).Error; err != nil {
			return err
		}
		if err := tx.Where("policy_id = ?", id).Delete(&n5model.TrafficPolicyBinding{}).Error; err != nil {
			return err
		}
		return tx.Delete(&n5model.TrafficPolicy{}, id).Error
	})
}

func (s *TrafficPolicyService) EnablePolicy(id int) (*n5model.TrafficPolicy, error) {
	return s.updatePolicyEnabled(id, true)
}

func (s *TrafficPolicyService) DisablePolicy(id int) (*n5model.TrafficPolicy, error) {
	return s.updatePolicyEnabled(id, false)
}

func (s *TrafficPolicyService) updatePolicyEnabled(id int, enabled bool) (*n5model.TrafficPolicy, error) {
	record, err := s.GetPolicy(id)
	if err != nil {
		return nil, err
	}
	record.Enabled = enabled
	if err := database.GetDB().Save(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *TrafficPolicyService) AddRule(rule *n5model.TrafficPolicyRule) (*n5model.TrafficPolicyRule, error) {
	if rule == nil || rule.PolicyId <= 0 {
		return nil, common.NewError("invalid traffic policy rule")
	}

	db := database.GetDB()
	var policyCount int64
	if err := db.Model(&n5model.TrafficPolicy{}).Where("id = ?", rule.PolicyId).Count(&policyCount).Error; err != nil {
		return nil, err
	}
	if policyCount == 0 {
		return nil, common.NewError("traffic policy not found")
	}

	ruleType := normalizeRuleType(rule.RuleType)
	matchMode := normalizeMatchMode(rule.MatchMode)
	matchValue := strings.TrimSpace(rule.MatchValue)
	targetType := normalizeTargetType(rule.TargetType)
	if err := validateTarget(targetType, rule.TargetId); err != nil {
		return nil, err
	}

	switch ruleType {
	case ruleTypeDomain:
		if err := validateDomainRule(matchMode, matchValue); err != nil {
			return nil, err
		}
	case ruleTypeIP:
		if err := validateIPRule(matchMode, matchValue); err != nil {
			return nil, err
		}
	default:
		return nil, common.NewError("invalid rule type")
	}

	record := &n5model.TrafficPolicyRule{
		PolicyId:   rule.PolicyId,
		RuleType:   ruleType,
		MatchMode:  matchMode,
		MatchValue: matchValue,
		TargetType: targetType,
		TargetId:   rule.TargetId,
		SortOrder:  rule.SortOrder,
		Enabled:    rule.Enabled,
	}
	if !rule.Enabled {
		record.Enabled = false
	} else {
		record.Enabled = true
	}
	if record.SortOrder <= 0 {
		var maxSort int
		db.Model(&n5model.TrafficPolicyRule{}).Where("policy_id = ?", rule.PolicyId).Select("coalesce(max(sort_order), 0)").Scan(&maxSort)
		record.SortOrder = maxSort + 1
	}

	if err := db.Create(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *TrafficPolicyService) DeleteRule(ruleId int) error {
	if ruleId <= 0 {
		return common.NewError("invalid rule id")
	}
	return database.GetDB().Delete(&n5model.TrafficPolicyRule{}, ruleId).Error
}

func (s *TrafficPolicyService) ListRules(policyId int) ([]*n5model.TrafficPolicyRule, error) {
	if policyId <= 0 {
		return nil, common.NewError("invalid policy id")
	}
	records := make([]*n5model.TrafficPolicyRule, 0)
	err := database.GetDB().Model(&n5model.TrafficPolicyRule{}).
		Where("policy_id = ?", policyId).
		Order("sort_order asc, id asc").
		Find(&records).Error
	return records, err
}

func (s *TrafficPolicyService) UpdateRule(rule *n5model.TrafficPolicyRule) (*n5model.TrafficPolicyRule, error) {
	if rule == nil || rule.Id <= 0 {
		return nil, common.NewError("invalid traffic policy rule")
	}

	record := &n5model.TrafficPolicyRule{}
	db := database.GetDB()
	if err := db.Model(&n5model.TrafficPolicyRule{}).Where("id = ?", rule.Id).First(record).Error; err != nil {
		return nil, err
	}

	ruleType := normalizeRuleType(rule.RuleType)
	matchMode := normalizeMatchMode(rule.MatchMode)
	matchValue := strings.TrimSpace(rule.MatchValue)
	targetType := normalizeTargetType(rule.TargetType)
	if err := validateTarget(targetType, rule.TargetId); err != nil {
		return nil, err
	}
	switch ruleType {
	case ruleTypeDomain:
		if err := validateDomainRule(matchMode, matchValue); err != nil {
			return nil, err
		}
	case ruleTypeIP:
		if err := validateIPRule(matchMode, matchValue); err != nil {
			return nil, err
		}
	default:
		return nil, common.NewError("invalid rule type")
	}

	record.RuleType = ruleType
	record.MatchMode = matchMode
	record.MatchValue = matchValue
	record.TargetType = targetType
	record.TargetId = rule.TargetId
	if rule.SortOrder > 0 {
		record.SortOrder = rule.SortOrder
	}
	if err := db.Save(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *TrafficPolicyService) EnableRule(id int) (*n5model.TrafficPolicyRule, error) {
	return s.updateRuleEnabled(id, true)
}

func (s *TrafficPolicyService) DisableRule(id int) (*n5model.TrafficPolicyRule, error) {
	return s.updateRuleEnabled(id, false)
}

func (s *TrafficPolicyService) updateRuleEnabled(id int, enabled bool) (*n5model.TrafficPolicyRule, error) {
	if id <= 0 {
		return nil, common.NewError("invalid rule id")
	}
	record := &n5model.TrafficPolicyRule{}
	db := database.GetDB()
	if err := db.Model(&n5model.TrafficPolicyRule{}).Where("id = ?", id).First(record).Error; err != nil {
		return nil, err
	}
	record.Enabled = enabled
	if err := db.Save(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *TrafficPolicyService) ReorderRules(policyId int, ruleIds []int) error {
	if policyId <= 0 {
		return common.NewError("invalid policy id")
	}
	if len(ruleIds) == 0 {
		return common.NewError("rule ids are required")
	}

	return database.GetDB().Transaction(func(tx *gorm.DB) error {
		records := make([]*n5model.TrafficPolicyRule, 0)
		if err := tx.Model(&n5model.TrafficPolicyRule{}).
			Where("policy_id = ?", policyId).
			Order("sort_order asc, id asc").
			Find(&records).Error; err != nil {
			return err
		}
		if len(records) != len(ruleIds) {
			return common.NewError("rule reorder count does not match")
		}

		recordMap := make(map[int]*n5model.TrafficPolicyRule, len(records))
		for _, record := range records {
			recordMap[record.Id] = record
		}
		for index, ruleId := range ruleIds {
			record, ok := recordMap[ruleId]
			if !ok {
				return common.NewError("rule not found in policy")
			}
			record.SortOrder = index + 1
			if err := tx.Save(record).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *TrafficPolicyService) BindInboundPolicy(inboundId int, policyId int) (*n5model.TrafficPolicyBinding, error) {
	return s.RebindInboundPolicy(inboundId, policyId)
}

func (s *TrafficPolicyService) RebindInboundPolicy(inboundId int, policyId int) (*n5model.TrafficPolicyBinding, error) {
	if inboundId <= 0 || policyId <= 0 {
		return nil, common.NewError("invalid policy binding")
	}
	if _, err := getInboundByID(inboundId); err != nil {
		return nil, err
	}
	db := database.GetDB()
	var policyCount int64
	if err := db.Model(&n5model.TrafficPolicy{}).Where("id = ?", policyId).Count(&policyCount).Error; err != nil {
		return nil, err
	}
	if policyCount == 0 {
		return nil, common.NewError("traffic policy not found")
	}

	record := &n5model.TrafficPolicyBinding{}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("inbound_id = ?", inboundId).Delete(&n5model.TrafficPolicyBinding{}).Error; err != nil {
			return err
		}
		record.InboundId = inboundId
		record.PolicyId = policyId
		record.Enabled = true
		return tx.Create(record).Error
	})
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *TrafficPolicyService) UnbindInboundPolicy(inboundId int) error {
	if inboundId <= 0 {
		return common.NewError("invalid inbound id")
	}
	if _, err := getInboundByID(inboundId); err != nil {
		return err
	}
	return database.GetDB().Where("inbound_id = ?", inboundId).Delete(&n5model.TrafficPolicyBinding{}).Error
}

func (s *TrafficPolicyService) ListBindings() ([]*n5model.TrafficPolicyBinding, error) {
	records := make([]*n5model.TrafficPolicyBinding, 0)
	err := database.GetDB().Model(&n5model.TrafficPolicyBinding{}).Order("inbound_id asc, id asc").Find(&records).Error
	return records, err
}

func (s *TrafficPolicyService) ListBindingsByPolicy(policyId int) ([]*n5model.TrafficPolicyBinding, error) {
	if policyId <= 0 {
		return nil, common.NewError("invalid policy id")
	}
	records := make([]*n5model.TrafficPolicyBinding, 0)
	err := database.GetDB().
		Model(&n5model.TrafficPolicyBinding{}).
		Where("policy_id = ?", policyId).
		Order("inbound_id asc, id asc").
		Find(&records).Error
	return records, err
}
