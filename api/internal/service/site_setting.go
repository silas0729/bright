package service

import (
	"context"
	"strings"

	"brights/api/internal/domain"
	"brights/api/internal/storage"
)

func (s *Service) GetSiteSetting(ctx context.Context) (domain.SiteSetting, error) {
	model, err := s.ensureSiteSetting(ctx)
	if err != nil {
		return domain.SiteSetting{}, err
	}
	return toSiteSetting(model), nil
}

func (s *Service) SaveSiteSetting(ctx context.Context, input domain.SaveSiteSettingInput) (domain.SiteSetting, error) {
	model, err := s.ensureSiteSetting(ctx)
	if err != nil {
		return domain.SiteSetting{}, err
	}

	model.SiteName = defaultIfTrimmedBlank(input.SiteName, defaultSiteSetting().SiteName)
	model.SiteIcon = strings.TrimSpace(input.SiteIcon)
	model.SiteTagline = strings.TrimSpace(input.SiteTagline)
	model.HeroTitle = defaultIfTrimmedBlank(input.HeroTitle, defaultSiteSetting().HeroTitle)
	model.HeroDescription = defaultIfTrimmedBlank(input.HeroDescription, defaultSiteSetting().HeroDescription)
	model.SEOHeadline = strings.TrimSpace(input.SEOHeadline)
	model.SEOTitle = defaultIfTrimmedBlank(input.SEOTitle, defaultSiteSetting().SEOTitle)
	model.SEODescription = defaultIfTrimmedBlank(input.SEODescription, defaultSiteSetting().SEODescription)
	model.SEOKeywords = strings.TrimSpace(input.SEOKeywords)
	model.FooterText = strings.TrimSpace(input.FooterText)
	model.ContactEmail = strings.TrimSpace(input.ContactEmail)
	model.InviteCommissionRate = input.InviteCommissionRate
	if model.InviteCommissionRate < 0 {
		model.InviteCommissionRate = 0
	}
	if model.InviteCommissionRate > 100 {
		model.InviteCommissionRate = 100
	}

	if err := s.db.WithContext(ctx).Save(&model).Error; err != nil {
		return domain.SiteSetting{}, err
	}
	return toSiteSetting(model), nil
}

func (s *Service) ensureSiteSetting(ctx context.Context) (storage.SiteSetting, error) {
	var model storage.SiteSetting
	err := s.db.WithContext(ctx).Order("id asc").First(&model).Error
	if err == nil {
		return model, nil
	}
	if err != nil && !isRecordNotFound(err) {
		return storage.SiteSetting{}, err
	}

	model = defaultSiteSetting()
	if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
		return storage.SiteSetting{}, err
	}
	return model, nil
}

func defaultSiteSetting() storage.SiteSetting {
	return storage.SiteSetting{
		SiteName:        "Brights 英语单词学习站",
		SiteIcon:        "",
		SiteTagline:     "先学真正会用到的词，再把词汇量慢慢做厚。",
		HeroTitle:       "高频英语单词，从真实场景开始学",
		HeroDescription: "围绕校园、日常、旅行、职场等高频场景整理常用英语单词，先学真正会遇到、会使用、会反复出现的词，再逐步扩展到更多学科和更系统的学习内容。",
		SEOHeadline:     "高频英语单词｜场景词汇｜会员制学习",
		SEOTitle:        "Brights 英语单词学习站 | 高频英语单词、场景词汇、会员制学习平台",
		SEODescription:  "Brights 专注高频英语单词学习，围绕校园、日常、旅行、职场等真实场景整理常用英语词汇，提供中文释义、分类学习、会员内容与多学科扩展能力，适合学生、成人和长期自学者系统积累词汇量。",
		SEOKeywords:     "英语单词学习,高频英语单词,场景英语词汇,英语词汇记忆,英语学习网站,英语会员学习,初中英语单词,高中英语单词,成人英语学习",
		FooterText:      "Brights 适合以英语高频词汇为主线持续学习，也支持后续扩展更多学科内容。",
		ContactEmail:    "support@brights.local",
		InviteCommissionRate: 10,
	}
}

func toSiteSetting(model storage.SiteSetting) domain.SiteSetting {
	return domain.SiteSetting{
		SiteName:        model.SiteName,
		SiteIcon:        model.SiteIcon,
		SiteTagline:     model.SiteTagline,
		HeroTitle:       model.HeroTitle,
		HeroDescription: model.HeroDescription,
		SEOHeadline:     model.SEOHeadline,
		SEOTitle:        model.SEOTitle,
		SEODescription:  model.SEODescription,
		SEOKeywords:     model.SEOKeywords,
		FooterText:      model.FooterText,
		ContactEmail:    model.ContactEmail,
		InviteCommissionRate: model.InviteCommissionRate,
	}
}

func defaultIfTrimmedBlank(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
