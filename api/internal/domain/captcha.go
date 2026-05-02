package domain

type CaptchaChallenge struct {
	Scene     string `json:"scene"`
	CaptchaID string `json:"captcha_id"`
	ImageData string `json:"image_data"`
	ExpiresIn int64  `json:"expires_in"`
}
