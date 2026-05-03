package service

import (
	"context"
	"crypto/rand"
	"crypto/rc4"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"brights/api/internal/domain"
	"brights/api/internal/storage"

	"gorm.io/gorm"
)

const miotSpecBaseURL = "https://home.miot-spec.com/spec/"

var miotSpecDataPagePattern = regexp.MustCompile(`data-page=\"(.*?)\">`)

func (s *Service) GetLearnerXiaomiConfig(ctx context.Context, learnerID uint) (domain.XiaomiConfig, error) {
	if learnerID == 0 {
		return domain.XiaomiConfig{}, errors.New("learner id is required")
	}
	model, exists, err := s.getLearnerXiaomiConfigModel(ctx, learnerID)
	if err != nil {
		return domain.XiaomiConfig{}, err
	}
	if !exists {
		return domain.XiaomiConfig{
			LearnerUserID: learnerID,
			Server:        "cn",
		}, nil
	}
	return toLearnerXiaomiConfig(model), nil
}

func (s *Service) SaveLearnerXiaomiConfig(ctx context.Context, learnerID uint, input domain.SaveXiaomiConfigInput) (domain.XiaomiConfig, error) {
	if learnerID == 0 {
		return domain.XiaomiConfig{}, errors.New("learner id is required")
	}

	serverValue := normalizeXiaomiServer(input.Server)
	now := time.Now().UTC()
	model, exists, err := s.getLearnerXiaomiConfigModel(ctx, learnerID)
	if err != nil {
		return domain.XiaomiConfig{}, err
	}
	if !exists {
		model = storage.XiaomiConfig{
			LearnerUserID: learnerID,
		}
	}

	model.Username = strings.TrimSpace(input.Username)
	model.XiaomiUserID = strings.TrimSpace(input.XiaomiUserID)
	model.Server = serverValue
	model.IsActive = input.IsActive
	if strings.TrimSpace(input.Ssecurity) != "" {
		model.Ssecurity = strings.TrimSpace(input.Ssecurity)
	}
	if strings.TrimSpace(input.ServiceToken) != "" {
		model.ServiceToken = strings.TrimSpace(input.ServiceToken)
	}
	if exists {
		if err := s.db.WithContext(ctx).Save(&model).Error; err != nil {
			return domain.XiaomiConfig{}, err
		}
	} else {
		if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
			return domain.XiaomiConfig{}, err
		}
	}
	if model.LastSyncAt == nil && model.DeviceList != "" {
		model.LastSyncAt = &now
	}
	return toLearnerXiaomiConfig(model), nil
}

func (s *Service) ClearLearnerXiaomiTokens(ctx context.Context, learnerID uint) error {
	if learnerID == 0 {
		return errors.New("learner id is required")
	}
	model, exists, err := s.getLearnerXiaomiConfigModel(ctx, learnerID)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	updates := map[string]interface{}{
		"ssecurity":    "",
		"service_token": "",
		"device_list":  "",
		"is_active":    false,
		"last_sync_at": nil,
	}
	return s.db.WithContext(ctx).Model(&model).Updates(updates).Error
}

func (s *Service) ListLearnerXiaomiHomes(ctx context.Context, learnerID uint) ([]domain.XiaomiHome, error) {
	client, apiBaseURL, cfg, err := s.newLearnerXiaomiCloudClient(ctx, learnerID)
	if err != nil {
		return nil, err
	}
	homes, err := s.getXiaomiHomes(client, apiBaseURL, cfg.Ssecurity)
	if err != nil {
		return nil, err
	}

	items := make([]domain.XiaomiHome, 0, len(homes))
	for _, home := range homes {
		items = append(items, domain.XiaomiHome{
			ID:      firstNonEmpty(mapString(home, "id"), mapString(home, "home_id")),
			Name:    firstNonEmpty(mapString(home, "name"), mapString(home, "home_name")),
			OwnerID: firstNonEmpty(mapString(home, "home_owner"), mapString(home, "uid")),
			Raw:     home,
		})
	}
	return items, nil
}

func (s *Service) ListLearnerXiaomiDevices(ctx context.Context, learnerID uint, refresh bool) (domain.XiaomiDeviceListResult, error) {
	model, exists, err := s.getLearnerXiaomiConfigModel(ctx, learnerID)
	if err != nil {
		return domain.XiaomiDeviceListResult{}, err
	}
	if !exists {
		return domain.XiaomiDeviceListResult{
			Account: domain.XiaomiAccountSnapshot{
				Server: "cn",
			},
			Devices: []domain.XiaomiDevice{},
		}, nil
	}

	refreshed := false
	if refresh || strings.TrimSpace(model.DeviceList) == "" {
		if _, err := s.RefreshLearnerXiaomiDevices(ctx, learnerID); err != nil {
			return domain.XiaomiDeviceListResult{}, err
		}
		model, _, err = s.getLearnerXiaomiConfigModel(ctx, learnerID)
		if err != nil {
			return domain.XiaomiDeviceListResult{}, err
		}
		refreshed = true
	}

	devices, err := parseStoredXiaomiDevices(model.DeviceList)
	if err != nil {
		return domain.XiaomiDeviceListResult{}, err
	}

	return domain.XiaomiDeviceListResult{
		Account:   toLearnerXiaomiAccountSnapshot(model),
		Devices:   devices,
		Total:     len(devices),
		Refreshed: refreshed,
	}, nil
}

func (s *Service) RefreshLearnerXiaomiDevices(ctx context.Context, learnerID uint) ([]domain.XiaomiDevice, error) {
	client, apiBaseURL, cfg, err := s.newLearnerXiaomiCloudClient(ctx, learnerID)
	if err != nil {
		return nil, err
	}

	deviceListJSON, devices, err := s.extractXiaomiDeviceList(client, apiBaseURL, cfg)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	updates := map[string]interface{}{
		"device_list":   deviceListJSON,
		"last_sync_at":  &now,
		"is_active":     true,
	}
	if err := s.db.WithContext(ctx).Model(&cfg).Updates(updates).Error; err != nil {
		return nil, err
	}
	return devices, nil
}

func (s *Service) LearnerXiaomiPropGet(ctx context.Context, learnerID uint, input domain.XiaomiPropGetInput) (interface{}, error) {
	if strings.TrimSpace(input.Did) == "" || input.Siid <= 0 || input.Piid <= 0 {
		return nil, errors.New("did, siid, and piid are required")
	}
	client, apiBaseURL, cfg, err := s.newLearnerXiaomiCloudClient(ctx, learnerID)
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"params": []map[string]interface{}{{
			"did":  input.Did,
			"siid": input.Siid,
			"piid": input.Piid,
		}},
		"datasource": 1,
	}
	body, _ := json.Marshal(payload)
	result, err := s.executeXiaomiAPICall(client, apiBaseURL+"/miotspec/prop/get", cfg.Ssecurity, map[string]string{
		"data": string(body),
	})
	if err != nil {
		return nil, err
	}
	return result["result"], nil
}

func (s *Service) LearnerXiaomiPropSet(ctx context.Context, learnerID uint, input domain.XiaomiPropSetInput) (interface{}, error) {
	if strings.TrimSpace(input.Did) == "" || input.Siid <= 0 || input.Piid <= 0 {
		return nil, errors.New("did, siid, and piid are required")
	}
	client, apiBaseURL, cfg, err := s.newLearnerXiaomiCloudClient(ctx, learnerID)
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"params": []map[string]interface{}{{
			"did":   input.Did,
			"siid":  input.Siid,
			"piid":  input.Piid,
			"value": input.Value,
		}},
	}
	body, _ := json.Marshal(payload)
	result, err := s.executeXiaomiAPICall(client, apiBaseURL+"/miotspec/prop/set", cfg.Ssecurity, map[string]string{
		"data": string(body),
	})
	if err != nil {
		return nil, err
	}
	return result["result"], nil
}

func (s *Service) LearnerXiaomiAction(ctx context.Context, learnerID uint, input domain.XiaomiActionInput) (interface{}, error) {
	if strings.TrimSpace(input.Did) == "" || input.Siid <= 0 || input.Aiid <= 0 {
		return nil, errors.New("did, siid, and aiid are required")
	}
	client, apiBaseURL, cfg, err := s.newLearnerXiaomiCloudClient(ctx, learnerID)
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"params": map[string]interface{}{
			"did":  input.Did,
			"siid": input.Siid,
			"aiid": input.Aiid,
		},
	}
	if len(input.In) > 0 {
		payload["params"].(map[string]interface{})["in"] = input.In
	}
	body, _ := json.Marshal(payload)
	result, err := s.executeXiaomiAPICall(client, apiBaseURL+"/miotspec/action", cfg.Ssecurity, map[string]string{
		"data": string(body),
	})
	if err != nil {
		return nil, err
	}
	return result["result"], nil
}

func (s *Service) LearnerXiaomiPropGetBatch(ctx context.Context, learnerID uint, items []domain.XiaomiBatchPropItem) (interface{}, error) {
	if len(items) == 0 {
		return []interface{}{}, nil
	}
	params := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Did) == "" || item.Siid <= 0 || item.Piid <= 0 {
			continue
		}
		params = append(params, map[string]interface{}{
			"did":  item.Did,
			"siid": item.Siid,
			"piid": item.Piid,
		})
	}
	if len(params) == 0 {
		return []interface{}{}, nil
	}
	client, apiBaseURL, cfg, err := s.newLearnerXiaomiCloudClient(ctx, learnerID)
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"params":     params,
		"datasource": 1,
	}
	body, _ := json.Marshal(payload)
	result, err := s.executeXiaomiAPICall(client, apiBaseURL+"/miotspec/prop/get", cfg.Ssecurity, map[string]string{
		"data": string(body),
	})
	if err != nil {
		return nil, err
	}
	return result["result"], nil
}

func (s *Service) FindLearnerXiaomiDevices(ctx context.Context, learnerID uint, query string, limit int) ([]domain.XiaomiDeviceMatch, error) {
	devices, err := s.ensureLearnerXiaomiDevices(ctx, learnerID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}
	needle := strings.ToLower(strings.TrimSpace(query))
	items := make([]domain.XiaomiDeviceMatch, 0, limit)
	for _, device := range devices {
		if needle != "" {
			if !strings.Contains(strings.ToLower(device.Name), needle) &&
				!strings.Contains(strings.ToLower(device.Model), needle) &&
				!strings.Contains(strings.ToLower(device.Did), needle) &&
				!strings.Contains(strings.ToLower(device.RoomName), needle) &&
				!strings.Contains(strings.ToLower(device.HomeName), needle) {
				continue
			}
		}
		items = append(items, domain.XiaomiDeviceMatch{
			Did:      device.Did,
			Name:     device.Name,
			Model:    device.Model,
			SpecType: device.SpecType,
		})
		if len(items) >= limit {
			break
		}
	}
	return items, nil
}

func (s *Service) GetLearnerMiotSpec(ctx context.Context, model string) (domain.MiotSpecParsed, []byte, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return domain.MiotSpecParsed{}, nil, errors.New("model is required")
	}
	var cache storage.MiotSpecCache
	if err := s.db.WithContext(ctx).Where("model = ?", model).First(&cache).Error; err == nil {
		var parsed domain.MiotSpecParsed
		if parseErr := json.Unmarshal([]byte(cache.SpecJSON), &parsed); parseErr == nil {
			return parsed, []byte(cache.SpecJSON), nil
		}
	}

	html, etag, err := httpGetXiaomiMiotSpecPage(model)
	if err != nil {
		return domain.MiotSpecParsed{}, nil, err
	}
	dataPageJSON, err := extractMiotSpecDataPageJSON(html)
	if err != nil {
		return domain.MiotSpecParsed{}, nil, err
	}
	parsed, err := parseMiotSpecFromDataPage(model, dataPageJSON)
	if err != nil {
		return domain.MiotSpecParsed{}, nil, err
	}
	raw, _ := json.Marshal(parsed)
	now := time.Now().UTC()
	_ = s.db.WithContext(ctx).Save(&storage.MiotSpecCache{
		Model:     model,
		SpecJSON:  string(raw),
		ETag:      etag,
		FetchedAt: &now,
	}).Error
	return parsed, raw, nil
}

func (s *Service) GetLearnerXiaomiDeviceStatus(
	ctx context.Context,
	learnerID uint,
	did string,
	properties []string,
	includeMetadata bool,
) (map[string]interface{}, error) {
	device, err := s.getLearnerXiaomiDeviceByDID(ctx, learnerID, did)
	if err != nil {
		return nil, err
	}
	spec, _, err := s.GetLearnerMiotSpec(ctx, device.Model)
	if err != nil {
		return nil, err
	}

	availableProps := make(map[string]interface{})
	for _, property := range spec.Properties {
		availableProps[property.Name] = summarizeMiotProperty(property)
	}
	availableActions := make(map[string]interface{})
	for _, action := range spec.Actions {
		availableActions[action.Name] = map[string]interface{}{
			"desc": action.Description,
			"siid": action.Method.SIID,
			"aiid": action.Method.AIID,
		}
	}

	response := map[string]interface{}{
		"success": true,
		"device": map[string]interface{}{
			"did":   device.Did,
			"name":  device.Name,
			"model": device.Model,
		},
	}

	if len(properties) > 0 {
		items := make([]domain.XiaomiBatchPropItem, 0, len(properties))
		propertyMap := make(map[string]domain.MiotSpecProperty)
		for _, name := range properties {
			property, err := pickMiotPropByCandidates(spec, []string{name})
			if err != nil {
				continue
			}
			if !strings.Contains(property.RW, "r") {
				continue
			}
			propertyMap[property.Name] = property
			items = append(items, domain.XiaomiBatchPropItem{
				Did:  device.Did,
				Siid: int64(property.Method.SIID),
				Piid: int64(property.Method.PIID),
			})
		}
		if len(items) > 0 {
			values, err := s.LearnerXiaomiPropGetBatch(ctx, learnerID, items)
			if err != nil {
				return nil, err
			}
			response["properties"] = values
		}
	}

	if includeMetadata {
		response["available_properties"] = availableProps
		response["available_actions"] = availableActions
	}
	return response, nil
}

func (s *Service) ControlLearnerXiaomiDevice(ctx context.Context, learnerID uint, input domain.XiaomiControlDeviceInput) (map[string]interface{}, error) {
	matches, err := s.FindLearnerXiaomiDevices(ctx, learnerID, input.Query, 1)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, errors.New("xiaomi device not found")
	}
	device, err := s.getLearnerXiaomiDeviceByDID(ctx, learnerID, matches[0].Did)
	if err != nil {
		return nil, err
	}
	spec, _, err := s.GetLearnerMiotSpec(ctx, device.Model)
	if err != nil {
		return nil, err
	}

	switch strings.TrimSpace(firstNonEmpty(input.Operation, "set_property")) {
	case "run_action":
		action, err := pickMiotAction(spec, input.ActionName)
		if err != nil {
			return nil, err
		}
		var in []interface{}
		if input.ActionValue != nil {
			switch typed := input.ActionValue.(type) {
			case []interface{}:
				in = typed
			default:
				in = []interface{}{typed}
			}
		}
		result, err := s.LearnerXiaomiAction(ctx, learnerID, domain.XiaomiActionInput{
			Did:  device.Did,
			Siid: int64(action.Method.SIID),
			Aiid: int64(action.Method.AIID),
			In:   in,
		})
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"success": true,
			"device":  device,
			"action": map[string]interface{}{
				"name": action.Name,
				"siid": action.Method.SIID,
				"aiid": action.Method.AIID,
			},
			"result": result,
		}, nil
	default:
		property, err := pickMiotPropByCandidates(spec, []string{input.PropName})
		if err != nil {
			return nil, err
		}
		if !strings.Contains(property.RW, "w") {
			return nil, errors.New("property is not writable")
		}
		if err := validateValueAgainstMiotProperty(property, input.Value); err != nil {
			return nil, err
		}
		result, err := s.LearnerXiaomiPropSet(ctx, learnerID, domain.XiaomiPropSetInput{
			Did:   device.Did,
			Siid:  int64(property.Method.SIID),
			Piid:  int64(property.Method.PIID),
			Value: input.Value,
		})
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"success": true,
			"device":  device,
			"property": map[string]interface{}{
				"name":  property.Name,
				"siid":  property.Method.SIID,
				"piid":  property.Method.PIID,
				"value": input.Value,
			},
			"result": result,
		}, nil
	}
}

func (s *Service) MijiaSwitchSet(ctx context.Context, learnerID uint, did string, on bool) (map[string]interface{}, error) {
	device, err := s.getLearnerXiaomiDeviceByDID(ctx, learnerID, did)
	if err != nil {
		return nil, err
	}
	spec, _, err := s.GetLearnerMiotSpec(ctx, device.Model)
	if err != nil {
		return nil, err
	}
	property, err := pickMiotSwitchProperty(spec)
	if err != nil {
		return nil, err
	}
	result, err := s.LearnerXiaomiPropSet(ctx, learnerID, domain.XiaomiPropSetInput{
		Did:   did,
		Siid:  int64(property.Method.SIID),
		Piid:  int64(property.Method.PIID),
		Value: on,
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"success": true,
		"device":  device,
		"set": map[string]interface{}{
			"property": property.Name,
			"siid":     property.Method.SIID,
			"piid":     property.Method.PIID,
			"value":    on,
		},
		"result": result,
	}, nil
}

func (s *Service) MijiaSensorGet(ctx context.Context, learnerID uint, did string) (interface{}, error) {
	device, err := s.getLearnerXiaomiDeviceByDID(ctx, learnerID, did)
	if err != nil {
		return nil, err
	}
	spec, _, err := s.GetLearnerMiotSpec(ctx, device.Model)
	if err != nil {
		return nil, err
	}
	properties := pickMiotSensorReadableProperties(spec)
	if len(properties) == 0 {
		return nil, errors.New("no common sensor properties found")
	}
	items := make([]domain.XiaomiBatchPropItem, 0, len(properties))
	for _, property := range properties {
		items = append(items, domain.XiaomiBatchPropItem{
			Did:  did,
			Siid: int64(property.Method.SIID),
			Piid: int64(property.Method.PIID),
		})
	}
	return s.LearnerXiaomiPropGetBatch(ctx, learnerID, items)
}

func (s *Service) MijiaPositionSet(ctx context.Context, learnerID uint, did string, position int) (map[string]interface{}, error) {
	device, err := s.getLearnerXiaomiDeviceByDID(ctx, learnerID, did)
	if err != nil {
		return nil, err
	}
	spec, _, err := s.GetLearnerMiotSpec(ctx, device.Model)
	if err != nil {
		return nil, err
	}
	property, err := pickMiotPositionProperty(spec)
	if err != nil {
		return nil, err
	}
	if err := validateValueAgainstMiotProperty(property, position); err != nil {
		return nil, err
	}
	result, err := s.LearnerXiaomiPropSet(ctx, learnerID, domain.XiaomiPropSetInput{
		Did:   did,
		Siid:  int64(property.Method.SIID),
		Piid:  int64(property.Method.PIID),
		Value: position,
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"success": true,
		"device":  device,
		"set": map[string]interface{}{
			"property": property.Name,
			"siid":     property.Method.SIID,
			"piid":     property.Method.PIID,
			"value":    position,
		},
		"result": result,
	}, nil
}

func (s *Service) MijiaActionCall(
	ctx context.Context,
	learnerID uint,
	did string,
	actionName string,
	in []interface{},
) (interface{}, error) {
	device, err := s.getLearnerXiaomiDeviceByDID(ctx, learnerID, did)
	if err != nil {
		return nil, err
	}
	spec, _, err := s.GetLearnerMiotSpec(ctx, device.Model)
	if err != nil {
		return nil, err
	}
	action, err := pickMiotAction(spec, actionName)
	if err != nil {
		return nil, err
	}
	return s.LearnerXiaomiAction(ctx, learnerID, domain.XiaomiActionInput{
		Did:  did,
		Siid: int64(action.Method.SIID),
		Aiid: int64(action.Method.AIID),
		In:   in,
	})
}

func (s *Service) MijiaHvacSet(
	ctx context.Context,
	learnerID uint,
	did string,
	params map[string]interface{},
) (map[string]interface{}, error) {
	device, err := s.getLearnerXiaomiDeviceByDID(ctx, learnerID, did)
	if err != nil {
		return nil, err
	}
	spec, _, err := s.GetLearnerMiotSpec(ctx, device.Model)
	if err != nil {
		return nil, err
	}
	results := make(map[string]interface{})
	fieldCandidates := map[string][]string{
		"power":       {"on", "power"},
		"mode":        {"mode"},
		"target_temp": {"target_temperature", "target-temperature", "target_temp"},
		"fan_level":   {"fan_level", "fan-level", "fan_speed", "fan-speed"},
		"swing":       {"horizontal_swing", "vertical_swing", "swing"},
	}
	for field, value := range params {
		candidates := fieldCandidates[field]
		if len(candidates) == 0 {
			results[field] = map[string]interface{}{"error": "unsupported field"}
			continue
		}
		property, err := pickMiotPropByCandidates(spec, candidates)
		if err != nil {
			results[field] = map[string]interface{}{"error": err.Error()}
			continue
		}
		if err := validateValueAgainstMiotProperty(property, value); err != nil {
			results[field] = map[string]interface{}{"error": err.Error(), "property": property.Name}
			continue
		}
		_, err = s.LearnerXiaomiPropSet(ctx, learnerID, domain.XiaomiPropSetInput{
			Did:   did,
			Siid:  int64(property.Method.SIID),
			Piid:  int64(property.Method.PIID),
			Value: value,
		})
		if err != nil {
			results[field] = map[string]interface{}{"error": err.Error(), "property": property.Name}
			continue
		}
		results[field] = map[string]interface{}{
			"property": property.Name,
			"siid":     property.Method.SIID,
			"piid":     property.Method.PIID,
			"value":    value,
		}
	}
	return map[string]interface{}{
		"success": true,
		"device":  device,
		"results": results,
	}, nil
}

func (s *Service) getLearnerXiaomiConfigModel(ctx context.Context, learnerID uint) (storage.XiaomiConfig, bool, error) {
	var model storage.XiaomiConfig
	if err := s.db.WithContext(ctx).Where("learner_user_id = ?", learnerID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return storage.XiaomiConfig{}, false, nil
		}
		return storage.XiaomiConfig{}, false, err
	}
	return model, true, nil
}

func (s *Service) newLearnerXiaomiCloudClient(ctx context.Context, learnerID uint) (*http.Client, string, storage.XiaomiConfig, error) {
	model, exists, err := s.getLearnerXiaomiConfigModel(ctx, learnerID)
	if err != nil {
		return nil, "", storage.XiaomiConfig{}, err
	}
	if !exists || strings.TrimSpace(model.Ssecurity) == "" || strings.TrimSpace(model.ServiceToken) == "" {
		return nil, "", storage.XiaomiConfig{}, errors.New("xiaomi credentials are not configured")
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, "", storage.XiaomiConfig{}, err
	}
	apiBaseURL := getXiaomiAPIURL(model.Server)
	parsed, err := url.Parse(apiBaseURL)
	if err != nil {
		return nil, "", storage.XiaomiConfig{}, err
	}
	jar.SetCookies(parsed, []*http.Cookie{{
		Name:     "serviceToken",
		Value:    model.ServiceToken,
		Path:     "/",
		Domain:   parsed.Hostname(),
		HttpOnly: true,
		Secure:   parsed.Scheme == "https",
	}})

	return &http.Client{
		Timeout: 20 * time.Second,
		Jar:     jar,
	}, apiBaseURL, model, nil
}

func (s *Service) extractXiaomiDeviceList(client *http.Client, apiBaseURL string, cfg storage.XiaomiConfig) (string, []domain.XiaomiDevice, error) {
	homes, err := s.getXiaomiHomes(client, apiBaseURL, cfg.Ssecurity)
	if err != nil {
		return "", nil, err
	}
	deviceCountResult, err := s.getXiaomiDeviceCount(client, apiBaseURL, cfg.Ssecurity)
	if err != nil {
		deviceCountResult = nil
	}

	allHomes := append([]map[string]interface{}{}, homes...)
	if deviceCountResult != nil {
		if result, ok := deviceCountResult["result"].(map[string]interface{}); ok {
			if share, ok := result["share"].(map[string]interface{}); ok {
				if families, ok := share["share_family"].([]interface{}); ok {
					for _, item := range families {
						if home, ok := item.(map[string]interface{}); ok {
							allHomes = append(allHomes, home)
						}
					}
				}
			}
		}
	}

	ownerFallback := firstNonEmpty(strings.TrimSpace(cfg.XiaomiUserID), strings.TrimSpace(cfg.Username))
	devicesRaw := make([]map[string]interface{}, 0, 128)
	for _, home := range allHomes {
		homeID := firstNonEmpty(mapString(home, "home_id"), mapString(home, "id"))
		homeOwner := firstNonEmpty(mapString(home, "home_owner"), ownerFallback)
		if homeID == "" || homeOwner == "" {
			continue
		}
		homeIDValue, err := strconv.ParseInt(homeID, 10, 64)
		if err != nil {
			continue
		}
		homeOwnerValue, err := strconv.ParseInt(homeOwner, 10, 64)
		if err != nil {
			continue
		}
		devices, err := s.getXiaomiDevices(client, apiBaseURL, cfg.Ssecurity, homeIDValue, homeOwnerValue)
		if err != nil {
			continue
		}
		for _, device := range devices {
			if _, exists := device["home_name"]; !exists {
				device["home_name"] = firstNonEmpty(mapString(home, "name"), mapString(home, "home_name"))
			}
			if _, exists := device["home_id"]; !exists {
				device["home_id"] = homeID
			}
			devicesRaw = append(devicesRaw, device)
		}
	}

	jsonBytes, err := json.MarshalIndent(devicesRaw, "", "  ")
	if err != nil {
		return "", nil, err
	}
	devices, err := parseStoredXiaomiDevices(string(jsonBytes))
	if err != nil {
		return "", nil, err
	}
	return string(jsonBytes), devices, nil
}

func (s *Service) getXiaomiHomes(client *http.Client, apiBaseURL, ssecurity string) ([]map[string]interface{}, error) {
	result, err := s.executeXiaomiAPICall(client, apiBaseURL+"/v2/homeroom/gethome", ssecurity, map[string]string{
		"data": `{"fg": true, "fetch_share": true, "fetch_share_dev": true, "limit": 300, "app_ver": 7}`,
	})
	if err != nil {
		return nil, err
	}
	items := make([]map[string]interface{}, 0, 16)
	if payload, ok := result["result"].(map[string]interface{}); ok {
		if homes, ok := payload["homelist"].([]interface{}); ok {
			for _, home := range homes {
				if item, ok := home.(map[string]interface{}); ok {
					items = append(items, item)
				}
			}
		}
	}
	return items, nil
}

func (s *Service) getXiaomiDeviceCount(client *http.Client, apiBaseURL, ssecurity string) (map[string]interface{}, error) {
	return s.executeXiaomiAPICall(client, apiBaseURL+"/v2/user/get_device_cnt", ssecurity, map[string]string{
		"data": `{"fetch_own": true, "fetch_share": true}`,
	})
}

func (s *Service) getXiaomiDevices(
	client *http.Client,
	apiBaseURL string,
	ssecurity string,
	homeID int64,
	homeOwner int64,
) ([]map[string]interface{}, error) {
	payload := fmt.Sprintf(`{"home_owner": %d, "home_id": %d, "limit": 200, "get_split_device": true, "support_smart_home": true}`, homeOwner, homeID)
	result, err := s.executeXiaomiAPICall(client, apiBaseURL+"/v2/home/home_device_list", ssecurity, map[string]string{
		"data": payload,
	})
	if err != nil {
		return nil, err
	}
	items := make([]map[string]interface{}, 0, 64)
	if data, ok := result["result"].(map[string]interface{}); ok {
		for _, key := range []string{"device_info", "device_info_new", "share_info", "list"} {
			list, ok := data[key].([]interface{})
			if !ok {
				continue
			}
			for _, device := range list {
				if item, ok := device.(map[string]interface{}); ok {
					items = append(items, item)
				}
			}
		}
	}
	return items, nil
}

func (s *Service) executeXiaomiAPICall(
	client *http.Client,
	apiURL string,
	ssecurity string,
	params map[string]string,
) (map[string]interface{}, error) {
	nonce := xiaomiGenerateNonce(time.Now().UnixMilli())
	signedNonce, err := xiaomiSignedNonce(nonce, ssecurity)
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	orderedParams := make([][2]string, 0, len(keys))
	for _, key := range keys {
		orderedParams = append(orderedParams, [2]string{key, params[key]})
	}

	encryptedParams, err := xiaomiGenerateEncryptedParams(apiURL, http.MethodPost, signedNonce, nonce, ssecurity, orderedParams)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	for key, value := range encryptedParams {
		form.Set(key, value)
	}
	request, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept-Encoding", "identity")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", generateXiaomiUserAgent())
	request.Header.Set("x-xiaomi-protocal-flag-cli", "PROTOCAL-HTTP2")
	request.Header.Set("MIOT-ENCRYPT-ALGORITHM", "ENCRYPT-RC4")

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("xiaomi api returned status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	decryptedBody, err := xiaomiRC4Decrypt(signedNonce, string(body))
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(decryptedBody, &result); err != nil {
		return nil, err
	}
	if codeValue, exists := result["code"]; exists {
		if codeNumber, ok := codeValue.(float64); ok && codeNumber != 0 {
			return nil, fmt.Errorf("xiaomi api error code=%v message=%v", codeNumber, result["message"])
		}
	}
	if _, exists := result["result"]; !exists {
		return nil, errors.New("xiaomi api response missing result field")
	}
	return result, nil
}

func (s *Service) ensureLearnerXiaomiDevices(ctx context.Context, learnerID uint) ([]domain.XiaomiDevice, error) {
	model, exists, err := s.getLearnerXiaomiConfigModel(ctx, learnerID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("xiaomi config is not configured")
	}
	if strings.TrimSpace(model.DeviceList) == "" {
		devices, err := s.RefreshLearnerXiaomiDevices(ctx, learnerID)
		if err != nil {
			return nil, err
		}
		return devices, nil
	}
	return parseStoredXiaomiDevices(model.DeviceList)
}

func (s *Service) getLearnerXiaomiDeviceByDID(ctx context.Context, learnerID uint, did string) (domain.XiaomiDevice, error) {
	devices, err := s.ensureLearnerXiaomiDevices(ctx, learnerID)
	if err != nil {
		return domain.XiaomiDevice{}, err
	}
	for _, device := range devices {
		if strings.TrimSpace(device.Did) == strings.TrimSpace(did) {
			return device, nil
		}
	}
	return domain.XiaomiDevice{}, errors.New("xiaomi device not found")
}

func toLearnerXiaomiConfig(model storage.XiaomiConfig) domain.XiaomiConfig {
	return domain.XiaomiConfig{
		ID:             model.ID,
		LearnerUserID:  model.LearnerUserID,
		Username:       model.Username,
		XiaomiUserID:   model.XiaomiUserID,
		Server:         normalizeXiaomiServer(model.Server),
		IsActive:       model.IsActive,
		HasCredentials: strings.TrimSpace(model.Ssecurity) != "" && strings.TrimSpace(model.ServiceToken) != "",
		DeviceCount:    countStoredXiaomiDevices(model.DeviceList),
		LastSyncAt:     model.LastSyncAt,
		CreatedAt:      model.CreatedAt,
		UpdatedAt:      model.UpdatedAt,
	}
}

func toLearnerXiaomiAccountSnapshot(model storage.XiaomiConfig) domain.XiaomiAccountSnapshot {
	return domain.XiaomiAccountSnapshot{
		Username:       model.Username,
		XiaomiUserID:   model.XiaomiUserID,
		Server:         normalizeXiaomiServer(model.Server),
		IsActive:       model.IsActive,
		HasCredentials: strings.TrimSpace(model.Ssecurity) != "" && strings.TrimSpace(model.ServiceToken) != "",
		DeviceCount:    countStoredXiaomiDevices(model.DeviceList),
		LastSyncAt:     model.LastSyncAt,
	}
}

func normalizeXiaomiServer(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "", "cn":
		return "cn"
	case "de", "i2", "ru", "sg", "us", "tw":
		return value
	default:
		return "cn"
	}
}

func getXiaomiAPIURL(server string) string {
	server = normalizeXiaomiServer(server)
	if server == "cn" {
		return "https://api.io.mi.com/app"
	}
	return fmt.Sprintf("https://%s.api.io.mi.com/app", server)
}

func parseStoredXiaomiDevices(raw string) ([]domain.XiaomiDevice, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []domain.XiaomiDevice{}, nil
	}
	var items []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, err
	}

	devices := make([]domain.XiaomiDevice, 0, len(items))
	for _, item := range items {
		devices = append(devices, domain.XiaomiDevice{
			Did:      mapString(item, "did"),
			Name:     firstNonEmpty(mapString(item, "name"), mapString(item, "device_name")),
			Model:    mapString(item, "model"),
			Token:    mapString(item, "token"),
			LocalIP:  firstNonEmpty(mapString(item, "localip"), mapString(item, "local_ip")),
			SpecType: firstNonEmpty(mapString(item, "spec_type"), mapString(item, "specType")),
			HomeID:   firstNonEmpty(mapString(item, "home_id"), mapString(item, "homeId")),
			HomeName: firstNonEmpty(mapString(item, "home_name"), mapString(item, "homeName")),
			RoomID:   firstNonEmpty(mapString(item, "room_id"), mapString(item, "roomId")),
			RoomName: firstNonEmpty(mapString(item, "room_name"), mapString(item, "roomName")),
			IsOnline: mapBool(item, "isOnline") || mapBool(item, "is_online"),
			IsShared: mapBool(item, "is_shared") || mapBool(item, "share"),
			Raw:      item,
		})
	}
	return devices, nil
}

func countStoredXiaomiDevices(raw string) int {
	devices, err := parseStoredXiaomiDevices(raw)
	if err != nil {
		return 0
	}
	return len(devices)
}

func mapString(values map[string]interface{}, key string) string {
	value, exists := values[key]
	if !exists || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		if math.Mod(typed, 1) == 0 {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case uint:
		return strconv.FormatUint(uint64(typed), 10)
	case uint64:
		return strconv.FormatUint(typed, 10)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func mapBool(values map[string]interface{}, key string) bool {
	value, exists := values[key]
	if !exists || value == nil {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(typed, "true") || typed == "1"
	case float64:
		return typed != 0
	case int:
		return typed != 0
	default:
		return false
	}
}

func xiaomiRC4Encrypt(password, payload string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(password)
	if err != nil {
		return "", err
	}
	cipher, err := rc4.NewCipher(key)
	if err != nil {
		return "", err
	}
	dummy := make([]byte, 1024)
	cipher.XORKeyStream(dummy, dummy)
	plaintext := []byte(payload)
	ciphertext := make([]byte, len(plaintext))
	cipher.XORKeyStream(ciphertext, plaintext)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func xiaomiRC4Decrypt(password, payload string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(password)
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, err
	}
	cipher, err := rc4.NewCipher(key)
	if err != nil {
		return nil, err
	}
	dummy := make([]byte, 1024)
	cipher.XORKeyStream(dummy, dummy)
	plaintext := make([]byte, len(ciphertext))
	cipher.XORKeyStream(plaintext, ciphertext)
	return plaintext, nil
}

func xiaomiSignedNonce(nonce, ssecurity string) (string, error) {
	nonceBytes, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return "", err
	}
	ssecurityBytes, err := base64.StdEncoding.DecodeString(ssecurity)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(append(ssecurityBytes, nonceBytes...))
	return base64.StdEncoding.EncodeToString(hash[:]), nil
}

func xiaomiGenerateNonce(millis int64) string {
	randomBytes := make([]byte, 8)
	_, _ = rand.Read(randomBytes)
	minutes := millis / 60000
	minutesBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(minutesBytes, uint32(minutes))
	return base64.StdEncoding.EncodeToString(append(randomBytes, minutesBytes...))
}

func xiaomiGenerateEncryptedSignature(uri, method, signedNonce string, paramsOrdered [][2]string) string {
	urlPath := uri
	if parsed, err := url.Parse(uri); err == nil {
		urlPath = parsed.Path
	}
	urlPath = strings.ReplaceAll(urlPath, "/app/", "/")

	parts := []string{strings.ToUpper(method), urlPath}
	for _, param := range paramsOrdered {
		parts = append(parts, fmt.Sprintf("%s=%s", param[0], param[1]))
	}
	parts = append(parts, signedNonce)

	hash := sha1.Sum([]byte(strings.Join(parts, "&")))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func xiaomiGenerateEncryptedParams(uri, method, signedNonce, nonce, ssecurity string, paramsOrdered [][2]string) (map[string]string, error) {
	rc4Hash := xiaomiGenerateEncryptedSignature(uri, method, signedNonce, paramsOrdered)
	paramsToEncrypt := append(paramsOrdered, [2]string{"rc4_hash__", rc4Hash})
	encryptedBody := make(map[string]string, len(paramsToEncrypt)+3)
	encryptedSignatureParams := make([][2]string, 0, len(paramsToEncrypt))
	for _, param := range paramsToEncrypt {
		encryptedValue, err := xiaomiRC4Encrypt(signedNonce, param[1])
		if err != nil {
			return nil, err
		}
		encryptedBody[param[0]] = encryptedValue
		encryptedSignatureParams = append(encryptedSignatureParams, [2]string{param[0], encryptedValue})
	}
	encryptedBody["signature"] = xiaomiGenerateEncryptedSignature(uri, method, signedNonce, encryptedSignatureParams)
	encryptedBody["ssecurity"] = ssecurity
	encryptedBody["_nonce"] = nonce
	return encryptedBody, nil
}

func generateXiaomiUserAgent() string {
	const charset = "abcdefghijklmnopqrstuvwxyz"
	var randomBuilder strings.Builder
	for index := 0; index < 18; index++ {
		number, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		randomBuilder.WriteByte(charset[number.Int64()])
	}
	agentID := make([]byte, 13)
	for index := range agentID {
		number, _ := rand.Int(rand.Reader, big.NewInt(5))
		agentID[index] = byte(65 + number.Int64())
	}
	return fmt.Sprintf("%s-%s APP/com.xiaomi.mihome APPV/10.5.201", randomBuilder.String(), string(agentID))
}

func httpGetXiaomiMiotSpecPage(model string) (string, string, error) {
	client := &http.Client{Timeout: 12 * time.Second}
	request, err := http.NewRequest(http.MethodGet, miotSpecBaseURL+strings.TrimSpace(model), nil)
	if err != nil {
		return "", "", err
	}
	request.Header.Set("User-Agent", "brights-mijia/1.0")
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	response, err := client.Do(request)
	if err != nil {
		return "", "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("miot spec page returned status %d", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", "", err
	}
	return string(body), response.Header.Get("ETag"), nil
}

func extractMiotSpecDataPageJSON(html string) (string, error) {
	match := miotSpecDataPagePattern.FindStringSubmatch(html)
	if len(match) < 2 {
		return "", errors.New("failed to extract miot data-page")
	}
	content := strings.ReplaceAll(match[1], "&quot;", "\"")
	return content, nil
}

func parseMiotSpecFromDataPage(model, dataPageJSON string) (domain.MiotSpecParsed, error) {
	var root map[string]interface{}
	if err := json.Unmarshal([]byte(dataPageJSON), &root); err != nil {
		return domain.MiotSpecParsed{}, err
	}
	props, _ := root["props"].(map[string]interface{})
	if props == nil {
		return domain.MiotSpecParsed{}, errors.New("miot spec props missing")
	}

	result := domain.MiotSpecParsed{Model: model, Name: model}
	if product, ok := props["product"].(map[string]interface{}); ok {
		if name := strings.TrimSpace(fmt.Sprintf("%v", product["name"])); name != "" && name != "<nil>" {
			result.Name = name
		}
		if value := strings.TrimSpace(fmt.Sprintf("%v", product["model"])); value != "" && value != "<nil>" {
			result.Model = value
		}
	}

	specNode, _ := props["spec"].(map[string]interface{})
	services, _ := specNode["services"].(map[string]interface{})
	if services == nil {
		return domain.MiotSpecParsed{}, errors.New("miot spec services missing")
	}

	propertyNames := map[string]bool{}
	actionNames := map[string]bool{}
	for siidString, serviceNode := range services {
		siid, _ := strconv.Atoi(siidString)
		serviceMap, _ := serviceNode.(map[string]interface{})
		serviceName := strings.TrimSpace(fmt.Sprintf("%v", serviceMap["name"]))

		if properties, ok := serviceMap["properties"].(map[string]interface{}); ok {
			for piidString, propertyNode := range properties {
				piid, _ := strconv.Atoi(piidString)
				propertyMap, _ := propertyNode.(map[string]interface{})
				name := strings.TrimSpace(fmt.Sprintf("%v", propertyMap["name"]))
				if name == "" {
					continue
				}
				format := strings.TrimSpace(fmt.Sprintf("%v", propertyMap["format"]))
				description := strings.TrimSpace(fmt.Sprintf("%v / %v", propertyMap["description"], propertyMap["desc_zh_cn"]))
				propertyName := name
				if propertyNames[propertyName] {
					propertyName = firstNonEmpty(serviceName, "service") + "-" + propertyName
				}
				propertyNames[propertyName] = true

				accessList, _ := propertyMap["access"].([]interface{})
				rw := ""
				for _, access := range accessList {
					switch strings.TrimSpace(fmt.Sprintf("%v", access)) {
					case "read":
						rw += "r"
					case "write":
						rw += "w"
					}
				}

				property := domain.MiotSpecProperty{
					Name:        propertyName,
					Description: description,
					Type:        normalizeMiotPropertyType(format),
					RW:          rw,
					Unit:        propertyMap["unit"],
					Method: domain.MiotSpecPropertyMethod{
						SIID: siid,
						PIID: piid,
					},
				}
				if rangeItems, ok := propertyMap["value-range"].([]interface{}); ok && len(rangeItems) > 0 {
					property.Range = rangeItems
				}
				if valueList, ok := propertyMap["value-list"].([]interface{}); ok && len(valueList) > 0 {
					values := make([]map[string]interface{}, 0, len(valueList))
					for _, item := range valueList {
						if valueMap, ok := item.(map[string]interface{}); ok {
							values = append(values, valueMap)
						}
					}
					property.ValueList = values
				}
				result.Properties = append(result.Properties, property)
			}
		}

		if actions, ok := serviceMap["actions"].(map[string]interface{}); ok {
			for aiidString, actionNode := range actions {
				aiid, _ := strconv.Atoi(aiidString)
				actionMap, _ := actionNode.(map[string]interface{})
				name := strings.TrimSpace(fmt.Sprintf("%v", actionMap["name"]))
				if name == "" {
					continue
				}
				actionName := name
				if actionNames[actionName] {
					actionName = firstNonEmpty(serviceName, "service") + "-" + actionName
				}
				actionNames[actionName] = true
				result.Actions = append(result.Actions, domain.MiotSpecAction{
					Name:        actionName,
					Description: strings.TrimSpace(fmt.Sprintf("%v / %v", actionMap["description"], actionMap["desc_zh_cn"])),
					Method: domain.MiotSpecActionMethod{
						SIID: siid,
						AIID: aiid,
					},
				})
			}
		}
	}

	sort.Slice(result.Properties, func(i, j int) bool { return result.Properties[i].Name < result.Properties[j].Name })
	sort.Slice(result.Actions, func(i, j int) bool { return result.Actions[i].Name < result.Actions[j].Name })
	return result, nil
}

func normalizeMiotPropertyType(format string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	switch {
	case strings.HasPrefix(format, "bool"):
		return "bool"
	case strings.HasPrefix(format, "int"):
		return "int"
	case strings.HasPrefix(format, "uint"):
		return "uint"
	case strings.HasPrefix(format, "float"), strings.HasPrefix(format, "double"):
		return "float"
	default:
		return "string"
	}
}

func summarizeMiotProperty(property domain.MiotSpecProperty) map[string]interface{} {
	summary := map[string]interface{}{
		"name": property.Name,
		"desc": property.Description,
		"type": property.Type,
		"rw":   property.RW,
		"siid": property.Method.SIID,
		"piid": property.Method.PIID,
	}
	if property.Unit != nil {
		summary["unit"] = property.Unit
	}
	if len(property.Range) > 0 {
		summary["range"] = property.Range
	}
	if len(property.ValueList) > 0 {
		summary["enum"] = property.ValueList
	}
	return summary
}

func pickMiotPropByCandidates(spec domain.MiotSpecParsed, candidates []string) (domain.MiotSpecProperty, error) {
	normalizedCandidates := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(candidate, "-", "_")))
		if candidate != "" {
			normalizedCandidates = append(normalizedCandidates, candidate)
		}
	}
	for _, candidate := range normalizedCandidates {
		for _, property := range spec.Properties {
			name := strings.ToLower(strings.ReplaceAll(property.Name, "-", "_"))
			if name == candidate {
				return property, nil
			}
		}
	}
	for _, candidate := range normalizedCandidates {
		for _, property := range spec.Properties {
			name := strings.ToLower(strings.ReplaceAll(property.Name, "-", "_"))
			if strings.Contains(name, candidate) || strings.Contains(strings.ToLower(property.Description), candidate) {
				return property, nil
			}
		}
	}
	return domain.MiotSpecProperty{}, errors.New("miot property not found")
}

func pickMiotAction(spec domain.MiotSpecParsed, actionName string) (domain.MiotSpecAction, error) {
	needle := strings.ToLower(strings.TrimSpace(actionName))
	for _, action := range spec.Actions {
		if strings.ToLower(action.Name) == needle {
			return action, nil
		}
	}
	for _, action := range spec.Actions {
		if strings.Contains(strings.ToLower(action.Name), needle) {
			return action, nil
		}
	}
	return domain.MiotSpecAction{}, errors.New("miot action not found")
}

func pickMiotSwitchProperty(spec domain.MiotSpecParsed) (domain.MiotSpecProperty, error) {
	property, err := pickMiotPropByCandidates(spec, []string{"on", "power"})
	if err == nil && strings.Contains(property.RW, "w") {
		return property, nil
	}
	for _, item := range spec.Properties {
		if item.Type == "bool" && strings.Contains(item.RW, "w") {
			return item, nil
		}
	}
	return domain.MiotSpecProperty{}, errors.New("writable switch property not found")
}

func pickMiotSensorReadableProperties(spec domain.MiotSpecParsed) []domain.MiotSpecProperty {
	keys := []string{"temperature", "relative_humidity", "humidity", "battery_level", "battery"}
	items := make([]domain.MiotSpecProperty, 0, len(keys))
	seen := map[string]bool{}
	for _, key := range keys {
		property, err := pickMiotPropByCandidates(spec, []string{key})
		if err != nil || !strings.Contains(property.RW, "r") || seen[property.Name] {
			continue
		}
		seen[property.Name] = true
		items = append(items, property)
	}
	return items
}

func pickMiotPositionProperty(spec domain.MiotSpecParsed) (domain.MiotSpecProperty, error) {
	property, err := pickMiotPropByCandidates(spec, []string{"target_position", "target-position", "position"})
	if err == nil && strings.Contains(property.RW, "w") {
		return property, nil
	}
	for _, item := range spec.Properties {
		if strings.Contains(strings.ToLower(item.Name), "position") && strings.Contains(item.RW, "w") {
			return item, nil
		}
	}
	return domain.MiotSpecProperty{}, errors.New("writable position property not found")
}

func validateValueAgainstMiotProperty(property domain.MiotSpecProperty, value interface{}) error {
	if len(property.Range) >= 2 {
		number, ok := toFloat64Value(value)
		if !ok {
			return errors.New("value must be numeric")
		}
		minimum, _ := toFloat64Value(property.Range[0])
		maximum, _ := toFloat64Value(property.Range[1])
		if number < minimum || number > maximum {
			return fmt.Errorf("value must be between %v and %v", minimum, maximum)
		}
	}
	if len(property.ValueList) > 0 {
		text := fmt.Sprintf("%v", value)
		for _, item := range property.ValueList {
			if fmt.Sprintf("%v", item["value"]) == text {
				return nil
			}
		}
		return errors.New("value is not in the allowed enum list")
	}
	return nil
}

func toFloat64Value(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case json.Number:
		number, err := typed.Float64()
		return number, err == nil
	case string:
		number, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return number, err == nil
	default:
		return 0, false
	}
}

