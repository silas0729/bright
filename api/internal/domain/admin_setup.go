package domain

type AdminSetupStatus struct {
	Initialized bool  `json:"initialized"`
	AdminCount  int64 `json:"admin_count"`
}

type InitializeSuperAdminInput struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}
