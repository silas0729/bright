import QRCode from "qrcode";
import { useDeferredValue, useEffect, useRef, useState, type FormEvent } from "react";

import {
  api,
  type CaptchaChallenge,
  type CatalogStats,
  type ClassificationStat,
  type LearnerSession,
  type LearnerUser,
  type PagedClassificationStats,
  type PagedWords,
  type PaymentOrderStatus,
  type Plan,
  type SiteSetting,
  type Subject,
  type WechatOrder,
} from "./api";

const publicPageSize = 18;
const publicClassificationPageSize = 8;
const publicUIStateStorageKey = "brights_public_ui_state";
const publicSessionStorageKey = "brights_public_session";

type AuthMode = "login" | "register";
type PublicView = "home" | "profile";

const fallbackSiteSettings: SiteSetting = {
  site_name: "Brights 英语单词学习站",
  site_icon: "",
  site_tagline: "先学真正会用到的词，再把词汇量慢慢做厚。",
  hero_title: "高频英语单词，从真实场景开始学",
  hero_description:
    "围绕校园、日常、旅行、职场等高频场景整理常用英语单词，先学真正会遇到、会使用、会反复出现的词，再逐步扩展到更多学科和更系统的学习内容。",
  seo_headline: "高频英语单词｜场景词汇｜会员制学习",
  seo_title: "Brights 英语单词学习站 | 高频英语单词、场景词汇、会员制学习平台",
  seo_description:
    "Brights 专注高频英语单词学习，围绕校园、日常、旅行、职场等真实场景整理常用英语词汇，提供中文释义、分类学习、会员内容与多学科扩展能力。",
  seo_keywords:
    "英语单词学习,高频英语单词,场景英语词汇,英语学习网站,英语会员学习,英语词汇记忆",
  footer_text: "Brights 适合以英语高频词汇为主线持续学习，也支持后续扩展更多学科内容。",
  contact_email: "support@brights.local",
};

export default function PublicSite() {
  const persistedUIState = readStoredState<{
    subjectKey?: string;
    classification?: string;
    classificationPage?: number;
    query?: string;
    page?: number;
  }>(publicUIStateStorageKey, {});
  const persistedSession = readStoredState<LearnerSession | null>(publicSessionStorageKey, null);

  const [session, setSession] = useState<LearnerSession | null>(persistedSession);
  const [currentUser, setCurrentUser] = useState<LearnerUser | null>(persistedSession?.user ?? null);
  const [authMode, setAuthMode] = useState<AuthMode>(persistedSession ? "login" : "register");
  const [authForm, setAuthForm] = useState({
    username: "",
    displayName: "",
    password: "",
    confirmPassword: "",
  });
  const [authCaptcha, setAuthCaptcha] = useState<CaptchaChallenge | null>(null);
  const [authCaptchaAnswer, setAuthCaptchaAnswer] = useState("");
  const [authBusy, setAuthBusy] = useState("");
  const [authError, setAuthError] = useState("");
  const [authNotice, setAuthNotice] = useState("");
  const [authCaptchaLoading, setAuthCaptchaLoading] = useState(false);

  const [siteSettings, setSiteSettings] = useState<SiteSetting>(fallbackSiteSettings);
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [stats, setStats] = useState<CatalogStats | null>(null);
  const [plans, setPlans] = useState<Plan[]>([]);
  const [classificationResult, setClassificationResult] = useState<PagedClassificationStats | null>(null);
  const [words, setWords] = useState<PagedWords | null>(null);
  const [subjectKey, setSubjectKey] = useState(persistedUIState.subjectKey ?? "english");
  const [classification, setClassification] = useState(persistedUIState.classification ?? "");
  const [classificationPage, setClassificationPage] = useState(Math.max(1, persistedUIState.classificationPage ?? 1));
  const [query, setQuery] = useState(persistedUIState.query ?? "");
  const [page, setPage] = useState(Math.max(1, persistedUIState.page ?? 1));
  const [currentHash, setCurrentHash] = useState(() => getCurrentHash());
  const [loadingClassifications, setLoadingClassifications] = useState(false);
  const [classificationError, setClassificationError] = useState("");
  const [loadingWords, setLoadingWords] = useState(false);
  const [error, setError] = useState("");
  const [speakingTerm, setSpeakingTerm] = useState("");
  const [accountMenuOpen, setAccountMenuOpen] = useState(false);

  const [checkoutPlan, setCheckoutPlan] = useState<Plan | null>(null);
  const [checkoutCustomerRef, setCheckoutCustomerRef] = useState("");
  const [checkoutDescription, setCheckoutDescription] = useState("");
  const [checkoutOrder, setCheckoutOrder] = useState<WechatOrder | null>(null);
  const [checkoutStatus, setCheckoutStatus] = useState<PaymentOrderStatus | null>(null);
  const [checkoutError, setCheckoutError] = useState("");
  const [checkoutBusy, setCheckoutBusy] = useState(false);
  const [qrCodeDataUrl, setQrCodeDataUrl] = useState("");

  const deferredQuery = useDeferredValue(query);
  const activeCheckoutOrder = checkoutStatus?.order ?? checkoutOrder;
  const currentSettings = siteSettings ?? fallbackSiteSettings;
  const learnerName = currentUser?.display_name || currentUser?.username || "";
  const speechSupported = canUseBrowserSpeech();
  const activeView: PublicView = resolvePublicView(currentHash);
  const classifications = classificationResult?.items ?? [];
  const classificationTotal = classificationResult?.total ?? 0;
  const speakTimerRef = useRef<number | null>(null);
  const speechTokenRef = useRef(0);
  const accountMenuRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    let active = true;

    Promise.all([api.getSiteSettings(), api.getSubjects(), api.getStats(), api.getPlans()])
      .then(([settingsData, subjectData, statsData, planData]) => {
        if (!active) {
          return;
        }
        setSiteSettings(settingsData);
        setSubjects(subjectData);
        setStats(statsData);
        setPlans(planData);
        if (subjectData.length > 0 && !subjectData.some((item) => item.key === subjectKey)) {
          setSubjectKey(subjectData[0].key);
        }
      })
      .catch((err: Error) => {
        if (!active) {
          return;
        }
        setError(err.message);
      });

    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    document.title = currentSettings.seo_title || currentSettings.site_name;
    applyMetaTag("description", currentSettings.seo_description);
    applyMetaTag("keywords", currentSettings.seo_keywords);
    applyIconLink(currentSettings.site_icon);
  }, [currentSettings]);

  useEffect(() => {
    const handleHashChange = () => {
      setCurrentHash(getCurrentHash());
    };

    handleHashChange();
    window.addEventListener("hashchange", handleHashChange);
    return () => {
      window.removeEventListener("hashchange", handleHashChange);
    };
  }, []);

  useEffect(() => {
    const targetId = currentHash.replace(/^#/, "");
    window.requestAnimationFrame(() => {
      if (!targetId || targetId === "home" || targetId === "profile" || targetId === "account") {
        window.scrollTo({ top: 0, left: 0, behavior: "auto" });
        return;
      }

      const targetElement = document.getElementById(targetId);
      if (targetElement) {
        targetElement.scrollIntoView({ block: "start" });
        return;
      }

      window.scrollTo({ top: 0, left: 0, behavior: "auto" });
    });
  }, [activeView, currentHash]);

  useEffect(() => {
    setAccountMenuOpen(false);
  }, [currentHash]);

  useEffect(() => {
    if (!accountMenuOpen) {
      return;
    }

    const handleDocumentClick = (event: MouseEvent) => {
      if (!(event.target instanceof Node)) {
        return;
      }
      if (!accountMenuRef.current?.contains(event.target)) {
        setAccountMenuOpen(false);
      }
    };

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setAccountMenuOpen(false);
      }
    };

    document.addEventListener("mousedown", handleDocumentClick);
    document.addEventListener("keydown", handleEscape);
    return () => {
      document.removeEventListener("mousedown", handleDocumentClick);
      document.removeEventListener("keydown", handleEscape);
    };
  }, [accountMenuOpen]);

  useEffect(() => {
    if (!session?.access_token) {
      setCurrentUser(null);
      return;
    }

    let active = true;
    api
      .learnerMe(session.access_token)
      .then((user) => {
        if (!active) {
          return;
        }
        setCurrentUser(user);
        window.localStorage.setItem(
          publicSessionStorageKey,
          JSON.stringify({
            ...session,
            user,
          }),
        );
      })
      .catch((err: Error) => {
        if (!active) {
          return;
        }
        clearLearnerSession();
        setSession(null);
        setCurrentUser(null);
        setAuthError(err.message || "登录状态已失效，请重新登录。");
      });

    return () => {
      active = false;
    };
  }, [session]);

  useEffect(() => {
    if (currentUser) {
      return;
    }
    void refreshAuthCaptcha(authMode);
  }, [authMode, currentUser]);

  useEffect(() => {
    let active = true;
    setLoadingClassifications(true);
    setClassificationError("");

    api
      .getClassifications({
        subjectKey,
        page: classificationPage,
        pageSize: publicClassificationPageSize,
      })
      .then((result) => {
        if (!active) {
          return;
        }
        setClassificationResult(result);
      })
      .catch((err: Error) => {
        if (!active) {
          return;
        }
        setClassificationError(err.message);
      })
      .finally(() => {
        if (!active) {
          return;
        }
        setLoadingClassifications(false);
      });

    return () => {
      active = false;
    };
  }, [classificationPage, subjectKey]);

  useEffect(() => {
    if (!classificationResult) {
      return;
    }

    const totalPages = Math.max(1, Math.ceil(classificationResult.total / Math.max(classificationResult.page_size, 1)));
    if (classificationPage <= totalPages) {
      return;
    }
    setClassificationPage(totalPages);
  }, [classificationPage, classificationResult]);

  useEffect(() => {
    let active = true;
    setLoadingWords(true);
    setError("");

    api
      .getWords({
        subjectKey,
        classification,
        query: deferredQuery,
        page,
        pageSize: publicPageSize,
      })
      .then((result) => {
        if (!active) {
          return;
        }
        setWords(result);
      })
      .catch((err: Error) => {
        if (!active) {
          return;
        }
        setError(err.message);
      })
      .finally(() => {
        if (!active) {
          return;
        }
        setLoadingWords(false);
      });

    return () => {
      active = false;
    };
  }, [classification, deferredQuery, page, subjectKey]);

  useEffect(() => {
    if (!activeCheckoutOrder?.code_url) {
      setQrCodeDataUrl("");
      return;
    }

    let active = true;
    QRCode.toDataURL(activeCheckoutOrder.code_url, {
      width: 300,
      margin: 1,
      color: {
        dark: "#1f2937",
        light: "#ffffff",
      },
    })
      .then((dataUrl: string) => {
        if (!active) {
          return;
        }
        setQrCodeDataUrl(dataUrl);
      })
      .catch(() => {
        if (!active) {
          return;
        }
        setQrCodeDataUrl("");
      });

    return () => {
      active = false;
    };
  }, [activeCheckoutOrder?.code_url]);

  useEffect(() => {
    if (!checkoutOrder?.order_no) {
      return;
    }
    if ((checkoutStatus?.order.status ?? checkoutOrder.status) !== "pending") {
      return;
    }

    let cancelled = false;
    const timer = window.setInterval(() => {
      api
        .getWechatOrderStatus(checkoutOrder.order_no, checkoutOrder.customer_ref)
        .then((result) => {
          if (cancelled) {
            return;
          }
          setCheckoutStatus(result);
          if (result.order.status !== "pending") {
            window.clearInterval(timer);
          }
        })
        .catch((err: Error) => {
          if (cancelled) {
            return;
          }
          setCheckoutError(err.message);
        });
    }, 3000);

    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [checkoutOrder, checkoutStatus?.order.status]);

  useEffect(() => {
    window.localStorage.setItem(
      publicUIStateStorageKey,
        JSON.stringify({
          subjectKey,
          classification,
          classificationPage,
          query,
          page,
        }),
    );
  }, [classification, classificationPage, page, query, subjectKey]);

  useEffect(() => {
    if (!checkoutPlan || checkoutOrder || !currentUser) {
      return;
    }
    setCheckoutCustomerRef(currentUser.username);
  }, [checkoutOrder, checkoutPlan, currentUser]);

  useEffect(() => {
    if (!speechSupported) {
      return;
    }

    window.speechSynthesis.getVoices();

    return () => {
      if (speakTimerRef.current !== null) {
        window.clearTimeout(speakTimerRef.current);
      }
      speechTokenRef.current += 1;
      window.speechSynthesis.cancel();
    };
  }, [speechSupported]);

  async function refreshAuthCaptcha(mode: AuthMode) {
    setAuthCaptchaLoading(true);
    try {
      const nextCaptcha = await api.getCaptcha(mode === "register" ? "learner_register" : "learner_login");
      setAuthCaptcha(nextCaptcha);
      setAuthCaptchaAnswer("");
    } catch (err) {
      setAuthError((err as Error).message);
    } finally {
      setAuthCaptchaLoading(false);
    }
  }

  function persistLearnerSession(nextSession: LearnerSession) {
    window.localStorage.setItem(publicSessionStorageKey, JSON.stringify(nextSession));
    setSession(nextSession);
    setCurrentUser(nextSession.user);
  }

  function clearLearnerSession() {
    window.localStorage.removeItem(publicSessionStorageKey);
  }

  function openCheckout(plan: Plan) {
    setCheckoutPlan(plan);
    setCheckoutCustomerRef(currentUser?.username ?? "");
    setCheckoutDescription(plan.name);
    setCheckoutOrder(null);
    setCheckoutStatus(null);
    setCheckoutError("");
    setCheckoutBusy(false);
    setQrCodeDataUrl("");
  }

  function closeCheckout() {
    setCheckoutPlan(null);
    setCheckoutCustomerRef("");
    setCheckoutDescription("");
    setCheckoutOrder(null);
    setCheckoutStatus(null);
    setCheckoutError("");
    setCheckoutBusy(false);
    setQrCodeDataUrl("");
  }

  function handleSpeakWord(term: string) {
    const nextTerm = term.trim();
    if (!nextTerm || !speechSupported) {
      return;
    }

    const synth = window.speechSynthesis;
    const isSameTermSpeaking = synth.speaking && speakingTerm === nextTerm;

    if (speakTimerRef.current !== null) {
      window.clearTimeout(speakTimerRef.current);
      speakTimerRef.current = null;
    }

    speechTokenRef.current += 1;
    const token = speechTokenRef.current;

    if (synth.speaking) {
      synth.cancel();
      setSpeakingTerm("");
      if (isSameTermSpeaking) {
        return;
      }
    }

    const utterance = new window.SpeechSynthesisUtterance(nextTerm);
    const preferredVoice = pickEnglishVoice(synth.getVoices());
    if (preferredVoice) {
      utterance.voice = preferredVoice;
      utterance.lang = preferredVoice.lang || "en-US";
    } else {
      utterance.lang = "en-US";
    }
    utterance.rate = 0.92;
    utterance.pitch = 1;
    utterance.volume = 1;
    utterance.onstart = () => {
      if (speechTokenRef.current === token) {
        setSpeakingTerm(nextTerm);
      }
    };
    utterance.onend = () => {
      if (speechTokenRef.current === token) {
        setSpeakingTerm("");
      }
    };
    utterance.onerror = () => {
      if (speechTokenRef.current === token) {
        setSpeakingTerm("");
      }
    };

    speakTimerRef.current = window.setTimeout(() => {
      if (speechTokenRef.current !== token) {
        return;
      }
      synth.speak(utterance);
      speakTimerRef.current = null;
    }, 40);
  }

  async function handleSubmitAuth(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAuthBusy(authMode);
    setAuthError("");
    setAuthNotice("");

    if (authMode === "register" && authForm.password !== authForm.confirmPassword) {
      setAuthError("两次输入的密码不一致。");
      setAuthBusy("");
      return;
    }
    if (!authCaptcha?.captcha_id || !authCaptchaAnswer.trim()) {
      setAuthError("请先填写图形验证码。");
      setAuthBusy("");
      return;
    }

    try {
      if (authMode === "register") {
        const nextSession = await api.learnerRegister({
          username: authForm.username,
          password: authForm.password,
          display_name: authForm.displayName,
          captcha_id: authCaptcha.captcha_id,
          captcha_answer: authCaptchaAnswer,
        });
        persistLearnerSession(nextSession);
        setCheckoutCustomerRef(nextSession.user.username);
        setAuthNotice("注册成功，已经自动登录。");
      } else {
        const nextSession = await api.learnerLogin(
          authForm.username,
          authForm.password,
          authCaptcha.captcha_id,
          authCaptchaAnswer,
        );
        persistLearnerSession(nextSession);
        setCheckoutCustomerRef(nextSession.user.username);
        setAuthNotice("登录成功。");
      }

      setAuthForm({
        username: "",
        displayName: "",
        password: "",
        confirmPassword: "",
      });
      setAuthCaptchaAnswer("");
    } catch (err) {
      setAuthError((err as Error).message);
      await refreshAuthCaptcha(authMode);
    } finally {
      setAuthBusy("");
    }
  }

  async function handleLogout() {
    const token = session?.access_token ?? "";
    try {
      if (token) {
        await api.learnerLogout(token);
      }
    } catch {
      // Ignore logout request failures and clear local session anyway.
    } finally {
      setAccountMenuOpen(false);
      clearLearnerSession();
      setSession(null);
      setCurrentUser(null);
      setAuthNotice("你已退出登录。");
      setCheckoutCustomerRef("");
      void refreshAuthCaptcha(authMode);
    }
  }

  async function handleCreateCheckoutOrder(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!checkoutPlan) {
      return;
    }
    if (!currentUser) {
      setCheckoutError("请先注册或登录学习账号，再继续购买。");
      return;
    }

    setCheckoutBusy(true);
    setCheckoutError("");

    try {
      const order = await api.createWechatOrder({
        plan_id: checkoutPlan.id,
        plan_key: checkoutPlan.key,
        subject_key: subjectKey,
        customer_ref: currentUser.username,
        description: checkoutDescription,
      });
      setCheckoutCustomerRef(currentUser.username);
      setCheckoutOrder(order);
      setCheckoutStatus({ order });
    } catch (err) {
      setCheckoutError((err as Error).message);
    } finally {
      setCheckoutBusy(false);
    }
  }

  async function handleCopyPaymentLink() {
    if (!activeCheckoutOrder?.code_url || !window.navigator.clipboard) {
      return;
    }

    try {
      await window.navigator.clipboard.writeText(activeCheckoutOrder.code_url);
    } catch {
      setCheckoutError("当前浏览器暂不支持复制支付链接，请直接扫码支付。");
    }
  }

  const formatSubjectLabel = (value?: string) => {
    const key = value?.trim() ?? "";
    if (!key) {
      return "-";
    }
    return subjects.find((item) => item.key === key)?.name ?? key;
  };

  const profileIntroTitle = currentUser ? "你的学习账号与会员安排" : "先注册学习账号，再开始持续学习";
  const profileIntroDescription = currentUser
    ? `你好，${learnerName}。这里集中放你的账号信息、购买入口和后续学习安排，后面扩展更多学科内容时也继续使用这一套学习身份。`
    : "先注册一个学习账号，后面不管是购买会员、切换设备继续学，还是扩展到其他学科内容，学习记录都会跟着你的账号一起保存。";
  const selectedSubjectLabel = formatSubjectLabel(subjectKey);
  const classificationOnCurrentPage = classification === "" || classifications.some((item) => item.name === classification);
  const classificationOptions: ClassificationStat[] =
    classification !== "" && !classifications.some((item) => item.name === classification)
      ? [{ name: classification, count: 0 }, ...classifications]
      : classifications;

  return (
    <div className="site-shell">
      <header className="site-header">
        <a className="site-brand" href="#home">
          <span className="site-logo">
            {currentSettings.site_icon ? (
              <img alt={`${currentSettings.site_name} 图标`} className="site-logo-image" src={currentSettings.site_icon} />
            ) : (
              "B"
            )}
          </span>
          <div>
            <strong>{currentSettings.site_name}</strong>
            <p>{currentSettings.site_tagline}</p>
          </div>
        </a>
        <nav className="site-topnav">
          <a href="#catalog">词库学习</a>
          <a href="#plans">会员方案</a>
        </nav>
        <div className="site-header-actions">
          {currentUser ? (
            <div className="site-account-menu" ref={accountMenuRef}>
              <button
                aria-controls="site-account-dropdown"
                aria-expanded={accountMenuOpen}
                aria-haspopup="menu"
                className={accountMenuOpen ? "site-account-trigger site-account-trigger-open" : "site-account-trigger"}
                onClick={() => {
                  setAccountMenuOpen((current) => !current);
                }}
                type="button"
              >
                <div className="site-account-meta">
                  <strong>{learnerName}</strong>
                  <span>{currentUser.username}</span>
                </div>
                <span aria-hidden="true" className="site-account-caret" />
              </button>

              {accountMenuOpen ? (
                <div className="site-account-dropdown" id="site-account-dropdown" role="menu">
                  <div className="site-account-dropdown-header">
                    <strong>{learnerName}</strong>
                    <span>{currentUser.username}</span>
                  </div>
                  <div className="site-account-dropdown-links">
                    <a
                      className="site-account-dropdown-link"
                      href="#profile"
                      onClick={() => {
                        setAccountMenuOpen(false);
                      }}
                      role="menuitem"
                    >
                      进入个人中心
                    </a>
                    <a
                      className="site-account-dropdown-link"
                      href="#catalog"
                      onClick={() => {
                        setAccountMenuOpen(false);
                      }}
                      role="menuitem"
                    >
                      返回词库学习
                    </a>
                    <a
                      className="site-account-dropdown-link"
                      href="#plans"
                      onClick={() => {
                        setAccountMenuOpen(false);
                      }}
                      role="menuitem"
                    >
                      开通 / 续费会员
                    </a>
                    <button
                      className="site-account-dropdown-link site-account-dropdown-link-danger"
                      onClick={() => {
                        void handleLogout();
                      }}
                      role="menuitem"
                      type="button"
                    >
                      退出登录
                    </button>
                  </div>
                </div>
              ) : null}
            </div>
          ) : (
            <>
              <a className="secondary-button site-account-link" href="#profile">
                登录 / 注册
              </a>
              <a className="primary-button site-header-buy" href="#plans">
                购买会员
              </a>
            </>
          )}
        </div>
      </header>

      {activeView === "profile" ? (
        <div className="site-main site-main-profile">
          <main className="site-content profile-page">
            <section className="content-card profile-hero-card" id="profile">
              <div className="section-header profile-hero-header">
                <div>
                  <p className="section-eyebrow">个人中心</p>
                  <h1>{profileIntroTitle}</h1>
                  <p className="helper-text">{profileIntroDescription}</p>
                </div>
                <div className="button-row">
                  <a className="secondary-button" href="#catalog">
                    去词库学习
                  </a>
                  <a className="primary-button" href="#plans">
                    查看会员方案
                  </a>
                </div>
              </div>

              <div className="profile-overview">
                <div>
                  <strong>{currentUser ? learnerName || currentUser.username : "未登录"}</strong>
                  <span>{currentUser ? "当前学习身份" : "登录后可保存学习记录"}</span>
                </div>
                <div>
                  <strong>{currentUser ? "可正常学习" : "等待注册 / 登录"}</strong>
                  <span>账号状态</span>
                </div>
                <div>
                  <strong>{selectedSubjectLabel}</strong>
                  <span>当前学习科目</span>
                </div>
                <div>
                  <strong>{plans.length > 0 ? `${plans.length} 种可选方案` : "暂未上架方案"}</strong>
                  <span>会员购买入口</span>
                </div>
              </div>
            </section>

            <div className="profile-grid">
              <section className="content-card profile-card">
                <div className="section-header">
                  <div>
                    <p className="section-eyebrow">账号概览</p>
                    <h2>{currentUser ? "学习资料和权益都跟着这个账号走" : "先准备好你的学习账号"}</h2>
                  </div>
                </div>

                {currentUser ? (
                  <>
                    <p className="helper-text">
                      你的购买记录、会员权益和后续学习进度都会绑定到这个账号，后续继续补充其他学科内容时也可以共用这一套学习身份。
                    </p>
                    <dl className="metric-list">
                      <div>
                        <dt>学习账号</dt>
                        <dd>{currentUser.username}</dd>
                      </div>
                      <div>
                        <dt>账号昵称</dt>
                        <dd>{currentUser.display_name || "-"}</dd>
                      </div>
                      <div>
                        <dt>注册时间</dt>
                        <dd>{formatDateTime(currentUser.created_at)}</dd>
                      </div>
                      <div>
                        <dt>当前学习科目</dt>
                        <dd>{selectedSubjectLabel}</dd>
                      </div>
                    </dl>
                    <div className="button-row">
                      <a className="primary-button" href="#plans">
                        去看会员方案
                      </a>
                      <a className="secondary-button" href="#catalog">
                        返回词库学习
                      </a>
                    </div>
                  </>
                ) : (
                  <>
                    <p className="helper-text">
                      注册之后，购买会员、切换设备继续学、后续增加其他科目内容，都会继续沿用同一个学习账号，不用重复建立新的学习身份。
                    </p>
                    <div className="tag-list">
                      <span className="tag">购买记录跟账号绑定</span>
                      <span className="tag">会员权益自动发放</span>
                      <span className="tag">后续学习进度可持续保留</span>
                    </div>
                    <div className="button-row">
                      <button
                        className={authMode === "register" ? "primary-button small-button" : "secondary-button small-button"}
                        onClick={() => setAuthMode("register")}
                        type="button"
                      >
                        注册账号
                      </button>
                      <button
                        className={authMode === "login" ? "primary-button small-button" : "secondary-button small-button"}
                        onClick={() => setAuthMode("login")}
                        type="button"
                      >
                        已有账号登录
                      </button>
                    </div>
                  </>
                )}
              </section>

              <section className="content-card profile-card">
                <div className="section-header">
                  <div>
                    <p className="section-eyebrow">{currentUser ? "会员与服务" : authMode === "register" ? "注册账号" : "账号登录"}</p>
                    <h2>{currentUser ? "购买后的权益会自动关联到你的账号" : authMode === "register" ? "填写资料，马上开始学习" : "输入账号信息，继续上次学习"}</h2>
                  </div>
                </div>

                {!currentUser ? (
                  <form className="setup-form" onSubmit={handleSubmitAuth}>
                    <label className="form-field">
                      <span>学习账号</span>
                      <input
                        value={authForm.username}
                        onChange={(event) => {
                          setAuthForm((current) => ({ ...current, username: event.target.value }));
                        }}
                        placeholder="例如：xiaoming"
                      />
                    </label>
                    {authMode === "register" ? (
                      <label className="form-field">
                        <span>昵称</span>
                        <input
                          value={authForm.displayName}
                          onChange={(event) => {
                            setAuthForm((current) => ({ ...current, displayName: event.target.value }));
                          }}
                          placeholder="例如：小明"
                        />
                      </label>
                    ) : null}
                    <label className="form-field">
                      <span>登录密码</span>
                      <input
                        type="password"
                        value={authForm.password}
                        onChange={(event) => {
                          setAuthForm((current) => ({ ...current, password: event.target.value }));
                        }}
                        placeholder="至少 8 位"
                      />
                    </label>
                    {authMode === "register" ? (
                      <label className="form-field">
                        <span>确认密码</span>
                        <input
                          type="password"
                          value={authForm.confirmPassword}
                          onChange={(event) => {
                            setAuthForm((current) => ({ ...current, confirmPassword: event.target.value }));
                          }}
                          placeholder="请再输入一次密码"
                        />
                      </label>
                    ) : null}
                    <div className="form-grid-two">
                      <label className="form-field">
                        <span>图形验证码</span>
                        <input
                          value={authCaptchaAnswer}
                          onChange={(event) => {
                            setAuthCaptchaAnswer(event.target.value);
                          }}
                          placeholder="请输入图中的字符"
                        />
                      </label>
                      <div className="form-field">
                        <span>验证码图片</span>
                        <div className="button-row">
                          <img
                            alt="图形验证码"
                            className="captcha-image"
                            src={authCaptcha?.image_data || ""}
                          />
                          <button
                            className="secondary-button small-button"
                            disabled={authCaptchaLoading}
                            onClick={() => {
                              void refreshAuthCaptcha(authMode);
                            }}
                            type="button"
                          >
                            {authCaptchaLoading ? "刷新中..." : "换一张"}
                          </button>
                        </div>
                      </div>
                    </div>
                    {authNotice ? <div className="feedback-banner feedback-success">{authNotice}</div> : null}
                    {authError ? <div className="feedback-banner feedback-error">{authError}</div> : null}
                    <button className="primary-button" disabled={authBusy !== ""} type="submit">
                      {authBusy === "register"
                        ? "注册中..."
                        : authBusy === "login"
                          ? "登录中..."
                          : authMode === "register"
                            ? "注册并开始学习"
                            : "进入个人中心"}
                    </button>
                  </form>
                ) : (
                  <div className="profile-card-body">
                    {authNotice ? <div className="feedback-banner feedback-success">{authNotice}</div> : null}
                    {authError ? <div className="feedback-banner feedback-error">{authError}</div> : null}
                    <div className="feedback-banner">
                      购买会员后，后台会直接把会员状态和有效期关联到账号 <strong>{currentUser.username}</strong>，你在前台继续学习时就能一直使用同一个账号。
                    </div>
                    <p className="helper-text">
                      如果你准备开通会员，可以从顶部导航、这里的按钮，或者会员方案页面进入购买；付款完成后，这个账号就是后续所有学习记录的承接入口。
                    </p>
                    <div className="button-row">
                      <a className="primary-button" href="#plans">
                        立即购买会员
                      </a>
                      <a className="secondary-button" href="#catalog">
                        返回词库学习
                      </a>
                    </div>
                  </div>
                )}
              </section>
            </div>

            <section className="content-card profile-card">
              <div className="section-header">
                <div>
                  <p className="section-eyebrow">学习概况</p>
                  <h2>当前词库与学习方向一目了然</h2>
                </div>
              </div>
              <dl className="metric-list">
                <div>
                  <dt>科目数量</dt>
                  <dd>{stats?.subject_count ?? 0}</dd>
                </div>
                <div>
                  <dt>词汇数量</dt>
                  <dd>{formatCount(stats?.word_count ?? 0)}</dd>
                </div>
                <div>
                  <dt>场景分类</dt>
                  <dd>{stats?.classification_count ?? 0}</dd>
                </div>
                <div>
                  <dt>年级维度</dt>
                  <dd>{stats?.grade_count ?? 0}</dd>
                </div>
              </dl>
              <p className="helper-text">{currentSettings.footer_text}</p>
            </section>

            <section className="content-card" id="plans">
              <div className="section-header">
                <div>
                  <p className="section-eyebrow">会员方案</p>
                  <h2>选择更适合你的学习节奏</h2>
                </div>
              </div>

              <div className="plan-table">
                {plans.map((plan) => (
                  <article className="plan-row" key={plan.key}>
                    <div>
                      <div className="plan-row-header">
                        <h3>{plan.name}</h3>
                        {plan.recommended ? <span className="pill pill-primary">推荐选择</span> : null}
                      </div>
                      <p
                        className="plan-row-meta"
                        data-billing-label={plan.billing_mode === "monthly" ? "按月会员" : "一次性买断"}
                        data-payment-channels={formatPaymentChannels(plan.payment_channels)}
                      >
                        {plan.billing_mode}
                      </p>
                      <p>{plan.description}</p>
                      <div className="tag-list">
                        {plan.features.map((feature) => (
                          <span className="tag" key={feature}>
                            {feature}
                          </span>
                        ))}
                      </div>
                    </div>
                    <div className="plan-row-side">
                      <strong>{formatPrice(plan.price_cents)}</strong>
                      <button className="primary-button" onClick={() => openCheckout(plan)} type="button">
                        立即购买
                      </button>
                    </div>
                  </article>
                ))}
              </div>
            </section>

            <section className="content-card profile-card">
              <div className="section-header">
                <div>
                  <p className="section-eyebrow">学习建议</p>
                  <h2>先学会用得上的，再慢慢学得更广</h2>
                </div>
              </div>
              <p className="helper-text">
                {currentSettings.seo_description}
                {currentSettings.contact_email ? ` 如需合作或内容支持，可联系：${currentSettings.contact_email}` : ""}
              </p>
            </section>
          </main>
        </div>
      ) : (
        <div className="site-main">
          <aside className="site-sidebar">
            <section className="sidebar-card">
              <h3>选择学习科目</h3>
              <label className="form-field">
                <span>正在学习</span>
                <select
                  value={subjectKey}
                  onChange={(event) => {
                    setSubjectKey(event.target.value);
                    setClassification("");
                    setClassificationPage(1);
                    setPage(1);
                  }}
                >
                  {subjects.map((subject) => (
                    <option key={subject.key} value={subject.key}>
                      {subject.name}
                    </option>
                  ))}
                </select>
              </label>
            </section>

            <section className="sidebar-card">
              <h3>场景分类</h3>
              {classification !== "" && !classificationOnCurrentPage ? (
                <div className="sidebar-selection-note">当前正在学习：{classification}</div>
              ) : null}
              <div className="sidebar-list">
                <button
                  className={classification === "" ? "sidebar-link sidebar-link-active" : "sidebar-link"}
                  onClick={() => {
                    setClassification("");
                    setPage(1);
                  }}
                  type="button"
                >
                  全部分类
                  <span>{formatCount(classificationTotal)}</span>
                </button>
                {classifications.map((item) => (
                  <button
                    className={classification === item.name ? "sidebar-link sidebar-link-active" : "sidebar-link"}
                    key={item.name}
                    onClick={() => {
                      setClassification(item.name);
                      setPage(1);
                    }}
                    type="button"
                  >
                    {item.name}
                    <span>{formatCount(item.count)}</span>
                  </button>
                ))}
              </div>
              {classificationError ? <div className="sidebar-feedback">{classificationError}</div> : null}
              {loadingClassifications ? <div className="sidebar-feedback">正在加载分类...</div> : null}
              <div className="sidebar-pagination">
                <PagerControls
                  className="pager-compact"
                  disabled={loadingClassifications}
                  page={classificationPage}
                  total={classificationTotal}
                  pageSize={classificationResult?.page_size ?? publicClassificationPageSize}
                  onChange={setClassificationPage}
                />
              </div>
            </section>

          </aside>

          <main className="site-content">
            <section className="content-card hero-card">
              <div>
                <p className="section-eyebrow">{currentSettings.seo_headline || "英语高频词学习"}</p>
                <h1>{currentSettings.hero_title}</h1>
                <p>{currentSettings.hero_description}</p>
              </div>
              <div className="hero-summary">
                <div>
                  <strong>{formatCount(stats?.word_count ?? 0)}</strong>
                  <span>已收录单词</span>
                </div>
                <div>
                  <strong>{formatCount(classificationTotal)}</strong>
                  <span>场景分类</span>
                </div>
                <div>
                  <strong>{formatCount(plans.length)}</strong>
                  <span>会员方案</span>
                </div>
              </div>
            </section>

            <section className="content-card" id="catalog">
              <div className="section-header">
                <div>
                  <p className="section-eyebrow">词库学习</p>
                  <h2>{classification || "全部单词"}</h2>
                  <p className="helper-text word-pronounce-tip">
                    {speechSupported
                      ? "点击单词即可调用浏览器朗读英文发音，再点一次同一个单词可停止。"
                      : "当前浏览器暂不支持朗读功能，建议换用支持语音合成的现代浏览器。"}
                  </p>
                </div>
              </div>

              <div className="section-toolbar catalog-toolbar">
                <div className="catalog-toolbar-fields">
                  <label className="form-field catalog-toolbar-field">
                    <span>场景分类</span>
                    <select
                      value={classification}
                      onChange={(event) => {
                        setClassification(event.target.value);
                        setPage(1);
                      }}
                    >
                      <option value="">全部分类</option>
                      {classificationOptions.map((item) => (
                        <option key={item.name} value={item.name}>
                          {item.name}
                        </option>
                      ))}
                    </select>
                  </label>

                  <label className="form-field catalog-toolbar-field catalog-toolbar-search">
                    <span>搜索单词内容</span>
                    <input
                      value={query}
                      onChange={(event) => {
                        setQuery(event.target.value);
                        setPage(1);
                      }}
                      placeholder="输入英文单词、中文释义或相关关键词"
                    />
                  </label>
                </div>

                <div className="catalog-toolbar-meta">
                  <span>当前共 {formatCount(words?.total ?? 0)} 条内容</span>
                  {(classification || query.trim()) && (
                    <button
                      className="secondary-button small-button"
                      onClick={() => {
                        setClassification("");
                        setQuery("");
                        setPage(1);
                      }}
                      type="button"
                    >
                      清空筛选
                    </button>
                  )}
                </div>
              </div>

              {error ? <div className="feedback-banner feedback-error">{error}</div> : null}
              {loadingWords ? <div className="feedback-banner">正在加载学习内容...</div> : null}

              <div className="word-table-wrap">
                <table className="word-table">
                  <thead>
                    <tr>
                      <th>单词</th>
                      <th>释义</th>
                      <th>场景</th>
                      <th>音标</th>
                      <th>来源</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(words?.items ?? []).map((word) => (
                      <tr key={`${word.id}-${word.term}`}>
                        <td>
                          <button
                            aria-pressed={speakingTerm === word.term}
                            className={
                              speakingTerm === word.term
                                ? "word-term-button word-term-button-active"
                                : "word-term-button"
                            }
                            disabled={!speechSupported}
                            onClick={() => handleSpeakWord(word.term)}
                            title={speechSupported ? `点击朗读 ${word.term}` : "当前浏览器暂不支持朗读"}
                            type="button"
                          >
                            <span>{word.term}</span>
                            <small>{speakingTerm === word.term ? "朗读中" : "点读"}</small>
                          </button>
                          {word.explanation ? <p>{word.explanation}</p> : null}
                        </td>
                        <td>{word.translation || "-"}</td>
                        <td>{word.classification || "-"}</td>
                        <td>{word.phonetics || "-"}</td>
                        <td>{word.source || "-"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {!loadingWords && (words?.items ?? []).length === 0 ? (
                  <div className="feedback-banner">当前筛选条件下还没有匹配内容，换个关键词试试看。</div>
                ) : null}
              </div>

              <div className="catalog-pagination">
                <PagerControls
                  page={page}
                  total={words?.total ?? 0}
                  pageSize={words?.page_size ?? publicPageSize}
                  onChange={setPage}
                />
              </div>
            </section>

          </main>
        </div>
      )}

      {checkoutPlan ? (
        <div
          className="payment-modal-backdrop"
          onClick={(event) => {
            if (event.target === event.currentTarget) {
              closeCheckout();
            }
          }}
        >
          <section className="payment-modal">
            <div className="section-header">
              <div>
                <p className="section-eyebrow">微信支付</p>
                <h2>{checkoutPlan.name}</h2>
              </div>
              <button className="secondary-button" onClick={closeCheckout} type="button">
                关闭
              </button>
            </div>

            {!currentUser ? (
              <div className="feedback-banner feedback-error">
                请先注册或登录学习账号，再继续购买会员。购买成功后的会员权益会直接绑定到你的账号。
              </div>
            ) : !checkoutOrder ? (
              <form className="setup-form" onSubmit={handleCreateCheckoutOrder}>
                <p className="helper-text">
                  当前订单会自动绑定到账号 <strong>{currentUser.username}</strong>，支付成功后，后台可以直接看到你的会员状态和有效期。
                </p>
                <label className="form-field">
                  <span>绑定账号</span>
                  <input disabled value={checkoutCustomerRef || currentUser.username} />
                </label>
                <label className="form-field">
                  <span>订单说明</span>
                  <input
                    value={checkoutDescription}
                    onChange={(event) => {
                      setCheckoutDescription(event.target.value);
                    }}
                    placeholder={checkoutPlan.name}
                  />
                </label>
                {checkoutError ? <div className="feedback-banner feedback-error">{checkoutError}</div> : null}
                <button className="primary-button" disabled={checkoutBusy} type="submit">
                  {checkoutBusy ? "正在生成订单..." : "生成支付二维码"}
                </button>
              </form>
            ) : (
              <div className="payment-panel-grid">
                <div className="payment-qr-card">
                  {qrCodeDataUrl ? (
                    <img alt="微信支付二维码" className="payment-qr-image" src={qrCodeDataUrl} />
                  ) : (
                    <div className="feedback-banner">正在生成支付二维码...</div>
                  )}
                  <div className="button-row">
                    <button className="secondary-button" onClick={handleCopyPaymentLink} type="button">
                      复制支付链接
                    </button>
                    <button
                      className="secondary-button"
                      onClick={() => {
                        setCheckoutOrder(null);
                        setCheckoutStatus(null);
                        setCheckoutError("");
                        setQrCodeDataUrl("");
                      }}
                      type="button"
                    >
                      重新下单
                    </button>
                  </div>
                </div>

                <div className="payment-info-card">
                  <dl className="detail-grid">
                    <div>
                      <dt>订单状态</dt>
                      <dd>
                        <span className={`pill ${paymentStatusClass(activeCheckoutOrder?.status)}`}>
                          {paymentStatusLabel(activeCheckoutOrder?.status)}
                        </span>
                      </dd>
                    </div>
                    <div>
                      <dt>订单号</dt>
                      <dd>{activeCheckoutOrder?.order_no}</dd>
                    </div>
                    <div>
                      <dt>支付金额</dt>
                      <dd>{formatPrice(activeCheckoutOrder?.amount_cents ?? 0)}</dd>
                    </div>
                    <div>
                      <dt>绑定账号</dt>
                      <dd>{activeCheckoutOrder?.customer_ref}</dd>
                    </div>
                    <div>
                      <dt>所属科目</dt>
                      <dd>{formatSubjectLabel(activeCheckoutOrder?.subject_key)}</dd>
                    </div>
                    <div>
                      <dt>二维码失效时间</dt>
                      <dd>{activeCheckoutOrder?.expires_at ? formatDateTime(activeCheckoutOrder.expires_at) : "-"}</dd>
                    </div>
                  </dl>

                  {checkoutStatus?.subscription ? (
                    <div className="feedback-banner feedback-success">
                      会员状态：{subscriptionStatusLabel(checkoutStatus.subscription.status)}
                      {checkoutStatus.subscription.current_period_end
                        ? `，有效期至 ${formatDateTime(checkoutStatus.subscription.current_period_end)}`
                        : "，当前为长期有效"}
                    </div>
                  ) : null}
                  {checkoutError ? <div className="feedback-banner feedback-error">{checkoutError}</div> : null}
                  {activeCheckoutOrder?.error_message ? (
                    <div className="feedback-banner feedback-error">{activeCheckoutOrder.error_message}</div>
                  ) : null}
                </div>
              </div>
            )}
          </section>
        </div>
      ) : null}
    </div>
  );
}

function PagerControls(props: {
  page: number;
  total: number;
  pageSize: number;
  onChange: (page: number) => void;
  className?: string;
  disabled?: boolean;
}) {
  const totalPages = Math.max(1, Math.ceil(props.total / Math.max(props.pageSize, 1)));
  const pagerClassName = props.className ? `pager ${props.className}` : "pager";

  return (
    <div className={pagerClassName}>
      <button
        className="secondary-button small-button"
        disabled={props.disabled || props.page <= 1}
        onClick={() => props.onChange(Math.max(1, props.page - 1))}
        type="button"
      >
        上一页
      </button>
      <span>
        第 {props.page} / {totalPages} 页
      </span>
      <button
        className="secondary-button small-button"
        disabled={props.disabled || props.page >= totalPages}
        onClick={() => props.onChange(Math.min(totalPages, props.page + 1))}
        type="button"
      >
        下一页
      </button>
    </div>
  );
}

function applyMetaTag(name: string, content: string) {
  const trimmed = content.trim();
  let meta = document.querySelector(`meta[name="${name}"]`);
  if (!meta) {
    meta = document.createElement("meta");
    meta.setAttribute("name", name);
    document.head.appendChild(meta);
  }
  meta.setAttribute("content", trimmed);
}

function applyIconLink(href?: string) {
  const normalized = (href ?? "").trim();
  let link = document.querySelector('link[rel="icon"]') as HTMLLinkElement | null;
  if (!link) {
    link = document.createElement("link");
    link.setAttribute("rel", "icon");
    document.head.appendChild(link);
  }

  if (!normalized) {
    link.removeAttribute("href");
    return;
  }

  link.setAttribute("href", normalized);
}

function readStoredState<T>(key: string, fallback: T): T {
  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) {
      return fallback;
    }
    return JSON.parse(raw) as T;
  } catch {
    return fallback;
  }
}

function formatCount(value: number) {
  return new Intl.NumberFormat("zh-CN").format(value);
}

function formatPrice(value: number) {
  return `${(value / 100).toFixed(2)} 元`;
}

function formatDateTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }
  return date.toLocaleString("zh-CN");
}

function getCurrentHash() {
  if (typeof window === "undefined") {
    return "#home";
  }
  return window.location.hash || "#home";
}

function resolvePublicView(hash: string): PublicView {
  if (hash === "#profile" || hash === "#account" || hash === "#plans") {
    return "profile";
  }
  return "home";
}

function canUseBrowserSpeech() {
  return typeof window !== "undefined" && "speechSynthesis" in window && "SpeechSynthesisUtterance" in window;
}

function pickEnglishVoice(voices: SpeechSynthesisVoice[]) {
  return (
    voices.find((voice) => /^en(-|_)/i.test(voice.lang) && voice.default) ||
    voices.find((voice) => /^en(-|_)/i.test(voice.lang)) ||
    voices.find((voice) => /english/i.test(voice.name)) ||
    null
  );
}

function formatPaymentChannels(channels: string[]) {
  if (channels.length === 0) {
    return "微信支付";
  }

  const labels = channels.map((item) => {
    switch (item) {
      case "wechat_native":
        return "微信扫码支付";
      case "wechat_contract_pay":
        return "微信续费";
      case "wechat_jsapi":
        return "微信内支付";
      case "wechat":
        return "微信支付";
      default:
        return item;
    }
  });
  return labels.join(" / ");
}

function paymentStatusLabel(status?: string) {
  switch (status) {
    case "success":
      return "支付成功";
    case "failed":
      return "支付失败";
    case "closed":
      return "已关闭";
    default:
      return "待支付";
  }
}

function paymentStatusClass(status?: string) {
  switch (status) {
    case "success":
      return "pill-success";
    case "failed":
      return "pill-danger";
    case "closed":
      return "pill-muted";
    default:
      return "pill-warning";
  }
}

function subscriptionStatusLabel(status?: string) {
  switch (status) {
    case "active":
      return "生效中";
    case "expired":
      return "已过期";
    case "pending":
      return "待生效";
    case "cancelled":
      return "已取消";
    default:
      return status || "-";
  }
}
