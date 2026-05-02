import { useDeferredValue, useEffect, useState, type FormEvent } from "react";

import {
  api,
  type AdminLearnerUser,
  type AdminRole,
  type AdminSession,
  type AdminSetupStatus,
  type AdminUser,
  type CaptchaChallenge,
  type CatalogStats,
  type PagedAdminUsers,
  type PagedCategories,
  type PagedGrades,
  type PagedLearnerUsers,
  type PagedPaymentOrders,
  type PagedSubscriptions,
  type PagedWords,
  type PaymentOrderStatus,
  type Plan,
  type SiteSetting,
  type Subject,
  type SubscriptionStatus,
  type WechatPayConfig,
} from "./api";

const adminPageSize = 10;
const adminSessionStorageKey = "brights_admin_session";
const adminUIStateStorageKey = "brights_admin_ui_state";

type AdminSection = "dashboard" | "import" | "catalog" | "site" | "payments" | "memberships" | "learners" | "admins";

const defaultSiteForm: SiteSetting = {
  site_name: "Brights 英语单词学习站",
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

export default function AdminPortal() {
  const persistedUIState = readStoredState<{
    activeSection?: AdminSection;
    adminUserPage?: number;
    learnerPage?: number;
    wordPage?: number;
    categoryPage?: number;
    gradePage?: number;
    paymentPage?: number;
    subscriptionPage?: number;
    adminUserQuery?: string;
    learnerQuery?: string;
    wordQuery?: string;
    categoryQuery?: string;
    gradeQuery?: string;
    paymentQuery?: string;
    subscriptionQuery?: string;
    learnerStatusFilter?: string;
    paymentStatusFilter?: string;
    subscriptionStatusFilter?: string;
    subjectFilter?: string;
  }>(adminUIStateStorageKey, {});

  const [token, setToken] = useState("");
  const [session, setSession] = useState<AdminSession | null>(null);
  const [currentAdmin, setCurrentAdmin] = useState<AdminUser | null>(null);
  const [setupStatus, setSetupStatus] = useState<AdminSetupStatus | null>(null);
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [plans, setPlans] = useState<Plan[]>([]);
  const [roles, setRoles] = useState<AdminRole[]>([]);
  const [adminUsers, setAdminUsers] = useState<PagedAdminUsers | null>(null);
  const [learners, setLearners] = useState<PagedLearnerUsers | null>(null);
  const [words, setWords] = useState<PagedWords | null>(null);
  const [categories, setCategories] = useState<PagedCategories | null>(null);
  const [grades, setGrades] = useState<PagedGrades | null>(null);
  const [stats, setStats] = useState<CatalogStats | null>(null);
  const [siteSettings, setSiteSettings] = useState<SiteSetting | null>(null);
  const [wechatPayConfig, setWechatPayConfig] = useState<WechatPayConfig | null>(null);
  const [wechatPayConfigExists, setWechatPayConfigExists] = useState(false);
  const [paymentOrders, setPaymentOrders] = useState<PagedPaymentOrders | null>(null);
  const [subscriptions, setSubscriptions] = useState<PagedSubscriptions | null>(null);
  const [selectedOrderDetail, setSelectedOrderDetail] = useState<PaymentOrderStatus | null>(null);
  const [selectedSubscription, setSelectedSubscription] = useState<SubscriptionStatus | null>(null);
  const [authLoading, setAuthLoading] = useState(true);
  const [dataLoading, setDataLoading] = useState(false);
  const [authError, setAuthError] = useState("");
  const [dataError, setDataError] = useState("");
  const [notice, setNotice] = useState("");
  const [busyAction, setBusyAction] = useState("");
  const [reloadKey, setReloadKey] = useState(0);
  const [activeSection, setActiveSection] = useState<AdminSection>(persistedUIState.activeSection ?? "dashboard");

  const [setupForm, setSetupForm] = useState({
    username: "",
    displayName: "",
    password: "",
    confirmPassword: "",
  });
  const [loginForm, setLoginForm] = useState({ username: "", password: "" });
  const [loginCaptcha, setLoginCaptcha] = useState<CaptchaChallenge | null>(null);
  const [loginCaptchaAnswer, setLoginCaptchaAnswer] = useState("");
  const [loginCaptchaLoading, setLoginCaptchaLoading] = useState(false);
  const [passwordForm, setPasswordForm] = useState({
    oldPassword: "",
    newPassword: "",
    confirmPassword: "",
  });
  const [importForm, setImportForm] = useState({
    file: null as File | null,
    fileName: "",
    subjectKey: "english",
    replace: true,
  });
  const [subjectForm, setSubjectForm] = useState({
    key: "",
    name: "",
    description: "",
    featured: false,
  });
  const [categoryForm, setCategoryForm] = useState({
    subjectKey: "english",
    kind: "topic",
    key: "",
    name: "",
    description: "",
  });
  const [gradeForm, setGradeForm] = useState({
    key: "",
    name: "",
    stage: "general",
    description: "",
  });
  const [siteForm, setSiteForm] = useState<SiteSetting>(defaultSiteForm);
  const [adminEditor, setAdminEditor] = useState({
    id: 0,
    username: "",
    displayName: "",
    password: "",
    role: "content_admin",
    status: "active",
    isSuper: false,
  });
  const [learnerEditor, setLearnerEditor] = useState({
    id: 0,
    username: "",
    displayName: "",
    status: "active",
  });
  const [roleEditor, setRoleEditor] = useState({
    id: 0,
    key: "",
    name: "",
    description: "",
    permissions: "admin.read\ncatalog.read\npayment.read\nplan.read",
    sort: "0",
  });
  const [wechatPayForm, setWechatPayForm] = useState({
    authMode: "public_key",
    mchId: "",
    appId: "",
    merchantSerialNo: "",
    apiV3Key: "",
    notifyURL: "",
    descriptionPrefix: "Brights 学习会员",
    timeExpireMinutes: "30",
    wechatPayPublicKeyID: "",
    wechatPayPublicKey: "",
    keyPem: "",
  });
  const [planEditor, setPlanEditor] = useState({
    id: 0,
    key: "",
    name: "",
    billingMode: "monthly",
    priceYuan: "19.90",
    description: "",
    recommended: false,
    paymentChannels: "wechat",
    features: "高频词学习\n场景分类\n持续更新",
  });
  const [subscriptionEditor, setSubscriptionEditor] = useState({
    id: 0,
    customerRef: "",
    planKey: "",
    subjectKey: "",
    status: "active",
    autoRenew: false,
    startedAt: "",
    currentPeriodStart: "",
    currentPeriodEnd: "",
    cancelledAt: "",
  });

  const [adminUserPage, setAdminUserPage] = useState(persistedUIState.adminUserPage ?? 1);
  const [learnerPage, setLearnerPage] = useState(persistedUIState.learnerPage ?? 1);
  const [wordPage, setWordPage] = useState(persistedUIState.wordPage ?? 1);
  const [categoryPage, setCategoryPage] = useState(persistedUIState.categoryPage ?? 1);
  const [gradePage, setGradePage] = useState(persistedUIState.gradePage ?? 1);
  const [paymentPage, setPaymentPage] = useState(persistedUIState.paymentPage ?? 1);
  const [subscriptionPage, setSubscriptionPage] = useState(persistedUIState.subscriptionPage ?? 1);
  const [adminUserQuery, setAdminUserQuery] = useState(persistedUIState.adminUserQuery ?? "");
  const [learnerQuery, setLearnerQuery] = useState(persistedUIState.learnerQuery ?? "");
  const [wordQuery, setWordQuery] = useState(persistedUIState.wordQuery ?? "");
  const [categoryQuery, setCategoryQuery] = useState(persistedUIState.categoryQuery ?? "");
  const [gradeQuery, setGradeQuery] = useState(persistedUIState.gradeQuery ?? "");
  const [paymentQuery, setPaymentQuery] = useState(persistedUIState.paymentQuery ?? "");
  const [subscriptionQuery, setSubscriptionQuery] = useState(persistedUIState.subscriptionQuery ?? "");
  const [learnerStatusFilter, setLearnerStatusFilter] = useState(persistedUIState.learnerStatusFilter ?? "");
  const [paymentStatusFilter, setPaymentStatusFilter] = useState(persistedUIState.paymentStatusFilter ?? "");
  const [subscriptionStatusFilter, setSubscriptionStatusFilter] =
    useState(persistedUIState.subscriptionStatusFilter ?? "");
  const [subjectFilter, setSubjectFilter] = useState(persistedUIState.subjectFilter ?? "english");

  const deferredAdminUserQuery = useDeferredValue(adminUserQuery);
  const deferredLearnerQuery = useDeferredValue(learnerQuery);
  const deferredWordQuery = useDeferredValue(wordQuery);
  const deferredCategoryQuery = useDeferredValue(categoryQuery);
  const deferredGradeQuery = useDeferredValue(gradeQuery);
  const deferredPaymentQuery = useDeferredValue(paymentQuery);
  const deferredSubscriptionQuery = useDeferredValue(subscriptionQuery);

  const currentRole = roles.find((role) => role.key === currentAdmin?.role) ?? null;
  const permissionSet = new Set(currentRole?.permissions ?? []);
  const canManageAdmins = currentAdmin?.is_super ?? false;
  const canViewLearners =
    currentAdmin?.is_super === true ||
    permissionSet.has("*") ||
    permissionSet.has("learner.read") ||
    permissionSet.has("learner.write");
  const canManageLearners =
    currentAdmin?.is_super === true || permissionSet.has("*") || permissionSet.has("learner.write");
  const canViewSiteSettings =
    currentAdmin?.is_super === true ||
    permissionSet.has("*") ||
    permissionSet.has("site.read") ||
    permissionSet.has("site.write");
  const canManageSiteSettings =
    currentAdmin?.is_super === true || permissionSet.has("*") || permissionSet.has("site.write");
  const canViewPayments =
    currentAdmin?.is_super === true ||
    permissionSet.has("*") ||
    permissionSet.has("payment.read") ||
    permissionSet.has("payment.write");
  const canViewPlans =
    currentAdmin?.is_super === true ||
    permissionSet.has("*") ||
    permissionSet.has("plan.read") ||
    permissionSet.has("plan.write");
  const canManagePayments =
    currentAdmin?.is_super === true || permissionSet.has("*") || permissionSet.has("payment.write");
  const canManagePlans =
    currentAdmin?.is_super === true || permissionSet.has("*") || permissionSet.has("plan.write");
  const useWechatPayPublicKeyMode = wechatPayForm.authMode !== "auto_certificate";

  const formatPlanLabel = (planKey: string) => {
    const match = plans.find((item) => item.key === planKey);
    if (!match) {
      return planKey || "-";
    }
    return match.name;
  };

  const formatSubjectLabel = (key: string) => {
    const match = subjects.find((item) => item.key === key);
    if (match?.name) {
      return match.name;
    }
    return key || "-";
  };

  async function refreshLoginCaptcha() {
    setLoginCaptchaLoading(true);
    try {
      const captcha = await api.adminCaptcha();
      setLoginCaptcha(captcha);
      setLoginCaptchaAnswer("");
    } catch (err) {
      setAuthError((err as Error).message);
    } finally {
      setLoginCaptchaLoading(false);
    }
  }

  const persistSession = (nextSession: AdminSession) => {
    window.localStorage.setItem(adminSessionStorageKey, JSON.stringify(nextSession));
    setSession(nextSession);
    setCurrentAdmin(nextSession.admin);
    setToken(nextSession.access_token);
  };

  const clearSession = () => {
    window.localStorage.removeItem(adminSessionStorageKey);
    setSession(null);
    setCurrentAdmin(null);
    setToken("");
  };

  async function loadSetupStatus() {
    const status = await api.adminSetupStatus();
    setSetupStatus(status);
    return status;
  }

  useEffect(() => {
    let active = true;

    const restore = async () => {
      const raw = window.localStorage.getItem(adminSessionStorageKey);
      if (!raw) {
        try {
          const status = await api.adminSetupStatus();
          if (active) {
            setSetupStatus(status);
          }
        } catch (err) {
          if (active) {
            setAuthError((err as Error).message);
          }
        } finally {
          if (active) {
            setAuthLoading(false);
          }
        }
        return;
      }

      let savedToken = "";
      try {
        const parsed = JSON.parse(raw) as Partial<AdminSession>;
        savedToken = parsed.access_token ?? "";
      } catch {
        window.localStorage.removeItem(adminSessionStorageKey);
      }

      if (!savedToken) {
        try {
          const status = await api.adminSetupStatus();
          if (active) {
            setSetupStatus(status);
          }
        } catch (err) {
          if (active) {
            setAuthError((err as Error).message);
          }
        } finally {
          if (active) {
            setAuthLoading(false);
          }
        }
        return;
      }

      try {
        const nextSession = await api.adminRefresh(savedToken);
        if (!active) {
          return;
        }
        persistSession(nextSession);
        setSetupStatus({ initialized: true, admin_count: 1 });
      } catch {
        if (!active) {
          return;
        }
        clearSession();
        try {
          const status = await api.adminSetupStatus();
          if (active) {
            setSetupStatus(status);
          }
        } catch (err) {
          if (active) {
            setAuthError((err as Error).message);
          }
        }
      } finally {
        if (active) {
          setAuthLoading(false);
        }
      }
    };

    void restore();

    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (currentAdmin || setupStatus?.initialized !== true) {
      return;
    }
    void refreshLoginCaptcha();
  }, [currentAdmin, setupStatus?.initialized]);

  useEffect(() => {
    if (!token || !currentAdmin) {
      setSiteSettings(null);
      setLearners(null);
      setWechatPayConfig(null);
      setWechatPayConfigExists(false);
      setPaymentOrders(null);
      setSubscriptions(null);
      setSelectedOrderDetail(null);
      setSelectedSubscription(null);
      return;
    }

    let active = true;
    setDataLoading(true);
    setDataError("");

    Promise.all([
      api.getSubjects(),
      api.getPlans(),
      api.getStats(),
      api.adminRoles(token),
      api.adminUsers(token, {
        page: adminUserPage,
        pageSize: adminPageSize,
        query: deferredAdminUserQuery,
      }),
      api.adminWords(token, {
        page: wordPage,
        pageSize: adminPageSize,
        query: deferredWordQuery,
        subjectKey: subjectFilter,
      }),
      api.adminCategories(token, {
        page: categoryPage,
        pageSize: adminPageSize,
        query: deferredCategoryQuery,
        kind: "topic",
        subject: subjectFilter,
      }),
      api.adminGrades(token, {
        page: gradePage,
        pageSize: adminPageSize,
        query: deferredGradeQuery,
      }),
      canViewSiteSettings ? api.adminSiteSettings(token) : Promise.resolve(null),
      canViewLearners
        ? api.adminLearners(token, {
            page: learnerPage,
            pageSize: adminPageSize,
            query: deferredLearnerQuery,
            status: learnerStatusFilter,
          })
        : Promise.resolve(null),
      canViewPayments ? api.adminWechatPayConfig(token) : Promise.resolve({ exists: false }),
      canViewPayments
        ? api.adminPaymentOrders(token, {
            page: paymentPage,
            pageSize: adminPageSize,
            query: deferredPaymentQuery,
            status: paymentStatusFilter,
            subject: subjectFilter,
          })
        : Promise.resolve(null),
      canViewPayments
        ? api.adminSubscriptions(token, {
            page: subscriptionPage,
            pageSize: adminPageSize,
            query: deferredSubscriptionQuery,
            status: subscriptionStatusFilter,
            subject: subjectFilter,
          })
        : Promise.resolve(null),
    ])
      .then(
        ([
          subjectData,
          planData,
          statsData,
          roleData,
          userData,
          wordData,
          categoryData,
          gradeData,
          siteData,
          learnerData,
          configResult,
          paymentData,
          subscriptionData,
        ]) => {
          if (!active) {
            return;
          }
          setSubjects(subjectData);
          setPlans(planData);
          setStats(statsData);
          setRoles(roleData);
          setAdminUsers(userData);
          setWords(wordData);
          setCategories(categoryData);
          setGrades(gradeData);
          setSiteSettings((siteData as SiteSetting | null) ?? null);
          if (siteData) {
            setSiteForm(siteData as SiteSetting);
          }
          setLearners((learnerData as PagedLearnerUsers | null) ?? null);

          const payConfig = configResult as { exists: boolean; config?: WechatPayConfig };
          setWechatPayConfigExists(payConfig.exists);
          setWechatPayConfig(payConfig.config ?? null);
          setPaymentOrders((paymentData as PagedPaymentOrders | null) ?? null);
          setSubscriptions((subscriptionData as PagedSubscriptions | null) ?? null);
        },
      )
      .catch((err: Error) => {
        if (!active) {
          return;
        }
        if (looksLikeAuthError(err.message)) {
          clearSession();
          setAuthError("登录已失效，请重新登录。");
          return;
        }
        setDataError(err.message);
      })
      .finally(() => {
        if (active) {
          setDataLoading(false);
        }
      });

    return () => {
      active = false;
    };
  }, [
    adminUserPage,
    canViewLearners,
    canViewPayments,
    canViewSiteSettings,
    categoryPage,
    currentAdmin,
    deferredAdminUserQuery,
    deferredCategoryQuery,
    deferredGradeQuery,
    deferredLearnerQuery,
    deferredPaymentQuery,
    deferredSubscriptionQuery,
    deferredWordQuery,
    gradePage,
    learnerPage,
    learnerStatusFilter,
    paymentPage,
    paymentStatusFilter,
    reloadKey,
    subjectFilter,
    subscriptionPage,
    subscriptionStatusFilter,
    token,
    wordPage,
  ]);

  useEffect(() => {
    if (!subjects.length) {
      return;
    }
    const defaultSubjectKey = subjects[0].key;
    const hasSubject = (key: string) => subjects.some((item) => item.key === key);

    if (!hasSubject(subjectFilter)) {
      setSubjectFilter(defaultSubjectKey);
    }

    setImportForm((current) => ({
      ...current,
      subjectKey: hasSubject(current.subjectKey) ? current.subjectKey : defaultSubjectKey,
    }));
    setCategoryForm((current) => ({
      ...current,
      subjectKey: hasSubject(current.subjectKey) ? current.subjectKey : defaultSubjectKey,
    }));
  }, [subjectFilter, subjects]);

  useEffect(() => {
    if (!wechatPayConfigExists || !wechatPayConfig) {
      setWechatPayForm({
        authMode: "public_key",
        mchId: "",
        appId: "",
        merchantSerialNo: "",
        apiV3Key: "",
        notifyURL: "",
        descriptionPrefix: "Brights 学习会员",
        timeExpireMinutes: "30",
        wechatPayPublicKeyID: "",
        wechatPayPublicKey: "",
        keyPem: "",
      });
      return;
    }

    setWechatPayForm({
      authMode: wechatPayConfig.auth_mode || "public_key",
      mchId: wechatPayConfig.mch_id,
      appId: wechatPayConfig.app_id,
      merchantSerialNo: wechatPayConfig.merchant_serial_no,
      apiV3Key: wechatPayConfig.apiv3_key || "",
      notifyURL: wechatPayConfig.notify_url,
      descriptionPrefix: wechatPayConfig.description_prefix || "Brights 学习会员",
      timeExpireMinutes: String(wechatPayConfig.time_expire_minutes || 30),
      wechatPayPublicKeyID: wechatPayConfig.wechatpay_public_key_id,
      wechatPayPublicKey: wechatPayConfig.wechatpay_public_key || "",
      keyPem: wechatPayConfig.key_pem || "",
    });
  }, [wechatPayConfig, wechatPayConfigExists]);

  useEffect(() => {
    window.localStorage.setItem(
      adminUIStateStorageKey,
      JSON.stringify({
        activeSection,
        adminUserPage,
        learnerPage,
        wordPage,
        categoryPage,
        gradePage,
        paymentPage,
        subscriptionPage,
        adminUserQuery,
        learnerQuery,
        wordQuery,
        categoryQuery,
        gradeQuery,
        paymentQuery,
        subscriptionQuery,
        learnerStatusFilter,
        paymentStatusFilter,
        subscriptionStatusFilter,
        subjectFilter,
      }),
    );
  }, [
    activeSection,
    adminUserPage,
    learnerPage,
    wordPage,
    categoryPage,
    gradePage,
    paymentPage,
    subscriptionPage,
    adminUserQuery,
    learnerQuery,
    wordQuery,
    categoryQuery,
    gradeQuery,
    paymentQuery,
    subscriptionQuery,
    learnerStatusFilter,
    paymentStatusFilter,
    subscriptionStatusFilter,
    subjectFilter,
  ]);

  async function handleSetupBootstrap(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (setupForm.password !== setupForm.confirmPassword) {
      setAuthError("两次输入的密码不一致。");
      return;
    }

    setBusyAction("setup");
    setAuthError("");
    setNotice("");
    try {
      const nextSession = await api.adminSetupBootstrap({
        username: setupForm.username,
        password: setupForm.password,
        display_name: setupForm.displayName,
      });
      persistSession(nextSession);
      setSetupStatus({ initialized: true, admin_count: 1 });
      setSetupForm((current) => ({ ...current, password: "", confirmPassword: "" }));
      setNotice("超级管理员已创建，后台已自动登录。");
    } catch (err) {
      setAuthError((err as Error).message);
    } finally {
      setBusyAction("");
    }
  }

  async function handleLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!loginCaptcha?.captcha_id || !loginCaptchaAnswer.trim()) {
      setAuthError("请先填写图形验证码。");
      return;
    }

    setBusyAction("login");
    setAuthError("");
    setNotice("");

    try {
      const nextSession = await api.adminLogin(
        loginForm.username,
        loginForm.password,
        loginCaptcha.captcha_id,
        loginCaptchaAnswer,
      );
      persistSession(nextSession);
      setSetupStatus({ initialized: true, admin_count: Math.max(setupStatus?.admin_count ?? 0, 1) });
      setLoginForm((current) => ({ ...current, password: "" }));
      setLoginCaptchaAnswer("");
      setNotice("后台登录成功。");
    } catch (err) {
      setAuthError((err as Error).message);
      await refreshLoginCaptcha();
    } finally {
      setBusyAction("");
      setAuthLoading(false);
    }
  }

  async function handleRefreshSession() {
    if (!token) {
      return;
    }
    setBusyAction("refresh");
    setAuthError("");
    setNotice("");
    try {
      const nextSession = await api.adminRefresh(token);
      persistSession(nextSession);
      setNotice(`登录状态已续期，本次登录有效至 ${formatDateTime(nextSession.expires_at)}。`);
    } catch (err) {
      clearSession();
      setAuthError((err as Error).message);
      try {
        await loadSetupStatus();
      } catch {
        // ignore
      }
    } finally {
      setBusyAction("");
    }
  }

  async function handleLogout() {
    setBusyAction("logout");
    setAuthError("");
    setDataError("");
    setNotice("");
    try {
      if (token) {
        await api.adminLogout(token);
      }
    } catch {
      // ignore
    } finally {
      clearSession();
      setLoginCaptcha(null);
      setLoginCaptchaAnswer("");
      setBusyAction("");
      setNotice("已退出登录。");
      try {
        await loadSetupStatus();
      } catch {
        // ignore
      }
    }
  }

  async function handleChangePassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) {
      return;
    }
    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      setAuthError("两次输入的新密码不一致。");
      return;
    }

    setBusyAction("password");
    setAuthError("");
    setNotice("");
    try {
      await api.adminChangePassword(token, passwordForm.oldPassword, passwordForm.newPassword);
      setPasswordForm({
        oldPassword: "",
        newPassword: "",
        confirmPassword: "",
      });
      setNotice("登录密码已更新。");
    } catch (err) {
      setAuthError((err as Error).message);
    } finally {
      setBusyAction("");
    }
  }

  async function handleImportLocal(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) {
      return;
    }
    if (!importForm.file) {
      setDataError("请先选择要导入的文件。");
      return;
    }

    setBusyAction("import");
    setDataError("");
    setNotice("");
    try {
      const result = await api.adminImportLocal(token, {
        file: importForm.file,
        subject_key: importForm.subjectKey || subjectFilter || "english",
        replace: importForm.replace,
      });
      setReloadKey((current) => current + 1);
      setNotice(
        `导入完成：文件 ${result.path} 共处理 ${result.imported_count} 条学习内容，自动匹配或创建 ${result.created_categories} 个内容分组。`,
      );
    } catch (err) {
      setDataError((err as Error).message);
    } finally {
      setBusyAction("");
    }
  }

  async function handleCreateSubject(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) {
      return;
    }
    setBusyAction("subject");
    setDataError("");
    setNotice("");
    try {
      await api.adminCreateSubject(token, {
        key: subjectForm.key,
        name: subjectForm.name,
        description: subjectForm.description,
        sort: 0,
        featured: subjectForm.featured,
      });
      setSubjectForm({
        key: "",
        name: "",
        description: "",
        featured: false,
      });
      setReloadKey((current) => current + 1);
      setNotice("学习科目已创建。");
    } catch (err) {
      setDataError((err as Error).message);
    } finally {
      setBusyAction("");
    }
  }

  async function handleCreateCategory(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) {
      return;
    }
    setBusyAction("category");
    setDataError("");
    setNotice("");
    try {
      await api.adminCreateCategory(token, {
        subject_key: categoryForm.subjectKey || subjectFilter || "english",
        kind: categoryForm.kind,
        key: categoryForm.key,
        name: categoryForm.name,
        description: categoryForm.description,
        sort: 0,
        enabled: true,
      });
      setCategoryForm((current) => ({
        ...current,
        key: "",
        name: "",
        description: "",
      }));
      setCategoryPage(1);
      setReloadKey((current) => current + 1);
      setNotice("内容分组已创建。");
    } catch (err) {
      setDataError((err as Error).message);
    } finally {
      setBusyAction("");
    }
  }

  async function handleCreateGrade(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) {
      return;
    }
    setBusyAction("grade");
    setDataError("");
    setNotice("");
    try {
      await api.adminCreateGrade(token, {
        key: gradeForm.key,
        name: gradeForm.name,
        stage: gradeForm.stage,
        description: gradeForm.description,
        sort: 0,
        enabled: true,
      });
      setGradeForm({
        key: "",
        name: "",
        stage: "general",
        description: "",
      });
      setGradePage(1);
      setReloadKey((current) => current + 1);
      setNotice("学习阶段已创建。");
    } catch (err) {
      setDataError((err as Error).message);
    } finally {
      setBusyAction("");
    }
  }

  async function handleSaveSiteSettings(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token || !canManageSiteSettings) {
      return;
    }
    setBusyAction("site-settings");
    setDataError("");
    setNotice("");
    try {
      const saved = await api.adminSaveSiteSettings(token, siteForm);
      setSiteSettings(saved);
      setSiteForm(saved);
      setNotice("前台展示文案和搜索优化设置已保存。");
    } catch (err) {
      setDataError((err as Error).message);
    } finally {
      setBusyAction("");
    }
  }

  async function handleSavePlan(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token || !canManagePlans) {
      return;
    }
    const priceCents = yuanToCents(planEditor.priceYuan);
    if (priceCents <= 0) {
      setDataError("会员方案售价必须大于 0 元。");
      return;
    }

    setBusyAction("plan");
    setDataError("");
    setNotice("");
    try {
      const payload = {
        key: planEditor.key,
        name: planEditor.name,
        billing_mode: planEditor.billingMode,
        price_cents: priceCents,
        description: planEditor.description,
        recommended: planEditor.recommended,
        payment_channels: parseLineList(planEditor.paymentChannels),
        features: parseLineList(planEditor.features),
      };

      if (planEditor.id > 0) {
        await api.adminUpdatePlan(token, planEditor.id, {
          name: payload.name,
          billing_mode: payload.billing_mode,
          price_cents: payload.price_cents,
          description: payload.description,
          recommended: payload.recommended,
          payment_channels: payload.payment_channels,
          features: payload.features,
        });
        setNotice("会员方案已更新。");
      } else {
        await api.adminCreatePlan(token, payload);
        setNotice("会员方案已创建。");
      }

      resetPlanEditor();
      setReloadKey((current) => current + 1);
    } catch (err) {
      setDataError((err as Error).message);
    } finally {
      setBusyAction("");
    }
  }

  async function handleDeletePlan(id: number) {
    if (!token || !canManagePlans || id <= 0) {
      return;
    }
    if (!window.confirm("确定删除这个会员方案吗？如果已经关联订单或会员记录，系统会阻止删除。")) {
      return;
    }
    setBusyAction("plan-delete");
    setDataError("");
    setNotice("");
    try {
      await api.adminDeletePlan(token, id);
      if (planEditor.id === id) {
        resetPlanEditor();
      }
      setReloadKey((current) => current + 1);
      setNotice("会员方案已删除。");
    } catch (err) {
      setDataError((err as Error).message);
    } finally {
      setBusyAction("");
    }
  }

  async function handleLoadOrderDetail(orderNo: string) {
    if (!token || !canViewPayments) {
      return;
    }
    setBusyAction(`order-${orderNo}`);
    setDataError("");
    try {
      const detail = await api.adminPaymentOrderDetail(token, orderNo);
      setSelectedOrderDetail(detail);
    } catch (err) {
      setDataError((err as Error).message);
    } finally {
      setBusyAction("");
    }
  }

  async function startEditSubscription(item: SubscriptionStatus) {
    if (!token || !canViewPayments) {
      return;
    }
    setBusyAction("subscription-detail");
    setDataError("");
    try {
      const detail = await api.adminSubscription(token, item.id);
      setSelectedSubscription(detail);
      setSubscriptionEditor({
        id: detail.id,
        customerRef: detail.customer_ref,
        planKey: detail.plan_key,
        subjectKey: detail.subject_key,
        status: detail.status,
        autoRenew: detail.auto_renew,
        startedAt: toDateTimeLocalValue(detail.started_at),
        currentPeriodStart: toDateTimeLocalValue(detail.current_period_start),
        currentPeriodEnd: toDateTimeLocalValue(detail.current_period_end),
        cancelledAt: toDateTimeLocalValue(detail.cancelled_at),
      });
    } catch (err) {
      setDataError((err as Error).message);
    } finally {
      setBusyAction("");
    }
  }

  async function handleSaveSubscription(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token || !canManagePayments || subscriptionEditor.id <= 0) {
      return;
    }
    setBusyAction("subscription-save");
    setDataError("");
    setNotice("");
    try {
      const updated = await api.adminUpdateSubscription(token, subscriptionEditor.id, {
        plan_key: subscriptionEditor.planKey,
        status: subscriptionEditor.status,
        auto_renew: subscriptionEditor.autoRenew,
        started_at: fromDateTimeLocalValue(subscriptionEditor.startedAt),
        current_period_start: fromDateTimeLocalValue(subscriptionEditor.currentPeriodStart),
        current_period_end: fromDateTimeLocalValue(subscriptionEditor.currentPeriodEnd),
        cancelled_at: fromDateTimeLocalValue(subscriptionEditor.cancelledAt),
        clear_started_at: subscriptionEditor.startedAt === "",
        clear_current_period_start: subscriptionEditor.currentPeriodStart === "",
        clear_current_period_end: subscriptionEditor.currentPeriodEnd === "",
        clear_cancelled_at: subscriptionEditor.cancelledAt === "",
      });
      setSelectedSubscription(updated);
      setSubscriptionEditor({
        id: updated.id,
        customerRef: updated.customer_ref,
        planKey: updated.plan_key,
        subjectKey: updated.subject_key,
        status: updated.status,
        autoRenew: updated.auto_renew,
        startedAt: toDateTimeLocalValue(updated.started_at),
        currentPeriodStart: toDateTimeLocalValue(updated.current_period_start),
        currentPeriodEnd: toDateTimeLocalValue(updated.current_period_end),
        cancelledAt: toDateTimeLocalValue(updated.cancelled_at),
      });
      setReloadKey((current) => current + 1);
      setNotice("会员服务状态已更新。");
    } catch (err) {
      setDataError((err as Error).message);
    } finally {
      setBusyAction("");
    }
  }

  async function handleSaveWechatPayConfig(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token || !canManagePayments) {
      return;
    }
    setBusyAction("wechatpay");
    setDataError("");
    setNotice("");
    try {
      const nextAPIv3Key = wechatPayForm.apiV3Key.trim();
      const nextWechatPublicKey = wechatPayForm.wechatPayPublicKey.trim();
      const nextMerchantPrivateKey = wechatPayForm.keyPem.trim();

      const config = await api.adminSaveWechatPayConfig(token, {
        auth_mode: wechatPayForm.authMode,
        mch_id: wechatPayForm.mchId,
        app_id: wechatPayForm.appId,
        merchant_serial_no: wechatPayForm.merchantSerialNo,
        apiv3_key: nextAPIv3Key || undefined,
        clear_apiv3_key: nextAPIv3Key === "" && !!wechatPayConfig?.has_apiv3_key,
        notify_url: wechatPayForm.notifyURL,
        description_prefix: wechatPayForm.descriptionPrefix,
        time_expire_minutes: Number.parseInt(wechatPayForm.timeExpireMinutes, 10) || 30,
        wechatpay_public_key_id: wechatPayForm.wechatPayPublicKeyID,
        wechatpay_public_key: nextWechatPublicKey || undefined,
        clear_wechatpay_public_key: nextWechatPublicKey === "" && !!wechatPayConfig?.has_wechatpay_public_key,
        key_pem: nextMerchantPrivateKey || undefined,
        clear_key_pem: nextMerchantPrivateKey === "" && !!wechatPayConfig?.has_key_pem,
      });
      setWechatPayConfig(config);
      setWechatPayConfigExists(true);
      setNotice("微信收款配置已保存。");
    } catch (err) {
      setDataError((err as Error).message);
    } finally {
      setBusyAction("");
    }
  }

  async function handleSaveLearnerUser(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token || !canManageLearners || learnerEditor.id <= 0) {
      return;
    }
    setBusyAction("learner-user");
    setDataError("");
    setNotice("");
    try {
      await api.adminUpdateLearner(token, learnerEditor.id, {
        display_name: learnerEditor.displayName,
        status: learnerEditor.status,
      });
      resetLearnerEditor();
      setReloadKey((current) => current + 1);
      setNotice("学员资料已更新。");
    } catch (err) {
      setDataError((err as Error).message);
    } finally {
      setBusyAction("");
    }
  }

  async function handleSaveAdminUser(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token || !canManageAdmins) {
      return;
    }
    setBusyAction("admin-user");
    setDataError("");
    setNotice("");
    try {
      if (adminEditor.id > 0) {
        await api.adminUpdateUser(token, adminEditor.id, {
          display_name: adminEditor.displayName,
          password: adminEditor.password || undefined,
          role: adminEditor.role,
          status: adminEditor.status,
          is_super: adminEditor.isSuper,
        });
        setNotice("后台账号已更新。");
      } else {
        await api.adminCreateUser(token, {
          username: adminEditor.username,
          password: adminEditor.password,
          display_name: adminEditor.displayName,
          role: adminEditor.role,
          status: adminEditor.status,
          is_super: adminEditor.isSuper,
        });
        setNotice("后台账号已创建。");
      }
      resetAdminEditor();
      setAdminUserPage(1);
      setReloadKey((current) => current + 1);
    } catch (err) {
      setDataError((err as Error).message);
    } finally {
      setBusyAction("");
    }
  }

  async function handleSaveRole(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token || !canManageAdmins) {
      return;
    }
    setBusyAction("admin-role");
    setDataError("");
    setNotice("");
    try {
      const permissions = parsePermissionInput(roleEditor.permissions);
      const sort = Number.parseInt(roleEditor.sort, 10) || 0;

      if (roleEditor.id > 0) {
        await api.adminUpdateRole(token, roleEditor.id, {
          name: roleEditor.name,
          description: roleEditor.description,
          permissions,
          sort,
        });
        setNotice("岗位权限模板已更新。");
      } else {
        await api.adminCreateRole(token, {
          key: roleEditor.key,
          name: roleEditor.name,
          description: roleEditor.description,
          permissions,
          sort,
        });
        setNotice("岗位权限模板已创建。");
      }
      resetRoleEditor();
      setReloadKey((current) => current + 1);
    } catch (err) {
      setDataError((err as Error).message);
    } finally {
      setBusyAction("");
    }
  }

  function startEditPlan(plan: Plan) {
    setPlanEditor({
      id: plan.id ?? 0,
      key: plan.key,
      name: plan.name,
      billingMode: plan.billing_mode,
      priceYuan: centsToYuanInput(plan.price_cents),
      description: plan.description,
      recommended: plan.recommended,
      paymentChannels: plan.payment_channels.join("\n"),
      features: plan.features.join("\n"),
    });
    setActiveSection("payments");
  }

  function startEditLearner(item: AdminLearnerUser) {
    setLearnerEditor({
      id: item.id,
      username: item.username,
      displayName: item.display_name,
      status: item.status,
    });
    setActiveSection("learners");
  }

  function startEditAdminUser(item: AdminUser) {
    setAdminEditor({
      id: item.id,
      username: item.username,
      displayName: item.display_name,
      password: "",
      role: item.role,
      status: item.status,
      isSuper: item.is_super,
    });
    setActiveSection("admins");
  }

  function startEditRole(role: AdminRole) {
    if (role.system) {
      setDataError("系统内置岗位模板只能查看，不能直接在这里修改。");
      return;
    }
    setRoleEditor({
      id: role.id,
      key: role.key,
      name: role.name,
      description: role.description,
      permissions: role.permissions.join("\n"),
      sort: String(role.sort),
    });
    setActiveSection("admins");
  }

  function resetPlanEditor() {
    setPlanEditor({
      id: 0,
      key: "",
      name: "",
      billingMode: "monthly",
      priceYuan: "19.90",
      description: "",
      recommended: false,
      paymentChannels: "wechat",
      features: "高频词学习\n场景分类\n持续更新",
    });
  }

  function resetSubscriptionEditor() {
    setSelectedSubscription(null);
    setSubscriptionEditor({
      id: 0,
      customerRef: "",
      planKey: "",
      subjectKey: "",
      status: "active",
      autoRenew: false,
      startedAt: "",
      currentPeriodStart: "",
      currentPeriodEnd: "",
      cancelledAt: "",
    });
  }

  function resetLearnerEditor() {
    setLearnerEditor({
      id: 0,
      username: "",
      displayName: "",
      status: "active",
    });
  }

  function resetAdminEditor() {
    setAdminEditor({
      id: 0,
      username: "",
      displayName: "",
      password: "",
      role: "content_admin",
      status: "active",
      isSuper: false,
    });
  }

  function resetRoleEditor() {
    setRoleEditor({
      id: 0,
      key: "",
      name: "",
      description: "",
      permissions: "admin.read\ncatalog.read\npayment.read\nplan.read",
      sort: "0",
    });
  }

  if (authLoading) {
    return (
      <div className="admin-auth-shell">
        <section className="auth-card">
          <p className="section-eyebrow">Brights 管理后台</p>
          <h1>正在确认后台访问状态</h1>
          <p>正在恢复上一次登录，或读取首次开通所需的信息。</p>
        </section>
      </div>
    );
  }

  if (!currentAdmin || !token) {
    const needsSetup = setupStatus?.initialized === false;
    return (
      <div className="admin-auth-shell">
        <section className="auth-card">
          <p className="section-eyebrow">Brights 管理后台</p>
          <h1>{needsSetup ? "首次开通后台管理" : "后台登录"}</h1>
          <p>
            {needsSetup
              ? "当前还没有后台账号，请先由运营负责人设置第一个超级管理员。首次开通后，后续账号都可以在后台继续新增和管理。"
              : "使用后台账号登录后，即可统一维护学习内容、站点展示、收费方案、会员服务和学员账号。"}
          </p>

          {notice ? <div className="feedback-banner feedback-success">{notice}</div> : null}
          {authError ? <div className="feedback-banner feedback-error">{authError}</div> : null}

          {needsSetup ? (
            <form className="setup-form" onSubmit={handleSetupBootstrap}>
              <label className="form-field">
                <span>后台登录账号</span>
                <input
                  value={setupForm.username}
                  onChange={(event) => {
                    setSetupForm((current) => ({ ...current, username: event.target.value }));
                  }}
                  placeholder="superadmin"
                />
              </label>
              <label className="form-field">
                <span>负责人姓名</span>
                <input
                  value={setupForm.displayName}
                  onChange={(event) => {
                    setSetupForm((current) => ({ ...current, displayName: event.target.value }));
                  }}
                  placeholder="站点运营负责人"
                />
              </label>
              <label className="form-field">
                <span>登录密码</span>
                <input
                  type="password"
                  value={setupForm.password}
                  onChange={(event) => {
                    setSetupForm((current) => ({ ...current, password: event.target.value }));
                  }}
                  placeholder="至少 8 位"
                />
              </label>
              <label className="form-field">
                <span>确认密码</span>
                <input
                  type="password"
                  value={setupForm.confirmPassword}
                  onChange={(event) => {
                    setSetupForm((current) => ({ ...current, confirmPassword: event.target.value }));
                  }}
                  placeholder="请再输入一次密码"
                />
              </label>
              <button className="primary-button" disabled={busyAction === "setup"} type="submit">
                {busyAction === "setup" ? "开通中..." : "创建超级管理员"}
              </button>
            </form>
          ) : (
            <form className="setup-form" onSubmit={handleLogin}>
              <label className="form-field">
                <span>后台登录账号</span>
                <input
                  value={loginForm.username}
                  onChange={(event) => {
                    setLoginForm((current) => ({ ...current, username: event.target.value }));
                  }}
                  placeholder="superadmin"
                />
              </label>
              <label className="form-field">
                <span>登录密码</span>
                <input
                  type="password"
                  value={loginForm.password}
                  onChange={(event) => {
                    setLoginForm((current) => ({ ...current, password: event.target.value }));
                  }}
                  placeholder="请输入后台登录密码"
                />
              </label>
              <div className="form-grid-two">
                <label className="form-field">
                  <span>图形验证码</span>
                  <input
                    value={loginCaptchaAnswer}
                    onChange={(event) => {
                      setLoginCaptchaAnswer(event.target.value);
                    }}
                    placeholder="请输入图中的字符"
                  />
                </label>
                <div className="form-field">
                  <span>验证码图片</span>
                  <div className="button-row">
                    <img alt="图形验证码" className="captcha-image" src={loginCaptcha?.image_data || ""} />
                    <button
                      className="secondary-button small-button"
                      disabled={loginCaptchaLoading}
                      onClick={() => {
                        void refreshLoginCaptcha();
                      }}
                      type="button"
                    >
                      {loginCaptchaLoading ? "刷新中..." : "换一张"}
                    </button>
                  </div>
                </div>
              </div>
              <button className="primary-button" disabled={busyAction === "login"} type="submit">
                {busyAction === "login" ? "登录中..." : "进入运营后台"}
              </button>
            </form>
          )}
        </section>
      </div>
    );
  }

  const navItems: Array<{ key: AdminSection; label: string; hidden?: boolean }> = [
    { key: "dashboard", label: "运营看板" },
    { key: "import", label: "内容导入" },
    { key: "catalog", label: "内容整理" },
    { key: "site", label: "站点展示", hidden: !canViewSiteSettings },
    { key: "payments", label: "收费方案", hidden: !canViewPayments && !canViewPlans },
    { key: "memberships", label: "会员服务", hidden: !canViewPayments },
    { key: "learners", label: "学员账号", hidden: !canViewLearners },
    { key: "admins", label: "团队与权限" },
  ];

  return (
    <div className="admin-shell">
      <header className="admin-topbar">
        <div className="admin-topbar-left">
          <div className="site-brand">
            <span className="site-logo">B</span>
            <div>
              <strong>{siteSettings?.site_name || "Brights 管理后台"}</strong>
              <p>内容上架、收费运营与学员服务中心</p>
            </div>
          </div>
          <div className="admin-topbar-meta">
            <span>{currentAdmin.display_name}</span>
            <span>{adminRoleLabel(currentAdmin.role, currentRole?.name)}</span>
            {session?.expires_at ? <span>本次登录有效至：{formatDateTime(session.expires_at)}</span> : null}
          </div>
        </div>
        <div className="button-row">
          <button className="secondary-button" disabled={busyAction === "refresh"} onClick={handleRefreshSession} type="button">
            {busyAction === "refresh" ? "处理中..." : "延长登录有效期"}
          </button>
          <button className="secondary-button" disabled={busyAction === "logout"} onClick={handleLogout} type="button">
            退出登录
          </button>
        </div>
      </header>

      <div className="admin-layout">
        <aside className="admin-sidebar">
          <div className="admin-sidebar-section">
            <h3>后台菜单</h3>
            <div className="admin-nav">
              {navItems
                .filter((item) => !item.hidden)
                .map((item) => (
                  <button
                    className={item.key === activeSection ? "admin-nav-link admin-nav-link-active" : "admin-nav-link"}
                    key={item.key}
                    onClick={() => setActiveSection(item.key)}
                    type="button"
                  >
                    {item.label}
                  </button>
                ))}
            </div>
          </div>

          <div className="admin-sidebar-section">
            <h3>正在管理的科目</h3>
            <select
              value={subjectFilter}
              onChange={(event) => {
                setSubjectFilter(event.target.value);
                setWordPage(1);
                setCategoryPage(1);
                setPaymentPage(1);
                setSubscriptionPage(1);
              }}
            >
              {subjects.map((subject) => (
                <option key={subject.key} value={subject.key}>
                  {subject.name}
                </option>
              ))}
            </select>
          </div>

          <div className="admin-sidebar-section">
            <h3>本页提醒</h3>
            <p>上传 CSV 后，系统会根据文件中的分类字段自动适配内容分组。后续新增其他学科时，也可以继续沿用同一套运营流程。</p>
          </div>
        </aside>

        <main className="admin-main">
          {notice ? <div className="feedback-banner feedback-success">{notice}</div> : null}
          {authError ? <div className="feedback-banner feedback-error">{authError}</div> : null}
          {dataError ? <div className="feedback-banner feedback-error">{dataError}</div> : null}
          {dataLoading ? <div className="feedback-banner">正在同步后台数据...</div> : null}

          {activeSection === "dashboard" ? (
            <section className="admin-section">
              <div className="section-header">
                <div>
                  <p className="section-eyebrow">运营看板</p>
                  <h1>运营总览</h1>
                </div>
              </div>

              <div className="summary-grid">
                <SummaryCard label="已开通科目" value={String(stats?.subject_count ?? subjects.length)} help="可继续扩展更多学习方向" />
                <SummaryCard label="内容总量" value={formatCount(stats?.word_count ?? 0)} help="当前已经上架的学习内容" />
                <SummaryCard label="收款订单" value={formatCount(paymentOrders?.total ?? 0)} help="已记录的付款与下单信息" />
                <SummaryCard label="学员人数" value={formatCount(learners?.total ?? 0)} help="已注册的学习账号总数" />
              </div>

              <div className="dashboard-grid">
                <article className="content-card">
                  <h2>当前后台可以处理</h2>
                  <ul className="detail-list">
                    <li>首次运行时由运营负责人自行创建超级管理员，不需要提前把账号密码写死在配置文件里。</li>
                    <li>导入 CSV 后自动匹配内容分组，适合持续补充英语高频词和后续其他学科资料。</li>
                    <li>后台可以统一维护首页展示、搜索优化、会员方案、微信收款、订单和会员状态。</li>
                  </ul>
                </article>
                <article className="content-card">
                  <h2>推荐操作顺序</h2>
                  <ol className="detail-list ordered-list">
                    <li>先上传英语词库 CSV，确认内容分组是否符合你的教学结构。</li>
                    <li>再到站点展示页完善站名、首页文案和默认搜索优化内容。</li>
                    <li>最后去收费方案页创建会员方案，并完整测试一次下单到会员生效的流程。</li>
                  </ol>
                </article>
              </div>
            </section>
          ) : null}

          {activeSection === "import" ? (
            <section className="admin-section">
              <div className="section-header">
                <div>
                  <p className="section-eyebrow">内容导入</p>
                  <h1>上传内容、整理科目与学习阶段</h1>
                </div>
              </div>

              <div className="admin-card-grid">
                <article className="content-card">
                  <h2>上传学习内容文件</h2>
                  <form className="setup-form" onSubmit={handleImportLocal}>
                    <label className="form-field">
                      <span>选择词库文件</span>
                      <label className={`upload-picker ${importForm.file ? "upload-picker-ready" : ""}`}>
                        <input
                          accept=".csv,.xlsx"
                          className="upload-picker-input"
                          onChange={(event) => {
                            const file = event.target.files?.[0] ?? null;
                            setImportForm((current) => ({
                              ...current,
                              file,
                              fileName: file?.name ?? "",
                            }));
                          }}
                          type="file"
                        />
                        <div className="upload-picker-main">
                          <div className="upload-picker-meta">
                            <strong>{importForm.fileName || "点击选择要导入的词库文件"}</strong>
                            <span>
                              {importForm.file
                                ? `文件大小 ${formatFileSize(importForm.file.size)}，导入后会自动匹配内容分组`
                                : "支持 CSV、Excel 文件，上传后系统会自动识别分类并归入当前科目"}
                            </span>
                          </div>
                          <span className="upload-picker-action">{importForm.file ? "重新选择" : "选择文件"}</span>
                        </div>
                      </label>
                    </label>
                    <div className="upload-hint-list">
                      <span className="tag">支持 CSV</span>
                      <span className="tag">支持 Excel</span>
                      <span className="tag">自动识别内容分组</span>
                    </div>
                    <label className="form-field">
                      <span>导入到哪个科目</span>
                      <select
                        value={importForm.subjectKey}
                        onChange={(event) => {
                          setImportForm((current) => ({ ...current, subjectKey: event.target.value }));
                        }}
                      >
                        {subjects.map((subject) => (
                          <option key={subject.key} value={subject.key}>
                            {subject.name}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="checkbox-field">
                      <input
                        checked={importForm.replace}
                        onChange={(event) => {
                          setImportForm((current) => ({ ...current, replace: event.target.checked }));
                        }}
                        type="checkbox"
                      />
                      <span>导入前先清空当前科目下的旧内容</span>
                    </label>
                    <button className="primary-button" disabled={busyAction === "import"} type="submit">
                      {busyAction === "import" ? "导入中..." : "开始导入"}
                    </button>
                  </form>
                </article>

                <article className="content-card">
                  <h2>新增学习科目</h2>
                  <form className="setup-form" onSubmit={handleCreateSubject}>
                    <label className="form-field">
                      <span>科目编码</span>
                      <input
                        value={subjectForm.key}
                        onChange={(event) => {
                          setSubjectForm((current) => ({ ...current, key: event.target.value }));
                        }}
                        placeholder="english"
                      />
                    </label>
                    <label className="form-field">
                      <span>前台展示名称</span>
                      <input
                        value={subjectForm.name}
                        onChange={(event) => {
                          setSubjectForm((current) => ({ ...current, name: event.target.value }));
                        }}
                        placeholder="英语高频词汇"
                      />
                    </label>
                    <label className="form-field">
                      <span>科目简介</span>
                      <textarea
                        rows={3}
                        value={subjectForm.description}
                        onChange={(event) => {
                          setSubjectForm((current) => ({ ...current, description: event.target.value }));
                        }}
                      />
                    </label>
                    <label className="checkbox-field">
                      <input
                        checked={subjectForm.featured}
                        onChange={(event) => {
                          setSubjectForm((current) => ({ ...current, featured: event.target.checked }));
                        }}
                        type="checkbox"
                      />
                      <span>前台优先展示这个科目</span>
                    </label>
                    <button className="primary-button" disabled={busyAction === "subject"} type="submit">
                      新增科目
                    </button>
                  </form>
                </article>

                <article className="content-card">
                  <h2>新增内容分组</h2>
                  <form className="setup-form" onSubmit={handleCreateCategory}>
                    <label className="form-field">
                      <span>归属科目</span>
                      <select
                        value={categoryForm.subjectKey}
                        onChange={(event) => {
                          setCategoryForm((current) => ({ ...current, subjectKey: event.target.value }));
                        }}
                      >
                        {subjects.map((subject) => (
                          <option key={subject.key} value={subject.key}>
                            {subject.name}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="form-field">
                      <span>分组类型</span>
                      <input
                        value={categoryForm.kind}
                        onChange={(event) => {
                          setCategoryForm((current) => ({ ...current, kind: event.target.value }));
                        }}
                        placeholder="topic"
                      />
                    </label>
                    <label className="form-field">
                      <span>分组编码</span>
                      <input
                        value={categoryForm.key}
                        onChange={(event) => {
                          setCategoryForm((current) => ({ ...current, key: event.target.value }));
                        }}
                        placeholder="travel"
                      />
                    </label>
                    <label className="form-field">
                      <span>前台展示名称</span>
                      <input
                        value={categoryForm.name}
                        onChange={(event) => {
                          setCategoryForm((current) => ({ ...current, name: event.target.value }));
                        }}
                        placeholder="旅行场景"
                      />
                    </label>
                    <label className="form-field">
                      <span>分组说明</span>
                      <textarea
                        rows={3}
                        value={categoryForm.description}
                        onChange={(event) => {
                          setCategoryForm((current) => ({ ...current, description: event.target.value }));
                        }}
                      />
                    </label>
                    <button className="primary-button" disabled={busyAction === "category"} type="submit">
                      新增分组
                    </button>
                  </form>
                </article>

                <article className="content-card">
                  <h2>新增学习阶段</h2>
                  <form className="setup-form" onSubmit={handleCreateGrade}>
                    <label className="form-field">
                      <span>阶段编码</span>
                      <input
                        value={gradeForm.key}
                        onChange={(event) => {
                          setGradeForm((current) => ({ ...current, key: event.target.value }));
                        }}
                        placeholder="junior-1"
                      />
                    </label>
                    <label className="form-field">
                      <span>阶段名称</span>
                      <input
                        value={gradeForm.name}
                        onChange={(event) => {
                          setGradeForm((current) => ({ ...current, name: event.target.value }));
                        }}
                        placeholder="初一"
                      />
                    </label>
                    <label className="form-field">
                      <span>阶段类型</span>
                      <input
                        value={gradeForm.stage}
                        onChange={(event) => {
                          setGradeForm((current) => ({ ...current, stage: event.target.value }));
                        }}
                        placeholder="junior"
                      />
                    </label>
                    <label className="form-field">
                      <span>阶段说明</span>
                      <textarea
                        rows={3}
                        value={gradeForm.description}
                        onChange={(event) => {
                          setGradeForm((current) => ({ ...current, description: event.target.value }));
                        }}
                      />
                    </label>
                    <button className="primary-button" disabled={busyAction === "grade"} type="submit">
                      新增阶段
                    </button>
                  </form>
                </article>
              </div>
            </section>
          ) : null}

          {activeSection === "catalog" ? (
            <section className="admin-section">
              <div className="section-header">
                <div>
                  <p className="section-eyebrow">内容整理</p>
                  <h1>词库、内容分组与学习阶段</h1>
                </div>
              </div>

              <div className="table-section">
                <div className="section-toolbar">
                  <h2>词库内容列表</h2>
                  <input
                    className="toolbar-search"
                    value={wordQuery}
                    onChange={(event) => {
                      setWordQuery(event.target.value);
                      setWordPage(1);
                    }}
                    placeholder="搜索单词、释义或音标"
                  />
                </div>
                <DataTable
                  columns={["单词", "释义", "内容分组", "来源", "音标"]}
                  rows={(words?.items ?? []).map((item) => [
                    item.term,
                    item.translation || "-",
                    item.classification || "-",
                    item.source || "-",
                    item.phonetics || "-",
                  ])}
                  emptyText="当前还没有符合条件的词库内容。"
                />
                <PagerControls
                  page={words?.page ?? 1}
                  total={words?.total ?? 0}
                  pageSize={words?.page_size ?? adminPageSize}
                  onChange={setWordPage}
                />
              </div>

              <div className="table-section">
                <div className="section-toolbar">
                  <h2>内容分组列表</h2>
                  <input
                    className="toolbar-search"
                    value={categoryQuery}
                    onChange={(event) => {
                      setCategoryQuery(event.target.value);
                      setCategoryPage(1);
                    }}
                    placeholder="搜索分组名称"
                  />
                </div>
                <DataTable
                  columns={["名称", "分组编码", "所属科目", "分组类型"]}
                  rows={(categories?.items ?? []).map((item) => [
                    item.name,
                    item.key,
                    formatSubjectLabel(item.subject_key ?? ""),
                    item.kind,
                  ])}
                  emptyText="当前还没有符合条件的内容分组。"
                />
                <PagerControls
                  page={categories?.page ?? 1}
                  total={categories?.total ?? 0}
                  pageSize={categories?.page_size ?? adminPageSize}
                  onChange={setCategoryPage}
                />
              </div>

              <div className="table-section">
                <div className="section-toolbar">
                  <h2>学习阶段列表</h2>
                  <input
                    className="toolbar-search"
                    value={gradeQuery}
                    onChange={(event) => {
                      setGradeQuery(event.target.value);
                      setGradePage(1);
                    }}
                    placeholder="搜索阶段名称"
                  />
                </div>
                <DataTable
                  columns={["名称", "编码", "阶段类型", "启用"]}
                  rows={(grades?.items ?? []).map((item) => [
                    item.name,
                    item.key,
                    item.stage || "-",
                    item.enabled ? "是" : "否",
                  ])}
                  emptyText="当前还没有符合条件的学习阶段。"
                />
                <PagerControls
                  page={grades?.page ?? 1}
                  total={grades?.total ?? 0}
                  pageSize={grades?.page_size ?? adminPageSize}
                  onChange={setGradePage}
                />
              </div>
            </section>
          ) : null}

          {activeSection === "site" && canViewSiteSettings ? (
            <section className="admin-section">
              <div className="section-header">
                <div>
                  <p className="section-eyebrow">站点展示</p>
                  <h1>前台文案、品牌信息与搜索优化</h1>
                </div>
              </div>

              <div className="admin-card-grid">
                <article className="content-card">
                  <h2>前台基础信息</h2>
                  <form className="setup-form" onSubmit={handleSaveSiteSettings}>
                    <label className="form-field">
                      <span>站点名称</span>
                      <input
                        disabled={!canManageSiteSettings}
                        value={siteForm.site_name}
                        onChange={(event) => {
                          setSiteForm((current) => ({ ...current, site_name: event.target.value }));
                        }}
                      />
                    </label>
                    <label className="form-field">
                      <span>站点副标题</span>
                      <input
                        disabled={!canManageSiteSettings}
                        value={siteForm.site_tagline}
                        onChange={(event) => {
                          setSiteForm((current) => ({ ...current, site_tagline: event.target.value }));
                        }}
                      />
                    </label>
                    <label className="form-field">
                      <span>首页主标题</span>
                      <input
                        disabled={!canManageSiteSettings}
                        value={siteForm.hero_title}
                        onChange={(event) => {
                          setSiteForm((current) => ({ ...current, hero_title: event.target.value }));
                        }}
                      />
                    </label>
                    <label className="form-field">
                      <span>首页介绍文案</span>
                      <textarea
                        disabled={!canManageSiteSettings}
                        rows={4}
                        value={siteForm.hero_description}
                        onChange={(event) => {
                          setSiteForm((current) => ({ ...current, hero_description: event.target.value }));
                        }}
                      />
                    </label>
                    <label className="form-field">
                      <span>页脚说明</span>
                      <textarea
                        disabled={!canManageSiteSettings}
                        rows={3}
                        value={siteForm.footer_text}
                        onChange={(event) => {
                          setSiteForm((current) => ({ ...current, footer_text: event.target.value }));
                        }}
                      />
                    </label>
                    <label className="form-field">
                      <span>联系邮箱</span>
                      <input
                        disabled={!canManageSiteSettings}
                        value={siteForm.contact_email}
                        onChange={(event) => {
                          setSiteForm((current) => ({ ...current, contact_email: event.target.value }));
                        }}
                      />
                    </label>
                    {canManageSiteSettings ? (
                      <button className="primary-button" disabled={busyAction === "site-settings"} type="submit">
                        {busyAction === "site-settings" ? "保存中..." : "保存前台展示"}
                      </button>
                    ) : (
                      <p className="helper-text">当前账号只能查看前台展示信息，暂时不能修改。</p>
                    )}
                  </form>
                </article>

                <article className="content-card">
                  <h2>搜索优化设置</h2>
                  <form className="setup-form" onSubmit={handleSaveSiteSettings}>
                    <label className="form-field">
                      <span>SEO 频道文案</span>
                      <input
                        disabled={!canManageSiteSettings}
                        value={siteForm.seo_headline}
                        onChange={(event) => {
                          setSiteForm((current) => ({ ...current, seo_headline: event.target.value }));
                        }}
                      />
                    </label>
                    <label className="form-field">
                      <span>SEO 标题</span>
                      <input
                        disabled={!canManageSiteSettings}
                        value={siteForm.seo_title}
                        onChange={(event) => {
                          setSiteForm((current) => ({ ...current, seo_title: event.target.value }));
                        }}
                      />
                    </label>
                    <label className="form-field">
                      <span>SEO 描述</span>
                      <textarea
                        disabled={!canManageSiteSettings}
                        rows={4}
                        value={siteForm.seo_description}
                        onChange={(event) => {
                          setSiteForm((current) => ({ ...current, seo_description: event.target.value }));
                        }}
                      />
                    </label>
                    <label className="form-field">
                      <span>SEO 关键词</span>
                      <textarea
                        disabled={!canManageSiteSettings}
                        rows={4}
                        value={siteForm.seo_keywords}
                        onChange={(event) => {
                          setSiteForm((current) => ({ ...current, seo_keywords: event.target.value }));
                        }}
                      />
                    </label>
                    <div className="feedback-banner">
                      默认 SEO 已按“高频英语单词 + 场景词汇 + 会员学习 + 多学科扩展”方向预设，后续可根据你的实际内容继续优化。
                    </div>
                  </form>
                </article>
              </div>
            </section>
          ) : null}

          {activeSection === "payments" ? (
            <section className="admin-section">
              <div className="section-header">
                <div>
                  <p className="section-eyebrow">收费方案</p>
                  <h1>会员方案、微信收款与订单跟进</h1>
                </div>
              </div>

              <div className="admin-card-grid">
                {canManagePlans ? (
                  <article className="content-card">
                    <h2>{planEditor.id > 0 ? "调整会员方案" : "新增会员方案"}</h2>
                    <form className="setup-form" onSubmit={handleSavePlan}>
                      <label className="form-field">
                        <span>方案编码</span>
                        <input
                          disabled={planEditor.id > 0}
                          value={planEditor.key}
                          onChange={(event) => {
                            setPlanEditor((current) => ({ ...current, key: event.target.value }));
                          }}
                          placeholder="english-monthly"
                        />
                      </label>
                      <label className="form-field">
                        <span>前台展示名称</span>
                        <input
                          value={planEditor.name}
                          onChange={(event) => {
                            setPlanEditor((current) => ({ ...current, name: event.target.value }));
                          }}
                          placeholder="英语月度会员"
                        />
                      </label>
                      <div className="form-grid-two">
                        <label className="form-field">
                          <span>收费方式</span>
                          <select
                            value={planEditor.billingMode}
                            onChange={(event) => {
                              setPlanEditor((current) => ({ ...current, billingMode: event.target.value }));
                            }}
                          >
                            <option value="monthly">按月会员</option>
                            <option value="lifetime">一次性买断</option>
                          </select>
                        </label>
                        <label className="form-field">
                          <span>售价（元）</span>
                          <input
                            inputMode="decimal"
                            value={planEditor.priceYuan}
                            onChange={(event) => {
                              setPlanEditor((current) => ({ ...current, priceYuan: event.target.value }));
                            }}
                            placeholder="19.90"
                          />
                        </label>
                      </div>
                      <label className="form-field">
                        <span>方案说明</span>
                        <textarea
                          rows={3}
                          value={planEditor.description}
                          onChange={(event) => {
                            setPlanEditor((current) => ({ ...current, description: event.target.value }));
                          }}
                        />
                      </label>
                      <label className="form-field">
                        <span>收款渠道编码</span>
                        <textarea
                          rows={3}
                          value={planEditor.paymentChannels}
                          onChange={(event) => {
                            setPlanEditor((current) => ({ ...current, paymentChannels: event.target.value }));
                          }}
                          placeholder={"wechat\nwechat_native"}
                        />
                      </label>
                      <label className="form-field">
                        <span>学员可获得的权益</span>
                        <textarea
                          rows={4}
                          value={planEditor.features}
                          onChange={(event) => {
                            setPlanEditor((current) => ({ ...current, features: event.target.value }));
                          }}
                          placeholder={"高频词学习\n场景分类\n持续更新"}
                        />
                      </label>
                      <label className="checkbox-field">
                        <input
                          checked={planEditor.recommended}
                          onChange={(event) => {
                            setPlanEditor((current) => ({ ...current, recommended: event.target.checked }));
                          }}
                          type="checkbox"
                        />
                        <span>前台标记为推荐方案</span>
                      </label>
                      <div className="button-row">
                        <button className="primary-button" disabled={busyAction === "plan"} type="submit">
                          {busyAction === "plan" ? "保存中..." : planEditor.id > 0 ? "保存方案" : "新增方案"}
                        </button>
                        <button className="secondary-button" onClick={resetPlanEditor} type="button">
                          清空表单
                        </button>
                      </div>
                    </form>
                  </article>
                ) : (
                  <article className="content-card">
                    <h2>方案查看说明</h2>
                    <p className="helper-text">当前岗位可以查看会员方案和订单信息，但暂时不能直接修改收费方案。</p>
                  </article>
                )}

                {canViewPayments ? (
                  <article className="content-card">
                    <h2>微信收款配置</h2>
                    <form className="setup-form" onSubmit={handleSaveWechatPayConfig}>
                      {wechatPayConfigExists && wechatPayConfig ? (
                        <div className={`feedback-banner ${wechatPayConfig.ready_for_checkout ? "feedback-success" : "feedback-error"}`}>
                          当前收款配置
                          {wechatPayConfig.ready_for_checkout ? "已经可以用于前台收款。" : "还未达到可收款状态。"}
                          {!wechatPayConfig.ready_for_checkout && wechatPayConfig.validation_error
                            ? ` 还需补充：${wechatPayConfig.validation_error}。`
                            : ""}
                        </div>
                      ) : (
                        <div className="feedback-banner">还没有保存微信收款配置，保存后前台才能发起支付。</div>
                      )}
                      <div className="form-grid-two">
                        <label className="form-field">
                          <span>验签方式</span>
                          <select
                            disabled={!canManagePayments}
                            value={wechatPayForm.authMode}
                            onChange={(event) => {
                              setWechatPayForm((current) => ({ ...current, authMode: event.target.value }));
                            }}
                          >
                            <option value="public_key">微信支付公钥模式</option>
                            <option value="auto_certificate">平台证书自动下载模式</option>
                          </select>
                        </label>
                        <label className="form-field">
                          <span>商户号</span>
                          <input
                            disabled={!canManagePayments}
                            value={wechatPayForm.mchId}
                            onChange={(event) => {
                              setWechatPayForm((current) => ({ ...current, mchId: event.target.value }));
                            }}
                          />
                        </label>
                        <label className="form-field">
                          <span>AppID</span>
                          <input
                            disabled={!canManagePayments}
                            value={wechatPayForm.appId}
                            onChange={(event) => {
                              setWechatPayForm((current) => ({ ...current, appId: event.target.value }));
                            }}
                          />
                        </label>
                        <label className="form-field">
                          <span>商户证书序列号</span>
                          <input
                            disabled={!canManagePayments}
                            value={wechatPayForm.merchantSerialNo}
                            onChange={(event) => {
                              setWechatPayForm((current) => ({ ...current, merchantSerialNo: event.target.value }));
                            }}
                          />
                        </label>
                        <label className="form-field">
                          <span>支付回调地址</span>
                          <input
                            disabled={!canManagePayments}
                            value={wechatPayForm.notifyURL}
                            onChange={(event) => {
                              setWechatPayForm((current) => ({ ...current, notifyURL: event.target.value }));
                            }}
                            placeholder="https://your-domain/api/v1/payments/wechat/notify"
                          />
                        </label>
                        <label className="form-field">
                          <span>订单名称前缀</span>
                          <input
                            disabled={!canManagePayments}
                            value={wechatPayForm.descriptionPrefix}
                            onChange={(event) => {
                              setWechatPayForm((current) => ({ ...current, descriptionPrefix: event.target.value }));
                            }}
                          />
                        </label>
                        <label className="form-field">
                          <span>二维码有效时长（分钟）</span>
                          <input
                            disabled={!canManagePayments}
                            inputMode="numeric"
                            type="number"
                            value={wechatPayForm.timeExpireMinutes}
                            onChange={(event) => {
                              setWechatPayForm((current) => ({ ...current, timeExpireMinutes: event.target.value }));
                            }}
                          />
                        </label>
                        <label className="form-field form-grid-span-two">
                          <span>APIv3 密钥</span>
                          <input
                            disabled={!canManagePayments}
                            value={wechatPayForm.apiV3Key}
                            onChange={(event) => {
                              setWechatPayForm((current) => ({ ...current, apiV3Key: event.target.value }));
                            }}
                            placeholder="直接粘贴 APIv3 密钥"
                          />
                        </label>
                        <label className="form-field form-grid-span-two">
                          <span>商户私钥内容</span>
                          <textarea
                            disabled={!canManagePayments}
                            rows={6}
                            value={wechatPayForm.keyPem}
                            onChange={(event) => {
                              setWechatPayForm((current) => ({ ...current, keyPem: event.target.value }));
                            }}
                            placeholder="直接粘贴商户私钥完整内容"
                          />
                        </label>
                        {useWechatPayPublicKeyMode ? (
                          <>
                            <label className="form-field">
                              <span>微信支付公钥编号</span>
                              <input
                                disabled={!canManagePayments}
                                value={wechatPayForm.wechatPayPublicKeyID}
                                onChange={(event) => {
                                  setWechatPayForm((current) => ({ ...current, wechatPayPublicKeyID: event.target.value }));
                                }}
                              />
                            </label>
                            <label className="form-field form-grid-span-two">
                              <span>微信支付公钥内容</span>
                              <textarea
                                disabled={!canManagePayments}
                                rows={5}
                                value={wechatPayForm.wechatPayPublicKey}
                                onChange={(event) => {
                                  setWechatPayForm((current) => ({ ...current, wechatPayPublicKey: event.target.value }));
                                }}
                                placeholder="直接粘贴微信支付公钥完整内容"
                              />
                            </label>
                          </>
                        ) : null}
                        <div className="feedback-banner form-grid-span-two">
                          当前页面会直接回显并保存密钥内容。如需清空某一项，把输入框删空后再保存即可。
                        </div>
                      </div>
                      {canManagePayments ? (
                        <button className="primary-button" disabled={busyAction === "wechatpay"} type="submit">
                          {busyAction === "wechatpay" ? "保存中..." : "保存收款配置"}
                        </button>
                      ) : (
                        <p className="helper-text">当前账号只能查看收款配置，暂时不能修改。</p>
                      )}
                    </form>
                  </article>
                ) : null}
              </div>

              <article className="content-card">
                <div className="section-toolbar">
                  <h2>会员方案列表</h2>
                </div>
                <div className="table-wrap">
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>方案编码</th>
                        <th>展示名称</th>
                        <th>收费方式</th>
                        <th>售价</th>
                        <th>推荐</th>
                        <th>收款渠道</th>
                        {canManagePlans ? <th>操作</th> : null}
                      </tr>
                    </thead>
                    <tbody>
                      {plans.map((item) => (
                        <tr key={item.key}>
                          <td>{item.key}</td>
                          <td>{item.name}</td>
                          <td>{item.billing_mode === "monthly" ? "按月会员" : "一次性买断"}</td>
                          <td>{formatPrice(item.price_cents)}</td>
                          <td>{item.recommended ? "是" : "否"}</td>
                          <td>{formatPaymentChannelLabels(item.payment_channels)}</td>
                          {canManagePlans ? (
                            <td>
                              <div className="button-row">
                                <button className="secondary-button small-button" onClick={() => startEditPlan(item)} type="button">
                                  调整
                                </button>
                                <button
                                  className="secondary-button small-button"
                                  disabled={busyAction === "plan-delete"}
                                  onClick={() => handleDeletePlan(item.id ?? 0)}
                                  type="button"
                                >
                                  删除
                                </button>
                              </div>
                            </td>
                          ) : null}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  {plans.length === 0 ? <div className="feedback-banner">当前还没有会员方案，请先新增一个方案。</div> : null}
                </div>
              </article>

              {canViewPayments ? (
                <article className="content-card">
                  <div className="section-toolbar">
                    <h2>收款订单</h2>
                    <div className="toolbar-controls">
                      <select
                        value={paymentStatusFilter}
                        onChange={(event) => {
                          setPaymentStatusFilter(event.target.value);
                          setPaymentPage(1);
                        }}
                      >
                        <option value="">全部状态</option>
                        <option value="pending">待支付</option>
                        <option value="success">支付成功</option>
                        <option value="failed">支付失败</option>
                        <option value="closed">已关闭</option>
                      </select>
                      <input
                        className="toolbar-search"
                        value={paymentQuery}
                        onChange={(event) => {
                          setPaymentQuery(event.target.value);
                          setPaymentPage(1);
                        }}
                        placeholder="搜索订单号、学员账号、会员方案"
                      />
                    </div>
                  </div>
                  <div className="table-wrap">
                    <table className="data-table">
                      <thead>
                        <tr>
                          <th>订单号</th>
                          <th>学员账号</th>
                          <th>会员方案</th>
                          <th>金额</th>
                          <th>状态</th>
                          <th>支付时间</th>
                          <th>创建时间</th>
                          <th>操作</th>
                        </tr>
                      </thead>
                      <tbody>
                        {(paymentOrders?.items ?? []).map((item) => (
                          <tr key={item.order_no}>
                            <td>{item.order_no}</td>
                            <td>{item.customer_ref}</td>
                            <td>{formatPlanLabel(item.plan_key)}</td>
                            <td>{formatPrice(item.amount_cents)}</td>
                            <td>
                              <span className={`pill ${paymentStatusClass(item.status)}`}>{paymentStatusLabel(item.status)}</span>
                            </td>
                            <td>{item.paid_at ? formatDateTime(item.paid_at) : "-"}</td>
                            <td>{formatDateTime(item.created_at)}</td>
                            <td>
                              <button
                                className="secondary-button small-button"
                                disabled={busyAction === `order-${item.order_no}`}
                                onClick={() => handleLoadOrderDetail(item.order_no)}
                                type="button"
                              >
                                查看订单
                              </button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                    {(paymentOrders?.items ?? []).length === 0 ? <div className="feedback-banner">没有符合条件的收款订单。</div> : null}
                  </div>
                  <PagerControls
                    page={paymentOrders?.page ?? 1}
                    total={paymentOrders?.total ?? 0}
                    pageSize={paymentOrders?.page_size ?? adminPageSize}
                    onChange={setPaymentPage}
                  />
                </article>
              ) : null}

              {selectedOrderDetail ? (
                <article className="content-card">
                  <div className="section-toolbar">
                    <h2>订单详情</h2>
                    <button className="secondary-button small-button" onClick={() => setSelectedOrderDetail(null)} type="button">
                        收起
                    </button>
                  </div>
                  <div className="detail-grid">
                    <div>
                      <dt>订单号</dt>
                      <dd>{selectedOrderDetail.order.order_no}</dd>
                    </div>
                    <div>
                      <dt>状态</dt>
                      <dd>
                        <span className={`pill ${paymentStatusClass(selectedOrderDetail.order.status)}`}>
                          {paymentStatusLabel(selectedOrderDetail.order.status)}
                        </span>
                      </dd>
                    </div>
                    <div>
                      <dt>学员账号</dt>
                      <dd>{selectedOrderDetail.order.customer_ref}</dd>
                    </div>
                    <div>
                      <dt>会员方案</dt>
                      <dd>{formatPlanLabel(selectedOrderDetail.order.plan_key)}</dd>
                    </div>
                    <div>
                      <dt>所属科目</dt>
                      <dd>{formatSubjectLabel(selectedOrderDetail.order.subject_key)}</dd>
                    </div>
                    <div>
                      <dt>金额</dt>
                      <dd>{formatPrice(selectedOrderDetail.order.amount_cents)}</dd>
                    </div>
                    <div>
                      <dt>支付渠道</dt>
                      <dd>{formatPaymentProviderLabel(selectedOrderDetail.order.provider)}</dd>
                    </div>
                    <div>
                      <dt>渠道流水号</dt>
                      <dd>{selectedOrderDetail.order.provider_trade_no || "-"}</dd>
                    </div>
                    <div>
                      <dt>创建时间</dt>
                      <dd>{formatDateTime(selectedOrderDetail.order.created_at)}</dd>
                    </div>
                    <div>
                      <dt>更新时间</dt>
                      <dd>{selectedOrderDetail.order.updated_at ? formatDateTime(selectedOrderDetail.order.updated_at) : "-"}</dd>
                    </div>
                    <div>
                      <dt>支付时间</dt>
                      <dd>{selectedOrderDetail.order.paid_at ? formatDateTime(selectedOrderDetail.order.paid_at) : "-"}</dd>
                    </div>
                    <div>
                      <dt>过期时间</dt>
                      <dd>{selectedOrderDetail.order.expires_at ? formatDateTime(selectedOrderDetail.order.expires_at) : "-"}</dd>
                    </div>
                  </div>
                  {selectedOrderDetail.order.description ? (
                    <div className="feedback-banner">下单说明：{selectedOrderDetail.order.description}</div>
                  ) : null}
                  {selectedOrderDetail.order.error_message ? (
                    <div className="feedback-banner feedback-error">支付失败原因：{selectedOrderDetail.order.error_message}</div>
                  ) : null}
                  {selectedOrderDetail.subscription ? (
                    <div className="feedback-banner feedback-success">
                      已关联会员权益：学员账号 {selectedOrderDetail.subscription.customer_ref}，状态{" "}
                      {subscriptionStatusLabel(selectedOrderDetail.subscription.status)}
                      {selectedOrderDetail.subscription.current_period_end
                        ? `，有效期至 ${formatDateTime(selectedOrderDetail.subscription.current_period_end)}`
                        : "，当前为长期有效"}
                    </div>
                  ) : (
                    <div className="feedback-banner">当前订单还没有关联到会员权益记录。</div>
                  )}
                </article>
              ) : null}
            </section>
          ) : null}

          {activeSection === "memberships" ? (
            <section className="admin-section">
              <div className="section-header">
                <div>
                  <p className="section-eyebrow">会员服务</p>
                  <h1>会员状态、有效期与人工处理</h1>
                </div>
              </div>

              <div className="admin-card-grid">
                <article className="content-card">
                  <h2>调整会员状态</h2>
                  {selectedSubscription ? (
                    <form className="setup-form" onSubmit={handleSaveSubscription}>
                      <div className="form-grid-two">
                        <label className="form-field">
                          <span>学员账号</span>
                          <input disabled value={subscriptionEditor.customerRef} />
                        </label>
                        <label className="form-field">
                          <span>所属科目</span>
                          <input disabled value={formatSubjectLabel(subscriptionEditor.subjectKey)} />
                        </label>
                      </div>
                      <div className="form-grid-two">
                        <label className="form-field">
                          <span>开通方案</span>
                          <select
                            value={subscriptionEditor.planKey}
                            onChange={(event) => {
                              setSubscriptionEditor((current) => ({ ...current, planKey: event.target.value }));
                            }}
                          >
                            {plans.map((plan) => (
                              <option key={plan.key} value={plan.key}>
                                {plan.name} ({plan.key})
                              </option>
                            ))}
                          </select>
                        </label>
                        <label className="form-field">
                          <span>服务状态</span>
                          <select
                            value={subscriptionEditor.status}
                            onChange={(event) => {
                              setSubscriptionEditor((current) => ({ ...current, status: event.target.value }));
                            }}
                          >
                            <option value="active">生效中</option>
                            <option value="pending">待生效</option>
                            <option value="expired">已过期</option>
                            <option value="cancelled">已取消</option>
                          </select>
                        </label>
                      </div>
                      <label className="checkbox-field">
                        <input
                          checked={subscriptionEditor.autoRenew}
                          onChange={(event) => {
                            setSubscriptionEditor((current) => ({ ...current, autoRenew: event.target.checked }));
                          }}
                          type="checkbox"
                        />
                        <span>标记为自动续费用户</span>
                      </label>
                      <div className="form-grid-two">
                        <label className="form-field">
                          <span>开通时间</span>
                          <input
                            type="datetime-local"
                            value={subscriptionEditor.startedAt}
                            onChange={(event) => {
                              setSubscriptionEditor((current) => ({ ...current, startedAt: event.target.value }));
                            }}
                          />
                        </label>
                        <label className="form-field">
                          <span>当前服务周期开始</span>
                          <input
                            type="datetime-local"
                            value={subscriptionEditor.currentPeriodStart}
                            onChange={(event) => {
                              setSubscriptionEditor((current) => ({ ...current, currentPeriodStart: event.target.value }));
                            }}
                          />
                        </label>
                      </div>
                      <div className="form-grid-two">
                        <label className="form-field">
                          <span>当前服务周期结束</span>
                          <input
                            type="datetime-local"
                            value={subscriptionEditor.currentPeriodEnd}
                            onChange={(event) => {
                              setSubscriptionEditor((current) => ({ ...current, currentPeriodEnd: event.target.value }));
                            }}
                          />
                        </label>
                        <label className="form-field">
                          <span>终止时间</span>
                          <input
                            type="datetime-local"
                            value={subscriptionEditor.cancelledAt}
                            onChange={(event) => {
                              setSubscriptionEditor((current) => ({ ...current, cancelledAt: event.target.value }));
                            }}
                          />
                        </label>
                      </div>
                      <div className="button-row">
                        <button className="primary-button" disabled={busyAction === "subscription-save"} type="submit">
                          {busyAction === "subscription-save" ? "保存中..." : "保存服务状态"}
                        </button>
                        <button className="secondary-button" onClick={resetSubscriptionEditor} type="button">
                          取消调整
                        </button>
                      </div>
                    </form>
                  ) : (
                    <p className="helper-text">从下方列表中选中一条会员记录后，就可以人工调整方案、状态和有效期。</p>
                  )}
                </article>

                <article className="content-card">
                  <h2>处理说明</h2>
                  <ul className="detail-list">
                    <li>按月会员建议维护“当前服务周期开始”和“当前服务周期结束”。</li>
                    <li>一次性买断方案通常不需要填写“当前服务周期结束”。</li>
                    <li>把状态改为“已过期”时，如果没有填写结束时间，系统会自动补上当前时间。</li>
                    <li>把状态改为“已取消”时，如果没有填写终止时间，系统会自动补上当前时间。</li>
                  </ul>
                </article>
              </div>

              <article className="content-card">
                <div className="section-toolbar">
                  <h2>会员记录列表</h2>
                  <div className="toolbar-controls">
                    <select
                      value={subscriptionStatusFilter}
                      onChange={(event) => {
                        setSubscriptionStatusFilter(event.target.value);
                        setSubscriptionPage(1);
                      }}
                    >
                      <option value="">全部状态</option>
                      <option value="active">生效中</option>
                      <option value="expired">已过期</option>
                      <option value="pending">待生效</option>
                      <option value="cancelled">已取消</option>
                    </select>
                    <input
                      className="toolbar-search"
                      value={subscriptionQuery}
                      onChange={(event) => {
                        setSubscriptionQuery(event.target.value);
                        setSubscriptionPage(1);
                      }}
                      placeholder="搜索学员账号或会员方案"
                    />
                  </div>
                </div>
                <div className="table-wrap">
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>学员账号</th>
                        <th>会员方案</th>
                        <th>科目</th>
                        <th>状态</th>
                        <th>到期时间</th>
                        <th>自动续费</th>
                        <th>开通时间</th>
                        <th>操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      {(subscriptions?.items ?? []).map((item) => (
                        <tr key={item.id}>
                          <td>{item.customer_ref}</td>
                          <td>{formatPlanLabel(item.plan_key)}</td>
                          <td>{formatSubjectLabel(item.subject_key)}</td>
                          <td>
                            <span className={`pill ${subscriptionStatusClass(item.status)}`}>{subscriptionStatusLabel(item.status)}</span>
                          </td>
                          <td>{item.current_period_end ? formatDateTime(item.current_period_end) : "长期有效"}</td>
                          <td>{item.auto_renew ? "是" : "否"}</td>
                          <td>{item.started_at ? formatDateTime(item.started_at) : "-"}</td>
                          <td>
                            <button className="secondary-button small-button" onClick={() => startEditSubscription(item)} type="button">
                              调整
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  {(subscriptions?.items ?? []).length === 0 ? <div className="feedback-banner">没有符合条件的会员记录。</div> : null}
                </div>
                <PagerControls
                  page={subscriptions?.page ?? 1}
                  total={subscriptions?.total ?? 0}
                  pageSize={subscriptions?.page_size ?? adminPageSize}
                  onChange={setSubscriptionPage}
                />
              </article>
            </section>
          ) : null}

          {activeSection === "learners" && canViewLearners ? (
            <section className="admin-section">
              <div className="section-header">
                <div>
                  <p className="section-eyebrow">学员账号</p>
                  <h1>学员账号、购买情况与服务状态</h1>
                </div>
              </div>

              <div className="admin-card-grid">
                <article className="content-card">
                  <h2>调整学员资料</h2>
                  {learnerEditor.id > 0 ? (
                    <form className="setup-form" onSubmit={handleSaveLearnerUser}>
                      <label className="form-field">
                        <span>学员账号</span>
                        <input disabled value={learnerEditor.username} />
                      </label>
                      <label className="form-field">
                        <span>昵称</span>
                        <input
                          disabled={!canManageLearners}
                          value={learnerEditor.displayName}
                          onChange={(event) => {
                            setLearnerEditor((current) => ({ ...current, displayName: event.target.value }));
                          }}
                        />
                      </label>
                      <label className="form-field">
                        <span>账号状态</span>
                        <select
                          disabled={!canManageLearners}
                          value={learnerEditor.status}
                          onChange={(event) => {
                            setLearnerEditor((current) => ({ ...current, status: event.target.value }));
                          }}
                        >
                          <option value="active">启用</option>
                          <option value="disabled">停用</option>
                        </select>
                      </label>
                      <div className="button-row">
                        <button className="primary-button" disabled={busyAction === "learner-user"} type="submit">
                          {busyAction === "learner-user" ? "保存中..." : "保存学员资料"}
                        </button>
                        <button className="secondary-button" onClick={resetLearnerEditor} type="button">
                          取消调整
                        </button>
                      </div>
                    </form>
                  ) : (
                    <p className="helper-text">从下方学员列表中选择一个账号后，就可以查看并调整昵称或状态。</p>
                  )}
                </article>

                <article className="content-card">
                  <h2>服务说明</h2>
                  <ul className="detail-list">
                    <li>可以查看学员的注册时间、购买次数、当前会员状态和正在使用的方案。</li>
                    <li>分页查询由前后端协同处理，刷新页面后会停留在当前分页和筛选条件。</li>
                    <li>后续继续扩展更多学科时，学员依旧可以共用同一个学习账号体系。</li>
                  </ul>
                </article>
              </div>

              <article className="content-card">
                <div className="section-toolbar">
                  <h2>学员列表</h2>
                  <div className="toolbar-controls">
                    <select
                      value={learnerStatusFilter}
                      onChange={(event) => {
                        setLearnerStatusFilter(event.target.value);
                        setLearnerPage(1);
                      }}
                    >
                      <option value="">全部状态</option>
                      <option value="active">启用</option>
                      <option value="disabled">停用</option>
                    </select>
                    <input
                      className="toolbar-search"
                      value={learnerQuery}
                      onChange={(event) => {
                        setLearnerQuery(event.target.value);
                        setLearnerPage(1);
                      }}
                      placeholder="搜索学员账号或昵称"
                    />
                  </div>
                </div>
                <div className="table-wrap">
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>学员账号</th>
                        <th>昵称</th>
                        <th>账号状态</th>
                        <th>注册时间</th>
                        <th>购买次数</th>
                        <th>是否开通过会员</th>
                        <th>会员状态</th>
                        <th>当前方案</th>
                        <th>到期时间</th>
                        {canManageLearners ? <th>操作</th> : null}
                      </tr>
                    </thead>
                    <tbody>
                      {(learners?.items ?? []).map((item) => (
                        <tr key={item.id}>
                          <td>{item.username}</td>
                          <td>{item.display_name || "-"}</td>
                          <td>{item.status === "active" ? "启用" : "停用"}</td>
                          <td>{formatDateTime(item.created_at)}</td>
                          <td>{formatCount(item.purchase_count)}</td>
                          <td>{item.has_membership ? "是" : "否"}</td>
                          <td>{item.membership_status ? subscriptionStatusLabel(item.membership_status) : "-"}</td>
                          <td>{item.current_plan_key ? formatPlanLabel(item.current_plan_key) : "-"}</td>
                          <td>{item.current_period_end ? formatDateTime(item.current_period_end) : "-"}</td>
                          {canManageLearners ? (
                            <td>
                              <button className="secondary-button small-button" onClick={() => startEditLearner(item)} type="button">
                                调整
                              </button>
                            </td>
                          ) : null}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  {(learners?.items ?? []).length === 0 ? <div className="feedback-banner">当前没有符合条件的学员。</div> : null}
                </div>
                <PagerControls
                  page={learners?.page ?? 1}
                  total={learners?.total ?? 0}
                  pageSize={learners?.page_size ?? adminPageSize}
                  onChange={setLearnerPage}
                />
              </article>
            </section>
          ) : null}

          {activeSection === "admins" ? (
            <section className="admin-section">
              <div className="section-header">
                <div>
                  <p className="section-eyebrow">团队与权限</p>
                  <h1>后台账号、岗位职责与操作范围</h1>
                </div>
              </div>

              <div className="admin-card-grid">
                <article className="content-card">
                  <h2>修改我的登录密码</h2>
                  <form className="setup-form" onSubmit={handleChangePassword}>
                    <label className="form-field">
                      <span>旧密码</span>
                      <input
                        type="password"
                        value={passwordForm.oldPassword}
                        onChange={(event) => {
                          setPasswordForm((current) => ({ ...current, oldPassword: event.target.value }));
                        }}
                      />
                    </label>
                    <label className="form-field">
                      <span>新密码</span>
                      <input
                        type="password"
                        value={passwordForm.newPassword}
                        onChange={(event) => {
                          setPasswordForm((current) => ({ ...current, newPassword: event.target.value }));
                        }}
                      />
                    </label>
                    <label className="form-field">
                      <span>确认新密码</span>
                      <input
                        type="password"
                        value={passwordForm.confirmPassword}
                        onChange={(event) => {
                          setPasswordForm((current) => ({ ...current, confirmPassword: event.target.value }));
                        }}
                      />
                    </label>
                    <button className="primary-button" disabled={busyAction === "password"} type="submit">
                      保存新密码
                    </button>
                  </form>
                </article>

                {canManageAdmins ? (
                  <article className="content-card">
                    <h2>{adminEditor.id > 0 ? "调整后台账号" : "新增后台账号"}</h2>
                    <form className="setup-form" onSubmit={handleSaveAdminUser}>
                      <label className="form-field">
                        <span>后台登录账号</span>
                        <input
                          disabled={adminEditor.id > 0}
                          value={adminEditor.username}
                          onChange={(event) => {
                            setAdminEditor((current) => ({ ...current, username: event.target.value }));
                          }}
                          placeholder="content-manager"
                        />
                      </label>
                      <label className="form-field">
                        <span>成员姓名</span>
                        <input
                          value={adminEditor.displayName}
                          onChange={(event) => {
                            setAdminEditor((current) => ({ ...current, displayName: event.target.value }));
                          }}
                          placeholder="内容运营"
                        />
                      </label>
                      <label className="form-field">
                        <span>{adminEditor.id > 0 ? "重置密码（可选）" : "登录密码"}</span>
                        <input
                          type="password"
                          value={adminEditor.password}
                          onChange={(event) => {
                            setAdminEditor((current) => ({ ...current, password: event.target.value }));
                          }}
                          placeholder="至少 8 位"
                        />
                      </label>
                      <div className="form-grid-two">
                        <label className="form-field">
                          <span>岗位权限</span>
                          <select
                            value={adminEditor.role}
                            onChange={(event) => {
                              const nextRole = event.target.value;
                              setAdminEditor((current) => ({
                                ...current,
                                role: nextRole,
                                isSuper: nextRole === "super_admin" ? true : current.isSuper,
                              }));
                            }}
                          >
                            {roles.map((role) => (
                              <option key={role.key} value={role.key}>
                                {adminRoleLabel(role.key, role.name)}
                              </option>
                            ))}
                          </select>
                        </label>
                        <label className="form-field">
                          <span>使用状态</span>
                          <select
                            value={adminEditor.status}
                            onChange={(event) => {
                              setAdminEditor((current) => ({ ...current, status: event.target.value }));
                            }}
                          >
                            <option value="active">启用</option>
                            <option value="disabled">停用</option>
                          </select>
                        </label>
                      </div>
                      <label className="checkbox-field">
                        <input
                          checked={adminEditor.isSuper}
                          disabled={adminEditor.role === "super_admin"}
                          onChange={(event) => {
                            setAdminEditor((current) => ({ ...current, isSuper: event.target.checked }));
                          }}
                          type="checkbox"
                        />
                        <span>授予全站最高权限</span>
                      </label>
                      <div className="button-row">
                        <button className="primary-button" disabled={busyAction === "admin-user"} type="submit">
                          {adminEditor.id > 0 ? "保存账号调整" : "新增后台账号"}
                        </button>
                        <button className="secondary-button" onClick={resetAdminEditor} type="button">
                          清空表单
                        </button>
                      </div>
                    </form>
                  </article>
                ) : null}

                {canManageAdmins ? (
                  <article className="content-card">
                    <h2>{roleEditor.id > 0 ? "调整岗位权限模板" : "新增岗位权限模板"}</h2>
                    <form className="setup-form" onSubmit={handleSaveRole}>
                      <label className="form-field">
                        <span>岗位编码</span>
                        <input
                          disabled={roleEditor.id > 0}
                          value={roleEditor.key}
                          onChange={(event) => {
                            setRoleEditor((current) => ({ ...current, key: event.target.value }));
                          }}
                          placeholder="ops-manager"
                        />
                      </label>
                      <label className="form-field">
                        <span>岗位名称</span>
                        <input
                          value={roleEditor.name}
                          onChange={(event) => {
                            setRoleEditor((current) => ({ ...current, name: event.target.value }));
                          }}
                          placeholder="运营经理"
                        />
                      </label>
                      <label className="form-field">
                        <span>职责说明</span>
                        <textarea
                          rows={3}
                          value={roleEditor.description}
                          onChange={(event) => {
                            setRoleEditor((current) => ({ ...current, description: event.target.value }));
                          }}
                        />
                      </label>
                      <label className="form-field">
                        <span>显示顺序</span>
                        <input
                          value={roleEditor.sort}
                          onChange={(event) => {
                            setRoleEditor((current) => ({ ...current, sort: event.target.value }));
                          }}
                          placeholder="0"
                        />
                      </label>
                      <label className="form-field">
                        <span>系统权限编码</span>
                        <textarea
                          rows={5}
                          value={roleEditor.permissions}
                          onChange={(event) => {
                            setRoleEditor((current) => ({ ...current, permissions: event.target.value }));
                          }}
                          placeholder={"admin.read\ncatalog.read\npayment.read\nplan.read"}
                        />
                      </label>
                      <p className="helper-text">每行填写一个系统权限编码，也可以用英文逗号分隔，用来定义这个岗位能查看和能处理的范围。</p>
                      <div className="button-row">
                        <button className="primary-button" disabled={busyAction === "admin-role"} type="submit">
                          {roleEditor.id > 0 ? "保存岗位模板" : "新增岗位模板"}
                        </button>
                        <button className="secondary-button" onClick={resetRoleEditor} type="button">
                          清空表单
                        </button>
                      </div>
                    </form>
                  </article>
                ) : null}
              </div>

              <article className="content-card">
                <div className="section-toolbar">
                  <h2>后台账号列表</h2>
                  <input
                    className="toolbar-search"
                    value={adminUserQuery}
                    onChange={(event) => {
                      setAdminUserQuery(event.target.value);
                      setAdminUserPage(1);
                    }}
                    placeholder="搜索账号或姓名"
                  />
                </div>
                <div className="table-wrap">
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>后台账号</th>
                        <th>成员姓名</th>
                        <th>岗位</th>
                        <th>使用状态</th>
                        <th>全站权限</th>
                        <th>最近登录</th>
                        {canManageAdmins ? <th>操作</th> : null}
                      </tr>
                    </thead>
                    <tbody>
                      {(adminUsers?.items ?? []).map((item) => (
                        <tr key={item.id}>
                          <td>{item.username}</td>
                          <td>{item.display_name}</td>
                          <td>{adminRoleLabel(item.role)}</td>
                          <td>{item.status === "active" ? "启用" : "停用"}</td>
                          <td>{item.is_super ? "是" : "否"}</td>
                          <td>{item.last_login_at ? formatDateTime(item.last_login_at) : "-"}</td>
                          {canManageAdmins ? (
                            <td>
                              <button className="secondary-button small-button" onClick={() => startEditAdminUser(item)} type="button">
                                调整
                              </button>
                            </td>
                          ) : null}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  {(adminUsers?.items ?? []).length === 0 ? <div className="feedback-banner">当前还没有后台账号。</div> : null}
                </div>
                <PagerControls
                  page={adminUsers?.page ?? 1}
                  total={adminUsers?.total ?? 0}
                  pageSize={adminUsers?.page_size ?? adminPageSize}
                  onChange={setAdminUserPage}
                />
              </article>

              <article className="content-card">
                <h2>岗位权限模板</h2>
                <div className="role-grid">
                  {roles.map((role) => (
                    <div className="role-card" key={role.key}>
                      <div className="role-card-header">
                        <strong>{adminRoleLabel(role.key, role.name)}</strong>
                        <span className="pill pill-muted">{role.key}</span>
                      </div>
                      <p>{role.description}</p>
                      <div className="tag-list">
                        {role.permissions.map((permission) => (
                          <span className="tag" key={permission} title={permission}>
                            {permissionLabel(permission)}
                          </span>
                        ))}
                      </div>
                      {!role.system && canManageAdmins ? (
                        <button className="secondary-button small-button" onClick={() => startEditRole(role)} type="button">
                          调整岗位模板
                        </button>
                      ) : (
                        <span className="helper-text">{role.system ? "系统内置岗位模板" : "只读"}</span>
                      )}
                    </div>
                  ))}
                </div>
              </article>
            </section>
          ) : null}
        </main>
      </div>
    </div>
  );
}

function SummaryCard(props: { label: string; value: string; help: string }) {
  return (
    <article className="summary-card">
      <p>{props.label}</p>
      <strong>{props.value}</strong>
      <span>{props.help}</span>
    </article>
  );
}

function DataTable(props: { columns: string[]; rows: string[][]; emptyText: string }) {
  if (props.rows.length === 0) {
    return <div className="feedback-banner">{props.emptyText}</div>;
  }

  return (
    <div className="table-wrap">
      <table className="data-table">
        <thead>
          <tr>
            {props.columns.map((column) => (
              <th key={column}>{column}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {props.rows.map((row, index) => (
            <tr key={index}>
              {row.map((cell, cellIndex) => (
                <td key={`${index}-${cellIndex}`}>{cell}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function PagerControls(props: {
  page: number;
  total: number;
  pageSize: number;
  onChange: (page: number) => void;
}) {
  const totalPages = Math.max(1, Math.ceil(props.total / Math.max(props.pageSize, 1)));

  return (
    <div className="pager">
      <button
        className="secondary-button small-button"
        disabled={props.page <= 1}
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
        disabled={props.page >= totalPages}
        onClick={() => props.onChange(Math.min(totalPages, props.page + 1))}
        type="button"
      >
        下一页
      </button>
    </div>
  );
}

function parsePermissionInput(value: string) {
  const seen = new Set<string>();
  const result: string[] = [];

  value
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean)
    .forEach((item) => {
      if (seen.has(item)) {
        return;
      }
      seen.add(item);
      result.push(item);
    });

  return result;
}

function parseLineList(value: string) {
  const seen = new Set<string>();
  const result: string[] = [];

  value
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean)
    .forEach((item) => {
      if (seen.has(item)) {
        return;
      }
      seen.add(item);
      result.push(item);
    });

  return result;
}

function toDateTimeLocalValue(value?: string) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const offset = date.getTimezoneOffset() * 60000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

function fromDateTimeLocalValue(value: string) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  return date.toISOString();
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

function yuanToCents(value: string) {
  const normalized = value.trim();
  if (!normalized) {
    return 0;
  }
  const amount = Number(normalized);
  if (!Number.isFinite(amount) || amount < 0) {
    return 0;
  }
  return Math.round(amount * 100);
}

function centsToYuanInput(value: number) {
  if (!Number.isFinite(value)) {
    return "0.00";
  }
  return (value / 100).toFixed(2);
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

function formatFileSize(bytes?: number) {
  if (!bytes || bytes <= 0) {
    return "0 B";
  }
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
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

function subscriptionStatusClass(status?: string) {
  switch (status) {
    case "active":
      return "pill-success";
    case "cancelled":
      return "pill-danger";
    case "expired":
      return "pill-muted";
    case "pending":
      return "pill-warning";
    default:
      return "pill-danger";
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

function looksLikeAuthError(message: string) {
  return /(authorization|token|expired|claims|unauthorized)/i.test(message);
}

function paymentChannelLabel(channel?: string) {
  switch (channel) {
    case "wechat_native":
      return "微信扫码支付";
    case "wechat_jsapi":
      return "微信内支付";
    case "wechat_contract_pay":
      return "微信自动续费";
    case "wechat":
      return "微信支付";
    case "bank":
      return "银行卡";
    case "alipay":
      return "支付宝";
    case "cash":
      return "线下收款";
    default:
      return channel || "-";
  }
}

function formatPaymentChannelLabels(channels: string[]) {
  if (!channels.length) {
    return "微信支付";
  }
  return channels.map((channel) => paymentChannelLabel(channel)).join(" / ");
}

function formatPaymentProviderLabel(provider?: string) {
  return paymentChannelLabel(provider);
}

function permissionLabel(permission: string) {
  switch (permission) {
    case "*":
      return "全部操作范围";
    case "admin.read":
      return "查看后台账号";
    case "admin.write":
      return "管理后台账号";
    case "catalog.read":
      return "查看学习内容";
    case "catalog.write":
      return "管理学习内容";
    case "site.read":
      return "查看站点展示";
    case "site.write":
      return "修改站点展示";
    case "payment.read":
      return "查看订单与收款";
    case "payment.write":
      return "处理订单与会员";
    case "plan.read":
      return "查看会员方案";
    case "plan.write":
      return "管理会员方案";
    case "learner.read":
      return "查看学员账号";
    case "learner.write":
      return "管理学员账号";
    default:
      return permission;
  }
}

function adminRoleLabel(roleKey: string, fallbackName?: string) {
  switch (roleKey) {
    case "super_admin":
      return "站点总负责人";
    case "content_admin":
      return "内容运营";
    case "viewer":
      return "数据查看";
    default:
      return fallbackName || roleKey;
  }
}
