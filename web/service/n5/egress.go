package n5

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"os"
	"strings"
	"time"
	"x-ui/database"
	n5model "x-ui/database/model/n5"
	"x-ui/util/common"
	"x-ui/util/json_util"
	"x-ui/xray"
)

type EgressService struct {
}

func (s *EgressService) GenerateStableTag(id int) string {
	return formatN5Tag("n5-egress", id)
}

func (s *EgressService) ValidateConfig(protocol string, outboundJSON string, tag string) (string, string, error) {
	obj, err := parseJSONObject(outboundJSON)
	if err != nil {
		return "", "", common.NewErrorf("invalid outbound json: %v", err)
	}

	jsonProtocol := ""
	if value, ok := obj["protocol"].(string); ok {
		jsonProtocol = normalizeProtocol(value)
	}
	protocol = normalizeProtocol(protocol)
	if protocol == "" {
		protocol = jsonProtocol
	}
	if protocol == "" {
		return "", "", common.NewError("protocol is required")
	}
	if jsonProtocol != "" && jsonProtocol != protocol {
		return "", "", common.NewError("protocol does not match outbound json")
	}
	if strings.TrimSpace(tag) == "" {
		return "", "", common.NewError("tag is required")
	}

	obj["protocol"] = protocol
	obj["tag"] = strings.TrimSpace(tag)

	data, err := json.Marshal(obj)
	if err != nil {
		return "", "", err
	}

	testConfig := &xray.Config{
		InboundConfigs:  []xray.InboundConfig{},
		OutboundConfigs: json_util.RawMessage("[" + string(data) + "]"),
	}
	if _, err := os.Stat(xray.GetBinaryPath()); err == nil {
		if err := xray.TestConfig(testConfig); err != nil {
			return "", "", err
		}
	} else {
		if err := validateOutboundShape(protocol, obj); err != nil {
			return "", "", err
		}
	}

	return protocol, string(data), nil
}

func (s *EgressService) Create(egress *n5model.Egress) (*n5model.Egress, error) {
	if egress == nil {
		return nil, common.NewError("egress is nil")
	}

	db := database.GetDB()
	now := time.Now().UnixNano()
	record := &n5model.Egress{
		Name:         normalizeName(egress.Name),
		Remark:       strings.TrimSpace(egress.Remark),
		Protocol:     normalizeProtocol(egress.Protocol),
		Enabled:      egress.Enabled,
		OutboundJSON: egress.OutboundJSON,
		Tag:          fmt.Sprintf("n5-egress-pending-%d", now),
	}
	if !egress.Enabled {
		record.Enabled = false
	} else {
		record.Enabled = true
	}
	if record.Name == "" {
		return nil, common.NewError("egress name is required")
	}
	if strings.TrimSpace(record.OutboundJSON) == "" {
		return nil, common.NewError("outbound json is required")
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(record).Error; err != nil {
			return err
		}

		tag := s.GenerateStableTag(record.Id)
		protocol, normalizedJSON, err := s.ValidateConfig(record.Protocol, record.OutboundJSON, tag)
		if err != nil {
			return err
		}

		record.Tag = tag
		record.Protocol = protocol
		record.OutboundJSON = normalizedJSON
		return tx.Save(record).Error
	})
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *EgressService) Update(egress *n5model.Egress) (*n5model.Egress, error) {
	if egress == nil || egress.Id <= 0 {
		return nil, common.NewError("invalid egress")
	}

	db := database.GetDB()
	record := &n5model.Egress{}
	if err := db.Model(&n5model.Egress{}).Where("id = ?", egress.Id).First(record).Error; err != nil {
		return nil, err
	}

	name := normalizeName(egress.Name)
	if name == "" {
		return nil, common.NewError("egress name is required")
	}
	outboundJSON := egress.OutboundJSON
	if strings.TrimSpace(outboundJSON) == "" {
		return nil, common.NewError("outbound json is required")
	}

	protocol, normalizedJSON, err := s.ValidateConfig(egress.Protocol, outboundJSON, record.Tag)
	if err != nil {
		return nil, err
	}

	record.Name = name
	record.Remark = strings.TrimSpace(egress.Remark)
	record.Protocol = protocol
	record.Enabled = egress.Enabled
	record.OutboundJSON = normalizedJSON
	if err := db.Save(record).Error; err != nil {
		return nil, err
	}

	return record, nil
}

func (s *EgressService) Delete(id int) error {
	if id <= 0 {
		return common.NewError("invalid egress id")
	}
	return database.GetDB().Delete(&n5model.Egress{}, id).Error
}

func (s *EgressService) Get(id int) (*n5model.Egress, error) {
	if id <= 0 {
		return nil, common.NewError("invalid egress id")
	}
	record := &n5model.Egress{}
	if err := database.GetDB().Model(&n5model.Egress{}).Where("id = ?", id).First(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *EgressService) List() ([]*n5model.Egress, error) {
	records := make([]*n5model.Egress, 0)
	err := database.GetDB().Model(&n5model.Egress{}).Order("id asc").Find(&records).Error
	return records, err
}
