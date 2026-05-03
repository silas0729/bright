import QRCode from "qrcode";
import { useDeferredValue, useEffect, useRef, useState, type FormEvent } from "react";

import {
  api,
  type APIConfig,
  type APIConfigTestResult,
  type CaptchaChallenge,
  type CatalogStats,
  type ClassificationStat,
  type InviteSummary,
  type KnowledgeBaseChunk,
  type KnowledgeBaseDocument,
  type LearningSummary,
  type LearnerSession,
  type LearnerUser,
  type PagedAPIConfigs,
  type PagedClassificationStats,
  type PagedKnowledgeBaseDocuments,
  type PagedLearningWordProgress,
  type PagedMCPMarketTools,
  type PagedPaymentOrders,
  type PagedSubscriptions,
  type PagedWords,
  type PaymentOrderStatus,
  type Plan,
  type SaveAPIConfigInput,
  type SiteSetting,
  type SubscriptionStatus,
  type Subject,
  type XiaomiDevice,
  type XiaomiConfig,
  type XiaomiDeviceListResult,
  type XiaomiHome,
  type XiaomiQRLoginResult,
  type WechatOrder,
} from "./api";
import MCPConsole from "./MCPConsole";

const publicPageSize = 18;
const publicClassificationPageSize = 8;
const profilePageSize = 5;
const marketPageSize = 12;
const publicUIStateStorageKey = "brights_public_ui_state";
const publicSessionStorageKey = "brights_public_session";
const inviteCodeSessionStorageKey = "brights_invite_code";

type AuthMode = "login" | "register";
type PublicView = "home" | "profile" | "mcp" | "market";
type ProfileWorkspaceTab =
  | "overview"
  | "memberships"
  | "orders"
  | "invite"
  | "knowledge-base"
  | "api-tools"
  | "xiaomi"
  | "plans"
  | "insights";
type NoticeDialogTone = "info" | "success" | "error";

type NoticeDialogState = {
  tone: NoticeDialogTone;
  title: string;
  message: string;
  actionLabel?: string;
  actionHref?: string;
};

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

const emptyAPIConfigEditor: SaveAPIConfigInput & { id: number } = {
  id: 0,
  name: "",
  tool_name: "",
  url: "",
  method: "GET",
  category: "custom",
  category_color: "",
  icon: "",
  description: "",
  headers: "{}",
  body: "",
  parameters: "[]",
  is_active: true,
  is_public: false,
  allow_admin_publish: false,
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
    inviteCode: "",
    password: "",
    confirmPassword: "",
  });
  const [authCaptcha, setAuthCaptcha] = useState<CaptchaChallenge | null>(null);
  const [authCaptchaAnswer, setAuthCaptchaAnswer] = useState("");
  const [authBusy, setAuthBusy] = useState("");
  const [authError, setAuthError] = useState("");
  const [authNotice, setAuthNotice] = useState("");
  const [authCaptchaLoading, setAuthCaptchaLoading] = useState(false);
  const [inviteCodeLocked, setInviteCodeLocked] = useState(false);

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
  const [noticeDialog, setNoticeDialog] = useState<NoticeDialogState | null>(null);
  const [inviteSummary, setInviteSummary] = useState<InviteSummary | null>(null);
  const [paymentOrders, setPaymentOrders] = useState<PagedPaymentOrders | null>(null);
  const [membershipHistory, setMembershipHistory] = useState<PagedSubscriptions | null>(null);
  const [knowledgeBaseDocuments, setKnowledgeBaseDocuments] = useState<PagedKnowledgeBaseDocuments | null>(null);
  const [knowledgeBaseDocumentPreview, setKnowledgeBaseDocumentPreview] = useState<KnowledgeBaseDocument | null>(null);
  const [knowledgeBaseDocumentChunks, setKnowledgeBaseDocumentChunks] = useState<KnowledgeBaseChunk[]>([]);
  const [knowledgeBaseDocumentPreviewLoading, setKnowledgeBaseDocumentPreviewLoading] = useState(false);
  const [profileBusyAction, setProfileBusyAction] = useState("");
  const [profileError, setProfileError] = useState("");
  const [profileNotice, setProfileNotice] = useState("");
  const [profileReloadKey, setProfileReloadKey] = useState(0);
  const [orderHistoryPage, setOrderHistoryPage] = useState(1);
  const [membershipHistoryPage, setMembershipHistoryPage] = useState(1);
  const [knowledgeBasePage, setKnowledgeBasePage] = useState(1);
  const [knowledgeBaseQuery, setKnowledgeBaseQuery] = useState("");
  const [knowledgeBaseForm, setKnowledgeBaseForm] = useState({
    file: null as File | null,
    fileName: "",
    title: "",
  });
  const [apiConfigs, setAPIConfigs] = useState<PagedAPIConfigs | null>(null);
  const [apiConfigPage, setAPIConfigPage] = useState(1);
  const [apiConfigQuery, setAPIConfigQuery] = useState("");
  const [apiConfigEditor, setAPIConfigEditor] = useState(emptyAPIConfigEditor);
  const [apiConfigTestArguments, setAPIConfigTestArguments] = useState("{}");
  const [apiConfigTestResult, setAPIConfigTestResult] = useState<APIConfigTestResult | null>(null);
  const [xiaomiConfig, setXiaomiConfig] = useState<XiaomiConfig | null>(null);
  const [xiaomiHomes, setXiaomiHomes] = useState<XiaomiHome[]>([]);
  const [xiaomiDevices, setXiaomiDevices] = useState<XiaomiDeviceListResult | null>(null);
  const [xiaomiSearchQuery, setXiaomiSearchQuery] = useState("");
  const [xiaomiQRSession, setXiaomiQRSession] = useState<XiaomiQRLoginResult | null>(null);
  const [xiaomiQRStatus, setXiaomiQRStatus] = useState("");
  const [marketResult, setMarketResult] = useState<PagedMCPMarketTools | null>(null);
  const [marketLoading, setMarketLoading] = useState(false);
  const [marketError, setMarketError] = useState("");
  const [marketPage, setMarketPage] = useState(1);
  const [marketQuery, setMarketQuery] = useState("");
  const [marketCategory, setMarketCategory] = useState("");
  const [learningSummary, setLearningSummary] = useState<LearningSummary | null>(null);
  const [learningProgress, setLearningProgress] = useState<PagedLearningWordProgress | null>(null);
  const [learningPage, setLearningPage] = useState(1);
  const [learningQuery, setLearningQuery] = useState("");
  const [learningLevelFilter, setLearningLevelFilter] = useState("");
  const [learningDifficultyFilter, setLearningDifficultyFilter] = useState("");

  const deferredQuery = useDeferredValue(query);
  const deferredKnowledgeBaseQuery = useDeferredValue(knowledgeBaseQuery);
  const deferredAPIConfigQuery = useDeferredValue(apiConfigQuery);
  const deferredMarketQuery = useDeferredValue(marketQuery);
  const deferredLearningQuery = useDeferredValue(learningQuery);
  const activeCheckoutOrder = checkoutStatus?.order ?? checkoutOrder;
  const currentSettings = siteSettings ?? fallbackSiteSettings;
  const learnerName = currentUser?.display_name || currentUser?.username || "";
  const currentMembership = currentUser?.membership ?? null;
  const hasActiveMembership = currentMembership?.status === "active";
  const membershipExpiryText = formatMembershipExpiry(currentMembership);
  const currentInviteCode = (inviteSummary?.invite_code || currentUser?.invite_code || "").trim();
  const inviteRegistrationLink = buildInviteRegistrationLink(currentInviteCode);
  const membershipBadgeText = currentMembership
    ? hasActiveMembership
      ? "\u4f1a\u5458\u5df2\u5f00\u901a"
      : subscriptionStatusLabel(currentMembership.status)
    : "\u666e\u901a\u7528\u6237";
  const learnerAccessToken = session?.access_token ?? "";
  const speechSupported = canUseBrowserSpeech();
  const activeView: PublicView = resolvePublicView(currentHash);
  const currentProfileTab: ProfileWorkspaceTab = resolveProfileWorkspaceTab(currentHash);
  const classifications = classificationResult?.items ?? [];
  const classificationTotal = classificationResult?.total ?? 0;
  const marketTools = marketResult?.items ?? [];
  const marketCategories = marketResult?.categories ?? [];
  const learningLevelCounts = learningSummary?.level_counts ?? [];
  const learningDifficultyCounts = learningSummary?.difficulty_counts ?? [];
  const learningCurvePoints = learningSummary?.curve_points ?? [];
  const learningCountByKey = (items: Array<{ key: string; count: number }>, key: string) =>
    items.find((item) => item.key === key)?.count ?? 0;
  const speakTimerRef = useRef<number | null>(null);
  const filteredMarketTools = marketTools;
  const speechTokenRef = useRef(0);
  const accountMenuRef = useRef<HTMLDivElement | null>(null);
  const checkoutNoticeKeyRef = useRef("");
  const profileTabItems: Array<{ key: ProfileWorkspaceTab; label: string; helper: string; href: string; count?: number }> = [
    { key: "overview", label: "账号总览", helper: currentUser ? "资料与权益" : "登录与注册", href: "#profile" },
    { key: "memberships", label: "会员记录", helper: "状态与有效期", href: "#profile-memberships", count: membershipHistory?.total ?? 0 },
    { key: "orders", label: "购买记录", helper: "订单与支付", href: "#profile-orders", count: paymentOrders?.total ?? 0 },
    { key: "invite", label: "邀请好友", helper: "邀请码与转化", href: "#profile-invite", count: Number(inviteSummary?.invited_count ?? 0) },
    { key: "knowledge-base", label: "我的知识库", helper: "上传与文档", href: "#profile-knowledge-base", count: knowledgeBaseDocuments?.total ?? 0 },
    { key: "api-tools", label: "API 工具", helper: "个人接口与 MCP", href: "#profile-api-tools", count: apiConfigs?.total ?? 0 },
    { key: "xiaomi", label: "米家智能", helper: "小米设备与控制", href: "#profile-xiaomi", count: xiaomiDevices?.total ?? xiaomiConfig?.device_count ?? 0 },
    { key: "plans", label: "会员方案", helper: "购买与续费", href: "#plans", count: plans.length },
    { key: "insights", label: "学习概况", helper: "词库与建议", href: "#profile-insights" },
  ];

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
    if (currentUser) {
      return;
    }

    const inviteCode = resolveInitialInviteCode();
    if (!inviteCode) {
      return;
    }

    setInviteCodeLocked(true);
    setAuthMode("register");
    setAuthForm((current) => ({ ...current, inviteCode }));

    if (resolvePublicView(getCurrentHash()) !== "profile") {
      window.location.hash = "#profile";
    }
  }, []);

  useEffect(() => {
    const targetId = currentHash.replace(/^#/, "");
    window.requestAnimationFrame(() => {
      if (!targetId || targetId === "home" || targetId === "profile" || targetId === "account" || targetId === "mcp" || targetId === "mcp-market") {
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
      .learnerMe(session.access_token, subjectKey)
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
  }, [session, subjectKey]);

  useEffect(() => {
    if (!session?.access_token) {
      setInviteSummary(null);
      setPaymentOrders(null);
      setMembershipHistory(null);
      setKnowledgeBaseDocuments(null);
      setKnowledgeBaseDocumentPreview(null);
      setKnowledgeBaseDocumentChunks([]);
      return;
    }

    let active = true;
    Promise.all([
      api.learnerInviteSummary(session.access_token),
      api.learnerPaymentOrders(session.access_token, {
        page: orderHistoryPage,
        pageSize: profilePageSize,
        subject: subjectKey,
      }),
      api.learnerMemberships(session.access_token, {
        page: membershipHistoryPage,
        pageSize: profilePageSize,
        subject: subjectKey,
      }),
      api.learnerKnowledgeBaseDocuments(session.access_token, {
        page: knowledgeBasePage,
        pageSize: profilePageSize,
        query: deferredKnowledgeBaseQuery,
        subjectKey,
      }),
    ])
      .then(([inviteData, orderData, membershipData, knowledgeBaseData]) => {
        if (!active) {
          return;
        }
        setInviteSummary(inviteData);
        setPaymentOrders(orderData);
        setMembershipHistory(membershipData);
        setKnowledgeBaseDocuments(knowledgeBaseData);
      })
      .catch((err: Error) => {
        if (!active) {
          return;
        }
        setProfileError(err.message);
      });

    return () => {
      active = false;
    };
  }, [
    deferredKnowledgeBaseQuery,
    knowledgeBasePage,
    membershipHistoryPage,
    orderHistoryPage,
    profileReloadKey,
    session?.access_token,
    subjectKey,
  ]);

  useEffect(() => {
    if (!session?.access_token) {
      setAPIConfigs(null);
      setAPIConfigTestResult(null);
      return;
    }

    let active = true;
    api
      .learnerAPIConfigs(session.access_token, {
        page: apiConfigPage,
        pageSize: profilePageSize,
        query: deferredAPIConfigQuery,
      })
      .then((result) => {
        if (active) {
          setAPIConfigs(result);
        }
      })
      .catch((err: Error) => {
        if (active) {
          setProfileError(err.message);
        }
      });

    return () => {
      active = false;
    };
  }, [apiConfigPage, deferredAPIConfigQuery, profileReloadKey, session?.access_token]);

  useEffect(() => {
    if (!session?.access_token) {
      setXiaomiConfig(null);
      setXiaomiHomes([]);
      setXiaomiDevices(null);
      setXiaomiQRSession(null);
      setXiaomiQRStatus("");
      return;
    }
    if (currentProfileTab !== "xiaomi") {
      return;
    }
    void loadXiaomiWorkspace(false);
  }, [currentProfileTab, profileReloadKey, session?.access_token]);

  useEffect(() => {
    setMarketPage(1);
  }, [deferredMarketQuery, marketCategory, subjectKey]);

  useEffect(() => {
    if (activeView !== "market") {
      return;
    }

    let active = true;
    setMarketLoading(true);
    setMarketError("");
    api
      .getMCPToolMarket({
        page: marketPage,
        pageSize: marketPageSize,
        query: deferredMarketQuery,
        category: marketCategory || undefined,
        subjectKey,
        token: learnerAccessToken || undefined,
      })
      .then((result) => {
        if (!active) {
          return;
        }
        setMarketResult(result);
      })
      .catch((err: Error) => {
        if (active) {
          setMarketError(err.message);
        }
      })
      .finally(() => {
        if (active) {
          setMarketLoading(false);
        }
      });

    return () => {
      active = false;
    };
  }, [activeView, deferredMarketQuery, learnerAccessToken, marketCategory, marketPage, profileReloadKey, subjectKey]);

  useEffect(() => {
    if (!session?.access_token) {
      setLearningSummary(null);
      setLearningProgress(null);
      return;
    }
    if (currentProfileTab !== "insights") {
      return;
    }

    let active = true;
    Promise.all([
      api.learnerLearningSummary(session.access_token, subjectKey),
      api.learnerLearningProgress(session.access_token, {
        page: learningPage,
        pageSize: profilePageSize,
        query: deferredLearningQuery,
        subjectKey,
        level: learningLevelFilter || undefined,
        difficulty: learningDifficultyFilter || undefined,
      }),
    ])
      .then(([summaryData, progressData]) => {
        if (!active) {
          return;
        }
        setLearningSummary(summaryData);
        setLearningProgress(progressData);
      })
      .catch((err: Error) => {
        if (active) {
          setProfileError(err.message);
        }
      });

    return () => {
      active = false;
    };
  }, [
    currentProfileTab,
    deferredLearningQuery,
    learningDifficultyFilter,
    learningLevelFilter,
    learningPage,
    profileReloadKey,
    session?.access_token,
    subjectKey,
  ]);

  useEffect(() => {
    if (!session?.access_token || !xiaomiQRSession?.session_id) {
      return;
    }

    let cancelled = false;
    const timer = window.setInterval(() => {
      api
        .learnerCheckXiaomiQRLogin(session.access_token, xiaomiQRSession.session_id)
        .then((result) => {
          if (cancelled) {
            return;
          }
          const nextStatus = result.status || (result.success ? "success" : "waiting");
          setXiaomiQRStatus(result.message || nextStatus);
          if (nextStatus === "success" || result.success) {
            setXiaomiQRSession(null);
            setProfileNotice(
              result.devices_synced
                ? `小米账号扫码登录成功，已同步 ${result.device_count ?? 0} 台设备。`
                : "小米账号扫码登录成功，正在载入设备列表。",
            );
            void loadXiaomiWorkspace(false);
          }
        })
        .catch((err: Error) => {
          if (cancelled) {
            return;
          }
          setXiaomiQRStatus(err.message);
        });
    }, 4000);

    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [session?.access_token, xiaomiQRSession?.session_id]);

  useEffect(() => {
    setOrderHistoryPage(1);
    setMembershipHistoryPage(1);
    setKnowledgeBasePage(1);
    setLearningPage(1);
  }, [subjectKey]);

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
        token: learnerAccessToken || undefined,
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
  }, [classificationPage, learnerAccessToken, subjectKey]);

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
        token: learnerAccessToken || undefined,
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
  }, [classification, deferredQuery, learnerAccessToken, page, subjectKey]);

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

  useEffect(() => {
    if (!authNotice) {
      return;
    }
    openNoticeDialog({
      tone: "success",
      title: "操作已完成",
      message: authNotice,
    });
    setAuthNotice("");
  }, [authNotice]);

  useEffect(() => {
    if (!profileNotice) {
      return;
    }
    openNoticeDialog({
      tone: "success",
      title: "操作已完成",
      message: profileNotice,
    });
    setProfileNotice("");
  }, [profileNotice]);

  useEffect(() => {
    if (!authError) {
      return;
    }
    openNoticeDialog({
      tone: "error",
      title: "请检查后再试",
      message: authError,
    });
    setAuthError("");
  }, [authError]);

  useEffect(() => {
    if (!profileError) {
      return;
    }
    openNoticeDialog({
      tone: "error",
      title: "个人中心操作未完成",
      message: profileError,
    });
    setProfileError("");
  }, [profileError]);

  useEffect(() => {
    if (!classificationError) {
      return;
    }
    openNoticeDialog({
      tone: "error",
      title: "分类加载失败",
      message: classificationError,
    });
    setClassificationError("");
  }, [classificationError]);

  useEffect(() => {
    if (!error) {
      return;
    }
    openNoticeDialog({
      tone: "error",
      title: "内容加载失败",
      message: error,
    });
    setError("");
  }, [error]);

  useEffect(() => {
    if (!checkoutError) {
      return;
    }
    openNoticeDialog({
      tone: "error",
      title: "当前操作未完成",
      message: checkoutError,
      actionLabel: currentUser ? "查看会员方案" : "先登录学习账号",
      actionHref: currentUser ? "#plans" : "#profile",
    });
    setCheckoutError("");
  }, [checkoutError, currentUser]);

  useEffect(() => {
    const order = checkoutStatus?.order;
    if (!order || order.status !== "success") {
      return;
    }

    const noticeKey = `${order.order_no}:${order.status}`;
    if (checkoutNoticeKeyRef.current === noticeKey) {
      return;
    }
    checkoutNoticeKeyRef.current = noticeKey;

    const membershipMessage = checkoutStatus?.subscription
      ? checkoutStatus.subscription.current_period_end
        ? `会员已生效，有效期至 ${formatDateTime(checkoutStatus.subscription.current_period_end)}。`
        : "会员已生效，当前为长期有效。"
      : "支付已成功，我们正在同步你的会员权益。";

    setProfileReloadKey((current) => current + 1);

    openNoticeDialog({
      tone: "success",
      title: "支付成功",
      message: membershipMessage,
      actionLabel: "回到词库继续学习",
      actionHref: "#catalog",
    });
  }, [checkoutStatus]);

  useEffect(() => {
    const order = checkoutStatus?.order;
    if (!session?.access_token || !currentUser?.username || !order || order.status !== "success") {
      return;
    }
    if (order.customer_ref !== currentUser.username || order.subject_key !== subjectKey) {
      return;
    }

    let active = true;
    api
      .learnerMe(session.access_token, subjectKey)
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
      .catch(() => {
        // Keep the checkout success state visible even if the profile refresh is delayed.
      });

    return () => {
      active = false;
    };
  }, [checkoutStatus, currentUser?.username, session, subjectKey]);

  function openNoticeDialog(nextNotice: NoticeDialogState) {
    setNoticeDialog(nextNotice);
  }

  function closeNoticeDialog() {
    setNoticeDialog(null);
  }

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
    checkoutNoticeKeyRef.current = "";
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
    checkoutNoticeKeyRef.current = "";
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
          invite_code: authForm.inviteCode.trim() || undefined,
          captcha_id: authCaptcha.captcha_id,
          captcha_answer: authCaptchaAnswer,
        });
        persistLearnerSession(nextSession);
        setCheckoutCustomerRef(nextSession.user.username);
        window.sessionStorage.removeItem(inviteCodeSessionStorageKey);
        clearInviteQueryParam();
        setInviteCodeLocked(false);
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
        inviteCode: "",
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
      setKnowledgeBaseForm({
        file: null,
        fileName: "",
        title: "",
      });
      void refreshAuthCaptcha(authMode);
    }
  }

  async function handleCopyInviteCode() {
    if (!currentInviteCode || !window.navigator.clipboard) {
      return;
    }
    const code = currentInviteCode;
    try {
      await window.navigator.clipboard.writeText(currentInviteCode);
      setProfileNotice(`邀请码 ${code} 已复制。`);
    } catch {
      setProfileError("复制邀请码失败，请手动复制。");
    }
  }

  async function handleCopyInviteLink() {
    if (!inviteRegistrationLink || !window.navigator.clipboard) {
      return;
    }
    try {
      await window.navigator.clipboard.writeText(inviteRegistrationLink);
      setProfileNotice("注册链接已复制。");
    } catch {
      setProfileError("复制注册链接失败，请手动复制。");
    }
  }

  async function handleSubmitKnowledgeBase(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!session?.access_token) {
      return;
    }
    if (!knowledgeBaseForm.file) {
      setProfileError("请先选择要上传的知识库文件。");
      return;
    }

    setProfileBusyAction("knowledge-base-upload");
    try {
      const result = await api.learnerImportKnowledgeBase(session.access_token, {
        file: knowledgeBaseForm.file,
        subject_key: subjectKey,
        title: knowledgeBaseForm.title.trim(),
      });
      setKnowledgeBaseForm({
        file: null,
        fileName: "",
        title: "",
      });
      setKnowledgeBasePage(1);
      setProfileReloadKey((current) => current + 1);
      setProfileNotice(`知识库文档《${result.document.title || result.document.source_file_name}》已上传。`);
    } catch (err) {
      setProfileError((err as Error).message);
    } finally {
      setProfileBusyAction("");
    }
  }

  async function handleToggleKnowledgeBaseDocument(item: KnowledgeBaseDocument) {
    if (!session?.access_token) {
      return;
    }

    const nextStatus = item.status === "active" ? "disabled" : "active";
    setProfileBusyAction(`knowledge-base-status-${item.id}`);
    try {
      await api.learnerUpdateKnowledgeBaseDocumentStatus(session.access_token, item.id, nextStatus);
      setProfileReloadKey((current) => current + 1);
      setProfileNotice(nextStatus === "active" ? "文档已启用。" : "文档已停用。");
    } catch (err) {
      setProfileError((err as Error).message);
    } finally {
      setProfileBusyAction("");
    }
  }

  async function handleDeleteKnowledgeBaseDocument(item: KnowledgeBaseDocument) {
    if (!session?.access_token) {
      return;
    }
    const confirmed = window.confirm(`确认删除知识库文档“${item.title || item.source_file_name}”吗？删除后对应检索片段也会移除。`);
    if (!confirmed) {
      return;
    }

    setProfileBusyAction(`knowledge-base-delete-${item.id}`);
    try {
      await api.learnerDeleteKnowledgeBaseDocument(session.access_token, item.id);
      setProfileReloadKey((current) => current + 1);
      setProfileNotice("知识库文档已删除。");
    } catch (err) {
      setProfileError((err as Error).message);
    } finally {
      setProfileBusyAction("");
    }
  }

  async function handleViewKnowledgeBaseDocument(item: KnowledgeBaseDocument) {
    if (!session?.access_token) {
      return;
    }

    setKnowledgeBaseDocumentPreview(item);
    setKnowledgeBaseDocumentPreviewLoading(true);
    setKnowledgeBaseDocumentChunks([]);
    try {
      const result = await api.learnerKnowledgeBaseDocumentChunks(session.access_token, item.id, {
        page: 1,
        pageSize: 200,
      });
      setKnowledgeBaseDocumentChunks(result.items);
    } catch (err) {
      setProfileError((err as Error).message);
      setKnowledgeBaseDocumentPreview(null);
    } finally {
      setKnowledgeBaseDocumentPreviewLoading(false);
    }
  }

  function closeKnowledgeBaseDocumentPreview() {
    setKnowledgeBaseDocumentPreview(null);
    setKnowledgeBaseDocumentChunks([]);
    setKnowledgeBaseDocumentPreviewLoading(false);
  }

  async function handleSaveLearningProgress(wordID: number, level: string, difficulty: string) {
    if (!session?.access_token || wordID <= 0) {
      return;
    }

    setProfileBusyAction(`learning-save-${wordID}`);
    try {
      await api.learnerSaveLearningProgress(session.access_token, {
        word_id: wordID,
        subject_key: subjectKey,
        level,
        difficulty,
      });
      setProfileReloadKey((current) => current + 1);
      setProfileNotice("学习级别与难度已更新。");
    } catch (err) {
      setProfileError((err as Error).message);
    } finally {
      setProfileBusyAction("");
    }
  }

  async function handleReviewLearningWord(
    word: { id: number },
    remembered: boolean,
    options?: { level?: string; difficulty?: string },
  ) {
    if (!session?.access_token || word.id <= 0) {
      return;
    }

    setProfileBusyAction(`learning-review-${word.id}`);
    try {
      await api.learnerReviewLearningWord(session.access_token, {
        word_id: word.id,
        subject_key: subjectKey,
        remembered,
        level: options?.level,
        difficulty: options?.difficulty,
      });
      setProfileReloadKey((current) => current + 1);
      setProfileNotice(remembered ? "已记录为记住，系统已生成下一次复习时间。" : "已记录为待复习，稍后会更早提醒。");
    } catch (err) {
      setProfileError((err as Error).message);
    } finally {
      setProfileBusyAction("");
    }
  }

  async function loadXiaomiWorkspace(refreshDevices: boolean) {
    if (!session?.access_token) {
      return;
    }
    try {
      const [configResult, deviceResult] = await Promise.all([
        api.learnerXiaomiConfig(session.access_token),
        api.learnerXiaomiDevices(session.access_token, refreshDevices),
      ]);
      setXiaomiConfig(configResult);
      setXiaomiDevices(deviceResult);
      setXiaomiHomes(buildXiaomiHomesFromDevices(deviceResult.devices));
    } catch (err) {
      setProfileError((err as Error).message);
    }
  }

  function buildXiaomiHomesFromDevices(devices: XiaomiDevice[]): XiaomiHome[] {
    const homes = new Map<string, XiaomiHome>();
    for (const item of devices) {
      const homeID = item.home_id?.trim() || "";
      const homeName = item.home_name?.trim() || "";
      if (!homeID && !homeName) {
        continue;
      }
      const key = homeID || homeName;
      if (homes.has(key)) {
        continue;
      }
      homes.set(key, {
        id: homeID || key,
        name: homeName || homeID || "未命名家庭",
      });
    }
    return Array.from(homes.values()).sort((left, right) => left.name.localeCompare(right.name, "zh-CN"));
  }

  function resetAPIConfigEditor() {
    setAPIConfigEditor(emptyAPIConfigEditor);
    setAPIConfigTestArguments("{}");
    setAPIConfigTestResult(null);
  }

  function toAPIConfigPayload(item: APIConfig | (SaveAPIConfigInput & { id?: number })): SaveAPIConfigInput {
    return {
      name: item.name,
      tool_name: item.tool_name,
      url: item.url,
      method: item.method,
      category: item.category,
      category_color: item.category_color,
      icon: item.icon,
      description: item.description,
      headers: item.headers,
      body: item.body,
      parameters: item.parameters,
      is_active: item.is_active,
      is_public: item.is_public,
      allow_admin_publish: item.allow_admin_publish,
    };
  }

  function startEditAPIConfig(item: APIConfig) {
    setAPIConfigEditor({
      id: item.id,
      ...toAPIConfigPayload(item),
    });
    setAPIConfigTestResult(null);
  }

  async function handleSaveAPIConfig(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!session?.access_token) {
      return;
    }

    setProfileBusyAction("api-config-save");
    setAPIConfigTestResult(null);
    try {
      const payload = toAPIConfigPayload(apiConfigEditor);
      if (apiConfigEditor.id > 0) {
        await api.learnerUpdateAPIConfig(session.access_token, apiConfigEditor.id, payload);
        setProfileNotice("API 工具已更新。");
      } else {
        await api.learnerCreateAPIConfig(session.access_token, payload);
        setProfileNotice("API 工具已创建。");
      }
      resetAPIConfigEditor();
      setAPIConfigPage(1);
      setProfileReloadKey((current) => current + 1);
    } catch (err) {
      setProfileError((err as Error).message);
    } finally {
      setProfileBusyAction("");
    }
  }

  async function handleToggleAPIConfig(item: APIConfig) {
    if (!session?.access_token) {
      return;
    }
    setProfileBusyAction(`api-config-toggle-${item.id}`);
    try {
      await api.learnerUpdateAPIConfig(session.access_token, item.id, {
        ...toAPIConfigPayload(item),
        is_active: !item.is_active,
      });
      setProfileReloadKey((current) => current + 1);
      setProfileNotice(!item.is_active ? "API 工具已启用。" : "API 工具已停用。");
    } catch (err) {
      setProfileError((err as Error).message);
    } finally {
      setProfileBusyAction("");
    }
  }

  async function handleDeleteAPIConfig(item: APIConfig) {
    if (!session?.access_token) {
      return;
    }
    const confirmed = window.confirm(`确认删除 API 工具“${item.name || item.resolved_tool_name}”吗？`);
    if (!confirmed) {
      return;
    }
    setProfileBusyAction(`api-config-delete-${item.id}`);
    try {
      await api.learnerDeleteAPIConfig(session.access_token, item.id);
      if (apiConfigEditor.id === item.id) {
        resetAPIConfigEditor();
      }
      setProfileReloadKey((current) => current + 1);
      setProfileNotice("API 工具已删除。");
    } catch (err) {
      setProfileError((err as Error).message);
    } finally {
      setProfileBusyAction("");
    }
  }

  async function handleTestAPIConfig(item: APIConfig) {
    if (!session?.access_token) {
      return;
    }

    let argumentsPayload: Record<string, unknown> = {};
    try {
      argumentsPayload = apiConfigTestArguments.trim() ? (JSON.parse(apiConfigTestArguments) as Record<string, unknown>) : {};
    } catch {
      setProfileError("测试参数必须是有效 JSON。");
      return;
    }

    setProfileBusyAction(`api-config-test-${item.id}`);
    try {
      const result = await api.learnerTestAPIConfig(
        session.access_token,
        item.id,
        { arguments: argumentsPayload },
        subjectKey,
      );
      setAPIConfigTestResult(result);
      setProfileNotice("API 工具测试请求已完成。");
    } catch (err) {
      setProfileError((err as Error).message);
    } finally {
      setProfileBusyAction("");
    }
  }

  async function handleSaveXiaomiConfig(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!session?.access_token || !xiaomiConfig) {
      return;
    }
    setProfileBusyAction("xiaomi-save");
    try {
      const saved = await api.learnerSaveXiaomiConfig(session.access_token, {
        username: xiaomiConfig.username || "",
        xiaomi_user_id: xiaomiConfig.xiaomi_user_id || "",
        server: xiaomiConfig.server || "cn",
        ssecurity: "",
        service_token: "",
        is_active: xiaomiConfig.is_active,
      });
      setXiaomiConfig(saved);
      setProfileNotice("小米配置已保存。");
    } catch (err) {
      setProfileError((err as Error).message);
    } finally {
      setProfileBusyAction("");
    }
  }

  async function handleStartXiaomiQRLogin() {
    if (!session?.access_token) {
      return;
    }
    setProfileBusyAction("xiaomi-qr-login");
    try {
      const result = await api.learnerStartXiaomiQRLogin(session.access_token, xiaomiConfig?.server || "cn");
      setXiaomiQRSession(result);
      setXiaomiQRStatus(result.message || "请使用米家 App 扫码。");
      setProfileNotice("二维码已生成，请使用米家 App 扫码登录。");
    } catch (err) {
      setProfileError((err as Error).message);
    } finally {
      setProfileBusyAction("");
    }
  }

  async function handleClearXiaomiTokens() {
    if (!session?.access_token) {
      return;
    }
    const confirmed = window.confirm("确认清空当前小米账号令牌吗？清空后需要重新扫码或重新填写凭证。");
    if (!confirmed) {
      return;
    }
    setProfileBusyAction("xiaomi-clear");
    try {
      await api.learnerClearXiaomiTokens(session.access_token);
      setXiaomiQRSession(null);
      setXiaomiQRStatus("");
      await loadXiaomiWorkspace(false);
      setProfileNotice("小米账号令牌已清空。");
    } catch (err) {
      setProfileError((err as Error).message);
    } finally {
      setProfileBusyAction("");
    }
  }

  async function handleRefreshXiaomiDevices() {
    if (!session?.access_token) {
      return;
    }
    setProfileBusyAction("xiaomi-refresh");
    try {
      const result = await api.learnerRefreshXiaomiDevices(session.access_token);
      setXiaomiDevices((current) =>
        current
          ? {
              ...current,
              devices: result.devices,
              total: result.device_count,
              refreshed: true,
              account: {
                ...current.account,
                device_count: result.device_count,
              },
            }
          : null,
      );
      setXiaomiHomes(buildXiaomiHomesFromDevices(result.devices));
      await loadXiaomiWorkspace(false);
      setProfileNotice(`设备列表已刷新，共 ${result.device_count} 台。`);
    } catch (err) {
      setProfileError((err as Error).message);
    } finally {
      setProfileBusyAction("");
    }
  }

  async function handleLoadXiaomiHomes() {
    if (!session?.access_token) {
      return;
    }
    setProfileBusyAction("xiaomi-homes");
    try {
      const items = await api.learnerXiaomiHomes(session.access_token);
      setXiaomiHomes(items);
      setProfileNotice(`已加载 ${items.length} 个家庭。`);
    } catch (err) {
      setProfileError((err as Error).message);
    } finally {
      setProfileBusyAction("");
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

  const findClassificationStat = (name: string) => {
    const trimmedName = name.trim();
    if (!trimmedName) {
      return null;
    }
    return classifications.find((item) => item.name === trimmedName) ?? null;
  };

  const handleSelectClassification = (nextClassification: string) => {
    setClassification(nextClassification);
    setPage(1);

    const targetClassification = findClassificationStat(nextClassification);
    if (!targetClassification) {
      return;
    }
    if (!targetClassification.requires_membership || targetClassification.accessible_count > 0) {
      return;
    }

    openNoticeDialog({
      tone: "info",
      title: "这个场景需要会员才能学习",
      message: `「${targetClassification.name}」当前属于会员专享内容，开通后可查看 ${formatCount(targetClassification.count)} 个场景词。`,
      actionLabel: currentUser ? "去看会员方案" : "先登录学习账号",
      actionHref: currentUser ? "#plans" : "#profile",
    });
  };

  const selectedSubjectLabel = formatSubjectLabel(subjectKey);
  const profileOverviewSnapshotItems: Array<{ label: string; value: string; note: string }> = currentUser
    ? [
        {
          label: "当前账号",
          value: learnerName || currentUser.username,
          note: "学习记录与购买信息都会绑定到这个账号",
        },
        {
          label: "会员状态",
          value: membershipBadgeText,
          note: membershipExpiryText || "去会员方案页签开通或续费",
        },
        {
          label: "当前科目",
          value: selectedSubjectLabel,
          note: "当前浏览与学习入口都会跟着这个学科切换",
        },
        {
          label: "可选方案",
          value: plans.length > 0 ? `${plans.length} 种方案` : "暂无方案",
          note: "支持按需购买、续费和升级",
        },
      ]
    : [
        {
          label: "当前状态",
          value: "未登录",
          note: "先登录后再保存学习记录和购买信息",
        },
        {
          label: "学习账号",
          value: "待创建",
          note: "注册后可跨设备继续学习",
        },
        {
          label: "会员权益",
          value: "未开通",
          note: "开通后会自动绑定到你的学习账号",
        },
        {
          label: "当前科目",
          value: selectedSubjectLabel,
          note: "先浏览词库，再决定是否开通会员",
        },
      ];
  const classificationOnCurrentPage = classification === "" || classifications.some((item) => item.name === classification);
  const classificationOptions: ClassificationStat[] =
    classification !== "" && !classifications.some((item) => item.name === classification)
      ? [
          {
            name: classification,
            count: 0,
            free_count: 0,
            vip_count: 0,
            accessible_count: 0,
            requires_membership: false,
            has_member_content: false,
          },
          ...classifications,
        ]
      : classifications;
  const selectedClassificationStat = classification ? findClassificationStat(classification) : null;
  const selectedClassificationLocked =
    !!selectedClassificationStat &&
    selectedClassificationStat.requires_membership &&
    selectedClassificationStat.accessible_count === 0;
  const visibleXiaomiDevices = (xiaomiDevices?.devices ?? []).filter((item) => {
    const queryText = xiaomiSearchQuery.trim().toLowerCase();
    if (!queryText) {
      return true;
    }
    return [item.name, item.model, item.did, item.home_name, item.room_name]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(queryText));
  });
  const apiConfigTestPreview = apiConfigTestResult
    ? JSON.stringify(apiConfigTestResult.body ?? apiConfigTestResult.raw_body ?? apiConfigTestResult, null, 2)
    : "";
  // These workspaces are mounted conditionally, so keep their state and helpers
  // referenced under strict TS unused-local checks.
  void [
    apiConfigs,
    setAPIConfigs,
    apiConfigPage,
    setAPIConfigPage,
    apiConfigQuery,
    setAPIConfigQuery,
    apiConfigEditor,
    setAPIConfigEditor,
    apiConfigTestArguments,
    setAPIConfigTestArguments,
    apiConfigTestResult,
    setAPIConfigTestResult,
    knowledgeBaseDocumentPreview,
    knowledgeBaseDocumentChunks,
    knowledgeBaseDocumentPreviewLoading,
    setKnowledgeBaseDocumentPreviewLoading,
    handleViewKnowledgeBaseDocument,
    closeKnowledgeBaseDocumentPreview,
    handleSaveLearningProgress,
    handleReviewLearningWord,
    xiaomiConfig,
    setXiaomiConfig,
    xiaomiHomes,
    setXiaomiHomes,
    xiaomiDevices,
    setXiaomiDevices,
    xiaomiSearchQuery,
    setXiaomiSearchQuery,
    xiaomiQRSession,
    setXiaomiQRSession,
    xiaomiQRStatus,
    setXiaomiQRStatus,
    marketResult,
    marketLoading,
    marketError,
    marketPage,
    setMarketPage,
    setMarketQuery,
    setMarketCategory,
    marketCategories,
    marketTools,
    learningSummary,
    learningProgress,
    setLearningQuery,
    setLearningLevelFilter,
    setLearningDifficultyFilter,
    deferredAPIConfigQuery,
    deferredMarketQuery,
    apiConfigs?.items?.[0] as APIConfig | undefined,
  ];

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
        <div className="site-header-actions">
          <a className="secondary-button site-account-link" href="#mcp-market">
            MCP 工具市场
          </a>
          {currentUser ? (
            <div className="site-account-menu" ref={accountMenuRef}>
              <button
                aria-controls="site-account-dropdown"
                aria-expanded={accountMenuOpen}
                aria-haspopup="menu"
                className={
                  accountMenuOpen
                    ? hasActiveMembership
                      ? "site-account-trigger site-account-trigger-open site-account-trigger-member"
                      : "site-account-trigger site-account-trigger-open"
                    : hasActiveMembership
                      ? "site-account-trigger site-account-trigger-member"
                      : "site-account-trigger"
                }
                onClick={() => {
                  setAccountMenuOpen((current) => !current);
                }}
                type="button"
              >
                <div className="site-account-meta">
                  <div className="site-account-name-row">
                    <strong>{learnerName}</strong>
                    <span className={hasActiveMembership ? "site-membership-badge site-membership-badge-active" : "site-membership-badge"}>
                      {membershipBadgeText}
                    </span>
                  </div>
                  <span>{hasActiveMembership && membershipExpiryText ? membershipExpiryText : currentUser.username}</span>
                </div>
                <span aria-hidden="true" className="site-account-caret" />
              </button>

              {accountMenuOpen ? (
                <div className="site-account-dropdown" id="site-account-dropdown" role="menu">
                  <div className="site-account-dropdown-header">
                    <div className="site-account-name-row">
                      <strong>{learnerName}</strong>
                      <span className={hasActiveMembership ? "site-membership-badge site-membership-badge-active" : "site-membership-badge"}>
                        {membershipBadgeText}
                      </span>
                    </div>
                    <span>{hasActiveMembership && membershipExpiryText ? membershipExpiryText : currentUser.username}</span>
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
                      href="#profile-knowledge-base"
                      onClick={() => {
                        setAccountMenuOpen(false);
                      }}
                      role="menuitem"
                    >
                      我的知识库
                    </a>
                    <a
                      className="site-account-dropdown-link"
                      href="#profile-api-tools"
                      onClick={() => {
                        setAccountMenuOpen(false);
                      }}
                      role="menuitem"
                    >
                      API 工具
                    </a>
                    <a
                      className="site-account-dropdown-link"
                      href="#mcp-market"
                      onClick={() => {
                        setAccountMenuOpen(false);
                      }}
                      role="menuitem"
                    >
                      MCP 工具市场
                    </a>
                    <a
                      className="site-account-dropdown-link"
                      href="#profile-xiaomi"
                      onClick={() => {
                        setAccountMenuOpen(false);
                      }}
                      role="menuitem"
                    >
                      米家智能
                    </a>
                    <a
                      className="site-account-dropdown-link"
                      href="#profile-invite"
                      onClick={() => {
                        setAccountMenuOpen(false);
                      }}
                      role="menuitem"
                    >
                      邀请好友
                    </a>
                    <a
                      className="site-account-dropdown-link"
                      href="#profile-orders"
                      onClick={() => {
                        setAccountMenuOpen(false);
                      }}
                      role="menuitem"
                    >
                      购买记录
                    </a>
                    <a
                      className="site-account-dropdown-link"
                      href="#profile-insights"
                      onClick={() => {
                        setAccountMenuOpen(false);
                      }}
                      role="menuitem"
                    >
                      学习进度
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
                    <a
                      className="site-account-dropdown-link"
                      href="#mcp"
                      onClick={() => {
                        setAccountMenuOpen(false);
                      }}
                      role="menuitem"
                    >
                      MCP 连接中心
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
            <nav className="profile-tabbar" aria-label="个人中心导航" id="profile">
              {profileTabItems.map((item) => (
                <a
                  className={currentProfileTab === item.key ? "profile-tab-link profile-tab-link-active" : "profile-tab-link"}
                  href={item.href}
                  key={item.key}
                >
                  <span>{item.label}</span>
                  <small>{item.helper}</small>
                  {item.count !== undefined ? <em>{formatCount(item.count)}</em> : null}
                </a>
              ))}
            </nav>

            <div className="profile-workbench">
              {currentProfileTab === "overview" ? (
                <div className="profile-panel-stack">
                  <div className="profile-snapshot-grid">
                    {profileOverviewSnapshotItems.map((item) => (
                      <article className="profile-snapshot-card" key={item.label}>
                        <span>{item.label}</span>
                        <strong>{item.value}</strong>
                        <small>{item.note}</small>
                      </article>
                    ))}
                  </div>
                  <div className="profile-grid">
                    <section className="content-card profile-card profile-panel-card">
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
                              <dt>会员状态</dt>
                              <dd>
                                <span className={hasActiveMembership ? "site-membership-badge site-membership-badge-active" : "site-membership-badge"}>
                                  {membershipBadgeText}
                                </span>
                              </dd>
                            </div>
                            {membershipExpiryText ? (
                              <div>
                                <dt>会员到期</dt>
                                <dd>{membershipExpiryText.replace("有效期至 ", "")}</dd>
                              </div>
                            ) : null}
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

                    <section className="content-card profile-card profile-panel-card">
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
                          {authMode === "register" ? (
                            <label className="form-field">
                              <span>邀请码</span>
                              <input
                                readOnly={inviteCodeLocked}
                                value={authForm.inviteCode}
                                onChange={(event) => {
                                  setAuthForm((current) => ({ ...current, inviteCode: event.target.value }));
                                }}
                                placeholder="有邀请码可填写，没有可留空"
                              />
                            </label>
                          ) : null}
                          {authMode === "register" ? (
                            <p className="helper-text">
                              {inviteCodeLocked
                                ? "通过邀请链接进入，邀请码已自动带入并锁定。注册成功后会自动建立邀请关系。"
                                : "有邀请码可以填写；如果从邀请链接进入，这里会自动带入。"}
                            </p>
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
                          <div className={hasActiveMembership ? "feedback-banner feedback-success membership-summary-banner" : "feedback-banner membership-summary-banner"}>
                            <div className="membership-summary-banner-head">
                              <span className={hasActiveMembership ? "site-membership-badge site-membership-badge-active" : "site-membership-badge"}>
                                {membershipBadgeText}
                              </span>
                              <strong>{currentMembership ? subscriptionStatusLabel(currentMembership.status) : "未开通"}</strong>
                            </div>
                            <p>
                              {hasActiveMembership
                                ? membershipExpiryText
                                  ? `你当前是按月会员，${membershipExpiryText}。`
                                  : "你当前会员权益已生效，当前为长期有效。"
                                : "当前账号还没有生效中的会员权益，购买后会自动关联到你的学习账号。"}
                            </p>
                          </div>
                          <div className="feedback-banner">
                            购买会员后，后台会直接把会员状态和有效期关联到账号 <strong>{currentUser.username}</strong>，你在前台继续学习时就能一直使用同一个账号。
                          </div>
                          <p className="helper-text">
                            如果你准备开通会员，可以直接从这里的按钮或会员方案页面进入购买；付款完成后，这个账号就是后续所有学习记录的承接入口。
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
                </div>
              ) : null}

              {currentProfileTab === "memberships" ? (
                <section className="content-card profile-card profile-panel-card" id="profile-memberships">
                  <div className="section-header">
                    <div>
                      <p className="section-eyebrow">会员记录</p>
                      <h2>按月会员到期时间和历史权益都在这里</h2>
                    </div>
                  </div>
                  {currentUser ? (
                    <>
                      <div className="feedback-banner">
                        当前学科：<strong>{selectedSubjectLabel}</strong>
                        {membershipExpiryText ? `，${membershipExpiryText}` : "，当前没有生效中的到期时间。"}
                      </div>
                      <div className="table-wrap">
                        <table className="data-table">
                          <thead>
                            <tr>
                              <th>方案</th>
                              <th>状态</th>
                              <th>学科</th>
                              <th>有效期至</th>
                              <th>开通时间</th>
                            </tr>
                          </thead>
                          <tbody>
                            {(membershipHistory?.items ?? []).map((item) => (
                              <tr key={item.id}>
                                <td>{item.plan_key || "-"}</td>
                                <td>
                                  <span className={`pill ${subscriptionStatusClass(item.status)}`}>
                                    {subscriptionStatusLabel(item.status)}
                                  </span>
                                </td>
                                <td>{formatSubjectLabel(item.subject_key)}</td>
                                <td>{item.current_period_end ? formatDateTime(item.current_period_end) : "长期有效"}</td>
                                <td>{item.started_at ? formatDateTime(item.started_at) : "-"}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                      {(membershipHistory?.items ?? []).length === 0 ? (
                        <div className="feedback-banner">当前还没有会员记录，购买后会自动同步到这里。</div>
                      ) : null}
                      <PagerControls
                        onChange={setMembershipHistoryPage}
                        page={membershipHistory?.page ?? membershipHistoryPage}
                        pageSize={membershipHistory?.page_size ?? profilePageSize}
                        total={membershipHistory?.total ?? 0}
                      />
                    </>
                  ) : (
                    <div className="feedback-banner">登录后可查看会员开通状态、按月到期时间和历史权益记录。</div>
                  )}
                </section>
              ) : null}

              {currentProfileTab === "orders" ? (
                <section className="content-card profile-card profile-panel-card" id="profile-orders">
                  <div className="section-header">
                    <div>
                      <p className="section-eyebrow">购买记录</p>
                      <h2>充值、下单和支付结果一目了然</h2>
                    </div>
                  </div>
                  {currentUser ? (
                    <>
                      <div className="table-wrap">
                        <table className="data-table">
                          <thead>
                            <tr>
                              <th>订单号</th>
                              <th>方案</th>
                              <th>金额</th>
                              <th>状态</th>
                              <th>支付时间</th>
                            </tr>
                          </thead>
                          <tbody>
                            {(paymentOrders?.items ?? []).map((item) => (
                              <tr key={item.order_no}>
                                <td>{item.order_no}</td>
                                <td>{item.description || item.plan_key}</td>
                                <td>{formatPrice(item.amount_cents)}</td>
                                <td>
                                  <span className={`pill ${paymentStatusClass(item.status)}`}>{paymentStatusLabel(item.status)}</span>
                                </td>
                                <td>{item.paid_at ? formatDateTime(item.paid_at) : item.created_at ? formatDateTime(item.created_at) : "-"}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                      {(paymentOrders?.items ?? []).length === 0 ? (
                        <div className="feedback-banner">当前还没有购买记录，支付成功后会自动显示在这里。</div>
                      ) : null}
                      <PagerControls
                        onChange={setOrderHistoryPage}
                        page={paymentOrders?.page ?? orderHistoryPage}
                        pageSize={paymentOrders?.page_size ?? profilePageSize}
                        total={paymentOrders?.total ?? 0}
                      />
                    </>
                  ) : (
                    <div className="feedback-banner">登录后可查看充值和会员购买记录。</div>
                  )}
                </section>
              ) : null}

              {currentProfileTab === "invite" ? (
                <section className="content-card profile-card profile-panel-card" id="profile-invite">
                  <div className="section-header">
                    <div>
                      <p className="section-eyebrow">邀请好友</p>
                      <h2>邀请码、邀请统计和转化记录集中查看</h2>
                    </div>
                    {currentUser ? (
                      <div className="button-row">
                        <button className="secondary-button small-button" onClick={() => void handleCopyInviteCode()} type="button">
                          复制邀请码
                        </button>
                      </div>
                    ) : null}
                  </div>
                  {currentUser ? (
                    <div className="profile-panel-stack">
                      <label className="form-field">
                        <span>注册链接</span>
                        <input readOnly value={inviteRegistrationLink} />
                        <small className="helper-text">把这个注册链接分享给好友，打开后会自动带入邀请码并进入注册流程。</small>
                      </label>
                      <div className="button-row">
                        <button className="secondary-button small-button" onClick={() => void handleCopyInviteCode()} type="button">
                          复制邀请码
                        </button>
                        <button className="secondary-button small-button" onClick={() => void handleCopyInviteLink()} type="button">
                          复制注册链接
                        </button>
                        {inviteRegistrationLink ? (
                          <a className="secondary-button small-button" href={inviteRegistrationLink} rel="noreferrer" target="_blank">
                            打开注册链接
                          </a>
                        ) : null}
                      </div>
                    </div>
                  ) : null}
                  {currentUser ? (
                    <>
                      <div className="profile-overview">
                        <div>
                          <strong>{inviteSummary?.invite_code || currentUser.invite_code || "-"}</strong>
                          <span>我的邀请码</span>
                        </div>
                        <div>
                          <strong>{formatCount(Number(inviteSummary?.invited_count ?? 0))}</strong>
                          <span>已邀请人数</span>
                        </div>
                        <div>
                          <strong>{formatCount(Number(inviteSummary?.paid_invite_count ?? 0))}</strong>
                          <span>已付费人数</span>
                        </div>
                        <div>
                          <strong>{formatPrice(inviteSummary?.total_recharge_cents ?? 0)}</strong>
                          <span>累计邀请充值</span>
                        </div>
                      </div>
                      <label className="form-field">
                        <span>邀请码</span>
                        <input readOnly value={currentInviteCode} />
                        <small className="helper-text">邀请码由系统自动生成，前台普通用户不可修改。</small>
                      </label>
                      <div className="table-wrap">
                        <table className="data-table">
                          <thead>
                            <tr>
                              <th>好友账号</th>
                              <th>显示名称</th>
                              <th>注册时间</th>
                              <th>付费次数</th>
                              <th>累计充值</th>
                            </tr>
                          </thead>
                          <tbody>
                            {(inviteSummary?.items ?? []).map((item) => (
                              <tr key={item.user_id}>
                                <td>{item.username}</td>
                                <td>{item.display_name || "-"}</td>
                                <td>{formatDateTime(item.created_at)}</td>
                                <td>{formatCount(item.paid_order_count)}</td>
                                <td>{formatPrice(item.total_recharge_cents)}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                      {(inviteSummary?.items ?? []).length === 0 ? (
                        <div className="feedback-banner">分享邀请码后，你邀请注册的用户和他们的付费记录会显示在这里。</div>
                      ) : null}
                    </>
                  ) : (
                    <div className="feedback-banner">登录后可查看邀请码、邀请人数和累计充值统计。</div>
                  )}
                </section>
              ) : null}

              {currentProfileTab === "knowledge-base" ? (
                <section className="content-card profile-card profile-panel-card" id="profile-knowledge-base">
                  <div className="section-header">
                    <div>
                      <p className="section-eyebrow">我的知识库</p>
                      <h2>上传文本、Markdown、CSV、Excel 作为你自己的 MCP 私有知识库</h2>
                    </div>
                  </div>
                  {currentUser ? (
                    <>
                      <div className="profile-grid">
                        <form className="setup-form" onSubmit={handleSubmitKnowledgeBase}>
                          <label className="form-field">
                            <span>选择知识库文件</span>
                            <label className={`upload-picker ${knowledgeBaseForm.file ? "upload-picker-ready" : ""}`}>
                              <input
                                accept=".txt,.md,.csv,.xlsx"
                                className="upload-picker-input"
                                onChange={(event) => {
                                  const file = event.target.files?.[0] ?? null;
                                  setKnowledgeBaseForm((current) => ({
                                    ...current,
                                    file,
                                    fileName: file?.name ?? "",
                                    title: current.title || (file?.name ? file.name.replace(/\.[^.]+$/, "") : ""),
                                  }));
                                }}
                                type="file"
                              />
                              <div className="upload-picker-main">
                                <div className="upload-picker-meta">
                                  <strong>{knowledgeBaseForm.fileName || "点击选择要上传的知识库文件"}</strong>
                                  <span>
                                    {knowledgeBaseForm.file
                                      ? `文件大小 ${formatFileSize(knowledgeBaseForm.file.size)}，上传后仅当前账号和对应 MCP 检索可见`
                                      : "支持 TXT、Markdown、CSV、Excel，上传后会自动切片用于检索"}
                                  </span>
                                </div>
                                <span className="upload-picker-action">{knowledgeBaseForm.file ? "重新选择" : "选择文件"}</span>
                              </div>
                            </label>
                          </label>
                          <div className="upload-hint-list">
                            <span className="tag">私有知识库</span>
                            <span className="tag">支持 TXT / Markdown</span>
                            <span className="tag">支持 CSV / Excel</span>
                            <span className="tag">支持 MCP 检索</span>
                          </div>
                          <label className="form-field">
                            <span>文档标题</span>
                            <input
                              value={knowledgeBaseForm.title}
                              onChange={(event) => {
                                setKnowledgeBaseForm((current) => ({ ...current, title: event.target.value }));
                              }}
                              placeholder="例如：我的产品知识库"
                            />
                          </label>
                          <div className="feedback-banner">
                            当前将上传到 <strong>{selectedSubjectLabel}</strong> 学科下，管理员也能在后台区分公共知识库和你的私有知识库。
                          </div>
                          <button className="primary-button" disabled={profileBusyAction === "knowledge-base-upload"} type="submit">
                            {profileBusyAction === "knowledge-base-upload" ? "上传处理中..." : "上传到我的知识库"}
                          </button>
                        </form>

                        <div className="profile-card-body">
                          <div className="feedback-banner feedback-success">
                            你的个人知识库上传后，会自动参与开放接口与 MCP 工具的检索，但不会暴露给其他普通用户。
                          </div>
                          <ul className="detail-list">
                            <li>支持上传管理员公共知识库之外的个人资料，适合整理自己的文档、表格和说明。</li>
                            <li>停用后不会参与检索；删除后文档和切片都会一起移除。</li>
                            <li>知识库检索结果会返回命中片段和文档来源，方便定位答案来自哪份文档。</li>
                          </ul>
                        </div>
                      </div>

                      <div className="section-toolbar">
                        <div>
                          <h2>我的知识库文档</h2>
                          <p className="helper-text">这里只展示当前账号上传的私有知识库文档。</p>
                        </div>
                        <input
                          className="toolbar-search"
                          onChange={(event) => {
                            setKnowledgeBaseQuery(event.target.value);
                            setKnowledgeBasePage(1);
                          }}
                          placeholder="搜索标题或文件名"
                          value={knowledgeBaseQuery}
                        />
                      </div>

                      <div className="table-wrap">
                        <table className="data-table">
                          <thead>
                            <tr>
                              <th>标题</th>
                              <th>文件名</th>
                              <th>格式</th>
                              <th>状态</th>
                              <th>片段数</th>
                              <th>更新时间</th>
                              <th>操作</th>
                            </tr>
                          </thead>
                          <tbody>
                            {(knowledgeBaseDocuments?.items ?? []).map((item) => {
                              const statusBusy = profileBusyAction === `knowledge-base-status-${item.id}`;
                              const deleteBusy = profileBusyAction === `knowledge-base-delete-${item.id}`;
                              const viewBusy = knowledgeBaseDocumentPreviewLoading && knowledgeBaseDocumentPreview?.id === item.id;
                              return (
                                <tr key={item.id}>
                                  <td>{item.title}</td>
                                  <td>{item.source_file_name}</td>
                                  <td>{item.source_type}</td>
                                  <td>
                                    <span className={item.status === "active" ? "pill pill-success" : "pill pill-muted"}>
                                      {item.status === "active" ? "启用中" : "已停用"}
                                    </span>
                                  </td>
                                  <td>{formatCount(item.chunk_count)}</td>
                                  <td>{formatDateTime(item.updated_at)}</td>
                                  <td>
                                    <div className="button-row">
                                      <button
                                        className="secondary-button small-button"
                                        disabled={statusBusy || deleteBusy || viewBusy}
                                        onClick={() => void handleViewKnowledgeBaseDocument(item)}
                                        type="button"
                                      >
                                        {viewBusy ? "加载中..." : "查看内容"}
                                      </button>
                                      <button
                                        className="secondary-button small-button"
                                        disabled={statusBusy || deleteBusy}
                                        onClick={() => void handleToggleKnowledgeBaseDocument(item)}
                                        type="button"
                                      >
                                        {statusBusy ? "处理中..." : item.status === "active" ? "停用" : "启用"}
                                      </button>
                                      <button
                                        className="secondary-button small-button"
                                        disabled={statusBusy || deleteBusy}
                                        onClick={() => void handleDeleteKnowledgeBaseDocument(item)}
                                        type="button"
                                      >
                                        {deleteBusy ? "删除中..." : "删除"}
                                      </button>
                                    </div>
                                  </td>
                                </tr>
                              );
                            })}
                          </tbody>
                        </table>
                      </div>
                      {(knowledgeBaseDocuments?.items ?? []).length === 0 ? (
                        <div className="feedback-banner">当前还没有上传个人知识库文件，先上传一份文本或表格试试。</div>
                      ) : null}
                      <PagerControls
                        onChange={setKnowledgeBasePage}
                        page={knowledgeBaseDocuments?.page ?? knowledgeBasePage}
                        pageSize={knowledgeBaseDocuments?.page_size ?? profilePageSize}
                        total={knowledgeBaseDocuments?.total ?? 0}
                      />
                    </>
                  ) : (
                    <div className="feedback-banner">登录后可上传你自己的知识库文件，并在 MCP 工具中按会员权限调用。</div>
                  )}
                </section>
              ) : null}

              {currentProfileTab === "api-tools" ? (
                <section className="content-card profile-card profile-panel-card" id="profile-api-tools">
                  <div className="section-header">
                    <div>
                      <p className="section-eyebrow">API 工具</p>
                      <h2>把你的开放接口配置成可供 MCP 直接调用的个人工具</h2>
                    </div>
                  </div>
                  {currentUser ? (
                    <>
                      <div className="profile-grid">
                        <form className="setup-form" onSubmit={handleSaveAPIConfig}>
                          <label className="form-field">
                            <span>工具名称</span>
                            <input
                              value={apiConfigEditor.name}
                              onChange={(event) => {
                                setAPIConfigEditor((current) => ({ ...current, name: event.target.value }));
                              }}
                              placeholder="例如：查询内部订单"
                            />
                          </label>
                          <label className="form-field">
                            <span>MCP 工具名</span>
                            <input
                              value={apiConfigEditor.tool_name}
                              onChange={(event) => {
                                setAPIConfigEditor((current) => ({ ...current, tool_name: event.target.value }));
                              }}
                              placeholder="例如：query_order_status"
                            />
                          </label>
                          <label className="form-field">
                            <span>请求地址</span>
                            <input
                              value={apiConfigEditor.url}
                              onChange={(event) => {
                                setAPIConfigEditor((current) => ({ ...current, url: event.target.value }));
                              }}
                              placeholder="https://example.com/api/orders 或 /api/v1/orders"
                            />
                          </label>
                          <div className="button-row">
                            <label className="form-field" style={{ flex: 1 }}>
                              <span>请求方法</span>
                              <select
                                value={apiConfigEditor.method}
                                onChange={(event) => {
                                  setAPIConfigEditor((current) => ({ ...current, method: event.target.value }));
                                }}
                              >
                                <option value="GET">GET</option>
                                <option value="POST">POST</option>
                                <option value="PUT">PUT</option>
                                <option value="PATCH">PATCH</option>
                                <option value="DELETE">DELETE</option>
                                <option value="HEAD">HEAD</option>
                              </select>
                            </label>
                            <label className="form-field" style={{ flex: 1 }}>
                              <span>分类</span>
                              <input
                                value={apiConfigEditor.category}
                                onChange={(event) => {
                                  setAPIConfigEditor((current) => ({ ...current, category: event.target.value }));
                                }}
                                placeholder="custom"
                              />
                            </label>
                          </div>
                          <label className="form-field">
                            <span>说明</span>
                            <textarea
                              rows={3}
                              value={apiConfigEditor.description}
                              onChange={(event) => {
                                setAPIConfigEditor((current) => ({ ...current, description: event.target.value }));
                              }}
                              placeholder="告诉 MCP 这个工具做什么。"
                            />
                          </label>
                          <label className="form-field">
                            <span>Headers JSON</span>
                            <textarea
                              rows={4}
                              value={apiConfigEditor.headers}
                              onChange={(event) => {
                                setAPIConfigEditor((current) => ({ ...current, headers: event.target.value }));
                              }}
                              placeholder='{"Authorization":"Bearer {{access_token}}"}'
                            />
                          </label>
                          <label className="form-field">
                            <span>Body JSON / 模板</span>
                            <textarea
                              rows={4}
                              value={apiConfigEditor.body}
                              onChange={(event) => {
                                setAPIConfigEditor((current) => ({ ...current, body: event.target.value }));
                              }}
                              placeholder='{"keyword":"{{query}}"}'
                            />
                          </label>
                          <label className="form-field">
                            <span>参数定义 JSON</span>
                            <textarea
                              rows={4}
                              value={apiConfigEditor.parameters}
                              onChange={(event) => {
                                setAPIConfigEditor((current) => ({ ...current, parameters: event.target.value }));
                              }}
                              placeholder='[{"name":"query","type":"string","required":true,"description":"搜索词"}]'
                            />
                          </label>
                          <div className="button-row">
                            <label className="tag">
                              <input
                                checked={apiConfigEditor.is_active}
                                onChange={(event) => {
                                  setAPIConfigEditor((current) => ({ ...current, is_active: event.target.checked }));
                                }}
                                type="checkbox"
                              />
                              启用
                            </label>
                            <label className="tag">
                              <input
                                checked={Boolean(apiConfigEditor.is_public)}
                                onChange={(event) => {
                                  setAPIConfigEditor((current) => ({ ...current, is_public: event.target.checked }));
                                }}
                                type="checkbox"
                              />
                              发布到工具市场
                            </label>
                            <label className="tag">
                              <input
                                checked={Boolean(apiConfigEditor.allow_admin_publish)}
                                onChange={(event) => {
                                  setAPIConfigEditor((current) => ({ ...current, allow_admin_publish: event.target.checked }));
                                }}
                                type="checkbox"
                              />
                              允许管理员代为公开
                            </label>
                          </div>
                          <div className="button-row">
                            <button className="primary-button" disabled={profileBusyAction === "api-config-save"} type="submit">
                              {profileBusyAction === "api-config-save"
                                ? "保存中..."
                                : apiConfigEditor.id > 0
                                  ? "更新 API 工具"
                                  : "创建 API 工具"}
                            </button>
                            <button className="secondary-button" onClick={resetAPIConfigEditor} type="button">
                              新建空白
                            </button>
                          </div>
                        </form>

                        <div className="profile-card-body">
                          <div className="feedback-banner feedback-success">
                            这里创建的是你自己的 API 工具。启用后可以进入 MCP 工具列表，并继续由管理员决定是否要加会员门槛。
                          </div>
                          <label className="form-field">
                            <span>测试参数 JSON</span>
                            <textarea
                              rows={8}
                              value={apiConfigTestArguments}
                              onChange={(event) => {
                                setAPIConfigTestArguments(event.target.value);
                              }}
                              placeholder='{"query":"hello"}'
                            />
                          </label>
                          {apiConfigTestResult ? (
                            <div className="feedback-banner">
                              <strong>最近一次测试返回状态：</strong> {apiConfigTestResult.status_code}
                              <pre className="code-panel">{apiConfigTestPreview}</pre>
                            </div>
                          ) : (
                            <div className="feedback-banner">
                              先从下面列表里选择一个已有工具进行测试，返回结果会显示在这里。
                            </div>
                          )}
                        </div>
                      </div>

                      <div className="section-toolbar">
                        <div>
                          <h2>我的 API 工具</h2>
                          <p className="helper-text">支持停用、删除、重新编辑，也支持作为 MCP 工具直接调用。</p>
                        </div>
                        <input
                          className="toolbar-search"
                          value={apiConfigQuery}
                          onChange={(event) => {
                            setAPIConfigQuery(event.target.value);
                            setAPIConfigPage(1);
                          }}
                          placeholder="搜索名称、工具名或分类"
                        />
                      </div>

                      <div className="table-wrap">
                        <table className="data-table">
                          <thead>
                            <tr>
                              <th>名称</th>
                              <th>工具名</th>
                              <th>方法</th>
                              <th>分类</th>
                              <th>状态</th>
                              <th>公开</th>
                              <th>操作</th>
                            </tr>
                          </thead>
                          <tbody>
                            {(apiConfigs?.items ?? []).map((item) => {
                              const toggleBusy = profileBusyAction === `api-config-toggle-${item.id}`;
                              const deleteBusy = profileBusyAction === `api-config-delete-${item.id}`;
                              const testBusy = profileBusyAction === `api-config-test-${item.id}`;
                              return (
                                <tr key={item.id}>
                                  <td>{item.name}</td>
                                  <td>{item.resolved_tool_name}</td>
                                  <td>{item.method}</td>
                                  <td>{item.category || "-"}</td>
                                  <td>
                                    <span className={item.is_active ? "pill pill-success" : "pill pill-muted"}>
                                      {item.is_active ? "已启用" : "已停用"}
                                    </span>
                                  </td>
                                  <td>{item.is_public ? "公开" : "私有"}</td>
                                  <td>
                                    <div className="button-row">
                                      <button className="secondary-button small-button" onClick={() => startEditAPIConfig(item)} type="button">
                                        编辑
                                      </button>
                                      <button
                                        className="secondary-button small-button"
                                        disabled={testBusy}
                                        onClick={() => void handleTestAPIConfig(item)}
                                        type="button"
                                      >
                                        {testBusy ? "测试中..." : "测试"}
                                      </button>
                                      <button
                                        className="secondary-button small-button"
                                        disabled={toggleBusy || deleteBusy}
                                        onClick={() => void handleToggleAPIConfig(item)}
                                        type="button"
                                      >
                                        {toggleBusy ? "处理中..." : item.is_active ? "停用" : "启用"}
                                      </button>
                                      <button
                                        className="secondary-button small-button"
                                        disabled={toggleBusy || deleteBusy}
                                        onClick={() => void handleDeleteAPIConfig(item)}
                                        type="button"
                                      >
                                        {deleteBusy ? "删除中..." : "删除"}
                                      </button>
                                    </div>
                                  </td>
                                </tr>
                              );
                            })}
                          </tbody>
                        </table>
                      </div>
                      {(apiConfigs?.items ?? []).length === 0 ? (
                        <div className="feedback-banner">当前还没有创建个人 API 工具，先配置一条接口试试。</div>
                      ) : null}
                      <PagerControls
                        onChange={setAPIConfigPage}
                        page={apiConfigs?.page ?? apiConfigPage}
                        pageSize={apiConfigs?.page_size ?? profilePageSize}
                        total={apiConfigs?.total ?? 0}
                      />
                    </>
                  ) : (
                    <div className="feedback-banner">登录后才可以创建和管理你自己的 API 工具。</div>
                  )}
                </section>
              ) : null}

              {currentProfileTab === "xiaomi" ? (
                <section className="content-card profile-card profile-panel-card" id="profile-xiaomi">
                  <div className="section-header">
                    <div>
                      <p className="section-eyebrow">米家智能</p>
                      <h2>绑定小米账号，刷新设备列表，并作为 MCP 工具直接调用</h2>
                    </div>
                  </div>
                  {currentUser ? (
                    <>
                      <div className="profile-grid">
                        <form className="setup-form" onSubmit={handleSaveXiaomiConfig}>
                          <label className="form-field">
                            <span>服务器区域</span>
                            <select
                              value={xiaomiConfig?.server || "cn"}
                              onChange={(event) => {
                                setXiaomiConfig((current) => ({
                                  ...(current || {
                                    learner_user_id: currentUser.id,
                                    username: "",
                                    xiaomi_user_id: "",
                                    server: "cn",
                                    is_active: false,
                                    has_credentials: false,
                                    device_count: 0,
                                  }),
                                  server: event.target.value,
                                }));
                              }}
                            >
                              <option value="cn">cn</option>
                              <option value="de">de</option>
                              <option value="sg">sg</option>
                              <option value="us">us</option>
                              <option value="ru">ru</option>
                              <option value="tw">tw</option>
                            </select>
                          </label>
                          <label className="form-field">
                            <span>账号备注</span>
                            <input
                              value={xiaomiConfig?.username || ""}
                              onChange={(event) => {
                                setXiaomiConfig((current) => ({
                                  ...(current || {
                                    learner_user_id: currentUser.id,
                                    xiaomi_user_id: "",
                                    server: "cn",
                                    is_active: false,
                                    has_credentials: false,
                                    device_count: 0,
                                    username: "",
                                  }),
                                  username: event.target.value,
                                }));
                              }}
                              placeholder="例如：我的米家主账号"
                            />
                          </label>
                          <label className="form-field">
                            <span>小米用户 ID</span>
                            <input
                              value={xiaomiConfig?.xiaomi_user_id || ""}
                              onChange={(event) => {
                                setXiaomiConfig((current) => ({
                                  ...(current || {
                                    learner_user_id: currentUser.id,
                                    username: "",
                                    server: "cn",
                                    is_active: false,
                                    has_credentials: false,
                                    device_count: 0,
                                    xiaomi_user_id: "",
                                  }),
                                  xiaomi_user_id: event.target.value,
                                }));
                              }}
                              placeholder="扫码成功后会自动回填"
                            />
                          </label>
                          <div className="button-row">
                            <label className="tag">
                              <input
                                checked={Boolean(xiaomiConfig?.is_active)}
                                onChange={(event) => {
                                  setXiaomiConfig((current) => ({
                                    ...(current || {
                                      learner_user_id: currentUser.id,
                                      username: "",
                                      xiaomi_user_id: "",
                                      server: "cn",
                                      is_active: false,
                                      has_credentials: false,
                                      device_count: 0,
                                    }),
                                    is_active: event.target.checked,
                                  }));
                                }}
                                type="checkbox"
                              />
                              启用此账号
                            </label>
                            <span className="pill pill-muted">
                              {xiaomiConfig?.has_credentials ? "已保存令牌" : "尚未保存令牌"}
                            </span>
                          </div>
                          <div className="button-row">
                            <button className="primary-button" disabled={profileBusyAction === "xiaomi-save"} type="submit">
                              {profileBusyAction === "xiaomi-save" ? "保存中..." : "保存基础配置"}
                            </button>
                            <button
                              className="secondary-button"
                              disabled={profileBusyAction === "xiaomi-qr-login"}
                              onClick={() => void handleStartXiaomiQRLogin()}
                              type="button"
                            >
                              {profileBusyAction === "xiaomi-qr-login" ? "生成中..." : "扫码登录"}
                            </button>
                            <button
                              className="secondary-button"
                              disabled={profileBusyAction === "xiaomi-clear"}
                              onClick={() => void handleClearXiaomiTokens()}
                              type="button"
                            >
                              {profileBusyAction === "xiaomi-clear" ? "处理中..." : "清空令牌"}
                            </button>
                          </div>
                        </form>

                        <div className="profile-card-body">
                          <div className="profile-overview">
                            <div>
                              <strong>{xiaomiConfig?.server || "cn"}</strong>
                              <span>当前区域</span>
                            </div>
                            <div>
                              <strong>{formatCount(xiaomiDevices?.total ?? xiaomiConfig?.device_count ?? 0)}</strong>
                              <span>设备数量</span>
                            </div>
                            <div>
                              <strong>{xiaomiHomes.length}</strong>
                              <span>家庭数量</span>
                            </div>
                            <div>
                              <strong>{xiaomiConfig?.last_sync_at ? formatDateTime(xiaomiConfig.last_sync_at) : "-"}</strong>
                              <span>最近同步</span>
                            </div>
                          </div>
                          {xiaomiQRSession?.qr_image ? (
                            <div className="feedback-banner">
                              <strong>扫码登录</strong>
                              <img alt="xiaomi qr" className="qr-preview-image" src={xiaomiQRSession.qr_image} />
                              <span>{xiaomiQRStatus || "等待扫码..."}</span>
                            </div>
                          ) : (
                            <div className="feedback-banner">
                              点击“扫码登录”后，会在这里显示二维码；成功后会自动保存小米令牌。
                            </div>
                          )}
                        </div>
                      </div>

                      <div className="section-toolbar">
                        <div>
                          <h2>设备与家庭</h2>
                          <p className="helper-text">刷新后即可在 MCP 中调用 `xiaomi_*` / `mijia_*` 工具操作这些设备。</p>
                        </div>
                        <div className="button-row">
                          <input
                            className="toolbar-search"
                            value={xiaomiSearchQuery}
                            onChange={(event) => {
                              setXiaomiSearchQuery(event.target.value);
                            }}
                            placeholder="搜索设备名称、型号、did"
                          />
                          <button
                            className="secondary-button"
                            disabled={profileBusyAction === "xiaomi-homes"}
                            onClick={() => void handleLoadXiaomiHomes()}
                            type="button"
                          >
                            {profileBusyAction === "xiaomi-homes" ? "加载中..." : "加载家庭"}
                          </button>
                          <button
                            className="primary-button"
                            disabled={profileBusyAction === "xiaomi-refresh"}
                            onClick={() => void handleRefreshXiaomiDevices()}
                            type="button"
                          >
                            {profileBusyAction === "xiaomi-refresh" ? "刷新中..." : "刷新设备"}
                          </button>
                        </div>
                      </div>

                      {xiaomiHomes.length > 0 ? (
                        <div className="tag-list">
                          {xiaomiHomes.map((item) => (
                            <span className="tag" key={item.id}>
                              {item.name}
                            </span>
                          ))}
                        </div>
                      ) : null}

                      <div className="table-wrap">
                        <table className="data-table">
                          <thead>
                            <tr>
                              <th>设备名</th>
                              <th>型号</th>
                              <th>DID</th>
                              <th>家庭</th>
                              <th>房间</th>
                              <th>在线</th>
                            </tr>
                          </thead>
                          <tbody>
                            {visibleXiaomiDevices.map((item) => (
                              <tr key={item.did}>
                                <td>{item.name}</td>
                                <td>{item.model}</td>
                                <td>{item.did}</td>
                                <td>{item.home_name || "-"}</td>
                                <td>{item.room_name || "-"}</td>
                                <td>
                                  <span className={item.is_online ? "pill pill-success" : "pill pill-muted"}>
                                    {item.is_online ? "在线" : "离线"}
                                  </span>
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                      {visibleXiaomiDevices.length === 0 ? (
                        <div className="feedback-banner">当前没有可展示的设备，先扫码登录并刷新设备列表。</div>
                      ) : null}
                    </>
                  ) : (
                    <div className="feedback-banner">登录后才可以绑定你自己的米家账号和设备。</div>
                  )}
                </section>
              ) : null}

              {currentProfileTab === "plans" ? (
                <section className="content-card profile-card profile-panel-card" id="plans">
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
              ) : null}

              {currentProfileTab === "insights" ? (
                <div className="profile-panel-stack" id="profile-insights">
                  {currentUser ? (
                    <>
                      <div className="profile-snapshot-grid">
                        <article className="profile-snapshot-card">
                          <span>待复习</span>
                          <strong>{formatCount(learningSummary?.due_reviews ?? 0)}</strong>
                          <small>到期的词条会优先出现在下面列表里。</small>
                        </article>
                        <article className="profile-snapshot-card">
                          <span>已追踪</span>
                          <strong>{formatCount(learningSummary?.tracked_words ?? 0)}</strong>
                          <small>在词库页点“记住了 / 待复习”后，这里会持续累计。</small>
                        </article>
                        <article className="profile-snapshot-card">
                          <span>已掌握</span>
                          <strong>{formatCount(learningSummary?.mastered_words ?? 0)}</strong>
                          <small>进入“已掌握”的词会获得更长的复习间隔。</small>
                        </article>
                        <article className="profile-snapshot-card">
                          <span>记忆正确率</span>
                          <strong>{formatPercent(learningSummary?.correct_rate ?? 0)}</strong>
                          <small>基于你的复习记录自动计算。</small>
                        </article>
                      </div>

                      <div className="profile-grid">
                        <section className="content-card profile-card profile-panel-card">
                          <div className="section-header">
                            <div>
                              <p className="section-eyebrow">学习等级</p>
                              <h2>初级、中级、高级、已掌握一眼看清</h2>
                            </div>
                          </div>
                          <div className="profile-overview">
                            <div>
                              <strong>{formatCount(learningCountByKey(learningLevelCounts, "beginner"))}</strong>
                              <span>初级</span>
                            </div>
                            <div>
                              <strong>{formatCount(learningCountByKey(learningLevelCounts, "intermediate"))}</strong>
                              <span>中级</span>
                            </div>
                            <div>
                              <strong>{formatCount(learningCountByKey(learningLevelCounts, "advanced"))}</strong>
                              <span>高级</span>
                            </div>
                            <div>
                              <strong>{formatCount(learningCountByKey(learningLevelCounts, "mastered"))}</strong>
                              <span>已掌握</span>
                            </div>
                          </div>
                          <div className="tag-list">
                            {learningDifficultyCounts.map((item) => (
                              <span className="tag" key={item.key}>
                                {item.label} {formatCount(item.count)}
                              </span>
                            ))}
                          </div>
                        </section>

                        <section className="content-card profile-card profile-panel-card">
                          <div className="section-header">
                            <div>
                              <p className="section-eyebrow">记忆曲线</p>
                              <h2>最近 14 天复习走势</h2>
                            </div>
                          </div>
                          <LearningCurveChart points={learningCurvePoints} />
                          <p className="helper-text">
                            绿色越高代表当天记住比例越高；如果你在词库页持续点击“记住了 / 待复习”，这条线会越来越有参考价值。
                          </p>
                        </section>
                      </div>

                      <section className="content-card profile-card profile-panel-card">
                        <div className="section-toolbar">
                          <div>
                            <h2>我的学习词条</h2>
                            <p className="helper-text">可以直接调整难度、等级，并记录本次是否记住。</p>
                          </div>
                          <div className="button-row learning-toolbar">
                            <input
                              className="toolbar-search"
                              onChange={(event) => {
                                setLearningQuery(event.target.value);
                                setLearningPage(1);
                              }}
                              placeholder="搜索单词或释义"
                              value={learningQuery}
                            />
                            <select
                              value={learningLevelFilter}
                              onChange={(event) => {
                                setLearningLevelFilter(event.target.value);
                                setLearningPage(1);
                              }}
                            >
                              <option value="">全部等级</option>
                              <option value="beginner">初级</option>
                              <option value="intermediate">中级</option>
                              <option value="advanced">高级</option>
                              <option value="mastered">已掌握</option>
                            </select>
                            <select
                              value={learningDifficultyFilter}
                              onChange={(event) => {
                                setLearningDifficultyFilter(event.target.value);
                                setLearningPage(1);
                              }}
                            >
                              <option value="">全部难度</option>
                              <option value="easy">简单</option>
                              <option value="medium">中等</option>
                              <option value="hard">困难</option>
                            </select>
                          </div>
                        </div>

                        <div className="table-wrap">
                          <table className="data-table">
                            <thead>
                              <tr>
                                <th>单词</th>
                                <th>释义</th>
                                <th>等级</th>
                                <th>难度</th>
                                <th>复习状态</th>
                                <th>操作</th>
                              </tr>
                            </thead>
                            <tbody>
                              {(learningProgress?.items ?? []).map((item) => {
                                const saveBusy = profileBusyAction === `learning-save-${item.word_id}`;
                                const reviewBusy = profileBusyAction === `learning-review-${item.word_id}`;
                                return (
                                  <tr key={item.id}>
                                    <td>
                                      <strong>{item.term}</strong>
                                      <div className="helper-text">{item.classification || "-"}</div>
                                    </td>
                                    <td>{item.translation || "-"}</td>
                                    <td>
                                      <select
                                        disabled={saveBusy || reviewBusy}
                                        value={item.level}
                                        onChange={(event) =>
                                          void handleSaveLearningProgress(item.word_id, event.target.value, item.difficulty)
                                        }
                                      >
                                        <option value="beginner">初级</option>
                                        <option value="intermediate">中级</option>
                                        <option value="advanced">高级</option>
                                        <option value="mastered">已掌握</option>
                                      </select>
                                    </td>
                                    <td>
                                      <select
                                        disabled={saveBusy || reviewBusy}
                                        value={item.difficulty}
                                        onChange={(event) =>
                                          void handleSaveLearningProgress(item.word_id, item.level, event.target.value)
                                        }
                                      >
                                        <option value="easy">简单</option>
                                        <option value="medium">中等</option>
                                        <option value="hard">困难</option>
                                      </select>
                                    </td>
                                    <td>
                                      <div>{formatLearningReviewStatus(item.next_review_at, item.is_due)}</div>
                                      <small className="helper-text">
                                        复习 {formatCount(item.review_count)} 次，连续记住 {formatCount(item.consecutive_correct)} 次
                                      </small>
                                    </td>
                                    <td>
                                      <div className="button-row">
                                        <button
                                          className="secondary-button small-button"
                                          disabled={saveBusy || reviewBusy}
                                          onClick={() =>
                                            void handleReviewLearningWord({ id: item.word_id }, true, {
                                              level: item.level,
                                              difficulty: item.difficulty,
                                            })
                                          }
                                          type="button"
                                        >
                                          {reviewBusy ? "处理中..." : "记住了"}
                                        </button>
                                        <button
                                          className="secondary-button small-button"
                                          disabled={saveBusy || reviewBusy}
                                          onClick={() =>
                                            void handleReviewLearningWord({ id: item.word_id }, false, {
                                              level: item.level,
                                              difficulty: item.difficulty,
                                            })
                                          }
                                          type="button"
                                        >
                                          {reviewBusy ? "处理中..." : "待复习"}
                                        </button>
                                      </div>
                                    </td>
                                  </tr>
                                );
                              })}
                            </tbody>
                          </table>
                        </div>
                        {(learningProgress?.items ?? []).length === 0 ? (
                          <div className="feedback-banner">
                            还没有学习记录。先去词库页点击“记住了 / 待复习”，系统就会开始记录你的等级、难度和记忆曲线。
                          </div>
                        ) : null}
                        <PagerControls
                          onChange={setLearningPage}
                          page={learningProgress?.page ?? learningPage}
                          pageSize={learningProgress?.page_size ?? profilePageSize}
                          total={learningProgress?.total ?? 0}
                        />
                      </section>
                    </>
                  ) : (
                    <div className="feedback-banner">登录后可查看学习等级、难度分布、记忆曲线和个人复习记录。</div>
                  )}
                </div>
              ) : null}
            </div>
          </main>
        </div>
      ) : activeView === "market" ? (
        <div className="site-main site-main-profile">
          <main className="site-content profile-page mcp-page">
            <section className="content-card profile-hero-card mcp-page-hero" id="mcp-market">
              <div className="section-header profile-hero-header">
                <div>
                  <p className="section-eyebrow">MCP 工具市场</p>
                  <h1>把当前项目里的 MCP 能力单独展示出来</h1>
                  <p className="helper-text">
                    这里集中查看内置工具、知识库工具、学习工具、米家工具和你自己的 API 工具，同时会标出是否需要登录、是否需要会员、当前能否直接使用。
                  </p>
                </div>
                <div className="button-row">
                  <a className="secondary-button" href="#profile">
                    返回个人中心
                  </a>
                  <a className="primary-button" href="#mcp">
                    去连接中心
                  </a>
                </div>
              </div>

              <div className="mcp-page-toolbar">
                <label className="form-field mcp-page-subject-field">
                  <span>当前工具学科</span>
                  <select
                    value={subjectKey}
                    onChange={(event) => {
                      setSubjectKey(event.target.value);
                      setMarketPage(1);
                    }}
                  >
                    {subjects.map((subject) => (
                      <option key={subject.key} value={subject.key}>
                        {subject.name}
                      </option>
                    ))}
                  </select>
                </label>
                <input
                  className="toolbar-search"
                  onChange={(event) => {
                    setMarketQuery(event.target.value);
                    setMarketPage(1);
                  }}
                  placeholder="搜索工具名称、描述或分类"
                  value={marketQuery}
                />
                <select
                  value={marketCategory}
                  onChange={(event) => {
                    setMarketCategory(event.target.value);
                    setMarketPage(1);
                  }}
                >
                  <option value="">全部分类</option>
                  {marketCategories.map((item) => (
                    <option key={item} value={item}>
                      {item}
                    </option>
                  ))}
                </select>
              </div>

              <div className="tag-list">
                <span className="tag">工具总数 {formatCount(marketTools.length)}</span>
                <span className="tag">当前可见 {formatCount(filteredMarketTools.length)}</span>
                <span className="tag">{currentUser ? `当前账号：${learnerName}` : "未登录访客"}</span>
              </div>
            </section>

            {marketError ? <div className="feedback-banner feedback-error">{marketError}</div> : null}
            {marketLoading ? <div className="feedback-banner">正在加载 MCP 工具市场...</div> : null}

            <section className="market-grid">
              {filteredMarketTools.map((tool) => (
                <article className="content-card market-card" key={tool.name}>
                  <div className="market-card-header">
                    <div>
                      <strong>{tool.title || tool.name}</strong>
                      <span>{tool.name}</span>
                    </div>
                    <span className={tool.canUse ? "pill pill-success" : "pill pill-warning"}>
                      {tool.canUse ? "可调用" : "受限"}
                    </span>
                  </div>
                  <p>{tool.description}</p>
                  <div className="tag-list">
                    <span className="tag">{tool.category || "general"}</span>
                    <span className="tag">{tool.sourceType || "builtin"}</span>
                    {tool.requiresAuth ? <span className="tag">需登录</span> : null}
                    {tool.requiresMembership ? <span className="tag">需会员</span> : null}
                  </div>
                  <div className="helper-text">
                    {tool.canUse
                      ? "当前条件下可以直接被 MCP 调用。"
                      : tool.requiresMembership
                        ? "管理员已配置为会员工具，未满足会员条件时不能调用。"
                        : tool.requiresAuth
                          ? "需要先登录学习账号后才能调用。"
                          : "当前工具暂不可用。"}
                  </div>
                </article>
              ))}
            </section>
            {marketResult && marketResult.total > 0 ? (
              <PagerControls
                disabled={marketLoading}
                onChange={setMarketPage}
                page={marketResult.page}
                pageSize={marketResult.page_size}
                total={marketResult.total}
              />
            ) : null}
            {!marketLoading && filteredMarketTools.length === 0 ? (
              <div className="feedback-banner">当前筛选条件下没有匹配的 MCP 工具。</div>
            ) : null}
          </main>
        </div>
      ) : activeView === "mcp" ? (
        <div className="site-main site-main-profile">
          <main className="site-content profile-page mcp-page">
            <section className="content-card profile-hero-card mcp-page-hero" id="mcp">
              <div className="section-header profile-hero-header">
                <div>
                  <p className="section-eyebrow">MCP 连接中心</p>
                  <h1>{currentUser ? "单独管理 Brights 到小智 AI 的远程 WSS 连接" : "先登录账号，再管理远程 MCP 连接"}</h1>
                  <p className="helper-text">
                    这里是独立的 MCP 页面，只负责维护 Brights 主动连接小智 AI 的远程 ws / wss 地址、连接状态和工具暴露情况，不再和个人中心内容混在一起。
                  </p>
                </div>
                <div className="button-row">
                  <a className="secondary-button" href="#profile">
                    返回个人中心
                  </a>
                  <a className="primary-button" href="#catalog">
                    返回词库学习
                  </a>
                </div>
              </div>

              <div className="mcp-page-toolbar">
                <label className="form-field mcp-page-subject-field">
                  <span>当前连接学科</span>
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

                <div className="mcp-page-identity-card">
                  <strong>{currentUser ? learnerName || currentUser.username : "未登录学员"}</strong>
                  <span>{currentUser ? `${selectedSubjectLabel} · ${membershipBadgeText}` : "登录后按当前学员会员权限返回数据"}</span>
                </div>
              </div>
            </section>

            <MCPConsole
              learnerName={learnerName || currentUser?.username || ""}
              subjectKey={subjectKey}
              token={learnerAccessToken}
            />
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
                      handleSelectClassification(item.name);
                    }}
                    type="button"
                  >
                    {formatClassificationButtonLabel(item)}
                    <span>{formatCount(item.count)}</span>
                  </button>
                ))}
              </div>
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
                  {selectedClassificationStat?.has_member_content ? (
                    <div className="catalog-membership-chip-row">
                      <span className={selectedClassificationLocked ? "pill pill-warning" : "pill pill-muted"}>
                        {selectedClassificationLocked ? "会员专享分类" : "本分类含会员内容"}
                      </span>
                      <span className="helper-text">
                        {selectedClassificationLocked
                          ? `开通后可学习 ${formatCount(selectedClassificationStat.count)} 个场景词。`
                          : `当前可先学习 ${formatCount(selectedClassificationStat.accessible_count)} 个可见词，开通后可查看完整内容。`}
                      </span>
                    </div>
                  ) : null}
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
                        handleSelectClassification(event.target.value);
                      }}
                    >
                      <option value="">全部分类</option>
                      {classificationOptions.map((item) => (
                        <option key={item.name} value={item.name}>
                          {formatClassificationOptionLabel(item)}
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

              {loadingWords ? <div className="feedback-banner">正在加载学习内容...</div> : null}

              <div className="word-table-wrap">
                {selectedClassificationLocked ? (
                  <div className="locked-classification-card">
                    <span className="pill pill-warning">会员专享</span>
                    <h3>{classification}</h3>
                    <p>
                      这个场景当前属于会员专享内容。开通后可学习{" "}
                      {formatCount(selectedClassificationStat?.count ?? 0)} 个场景词，并继续使用点读与检索。
                    </p>
                    <div className="button-row">
                      <a className="primary-button" href={currentUser ? "#plans" : "#profile"}>
                        {currentUser ? "去看会员方案" : "先登录学习账号"}
                      </a>
                      <button
                        className="secondary-button"
                        onClick={() => {
                          setClassification("");
                          setPage(1);
                        }}
                        type="button"
                      >
                        先看可免费学习的分类
                      </button>
                    </div>
                  </div>
                ) : null}
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
                          {currentUser ? (
                            <div className="button-row word-learning-actions">
                              <button
                                className="secondary-button small-button"
                                disabled={profileBusyAction === `learning-review-${word.id}`}
                                onClick={() => void handleReviewLearningWord(word, true)}
                                type="button"
                              >
                                {profileBusyAction === `learning-review-${word.id}` ? "处理中..." : "记住了"}
                              </button>
                              <button
                                className="secondary-button small-button"
                                disabled={profileBusyAction === `learning-review-${word.id}`}
                                onClick={() => void handleReviewLearningWord(word, false)}
                                type="button"
                              >
                                {profileBusyAction === `learning-review-${word.id}` ? "处理中..." : "待复习"}
                              </button>
                            </div>
                          ) : null}
                        </td>
                        <td>{word.translation || "-"}</td>
                        <td>{word.classification || "-"}</td>
                        <td>{word.phonetics || "-"}</td>
                        <td>{word.source || "-"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {!selectedClassificationLocked && !loadingWords && (words?.items ?? []).length === 0 ? (
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

      {knowledgeBaseDocumentPreview ? (
        <div
          className="document-preview-backdrop"
          onClick={(event) => {
            if (event.target === event.currentTarget) {
              closeKnowledgeBaseDocumentPreview();
            }
          }}
        >
          <section className="document-preview-card">
            <div className="section-header">
              <div>
                <p className="section-eyebrow">知识库文档内容</p>
                <h2>{knowledgeBaseDocumentPreview.title || knowledgeBaseDocumentPreview.source_file_name}</h2>
                <p className="helper-text">
                  {knowledgeBaseDocumentPreview.source_file_name} · {formatCount(knowledgeBaseDocumentPreview.chunk_count)} 个片段
                </p>
              </div>
              <button className="secondary-button" onClick={closeKnowledgeBaseDocumentPreview} type="button">
                关闭
              </button>
            </div>
            {knowledgeBaseDocumentPreviewLoading ? (
              <div className="feedback-banner">正在加载文档内容...</div>
            ) : (
              <div className="document-preview-body">
                {knowledgeBaseDocumentChunks.map((chunk) => (
                  <article className="document-preview-chunk" key={chunk.id}>
                    <div className="document-preview-chunk-meta">
                      <strong>{chunk.title || knowledgeBaseDocumentPreview.title}</strong>
                      <span>片段 #{chunk.chunk_index}</span>
                    </div>
                    <pre>{chunk.content}</pre>
                  </article>
                ))}
                {knowledgeBaseDocumentChunks.length === 0 ? (
                  <div className="feedback-banner">这份文档暂时没有可展示的内容片段。</div>
                ) : null}
              </div>
            )}
          </section>
        </div>
      ) : null}

      {noticeDialog ? (
        <div
          className="notice-dialog-backdrop"
          onClick={(event) => {
            if (event.target === event.currentTarget) {
              closeNoticeDialog();
            }
          }}
        >
          <section className="notice-dialog-card">
            <div className="notice-dialog-header">
              <span className={`pill ${noticeDialogToneClass(noticeDialog.tone)}`}>{noticeDialogTitleTag(noticeDialog.tone)}</span>
              <button className="secondary-button small-button" onClick={closeNoticeDialog} type="button">
                关闭
              </button>
            </div>
            <div className="notice-dialog-body">
              <h3>{noticeDialog.title}</h3>
              <p>{noticeDialog.message}</p>
            </div>
            <div className="button-row notice-dialog-actions">
              {noticeDialog.actionLabel && noticeDialog.actionHref ? (
                <a className="primary-button" href={noticeDialog.actionHref} onClick={closeNoticeDialog}>
                  {noticeDialog.actionLabel}
                </a>
              ) : null}
              <button className="secondary-button" onClick={closeNoticeDialog} type="button">
                我知道了
              </button>
            </div>
          </section>
        </div>
      ) : null}

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

function LearningCurveChart(props: { points: Array<{ date: string; retention_rate: number; review_count: number }> }) {
  if (props.points.length === 0) {
    return <div className="feedback-banner">还没有复习记录，记忆曲线会在你开始复习后自动生成。</div>;
  }

  const width = 640;
  const height = 220;
  const paddingX = 24;
  const paddingY = 24;
  const usableWidth = width - paddingX * 2;
  const usableHeight = height - paddingY * 2;
  const stepX = props.points.length > 1 ? usableWidth / (props.points.length - 1) : usableWidth;
  const points = props.points.map((item, index) => {
    const x = paddingX + index * stepX;
    const y = paddingY + usableHeight - Math.max(0, Math.min(1, item.retention_rate)) * usableHeight;
    return { ...item, x, y };
  });
  const linePath = points
    .map((item, index) => `${index === 0 ? "M" : "L"} ${item.x.toFixed(2)} ${item.y.toFixed(2)}`)
    .join(" ");

  return (
    <div className="learning-curve-card">
      <svg aria-label="学习记忆曲线" className="learning-curve-svg" viewBox={`0 0 ${width} ${height}`} role="img">
        <path className="learning-curve-grid" d={`M ${paddingX} ${paddingY} H ${width - paddingX}`} />
        <path className="learning-curve-grid" d={`M ${paddingX} ${paddingY + usableHeight / 2} H ${width - paddingX}`} />
        <path className="learning-curve-grid" d={`M ${paddingX} ${height - paddingY} H ${width - paddingX}`} />
        <path className="learning-curve-line" d={linePath} />
        {points.map((item) => (
          <g key={item.date}>
            <circle className="learning-curve-point" cx={item.x} cy={item.y} r={4} />
          </g>
        ))}
      </svg>
      <div className="learning-curve-labels">
        {points.map((item) => (
          <span key={item.date}>
            {item.date.slice(5)} · {formatPercent(item.retention_rate)}
          </span>
        ))}
      </div>
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

function formatPercent(value: number) {
  return `${Math.round(Math.max(0, value) * 100)}%`;
}

function formatFileSize(value: number) {
  if (value <= 0) {
    return "0 B";
  }
  if (value < 1024) {
    return `${value} B`;
  }
  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)} KB`;
  }
  return `${(value / (1024 * 1024)).toFixed(1)} MB`;
}

function formatDateTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }
  return date.toLocaleString("zh-CN");
}

function formatMembershipExpiry(membership?: SubscriptionStatus | null) {
  if (!membership?.current_period_end) {
    return "";
  }
  return `有效期至 ${formatDateTime(membership.current_period_end)}`;
}

function formatLearningReviewStatus(value?: string, isDue?: boolean) {
  if (!value) {
    return "尚未安排复习";
  }
  const text = formatDateTime(value);
  if (isDue) {
    return `待复习 · ${text}`;
  }
  return `下次复习 · ${text}`;
}

function getCurrentHash() {
  if (typeof window === "undefined") {
    return "#home";
  }
  return window.location.hash || "#home";
}

function resolvePublicView(hash: string): PublicView {
  if (hash === "#mcp-market") {
    return "market";
  }
  if (hash === "#mcp") {
    return "mcp";
  }
  if (hash === "#profile" || hash === "#account" || hash === "#plans" || hash.startsWith("#profile-")) {
    return "profile";
  }
  return "home";
}

function resolveProfileWorkspaceTab(hash: string): ProfileWorkspaceTab {
  switch (hash) {
    case "#profile-memberships":
      return "memberships";
    case "#profile-orders":
      return "orders";
    case "#profile-invite":
      return "invite";
    case "#profile-knowledge-base":
      return "knowledge-base";
    case "#profile-api-tools":
      return "api-tools";
    case "#profile-xiaomi":
      return "xiaomi";
    case "#plans":
      return "plans";
    case "#profile-insights":
      return "insights";
    default:
      return "overview";
  }
}

function resolveInitialInviteCode() {
  if (typeof window === "undefined") {
    return "";
  }

  const search = new URLSearchParams(window.location.search);
  const inviteCode = (search.get("invite_code") || search.get("invite") || search.get("code") || "").trim();
  if (inviteCode) {
    window.sessionStorage.setItem(inviteCodeSessionStorageKey, inviteCode);
    return inviteCode;
  }

  return window.sessionStorage.getItem(inviteCodeSessionStorageKey)?.trim() || "";
}

function clearInviteQueryParam() {
  if (typeof window === "undefined") {
    return;
  }

  const url = new URL(window.location.href);
  url.searchParams.delete("invite_code");
  url.searchParams.delete("invite");
  url.searchParams.delete("code");
  window.history.replaceState({}, "", url.toString());
}

function buildInviteRegistrationLink(inviteCode: string) {
  const code = inviteCode.trim();
  if (!code || typeof window === "undefined") {
    return "";
  }

  const url = new URL(window.location.origin + window.location.pathname);
  url.searchParams.set("invite", code);
  url.hash = "profile";
  return url.toString();
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

function subscriptionStatusClass(status?: string) {
  switch (status) {
    case "active":
      return "pill-success";
    case "expired":
      return "pill-muted";
    case "pending":
      return "pill-warning";
    case "cancelled":
      return "pill-danger";
    default:
      return "pill-muted";
  }
}

function classificationBadgeLabel(item: ClassificationStat) {
  if (item.requires_membership) {
    return "会员专享";
  }
  if (item.has_member_content) {
    return "含会员内容";
  }
  return "";
}

function formatClassificationButtonLabel(item: ClassificationStat) {
  const badge = classificationBadgeLabel(item);
  if (!badge) {
    return item.name;
  }
  return `${item.name} · ${badge}`;
}

function formatClassificationOptionLabel(item: ClassificationStat) {
  const badge = classificationBadgeLabel(item);
  if (!badge) {
    return item.name;
  }
  return `${item.name}（${badge}）`;
}

function noticeDialogToneClass(tone: NoticeDialogTone) {
  switch (tone) {
    case "success":
      return "pill-success";
    case "error":
      return "pill-danger";
    default:
      return "pill-warning";
  }
}

function noticeDialogTitleTag(tone: NoticeDialogTone) {
  switch (tone) {
    case "success":
      return "已完成";
    case "error":
      return "请处理";
    default:
      return "提示";
  }
}
