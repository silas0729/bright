package domain

type SiteSetting struct {
	SiteName        string `json:"site_name"`
	SiteIcon        string `json:"site_icon"`
	SiteTagline     string `json:"site_tagline"`
	HeroTitle       string `json:"hero_title"`
	HeroDescription string `json:"hero_description"`
	SEOHeadline     string `json:"seo_headline"`
	SEOTitle        string `json:"seo_title"`
	SEODescription  string `json:"seo_description"`
	SEOKeywords     string `json:"seo_keywords"`
	FooterText      string `json:"footer_text"`
	ContactEmail    string `json:"contact_email"`
}

type SaveSiteSettingInput struct {
	SiteName        string `json:"site_name"`
	SiteIcon        string `json:"site_icon"`
	SiteTagline     string `json:"site_tagline"`
	HeroTitle       string `json:"hero_title"`
	HeroDescription string `json:"hero_description"`
	SEOHeadline     string `json:"seo_headline"`
	SEOTitle        string `json:"seo_title"`
	SEODescription  string `json:"seo_description"`
	SEOKeywords     string `json:"seo_keywords"`
	FooterText      string `json:"footer_text"`
	ContactEmail    string `json:"contact_email"`
}
