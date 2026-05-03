export interface Subject {
  id?: number;
  key: string;
  name: string;
  description: string;
  sort: number;
  featured: boolean;
}

export interface Category {
  id: number;
  subject_id: number;
  subject_key?: string;
  kind: string;
  key: string;
  name: string;
  description: string;
  sort: number;
  enabled: boolean;
}

export interface Grade {
  id: number;
  key: string;
  name: string;
  stage: string;
  description: string;
  sort: number;
  enabled: boolean;
}

export interface ClassificationStat {
  name: string;
  count: number;
  free_count: number;
  vip_count: number;
  accessible_count: number;
  requires_membership: boolean;
  has_member_content: boolean;
}

export interface Word {
  id: number;
  legacy_id?: number;
  subject_id?: number;
  subject_key: string;
  category_id?: number;
  category_name?: string;
  grade_id?: number;
  grade_name?: string;
  term: string;
  translation: string;
  classification: string;
  source?: string;
  phonetics?: string;
  explanation?: string;
  is_vip?: boolean;
}

export interface PagedWords {
  items: Word[];
  total: number;
  page: number;
  page_size: number;
}

export interface PagedClassificationStats {
  items: ClassificationStat[];
  total: number;
  page: number;
  page_size: number;
}

export interface PagedCategories {
  items: Category[];
  total: number;
  page: number;
  page_size: number;
}

export interface PagedGrades {
  items: Grade[];
  total: number;
  page: number;
  page_size: number;
}

export interface CatalogStats {
  subject_count: number;
  word_count: number;
  classification_count: number;
  grade_count?: number;
  admin_count?: number;
  data_source: string;
  sample_data: boolean;
  super_admin_initialized?: boolean;
}

export interface Plan {
  id?: number;
  key: string;
  name: string;
  billing_mode: string;
  price_cents: number;
  description: string;
  recommended: boolean;
  payment_channels: string[];
  features: string[];
}

export interface AdminUser {
  id: number;
  username: string;
  display_name: string;
  role: string;
  is_super: boolean;
  status: string;
  last_login_at?: string;
}

export interface AdminRole {
  id: number;
  key: string;
  name: string;
  description: string;
  permissions: string[];
  system: boolean;
  sort: number;
}

export interface AdminSession {
  access_token: string;
  token_type: string;
  expires_at: string;
  admin: AdminUser;
}

export interface LearnerUser {
  id: number;
  username: string;
  display_name: string;
  status: string;
  created_at: string;
  membership?: SubscriptionStatus;
}

export interface CaptchaChallenge {
  scene: string;
  captcha_id: string;
  image_data: string;
  expires_in: number;
}

export interface SiteSetting {
  site_name: string;
  site_icon: string;
  site_tagline: string;
  hero_title: string;
  hero_description: string;
  seo_headline: string;
  seo_title: string;
  seo_description: string;
  seo_keywords: string;
  footer_text: string;
  contact_email: string;
}

export interface SaveSiteSettingInput {
  site_name: string;
  site_icon: string;
  site_tagline: string;
  hero_title: string;
  hero_description: string;
  seo_headline: string;
  seo_title: string;
  seo_description: string;
  seo_keywords: string;
  footer_text: string;
  contact_email: string;
}

export interface LearnerSession {
  access_token: string;
  token_type: string;
  expires_at: string;
  user: LearnerUser;
}

export interface AdminLearnerUser {
  id: number;
  username: string;
  display_name: string;
  status: string;
  created_at: string;
  purchase_count: number;
  has_membership: boolean;
  membership_status: string;
  current_plan_key: string;
  current_period_end?: string;
  last_order_paid_at?: string;
  last_membership_at?: string;
}

export interface PagedLearnerUsers {
  items: AdminLearnerUser[];
  total: number;
  page: number;
  page_size: number;
}

export interface UpdateLearnerUserInput {
  display_name: string;
  status: string;
}

export interface AdminSetupStatus {
  initialized: boolean;
  admin_count: number;
}

export interface PagedAdminUsers {
  items: AdminUser[];
  total: number;
  page: number;
  page_size: number;
}

export interface ImportResult {
  imported_count: number;
  created_categories: number;
  subject_key: string;
  path: string;
  replace: boolean;
}

export interface WechatPayConfig {
  id: number;
  auth_mode: string;
  mch_id: string;
  app_id: string;
  merchant_serial_no: string;
  notify_url: string;
  description_prefix: string;
  time_expire_minutes: number;
  wechatpay_public_key_id: string;
  apiv3_key: string;
  wechatpay_public_key: string;
  key_pem: string;
  has_apiv3_key: boolean;
  has_wechatpay_public_key: boolean;
  has_cert_pem: boolean;
  has_key_pem: boolean;
  has_platform_cert: boolean;
  ready_for_checkout: boolean;
  validation_error?: string;
  updated_at: string;
}

export interface SaveWechatPayConfigInput {
  mch_id: string;
  app_id: string;
  auth_mode: string;
  merchant_serial_no: string;
  apiv3_key?: string;
  clear_apiv3_key?: boolean;
  platform_cert_serial_no?: string;
  notify_url: string;
  description_prefix: string;
  time_expire_minutes: number;
  wechatpay_public_key_id: string;
  wechatpay_public_key?: string;
  clear_wechatpay_public_key?: boolean;
  cert_pem?: string;
  clear_cert_pem?: boolean;
  key_pem?: string;
  clear_key_pem?: boolean;
  platform_cert?: string;
  clear_platform_cert?: boolean;
}

export interface WechatOrder {
  order_no: string;
  plan_id?: number;
  plan_key: string;
  subject_key: string;
  customer_ref: string;
  description: string;
  billing_mode: string;
  amount_cents: number;
  currency: string;
  provider?: string;
  provider_trade_no?: string;
  code_url: string;
  status: string;
  error_message?: string;
  paid_at?: string;
  expires_at?: string;
  created_at: string;
  updated_at?: string;
}

export interface SubscriptionStatus {
  id: number;
  customer_ref: string;
  plan_id?: number;
  plan_key: string;
  subject_key: string;
  status: string;
  auto_renew: boolean;
  provider?: string;
  provider_contract_id?: string;
  started_at?: string;
  current_period_start?: string;
  current_period_end?: string;
  cancelled_at?: string;
  created_at?: string;
  updated_at?: string;
}

export interface PaymentOrderStatus {
  order: WechatOrder;
  subscription?: SubscriptionStatus;
}

export interface PagedPaymentOrders {
  items: WechatOrder[];
  total: number;
  page: number;
  page_size: number;
}

export interface PagedSubscriptions {
  items: SubscriptionStatus[];
  total: number;
  page: number;
  page_size: number;
}

export interface CreateSubjectInput {
  key: string;
  name: string;
  description: string;
  sort: number;
  featured: boolean;
}

export interface UpdateSubjectInput {
  key: string;
  name: string;
  description: string;
  sort: number;
  featured: boolean;
}

export interface CreateCategoryInput {
  subject_key: string;
  kind: string;
  key: string;
  name: string;
  description: string;
  sort: number;
  enabled: boolean;
}

export interface UpdateCategoryInput {
  subject_key: string;
  kind: string;
  key: string;
  name: string;
  description: string;
  sort: number;
  enabled: boolean;
}

export interface CreateGradeInput {
  key: string;
  name: string;
  stage: string;
  description: string;
  sort: number;
  enabled: boolean;
}

export interface UpdateGradeInput {
  key: string;
  name: string;
  stage: string;
  description: string;
  sort: number;
  enabled: boolean;
}

export interface CreateWordInput {
  legacy_id?: number;
  subject_key: string;
  classification?: string;
  category_name?: string;
  grade_id?: number | null;
  term: string;
  translation: string;
  source: string;
  phonetics: string;
  explanation: string;
  is_vip: boolean;
}

export interface UpdateWordInput {
 legacy_id?: number;
 subject_key: string;
  classification?: string;
  category_name?: string;
  grade_id?: number | null;
  term: string;
  translation: string;
  source: string;
  phonetics: string;
  explanation: string;
 is_vip: boolean;
}

export interface BatchUpdateWordVIPInput {
  subject_key: string;
  category_id?: number;
  classification?: string;
  is_vip: boolean;
}

export interface BatchUpdateWordVIPResult {
  subject_key: string;
  category_id?: number;
  classification: string;
  is_vip: boolean;
  updated_count: number;
}

export interface CreateAdminUserInput {
  username: string;
  password: string;
  display_name: string;
  role: string;
  status: string;
  is_super?: boolean;
}

export interface UpdateAdminUserInput {
  display_name: string;
  password?: string;
  role: string;
  status: string;
  is_super?: boolean;
}

export interface CreateAdminRoleInput {
  key: string;
  name: string;
  description: string;
  permissions: string[];
  sort: number;
}

export interface UpdateAdminRoleInput {
  name: string;
  description: string;
  permissions: string[];
  sort: number;
}

export interface UpdatePlanInput {
  name: string;
  billing_mode: string;
  price_cents: number;
  description: string;
  recommended: boolean;
  payment_channels: string[];
  features: string[];
}

export interface UpdateSubscriptionInput {
  plan_key: string;
  status: string;
  auto_renew: boolean;
  started_at: string;
  current_period_start: string;
  current_period_end: string;
  cancelled_at: string;
  clear_started_at?: boolean;
  clear_current_period_start?: boolean;
  clear_current_period_end?: boolean;
  clear_cancelled_at?: boolean;
}

export const apiBaseUrl = (import.meta.env.VITE_API_BASE_URL ?? "").replace(/\/$/, "");
const baseUrl = apiBaseUrl;

export interface MCPInfoTool {
  name: string;
  title?: string;
  description: string;
  inputSchema: Record<string, unknown>;
  outputSchema?: Record<string, unknown>;
}

export interface MCPInfo {
  name: string;
  version: string;
  protocolVersion: string;
  websocketPath: string;
  websocketURL: string;
  availableMethods: string[];
  tools: MCPInfoTool[];
  auth?: {
    mode?: string;
    queryTokenParam?: string;
    querySubjectParam?: string;
    requiresMembership?: boolean;
  };
  examples?: Record<string, unknown>;
}

export interface MCPEndpoint {
  id: number;
  learner_user_id?: number;
  name: string;
  url: string;
  description: string;
  enabled: boolean;
  token_query_param: string;
  subject_query_param: string;
  connection_status?: string;
  is_connected?: boolean;
  last_error?: string;
  connected_at?: string;
  created_at: string;
  updated_at: string;
}

export interface SaveMCPEndpointInput {
  name: string;
  url: string;
  description: string;
  enabled: boolean;
  token_query_param: string;
  subject_query_param: string;
}

export interface MCPEndpointToolsResponse {
  endpoint_id: number;
  endpoint_name: string;
  tool_count: number;
  tools: MCPInfoTool[];
}

export interface RefreshLearnerMCPConnectionsResponse {
  success: boolean;
  endpoints: MCPEndpoint[];
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${baseUrl}${path}`, init);
  if (!response.ok) {
    let message = `Request failed: ${response.status}`;
    try {
      const payload = (await response.json()) as { error?: string };
      if (payload.error) {
        message = payload.error;
      }
    } catch {
      // Ignore JSON parsing errors for non-JSON responses.
    }
    throw new Error(message);
  }
  return response.json() as Promise<T>;
}

function authHeaders(token: string) {
  return {
    Authorization: `Bearer ${token}`,
    "Content-Type": "application/json",
  };
}

function resolveHttpBase() {
  if (baseUrl) {
    return baseUrl;
  }
  if (typeof window !== "undefined") {
    return window.location.origin;
  }
  return "";
}

export function buildMCPWebSocketUrl(subjectKey: string) {
  const httpBase = resolveHttpBase();
  if (!httpBase) {
    return "";
  }

  const url = new URL(httpBase);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  url.pathname = "/mcp";
  url.search = "";

  const trimmedSubject = subjectKey.trim();
  if (trimmedSubject) {
    url.searchParams.set("subject", trimmedSubject);
  }

  return url.toString();
}

export function buildMCPWebSocketUrlWithToken(subjectKey: string, token: string) {
  const urlString = buildMCPWebSocketUrl(subjectKey);
  if (!urlString) {
    return "";
  }

  const url = new URL(urlString);
  if (token.trim()) {
    url.searchParams.set("token", token.trim());
  }
  return url.toString();
}

export function buildRemoteMCPWebSocketUrl(
  endpoint: Pick<MCPEndpoint, "url" | "token_query_param" | "subject_query_param">,
  options: {
    token?: string;
    subjectKey?: string;
  },
) {
  const rawURL = endpoint.url.trim();
  if (!rawURL) {
    return "";
  }

  let url: URL;
  try {
    url = new URL(rawURL);
  } catch {
    return rawURL;
  }

  if (options.token?.trim() && endpoint.token_query_param.trim()) {
    url.searchParams.set(endpoint.token_query_param.trim(), options.token.trim());
  }
  if (options.subjectKey?.trim() && endpoint.subject_query_param.trim()) {
    url.searchParams.set(endpoint.subject_query_param.trim(), options.subjectKey.trim());
  }

  return url.toString();
}

export const api = {
  getSubjects() {
    return request<Subject[]>("/api/v1/subjects");
  },
  getStats() {
    return request<CatalogStats>("/api/v1/stats");
  },
  getClassifications(params: { subjectKey: string; page: number; pageSize: number; token?: string }) {
    const search = new URLSearchParams({
      subject: params.subjectKey,
      page: String(params.page),
      page_size: String(params.pageSize),
    });
    return request<PagedClassificationStats>(`/api/v1/classifications?${search.toString()}`, {
      headers: params.token
        ? {
            Authorization: `Bearer ${params.token}`,
          }
        : undefined,
    });
  },
  getCategories(subjectKey: string, kind = "topic") {
    const search = new URLSearchParams();
    if (subjectKey) {
      search.set("subject", subjectKey);
    }
    if (kind) {
      search.set("kind", kind);
    }
    return request<Category[]>(`/api/v1/categories?${search.toString()}`);
  },
  getWords(params: {
    subjectKey: string;
    classification: string;
    query: string;
    page: number;
    pageSize: number;
    token?: string;
  }) {
    const search = new URLSearchParams({
      subject: params.subjectKey,
      page: String(params.page),
      page_size: String(params.pageSize),
    });
    if (params.classification) {
      search.set("classification", params.classification);
    }
    if (params.query) {
      search.set("q", params.query);
    }
    return request<PagedWords>(`/api/v1/words?${search.toString()}`, {
      headers: params.token
        ? {
            Authorization: `Bearer ${params.token}`,
          }
        : undefined,
    });
  },
  getPlans() {
    return request<Plan[]>("/api/v1/plans");
  },
  getMCPInfo(subjectKey?: string) {
    const search = new URLSearchParams();
    if (subjectKey?.trim()) {
      search.set("subject", subjectKey.trim());
    }
    const query = search.toString();
    return request<MCPInfo>(query ? `/mcp/info?${query}` : "/mcp/info");
  },
  listLearnerMCPEndpoints(token: string) {
    return request<MCPEndpoint[]>("/api/v1/auth/mcp/endpoints", {
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  createLearnerMCPEndpoint(token: string, payload: SaveMCPEndpointInput) {
    return request<MCPEndpoint>("/api/v1/auth/mcp/endpoints", {
      method: "POST",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  updateLearnerMCPEndpoint(token: string, id: number, payload: SaveMCPEndpointInput) {
    return request<MCPEndpoint>(`/api/v1/auth/mcp/endpoints/${id}`, {
      method: "PUT",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  getLearnerMCPEndpointTools(token: string, id: number) {
    return request<MCPEndpointToolsResponse>(`/api/v1/auth/mcp/endpoints/${id}/tools`, {
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  getLearnerMCPEndpointStatus(token: string, id: number) {
    return request<MCPEndpoint>(`/api/v1/auth/mcp/endpoints/${id}/status`, {
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  refreshLearnerMCPConnections(token: string) {
    return request<RefreshLearnerMCPConnectionsResponse>("/api/v1/auth/mcp/refresh", {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  deleteLearnerMCPEndpoint(token: string, id: number) {
    return request<{ success: boolean }>(`/api/v1/auth/mcp/endpoints/${id}`, {
      method: "DELETE",
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  getSiteSettings() {
    return request<SiteSetting>("/api/v1/site/settings");
  },
  getCaptcha(scene: "learner_register" | "learner_login") {
    return request<CaptchaChallenge>(`/api/v1/auth/captcha?scene=${encodeURIComponent(scene)}`);
  },
  adminSetupStatus() {
    return request<AdminSetupStatus>("/api/v1/admin/setup/status");
  },
  adminSetupBootstrap(payload: { username: string; password: string; display_name: string }) {
    return request<AdminSession>("/api/v1/admin/setup/bootstrap", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  },
  learnerRegister(payload: {
    username: string;
    password: string;
    display_name: string;
    captcha_id: string;
    captcha_answer: string;
  }) {
    return request<LearnerSession>("/api/v1/auth/register", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  },
  learnerLogin(username: string, password: string, captcha_id: string, captcha_answer: string) {
    return request<LearnerSession>("/api/v1/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password, captcha_id, captcha_answer }),
    });
  },
  learnerMe(token: string, subjectKey?: string) {
    const search = new URLSearchParams();
    if (subjectKey) {
      search.set("subject", subjectKey);
    }
    return request<LearnerUser>(`/api/v1/auth/me${search.toString() ? `?${search.toString()}` : ""}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  learnerLogout(token: string) {
    return request<{ success: boolean }>("/api/v1/auth/logout", {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  adminCaptcha() {
    return request<CaptchaChallenge>("/api/v1/admin/auth/captcha");
  },
  adminLogin(username: string, password: string, captcha_id: string, captcha_answer: string) {
    return request<AdminSession>("/api/v1/admin/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password, captcha_id, captcha_answer }),
    });
  },
  adminMe(token: string) {
    return request<AdminUser>("/api/v1/admin/auth/me", {
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  adminRefresh(token: string) {
    return request<AdminSession>("/api/v1/admin/auth/refresh", {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  adminLogout(token: string) {
    return request<{ success: boolean }>("/api/v1/admin/auth/logout", {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  adminChangePassword(token: string, oldPassword: string, newPassword: string) {
    return request<{ success: boolean }>("/api/v1/admin/auth/change-password", {
      method: "POST",
      headers: authHeaders(token),
      body: JSON.stringify({
        old_password: oldPassword,
        new_password: newPassword,
      }),
    });
  },
  adminRoles(token: string) {
    return request<AdminRole[]>("/api/v1/admin/roles", {
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  adminUsers(
    token: string,
    params: { page: number; pageSize: number; query?: string; role?: string; status?: string },
  ) {
    const search = new URLSearchParams({
      page: String(params.page),
      page_size: String(params.pageSize),
    });
    if (params.query) {
      search.set("q", params.query);
    }
    if (params.role) {
      search.set("role", params.role);
    }
    if (params.status) {
      search.set("status", params.status);
    }
    return request<PagedAdminUsers>(`/api/v1/admin/users?${search.toString()}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  adminLearners(token: string, params: { page: number; pageSize: number; query?: string; status?: string }) {
    const search = new URLSearchParams({
      page: String(params.page),
      page_size: String(params.pageSize),
    });
    if (params.query) {
      search.set("q", params.query);
    }
    if (params.status) {
      search.set("status", params.status);
    }
    return request<PagedLearnerUsers>(`/api/v1/admin/learners?${search.toString()}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  adminUpdateLearner(token: string, id: number, payload: UpdateLearnerUserInput) {
    return request<LearnerUser>(`/api/v1/admin/learners/${id}`, {
      method: "PUT",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  adminSiteSettings(token: string) {
    return request<SiteSetting>("/api/v1/admin/site/settings", {
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  adminSaveSiteSettings(token: string, payload: SaveSiteSettingInput) {
    return request<SiteSetting>("/api/v1/admin/site/settings", {
      method: "PUT",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  adminWords(
    token: string,
    params: {
      page: number;
      pageSize: number;
      query: string;
      subjectKey?: string;
      classification?: string;
    },
  ) {
    const search = new URLSearchParams({
      page: String(params.page),
      page_size: String(params.pageSize),
    });
    if (params.query) {
      search.set("q", params.query);
    }
    if (params.subjectKey) {
      search.set("subject", params.subjectKey);
    }
    if (params.classification) {
      search.set("classification", params.classification);
    }
    return request<PagedWords>(`/api/v1/admin/words?${search.toString()}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  adminCategories(
    token: string,
    params: { page: number; pageSize: number; query: string; kind: string; subject: string },
  ) {
    const search = new URLSearchParams({
      page: String(params.page),
      page_size: String(params.pageSize),
      kind: params.kind,
    });
    if (params.query) {
      search.set("q", params.query);
    }
    if (params.subject) {
      search.set("subject", params.subject);
    }
    return request<PagedCategories>(`/api/v1/admin/categories?${search.toString()}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  adminGrades(token: string, params: { page: number; pageSize: number; query: string }) {
    const search = new URLSearchParams({
      page: String(params.page),
      page_size: String(params.pageSize),
    });
    if (params.query) {
      search.set("q", params.query);
    }
    return request<PagedGrades>(`/api/v1/admin/grades?${search.toString()}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  adminImportLocal(
    token: string,
    payload: { file: File; subject_key: string; replace: boolean },
  ) {
    const formData = new FormData();
    formData.set("file", payload.file);
    formData.set("subject_key", payload.subject_key);
    formData.set("replace", String(payload.replace));
    return request<ImportResult>("/api/v1/admin/import/local", {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
      },
      body: formData,
    });
  },
  adminCreateSubject(token: string, payload: CreateSubjectInput) {
    return request<Subject>("/api/v1/admin/subjects", {
      method: "POST",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  adminUpdateSubject(token: string, id: number, payload: UpdateSubjectInput) {
    return request<Subject>(`/api/v1/admin/subjects/${id}`, {
      method: "PUT",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  adminCreateCategory(token: string, payload: CreateCategoryInput) {
    return request<Category>("/api/v1/admin/categories", {
      method: "POST",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  adminUpdateCategory(token: string, id: number, payload: UpdateCategoryInput) {
    return request<Category>(`/api/v1/admin/categories/${id}`, {
      method: "PUT",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  adminCreateGrade(token: string, payload: CreateGradeInput) {
    return request<Grade>("/api/v1/admin/grades", {
      method: "POST",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  adminUpdateGrade(token: string, id: number, payload: UpdateGradeInput) {
    return request<Grade>(`/api/v1/admin/grades/${id}`, {
      method: "PUT",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  adminCreateWord(token: string, payload: CreateWordInput) {
    return request<Word>("/api/v1/admin/words", {
      method: "POST",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  adminUpdateWord(token: string, id: number, payload: UpdateWordInput) {
    return request<Word>(`/api/v1/admin/words/${id}`, {
      method: "PUT",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  adminBatchUpdateWordVIP(token: string, payload: BatchUpdateWordVIPInput) {
    return request<BatchUpdateWordVIPResult>("/api/v1/admin/words/batch-vip", {
      method: "PUT",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  adminPlans(token: string) {
    return request<Plan[]>("/api/v1/admin/plans", {
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  adminCreatePlan(token: string, payload: Plan) {
    return request<Plan>("/api/v1/admin/plans", {
      method: "POST",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  adminUpdatePlan(token: string, id: number, payload: UpdatePlanInput) {
    return request<Plan>(`/api/v1/admin/plans/${id}`, {
      method: "PUT",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  adminDeletePlan(token: string, id: number) {
    return request<{ success: boolean }>(`/api/v1/admin/plans/${id}`, {
      method: "DELETE",
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  adminCreateUser(token: string, payload: CreateAdminUserInput) {
    return request<AdminUser>("/api/v1/admin/users", {
      method: "POST",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  adminUpdateUser(token: string, id: number, payload: UpdateAdminUserInput) {
    return request<AdminUser>(`/api/v1/admin/users/${id}`, {
      method: "PUT",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  adminCreateRole(token: string, payload: CreateAdminRoleInput) {
    return request<AdminRole>("/api/v1/admin/roles", {
      method: "POST",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  adminUpdateRole(token: string, id: number, payload: UpdateAdminRoleInput) {
    return request<AdminRole>(`/api/v1/admin/roles/${id}`, {
      method: "PUT",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  adminWechatPayConfig(token: string) {
    return request<{ exists: boolean; config?: WechatPayConfig }>("/api/v1/admin/wechatpay/config", {
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  adminSaveWechatPayConfig(token: string, payload: SaveWechatPayConfigInput) {
    return request<WechatPayConfig>("/api/v1/admin/wechatpay/config", {
      method: "POST",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  adminPaymentOrders(
    token: string,
    params: {
      page: number;
      pageSize: number;
      query?: string;
      status?: string;
      subject?: string;
      customerRef?: string;
      planKey?: string;
    },
  ) {
    const search = new URLSearchParams({
      page: String(params.page),
      page_size: String(params.pageSize),
    });
    if (params.query) {
      search.set("q", params.query);
    }
    if (params.status) {
      search.set("status", params.status);
    }
    if (params.subject) {
      search.set("subject", params.subject);
    }
    if (params.customerRef) {
      search.set("customer_ref", params.customerRef);
    }
    if (params.planKey) {
      search.set("plan_key", params.planKey);
    }
    return request<PagedPaymentOrders>(`/api/v1/admin/payments/orders?${search.toString()}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  adminPaymentOrderDetail(token: string, orderNo: string) {
    return request<PaymentOrderStatus>(`/api/v1/admin/payments/orders/${encodeURIComponent(orderNo)}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  adminSubscriptions(
    token: string,
    params: {
      page: number;
      pageSize: number;
      query?: string;
      status?: string;
      subject?: string;
      customerRef?: string;
      planKey?: string;
    },
  ) {
    const search = new URLSearchParams({
      page: String(params.page),
      page_size: String(params.pageSize),
    });
    if (params.query) {
      search.set("q", params.query);
    }
    if (params.status) {
      search.set("status", params.status);
    }
    if (params.subject) {
      search.set("subject", params.subject);
    }
    if (params.customerRef) {
      search.set("customer_ref", params.customerRef);
    }
    if (params.planKey) {
      search.set("plan_key", params.planKey);
    }
    return request<PagedSubscriptions>(`/api/v1/admin/payments/subscriptions?${search.toString()}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  adminSubscription(token: string, id: number) {
    return request<SubscriptionStatus>(`/api/v1/admin/payments/subscriptions/${id}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
  },
  adminUpdateSubscription(token: string, id: number, payload: UpdateSubscriptionInput) {
    return request<SubscriptionStatus>(`/api/v1/admin/payments/subscriptions/${id}`, {
      method: "PUT",
      headers: authHeaders(token),
      body: JSON.stringify(payload),
    });
  },
  createWechatOrder(payload: {
    plan_id?: number;
    plan_key?: string;
    subject_key?: string;
    customer_ref: string;
    description?: string;
  }) {
    return request<WechatOrder>("/api/v1/payments/wechat/orders", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  },
  getWechatOrderStatus(orderNo: string, customerRef?: string) {
    const search = new URLSearchParams();
    if (customerRef) {
      search.set("customer_ref", customerRef);
    }
    const query = search.toString();
    const path = query
      ? `/api/v1/payments/wechat/orders/${encodeURIComponent(orderNo)}?${query}`
      : `/api/v1/payments/wechat/orders/${encodeURIComponent(orderNo)}`;
    return request<PaymentOrderStatus>(path);
  },
};
