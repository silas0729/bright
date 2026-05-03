package domain

import "time"

type XiaomiConfig struct {
	ID              uint      `json:"id"`
	LearnerUserID   uint      `json:"learner_user_id"`
	Username        string    `json:"username"`
	XiaomiUserID    string    `json:"xiaomi_user_id"`
	Server          string    `json:"server"`
	IsActive        bool      `json:"is_active"`
	HasCredentials  bool      `json:"has_credentials"`
	DeviceCount     int       `json:"device_count"`
	LastSyncAt      *time.Time `json:"last_sync_at,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type SaveXiaomiConfigInput struct {
	Username     string `json:"username"`
	XiaomiUserID string `json:"xiaomi_user_id"`
	Server       string `json:"server"`
	Ssecurity    string `json:"ssecurity"`
	ServiceToken string `json:"service_token"`
	IsActive     bool   `json:"is_active"`
}

type XiaomiAccountSnapshot struct {
	Username       string     `json:"username"`
	XiaomiUserID   string     `json:"xiaomi_user_id"`
	Server         string     `json:"server"`
	IsActive       bool       `json:"is_active"`
	HasCredentials bool       `json:"has_credentials"`
	DeviceCount    int        `json:"device_count"`
	LastSyncAt     *time.Time `json:"last_sync_at,omitempty"`
}

type XiaomiDevice struct {
	Did       string `json:"did"`
	Name      string `json:"name"`
	Model     string `json:"model"`
	Token     string `json:"token,omitempty"`
	LocalIP   string `json:"localip,omitempty"`
	SpecType  string `json:"spec_type,omitempty"`
	HomeID    string `json:"home_id,omitempty"`
	HomeName  string `json:"home_name,omitempty"`
	RoomID    string `json:"room_id,omitempty"`
	RoomName  string `json:"room_name,omitempty"`
	IsOnline  bool   `json:"is_online"`
	IsShared  bool   `json:"is_shared,omitempty"`
	Raw       map[string]interface{} `json:"raw,omitempty"`
}

type XiaomiHome struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	OwnerID  string                 `json:"owner_id,omitempty"`
	Raw      map[string]interface{} `json:"raw,omitempty"`
}

type XiaomiDeviceListResult struct {
	Account   XiaomiAccountSnapshot `json:"account"`
	Devices   []XiaomiDevice        `json:"devices"`
	Total     int                   `json:"total"`
	Refreshed bool                  `json:"refreshed"`
}

type XiaomiPropGetInput struct {
	Did  string `json:"did"`
	Siid int64  `json:"siid"`
	Piid int64  `json:"piid"`
}

type XiaomiPropSetInput struct {
	Did   string      `json:"did"`
	Siid  int64       `json:"siid"`
	Piid  int64       `json:"piid"`
	Value interface{} `json:"value"`
}

type XiaomiActionInput struct {
	Did  string        `json:"did"`
	Siid int64         `json:"siid"`
	Aiid int64         `json:"aiid"`
	In   []interface{} `json:"in"`
}

type XiaomiBatchPropItem struct {
	Did  string `json:"did"`
	Siid int64  `json:"siid"`
	Piid int64  `json:"piid"`
}

type XiaomiDeviceMatch struct {
	Did      string `json:"did"`
	Name     string `json:"name"`
	Model    string `json:"model"`
	SpecType string `json:"spec_type,omitempty"`
}

type XiaomiControlDeviceInput struct {
	Query       string      `json:"query"`
	Operation   string      `json:"operation"`
	PropName    string      `json:"prop_name"`
	Value       interface{} `json:"value"`
	ActionName  string      `json:"action_name"`
	ActionValue interface{} `json:"action_value"`
}

type MiotSpecParsed struct {
	Name       string             `json:"name"`
	Model      string             `json:"model"`
	Properties []MiotSpecProperty `json:"properties"`
	Actions    []MiotSpecAction   `json:"actions"`
}

type MiotSpecProperty struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Type        string                   `json:"type"`
	RW          string                   `json:"rw"`
	Unit        interface{}              `json:"unit,omitempty"`
	Range       []interface{}            `json:"range,omitempty"`
	ValueList   []map[string]interface{} `json:"value_list,omitempty"`
	Method      MiotSpecPropertyMethod   `json:"method"`
}

type MiotSpecPropertyMethod struct {
	SIID int `json:"siid"`
	PIID int `json:"piid"`
}

type MiotSpecAction struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Method      MiotSpecActionMethod `json:"method"`
}

type MiotSpecActionMethod struct {
	SIID int `json:"siid"`
	AIID int `json:"aiid"`
}

