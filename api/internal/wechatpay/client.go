package wechatpay

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

const (
	AuthModePublicKey        = "public_key"
	AuthModeAutoCertificate  = "auto_certificate"
	DefaultDescriptionPrefix = "Brights 学习会员"
	DefaultTimeExpireMinutes = 30
)

type Config struct {
	AuthMode               string
	MchID                  string
	AppID                  string
	MerchantSerialNo       string
	APIv3KeyEnc            string
	PlatformCertSerialNo   string
	NotifyURL              string
	DescriptionPrefix      string
	TimeExpireMinutes      int
	WechatPayPublicKeyID   string
	WechatPayPublicKey     string
	WechatPayPublicKeyPath string
	P12Path                string
	CertPem                string
	CertPemPath            string
	KeyPem                 string
	KeyPemPath             string
	PlatformCert           string
	PlatformCertPath       string
}

func NormalizeAuthMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", AuthModePublicKey:
		return AuthModePublicKey
	case "auto", AuthModeAutoCertificate, "certificate", "cert":
		return AuthModeAutoCertificate
	default:
		return ""
	}
}

func UsesPublicKeyMode(cfg Config) bool {
	return NormalizeAuthMode(cfg.AuthMode) != AuthModeAutoCertificate
}

func NormalizeDescriptionPrefix(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return DefaultDescriptionPrefix
	}
	return value
}

func NormalizeTimeExpireMinutes(value int) int {
	if value <= 0 {
		return DefaultTimeExpireMinutes
	}
	return value
}

func ValidateNotifyURL(raw string) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("支付结果回调地址必须是有效的 http 或 https 地址")
	}

	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return errors.New("支付结果回调地址必须是有效的 http 或 https 地址")
	}
	if strings.TrimSpace(parsed.RawQuery) != "" {
		return errors.New("支付结果回调地址不能带查询参数")
	}
	return nil
}

func ValidateConfig(cfg Config, apiV3Key string, strict bool) error {
	authMode := NormalizeAuthMode(cfg.AuthMode)
	if authMode == "" {
		return errors.New("验签方式仅支持“微信支付公钥模式”或“平台证书自动下载模式”")
	}

	cfg.DescriptionPrefix = NormalizeDescriptionPrefix(cfg.DescriptionPrefix)
	cfg.TimeExpireMinutes = NormalizeTimeExpireMinutes(cfg.TimeExpireMinutes)

	if err := ValidateNotifyURL(cfg.NotifyURL); err != nil {
		return err
	}

	if strings.TrimSpace(apiV3Key) != "" && len(strings.TrimSpace(apiV3Key)) != 32 {
		return errors.New("APIv3 密钥长度必须为 32 位")
	}
	if strict && strings.TrimSpace(apiV3Key) == "" {
		return errors.New("请先填写 APIv3 密钥")
	}
	if strict && strings.TrimSpace(cfg.MchID) == "" {
		return errors.New("请先填写商户号")
	}
	if strict && strings.TrimSpace(cfg.AppID) == "" {
		return errors.New("请先填写收款应用 AppID")
	}
	if strict && strings.TrimSpace(cfg.MerchantSerialNo) == "" {
		return errors.New("请先填写商户证书序列号")
	}
	if strict && strings.TrimSpace(cfg.NotifyURL) == "" {
		return errors.New("请先填写支付结果回调地址")
	}

	if strings.TrimSpace(cfg.WechatPayPublicKeyID) != "" && strings.Contains(cfg.WechatPayPublicKeyID, " ") {
		return errors.New("微信支付公钥编号不能包含空格")
	}

	if strings.TrimSpace(cfg.KeyPem) != "" || strings.TrimSpace(cfg.KeyPemPath) != "" {
		if _, err := loadPrivateKey(cfg); err != nil {
			return fmt.Errorf("商户私钥内容格式不正确：%w", err)
		}
	} else if strict {
		return errors.New("请先填写商户私钥内容")
	}

	if strings.TrimSpace(cfg.WechatPayPublicKey) != "" || strings.TrimSpace(cfg.WechatPayPublicKeyPath) != "" {
		if _, err := loadRSAPublicKey(cfg); err != nil {
			return fmt.Errorf("微信支付公钥内容格式不正确：%w", err)
		}
	}

	if strings.TrimSpace(cfg.PlatformCert) != "" || strings.TrimSpace(cfg.PlatformCertPath) != "" {
		if _, err := loadCertificate(cfg); err != nil {
			return fmt.Errorf("平台证书内容格式不正确：%w", err)
		}
	}

	if UsesPublicKeyMode(cfg) {
		if strict && strings.TrimSpace(cfg.WechatPayPublicKeyID) == "" {
			return errors.New("请先填写微信支付公钥编号")
		}
		if strict && strings.TrimSpace(cfg.WechatPayPublicKey) == "" && strings.TrimSpace(cfg.WechatPayPublicKeyPath) == "" {
			return errors.New("请先填写微信支付公钥内容")
		}
	}

	return nil
}

func CheckoutState(cfg Config, apiV3Key string) (bool, string) {
	if err := ValidateConfig(cfg, apiV3Key, true); err != nil {
		return false, err.Error()
	}
	return true, ""
}

func NormalizeAPIError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr *core.APIError
	if !errors.As(err, &apiErr) {
		return err
	}

	code := strings.ToUpper(strings.TrimSpace(apiErr.Code))
	message := strings.TrimSpace(apiErr.Message)
	switch code {
	case "SIGN_ERROR":
		return errors.New("微信支付签名校验失败，请检查商户号、商户证书序列号和商户私钥是否与微信支付后台一致")
	case "PARAM_ERROR", "INVALID_REQUEST":
		if message != "" {
			return fmt.Errorf("微信支付参数有误：%s", message)
		}
		return errors.New("微信支付参数有误")
	case "APPID_MCHID_NOT_MATCH":
		return errors.New("收款应用 AppID 与商户号不匹配")
	case "ORDERNOTEXIST":
		return errors.New("微信支付订单不存在")
	case "NOAUTH":
		return errors.New("当前商户号没有对应接口权限，请检查微信支付后台开通状态")
	}

	if code != "" && message != "" {
		return fmt.Errorf("微信支付返回 %s：%s", strings.ToLower(code), message)
	}
	if message != "" {
		return fmt.Errorf("微信支付返回错误：%s", message)
	}
	return err
}

func ConfigEncKey() []byte {
	candidates := []string{
		strings.TrimSpace(os.Getenv("BRIGHTS_CONFIG_ENC_KEY")),
		strings.TrimSpace(os.Getenv("CONFIG_ENC_KEY")),
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(candidate)
		if err == nil && len(decoded) > 0 {
			return decoded
		}
		return []byte(candidate)
	}

	return nil
}

func EncryptConfigValue(plain string, key []byte) string {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return ""
	}
	if len(key) == 0 {
		return "b64:" + base64.StdEncoding.EncodeToString([]byte(plain))
	}

	input := []byte(plain)
	output := make([]byte, len(input))
	for i := range input {
		output[i] = input[i] ^ key[i%len(key)]
	}
	return "xorb64:" + base64.StdEncoding.EncodeToString(output)
}

func DecryptConfigValue(encrypted string, key []byte) string {
	encrypted = strings.TrimSpace(encrypted)
	if encrypted == "" {
		return ""
	}

	switch {
	case strings.HasPrefix(encrypted, "b64:"):
		raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encrypted, "b64:"))
		if err != nil {
			return ""
		}
		return string(raw)
	case strings.HasPrefix(encrypted, "xorb64:"):
		raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encrypted, "xorb64:"))
		if err != nil || len(key) == 0 {
			return ""
		}
		output := make([]byte, len(raw))
		for i := range raw {
			output[i] = raw[i] ^ key[i%len(key)]
		}
		return string(output)
	default:
		return ""
	}
}

func IsEncryptedConfigValue(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "b64:") || strings.HasPrefix(value, "xorb64:")
}

func BuildNativeService(ctx context.Context, cfg Config, apiV3Key string) (*native.NativeApiService, error) {
	if strings.TrimSpace(cfg.MchID) == "" {
		return nil, errors.New("请先填写商户号")
	}
	if strings.TrimSpace(cfg.MerchantSerialNo) == "" {
		return nil, errors.New("请先填写商户证书序列号")
	}
	if strings.TrimSpace(cfg.KeyPem) == "" && strings.TrimSpace(cfg.KeyPemPath) == "" {
		return nil, errors.New("请先填写商户私钥内容")
	}

	privateKey, err := loadPrivateKey(cfg)
	if err != nil {
		return nil, fmt.Errorf("读取商户私钥失败：%w", err)
	}

	var client *core.Client
	if UsesPublicKeyMode(cfg) {
		if strings.TrimSpace(cfg.WechatPayPublicKeyID) == "" {
			return nil, errors.New("请先填写微信支付公钥编号")
		}
		if strings.TrimSpace(cfg.WechatPayPublicKey) == "" && strings.TrimSpace(cfg.WechatPayPublicKeyPath) == "" {
			return nil, errors.New("请先填写微信支付公钥内容")
		}
		publicKey, err := loadRSAPublicKey(cfg)
		if err != nil {
			return nil, fmt.Errorf("读取微信支付公钥失败：%w", err)
		}
		client, err = core.NewClient(ctx, option.WithWechatPayPublicKeyAuthCipher(
			cfg.MchID,
			cfg.MerchantSerialNo,
			privateKey,
			cfg.WechatPayPublicKeyID,
			publicKey,
		))
		if err != nil {
			return nil, NormalizeAPIError(fmt.Errorf("创建微信支付公钥模式客户端失败：%w", err))
		}
	} else {
		if strings.TrimSpace(apiV3Key) == "" {
			return nil, errors.New("请先填写 APIv3 密钥")
		}
		client, err = core.NewClient(ctx, option.WithWechatPayAutoAuthCipher(
			cfg.MchID,
			cfg.MerchantSerialNo,
			privateKey,
			apiV3Key,
		))
		if err != nil {
			return nil, NormalizeAPIError(fmt.Errorf("创建微信支付自动验签客户端失败：%w", err))
		}
	}

	service := native.NativeApiService{Client: client}
	return &service, nil
}

func BuildNotifyHandler(ctx context.Context, cfg Config, apiV3Key string) (*notify.Handler, error) {
	if strings.TrimSpace(apiV3Key) == "" {
		return nil, errors.New("请先填写 APIv3 密钥")
	}

	verifier, err := buildVerifier(ctx, cfg, apiV3Key)
	if err != nil {
		return nil, err
	}
	return notify.NewRSANotifyHandler(apiV3Key, verifier)
}

func buildVerifier(ctx context.Context, cfg Config, apiV3Key string) (auth.Verifier, error) {
	if UsesPublicKeyMode(cfg) {
		if strings.TrimSpace(cfg.WechatPayPublicKeyID) == "" {
			return nil, errors.New("请先填写微信支付公钥编号")
		}
		if strings.TrimSpace(cfg.WechatPayPublicKey) == "" && strings.TrimSpace(cfg.WechatPayPublicKeyPath) == "" {
			return nil, errors.New("请先填写微信支付公钥内容")
		}
		publicKey, err := loadRSAPublicKey(cfg)
		if err != nil {
			return nil, fmt.Errorf("读取微信支付公钥失败：%w", err)
		}
		return verifiers.NewSHA256WithRSAPubkeyVerifier(cfg.WechatPayPublicKeyID, *publicKey), nil
	}

	if strings.TrimSpace(cfg.PlatformCert) != "" || strings.TrimSpace(cfg.PlatformCertPath) != "" {
		certificate, err := loadCertificate(cfg)
		if err != nil {
			return nil, fmt.Errorf("读取平台证书失败：%w", err)
		}
		publicKey, ok := certificate.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("平台证书中的公钥必须是 RSA")
		}

		keyID := strings.TrimSpace(cfg.PlatformCertSerialNo)
		if keyID == "" {
			keyID = strings.ToUpper(certificate.SerialNumber.Text(16))
		}
		return verifiers.NewSHA256WithRSAPubkeyVerifier(keyID, *publicKey), nil
	}

	if strings.TrimSpace(cfg.KeyPem) == "" && strings.TrimSpace(cfg.KeyPemPath) == "" {
		return nil, errors.New("请先填写商户私钥内容")
	}
	privateKey, err := loadPrivateKey(cfg)
	if err != nil {
		return nil, fmt.Errorf("读取商户私钥失败：%w", err)
	}

	certificateDownloader, err := downloader.NewCertificateDownloader(
		ctx,
		cfg.MchID,
		privateKey,
		cfg.MerchantSerialNo,
		apiV3Key,
	)
	if err != nil {
		return nil, fmt.Errorf("创建平台证书下载器失败：%w", err)
	}
	return verifiers.NewSHA256WithRSAVerifier(certificateDownloader), nil
}

func loadPrivateKey(cfg Config) (*rsa.PrivateKey, error) {
	if strings.TrimSpace(cfg.KeyPem) != "" {
		return parseRSAPrivateKey([]byte(cfg.KeyPem))
	}
	return utils.LoadPrivateKeyWithPath(strings.TrimSpace(cfg.KeyPemPath))
}

func loadCertificate(cfg Config) (*x509.Certificate, error) {
	if strings.TrimSpace(cfg.PlatformCert) != "" {
		return parseCertificate([]byte(cfg.PlatformCert))
	}
	return utils.LoadCertificateWithPath(strings.TrimSpace(cfg.PlatformCertPath))
}

func loadRSAPublicKey(cfg Config) (*rsa.PublicKey, error) {
	if strings.TrimSpace(cfg.WechatPayPublicKey) != "" {
		return parseRSAPublicKey([]byte(cfg.WechatPayPublicKey))
	}

	content, err := os.ReadFile(strings.TrimSpace(cfg.WechatPayPublicKeyPath))
	if err != nil {
		return nil, err
	}
	return parseRSAPublicKey(content)
}

func parseRSAPrivateKey(content []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(content)
	if block == nil {
		return nil, errors.New("invalid pem private key")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	privateKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not rsa")
	}
	return privateKey, nil
}

func parseCertificate(content []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(content)
	if block == nil {
		certificate, err := x509.ParseCertificate(content)
		if err != nil {
			return nil, errors.New("invalid platform certificate")
		}
		return certificate, nil
	}

	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return certificate, nil
}

func parseRSAPublicKey(content []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(content)
	if block == nil {
		return nil, errors.New("invalid pem public key")
	}

	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		publicKey, ok := parsed.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key is not rsa")
		}
		return publicKey, nil
	}

	certificate, certErr := x509.ParseCertificate(block.Bytes)
	if certErr != nil {
		return nil, err
	}
	publicKey, ok := certificate.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("certificate public key is not rsa")
	}
	return publicKey, nil
}
