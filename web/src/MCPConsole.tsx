import { useEffect, useMemo, useState } from "react";

import {
  api,
  type MCPEndpoint,
  type MCPInfo,
  type MCPInfoTool,
  type SaveMCPEndpointInput,
} from "./api";

type MCPConsoleProps = {
  subjectKey: string;
  token: string;
  learnerName: string;
};

type EndpointFormState = SaveMCPEndpointInput;

const emptyEndpointForm: EndpointFormState = {
  name: "",
  url: "",
  description: "",
  enabled: true,
  token_query_param: "token",
  subject_query_param: "subject",
};

export default function MCPConsole(props: MCPConsoleProps) {
  const [mcpInfo, setMcpInfo] = useState<MCPInfo | null>(null);
  const [infoLoading, setInfoLoading] = useState(false);
  const [infoError, setInfoError] = useState("");

  const [endpoints, setEndpoints] = useState<MCPEndpoint[]>([]);
  const [endpointsLoading, setEndpointsLoading] = useState(false);
  const [endpointsError, setEndpointsError] = useState("");
  const [refreshing, setRefreshing] = useState(false);

  const [selectedEndpointID, setSelectedEndpointID] = useState<number | null>(null);
  const [toolPreviewMap, setToolPreviewMap] = useState<Record<number, MCPInfoTool[]>>({});
  const [toolPreviewVersionMap, setToolPreviewVersionMap] = useState<Record<number, string>>({});
  const [toolPreviewLoadingID, setToolPreviewLoadingID] = useState<number | null>(null);
  const [toolPreviewError, setToolPreviewError] = useState("");

  const [endpointModalOpen, setEndpointModalOpen] = useState(false);
  const [editingEndpointID, setEditingEndpointID] = useState<number | null>(null);
  const [endpointForm, setEndpointForm] = useState<EndpointFormState>(emptyEndpointForm);
  const [savingEndpoint, setSavingEndpoint] = useState(false);
  const [endpointActionBusyID, setEndpointActionBusyID] = useState<number | null>(null);
  const [endpointBusyMessage, setEndpointBusyMessage] = useState("");
  const [copiedMessage, setCopiedMessage] = useState("");

  const subjectLabel = props.subjectKey.trim() || "-";
  const learnerLabel = props.learnerName.trim() || "未登录";

  const enabledEndpoints = useMemo(() => endpoints.filter((item) => item.enabled), [endpoints]);
  const connectedEndpoints = useMemo(() => endpoints.filter((item) => item.is_connected), [endpoints]);
  const globalTools = useMemo(() => mcpInfo?.tools ?? [], [mcpInfo]);
  const selectedEndpoint = useMemo(
    () => endpoints.find((item) => item.id === selectedEndpointID) ?? endpoints[0] ?? null,
    [endpoints, selectedEndpointID],
  );
  const selectedEndpointTools = useMemo(() => {
    if (!selectedEndpoint) {
      return globalTools;
    }
    return toolPreviewMap[selectedEndpoint.id] ?? globalTools;
  }, [globalTools, selectedEndpoint, toolPreviewMap]);
  const selectedEndpointVersion = selectedEndpoint ? buildEndpointPreviewVersion(selectedEndpoint) : "";

  useEffect(() => {
    let active = true;
    setInfoLoading(true);
    setInfoError("");

    api
      .getMCPInfo(props.subjectKey)
      .then((payload) => {
        if (!active) {
          return;
        }
        setMcpInfo(payload);
      })
      .catch((error: Error) => {
        if (!active) {
          return;
        }
        setInfoError(error.message);
      })
      .finally(() => {
        if (!active) {
          return;
        }
        setInfoLoading(false);
      });

    return () => {
      active = false;
    };
  }, [props.subjectKey]);

  useEffect(() => {
    if (!props.token.trim()) {
      setEndpoints([]);
      setSelectedEndpointID(null);
      setToolPreviewMap({});
      setToolPreviewVersionMap({});
      return;
    }
    void loadEndpoints();
  }, [props.token]);

  useEffect(() => {
    if (endpoints.length === 0) {
      if (selectedEndpointID !== null) {
        setSelectedEndpointID(null);
      }
      return;
    }

    if (!selectedEndpointID || !endpoints.some((item) => item.id === selectedEndpointID)) {
      setSelectedEndpointID(endpoints[0].id);
    }
  }, [endpoints, selectedEndpointID]);

  useEffect(() => {
    if (!props.token.trim()) {
      return;
    }

    const timer = window.setInterval(() => {
      void loadEndpoints({ silent: true });
    }, 15000);

    return () => {
      window.clearInterval(timer);
    };
  }, [props.token]);

  useEffect(() => {
    if (!props.token.trim() || !selectedEndpoint) {
      return;
    }
    if (toolPreviewVersionMap[selectedEndpoint.id] === selectedEndpointVersion) {
      return;
    }
    void loadEndpointToolPreview(selectedEndpoint);
  }, [props.token, selectedEndpoint, selectedEndpointVersion, toolPreviewVersionMap]);

  async function loadEndpoints(options?: { silent?: boolean }) {
    if (!props.token.trim()) {
      return;
    }

    if (!options?.silent) {
      setEndpointsLoading(true);
      setEndpointsError("");
    }

    try {
      const items = await api.listLearnerMCPEndpoints(props.token);
      setEndpoints(items);
    } catch (error) {
      setEndpointsError(parseErrorMessage(error));
    } finally {
      if (!options?.silent) {
        setEndpointsLoading(false);
      }
    }
  }

  async function loadEndpointToolPreview(endpoint: MCPEndpoint, options?: { force?: boolean }) {
    if (!props.token.trim()) {
      return;
    }

    const version = buildEndpointPreviewVersion(endpoint);
    if (!options?.force && toolPreviewVersionMap[endpoint.id] === version) {
      return;
    }

    setToolPreviewLoadingID(endpoint.id);
    setToolPreviewError("");

    try {
      const payload = await api.getLearnerMCPEndpointToolsWithSubject(props.token, endpoint.id, props.subjectKey);
      setToolPreviewMap((current) => ({
        ...current,
        [endpoint.id]: payload.tools,
      }));
      setToolPreviewVersionMap((current) => ({
        ...current,
        [endpoint.id]: version,
      }));
    } catch (error) {
      setToolPreviewError(parseErrorMessage(error));
    } finally {
      setToolPreviewLoadingID(null);
    }
  }

  function selectEndpoint(endpoint: MCPEndpoint) {
    setSelectedEndpointID(endpoint.id);
    setToolPreviewError("");
    void loadEndpointToolPreview(endpoint);
  }

  async function refreshAllConnections() {
    if (!props.token.trim()) {
      setEndpointBusyMessage("请先登录学员账号，再刷新远程 WSS 连接状态。");
      return;
    }

    setRefreshing(true);
    setEndpointBusyMessage("");
    setEndpointsError("");
    setCopiedMessage("");

    try {
      const payload = await api.refreshLearnerMCPConnections(props.token);
      setEndpoints(payload.endpoints);
      setEndpointBusyMessage("已刷新所有远程接入点的连接状态。");
    } catch (error) {
      setEndpointsError(parseErrorMessage(error));
    } finally {
      setRefreshing(false);
    }
  }

  async function refreshSelectedEndpointStatus() {
    if (!props.token.trim() || !selectedEndpoint) {
      return;
    }

    setEndpointBusyMessage("");
    try {
      const payload = await api.getLearnerMCPEndpointStatus(props.token, selectedEndpoint.id);
      setEndpoints((current) => current.map((item) => (item.id === payload.id ? payload : item)));
      setToolPreviewVersionMap((current) => {
        const next = { ...current };
        delete next[payload.id];
        return next;
      });
    } catch (error) {
      setEndpointBusyMessage(parseErrorMessage(error));
    }
  }

  function resetEndpointForm() {
    setEndpointForm(emptyEndpointForm);
    setEditingEndpointID(null);
    setEndpointBusyMessage("");
  }

  function openCreateModal() {
    resetEndpointForm();
    setEndpointModalOpen(true);
  }

  function openEditModal(endpoint: MCPEndpoint) {
    setEditingEndpointID(endpoint.id);
    setEndpointForm({
      name: endpoint.name,
      url: endpoint.url,
      description: endpoint.description,
      enabled: endpoint.enabled,
      token_query_param: endpoint.token_query_param || "token",
      subject_query_param: endpoint.subject_query_param || "subject",
    });
    setEndpointBusyMessage("");
    setEndpointModalOpen(true);
  }

  function closeEndpointModal() {
    if (savingEndpoint) {
      return;
    }
    setEndpointModalOpen(false);
    resetEndpointForm();
  }

  async function saveEndpoint() {
    if (!props.token.trim()) {
      setEndpointBusyMessage("请先登录学员账号，再保存远程 ws / wss 地址。");
      return;
    }

    setSavingEndpoint(true);
    setEndpointBusyMessage("");

    try {
      const payload = {
        ...endpointForm,
        name: endpointForm.name.trim(),
        url: endpointForm.url.trim(),
        description: endpointForm.description.trim(),
        token_query_param: endpointForm.token_query_param.trim() || "token",
        subject_query_param: endpointForm.subject_query_param.trim() || "subject",
      };

      const saved = editingEndpointID
        ? await api.updateLearnerMCPEndpoint(props.token, editingEndpointID, payload)
        : await api.createLearnerMCPEndpoint(props.token, payload);

      setEndpoints((current) => {
        if (editingEndpointID) {
          return current.map((item) => (item.id === saved.id ? saved : item));
        }
        return [saved, ...current];
      });
      setSelectedEndpointID(saved.id);
      setToolPreviewMap((current) => {
        const next = { ...current };
        delete next[saved.id];
        return next;
      });
      setToolPreviewVersionMap((current) => {
        const next = { ...current };
        delete next[saved.id];
        return next;
      });
      setEndpointModalOpen(false);
      resetEndpointForm();
      setEndpointBusyMessage(editingEndpointID ? "远程接入点已更新。" : "新的远程接入点已添加。");
      void loadEndpoints({ silent: true });
    } catch (error) {
      setEndpointBusyMessage(parseErrorMessage(error));
    } finally {
      setSavingEndpoint(false);
    }
  }

  async function toggleEndpointEnabled(endpoint: MCPEndpoint) {
    if (!props.token.trim()) {
      setEndpointBusyMessage("请先登录学员账号，再变更远程接入点状态。");
      return;
    }

    setEndpointActionBusyID(endpoint.id);
    setEndpointBusyMessage("");

    try {
      const payload: SaveMCPEndpointInput = {
        name: endpoint.name,
        url: endpoint.url,
        description: endpoint.description,
        enabled: !endpoint.enabled,
        token_query_param: endpoint.token_query_param || "token",
        subject_query_param: endpoint.subject_query_param || "subject",
      };

      const saved = await api.updateLearnerMCPEndpoint(props.token, endpoint.id, payload);
      setEndpoints((current) => current.map((item) => (item.id === saved.id ? saved : item)));
      setToolPreviewMap((current) => {
        const next = { ...current };
        delete next[saved.id];
        return next;
      });
      setToolPreviewVersionMap((current) => {
        const next = { ...current };
        delete next[saved.id];
        return next;
      });
      setEndpointBusyMessage(saved.enabled ? `已启动接入点“${saved.name}”。` : `已停用接入点“${saved.name}”。`);
    } catch (error) {
      setEndpointBusyMessage(parseErrorMessage(error));
    } finally {
      setEndpointActionBusyID(null);
    }
  }

  async function deleteEndpoint(endpoint: MCPEndpoint) {
    if (!props.token.trim()) {
      setEndpointBusyMessage("请先登录学员账号，再删除远程接入点。");
      return;
    }
    if (!window.confirm(`确认删除接入点“${endpoint.name}”吗？`)) {
      return;
    }

    setEndpointBusyMessage("");
    try {
      await api.deleteLearnerMCPEndpoint(props.token, endpoint.id);
      setEndpoints((current) => current.filter((item) => item.id !== endpoint.id));
      setToolPreviewMap((current) => {
        const next = { ...current };
        delete next[endpoint.id];
        return next;
      });
      setToolPreviewVersionMap((current) => {
        const next = { ...current };
        delete next[endpoint.id];
        return next;
      });
      if (selectedEndpointID === endpoint.id) {
        setSelectedEndpointID(null);
      }
      setEndpointBusyMessage("远程接入点已删除。");
    } catch (error) {
      setEndpointBusyMessage(parseErrorMessage(error));
    }
  }

  async function copyText(text: string, message: string) {
    if (!text.trim()) {
      return;
    }

    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(text);
      } else {
        const textarea = document.createElement("textarea");
        textarea.value = text;
        textarea.setAttribute("readonly", "true");
        textarea.style.position = "absolute";
        textarea.style.left = "-9999px";
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand("copy");
        textarea.remove();
      }
      setCopiedMessage(message);
      setEndpointBusyMessage("");
    } catch {
      setCopiedMessage("复制失败，请手动复制。");
    }
  }

  const overallStatus = resolveOverallStatus(endpoints, props.token, endpointsLoading || refreshing);
  const overallStatusClass = overallStatusToClassName(overallStatus);
  const overallStatusLabel = overallStatusToLabel(overallStatus);

  return (
    <>
      <section className="content-card profile-card mcp-hub-card">
        <div className="section-header">
          <div>
            <p className="section-eyebrow">MCP Remote</p>
            <h2>远程 WSS 接入管理</h2>
            <p className="helper-text mcp-hero-note">
              这里维护的是小智 AI 那边的远程 ws / wss 地址。真正的连接动作由 Brights 后端主动发起，连接建立后，小智侧会通过这一条远程连接一次性调用当前学员可用的全部 Brights 工具。
            </p>
          </div>
          <div className="mcp-header-actions">
            <span className={`pill ${overallStatusClass}`}>{overallStatusLabel}</span>
            <button className="primary-button small-button" onClick={openCreateModal} type="button">
              添加接入点
            </button>
          </div>
        </div>

        <div className="mcp-console-stack">
          <section className="mcp-connection-hero">
            <div className="mcp-hero-topline">
              <div className="mcp-connection-hero-copy">
                <p className="section-eyebrow">Remote WSS Bridge</p>
                <h3>Brights 主动连接小智 AI 远程 WSS</h3>
                <p className="helper-text">
                  当前学员 <strong>{learnerLabel}</strong>
                  {" · "}
                  当前学科 <strong>{subjectLabel}</strong>
                </p>
              </div>

              <div className="mcp-hero-side">
                <span className={`pill ${overallStatusClass}`}>{overallStatusLabel}</span>
                <div className="mcp-hero-actions">
                  <button className="primary-button" onClick={openCreateModal} type="button">
                    添加远程地址
                  </button>
                  <button className="secondary-button" onClick={() => void refreshAllConnections()} type="button">
                    {refreshing ? "刷新中..." : "刷新连接"}
                  </button>
                </div>
              </div>
            </div>

            <div className="mcp-mode-pills">
              <span className="mcp-mode-pill mcp-mode-pill-active">Brights 后端主动外连</span>
              <span className="mcp-mode-pill">小智侧连上后可直接调用全部工具</span>
              <span className="mcp-mode-pill">返回数据按当前用户会员权限控制</span>
            </div>
          </section>

          <div className="mcp-summary-grid">
            <article className="mcp-summary-card">
              <span>连接模式</span>
              <strong>远程 WSS 主动外连</strong>
            </article>
            <article className="mcp-summary-card">
              <span>全部工具</span>
              <strong>{globalTools.length} 个工具</strong>
            </article>
            <article className="mcp-summary-card">
              <span>已启用接入点</span>
              <strong>{enabledEndpoints.length} 个</strong>
            </article>
            <article className="mcp-summary-card">
              <span>当前在线</span>
              <strong>{connectedEndpoints.length} 个</strong>
            </article>
          </div>

          <section className="mcp-console-panel mcp-connect-panel">
            <div className="section-header">
              <div>
                <p className="section-eyebrow">Bridge Status</p>
                <h3>桥接状态与能力概览</h3>
                <p className="helper-text">这里展示当前桥接能力，不再对外提供 Brights 本地 ws 入口复制。</p>
              </div>
            </div>

            <div className="mcp-bridge-grid">
              <article className="mcp-bridge-card">
                <span>当前身份</span>
                <strong>{learnerLabel}</strong>
                <p className="helper-text">小智侧发起工具调用时，会按当前学员和当前学科的权限返回数据。</p>
              </article>
              <article className="mcp-bridge-card">
                <span>远程工具暴露</span>
                <strong>{globalTools.length} 个 Brights 工具</strong>
                <p className="helper-text">连接成功后，无需在网页端逐个勾选工具，默认统一暴露全部可用工具。</p>
              </article>
              <article className="mcp-bridge-card">
                <span>协议能力</span>
                <strong>{mcpInfo ? `${mcpInfo.name} ${mcpInfo.version}` : "读取中..."}</strong>
                <p className="helper-text">
                  {mcpInfo ? `支持 ${mcpInfo.availableMethods.join(" / ")}` : "正在读取 MCP 能力说明。"}
                </p>
              </article>
            </div>

            {infoLoading ? <div className="feedback-banner">正在读取 MCP 能力信息...</div> : null}
            {infoError ? <div className="feedback-banner feedback-error">{infoError}</div> : null}
            {copiedMessage ? <div className="feedback-banner feedback-success">{copiedMessage}</div> : null}
            {endpointBusyMessage ? <div className="feedback-banner">{endpointBusyMessage}</div> : null}
            {endpointsError ? <div className="feedback-banner feedback-error">{endpointsError}</div> : null}

            {mcpInfo ? (
              <div className="mcp-service-strip">
                <strong>{mcpInfo.name}</strong>
                <span>{mcpInfo.version}</span>
                <span>协议 {mcpInfo.protocolVersion}</span>
                <span>学科 {subjectLabel}</span>
              </div>
            ) : null}
          </section>

          <section className="mcp-console-panel mcp-endpoints-panel">
            <div className="section-header">
              <div>
                <p className="section-eyebrow">Remote Endpoints</p>
                <h3>小智远程 WSS 地址列表</h3>
                <p className="helper-text">保存的是小智侧提供的远程 ws / wss 地址，浏览器不会直接连，真正的连接由 Brights 后端完成。</p>
              </div>
              <div className="mcp-header-actions">
                <button className="secondary-button small-button" onClick={() => void refreshAllConnections()} type="button">
                  {refreshing ? "刷新中..." : "刷新状态"}
                </button>
                <button className="primary-button small-button" onClick={openCreateModal} type="button">
                  新增接入点
                </button>
              </div>
            </div>

            <div className="mcp-management-band">
              <div className="mcp-management-copy">
                <strong>一条远程连接，统一调用全部工具</strong>
                <p className="helper-text">点击任意一行即可查看连接详情、错误信息和当前暴露的工具列表。</p>
              </div>
              <div className="mcp-management-stats">
                <div className="mcp-management-stat">
                  <strong>{endpoints.length}</strong>
                  <span>总接入点</span>
                </div>
                <div className="mcp-management-stat">
                  <strong>{connectedEndpoints.length}</strong>
                  <span>当前在线</span>
                </div>
                <div className="mcp-management-stat">
                  <strong>{globalTools.length}</strong>
                  <span>统一工具数</span>
                </div>
              </div>
            </div>

            {endpointsLoading ? <div className="feedback-banner">正在加载远程接入点列表...</div> : null}

            {endpoints.length === 0 ? (
              <div className="mcp-empty-state">
                <strong>还没有远程接入点</strong>
                <p className="helper-text">点击“新增接入点”，手动填写一个小智侧的 ws / wss 地址后，Brights 后端就会主动尝试连接。</p>
              </div>
            ) : (
              <div className="table-wrap mcp-table-wrap">
                <table className="data-table mcp-data-table mcp-endpoint-table">
                  <thead>
                    <tr>
                      <th scope="col">接入点</th>
                      <th scope="col">远程地址</th>
                      <th scope="col">状态与工具</th>
                      <th scope="col">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {endpoints.map((endpoint) => {
                      const endpointTools = toolPreviewMap[endpoint.id] ?? globalTools;
                      const isSelected = endpoint.id === selectedEndpoint?.id;
                      const isActionBusy = endpointActionBusyID === endpoint.id;

                      return (
                        <tr
                          className={isSelected ? "mcp-table-row-selected mcp-table-row-clickable" : "mcp-table-row-clickable"}
                          key={endpoint.id}
                          onClick={() => {
                            selectEndpoint(endpoint);
                          }}
                          onKeyDown={(event) => {
                            if (event.key === "Enter" || event.key === " ") {
                              event.preventDefault();
                              selectEndpoint(endpoint);
                            }
                          }}
                          tabIndex={0}
                        >
                          <td>
                            <div className="mcp-cell-stack">
                              <div className="mcp-name-row">
                                <strong>{endpoint.name}</strong>
                                {isSelected ? <span className="mcp-mini-badge">当前查看</span> : null}
                              </div>
                              <span className="mcp-table-muted">{endpoint.description || "未填写说明"}</span>
                            </div>
                          </td>
                          <td>
                            <div className="mcp-url-cell">
                              <code className="mcp-url-text" title={endpoint.url}>
                                {endpoint.url}
                              </code>
                              <div className="mcp-url-meta">
                                <span className="mcp-table-muted">
                                  参数：{endpoint.token_query_param || "token"} / {endpoint.subject_query_param || "subject"}
                                </span>
                                <button
                                  className="secondary-button small-button"
                                  onClick={(event) => {
                                    event.stopPropagation();
                                    void copyText(endpoint.url, "远程地址已复制。");
                                  }}
                                  type="button"
                                >
                                  复制
                                </button>
                              </div>
                            </div>
                          </td>
                          <td>
                            <div className="mcp-cell-stack">
                              <span className={`pill ${statusPillClass(endpoint.connection_status, endpoint.enabled)}`}>
                                {statusLabel(endpoint.connection_status, endpoint.enabled)}
                              </span>
                              <span className="mcp-table-muted">工具 {endpointTools.length} 个</span>
                              <span className="mcp-table-muted">最近连接：{formatDateTime(endpoint.connected_at)}</span>
                              {endpoint.last_error ? (
                                <span className="mcp-table-error">{endpoint.last_error}</span>
                              ) : (
                                <span className="mcp-table-muted">{endpoint.enabled ? "由 Brights 后端自动维护连接" : "当前已停用"}</span>
                              )}
                            </div>
                          </td>
                          <td>
                            <div className="mcp-action-group">
                              <button
                                className="secondary-button small-button"
                                disabled={isActionBusy}
                                onClick={(event) => {
                                  event.stopPropagation();
                                  void toggleEndpointEnabled(endpoint);
                                }}
                                type="button"
                              >
                                {isActionBusy ? "处理中..." : endpoint.enabled ? "停用" : "启动"}
                              </button>
                              <button
                                className="secondary-button small-button"
                                disabled={isActionBusy}
                                onClick={(event) => {
                                  event.stopPropagation();
                                  openEditModal(endpoint);
                                }}
                                type="button"
                              >
                                编辑
                              </button>
                              <button
                                className="secondary-button small-button"
                                disabled={isActionBusy}
                                onClick={(event) => {
                                  event.stopPropagation();
                                  void deleteEndpoint(endpoint);
                                }}
                                type="button"
                              >
                                删除
                              </button>
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          <div className="mcp-inspector-grid">
            <section className="mcp-console-panel">
              <div className="section-header">
                <div>
                  <p className="section-eyebrow">Selected Endpoint</p>
                  <h3>当前接入点详情</h3>
                </div>
                {selectedEndpoint ? (
                  <div className="button-row">
                    <button className="secondary-button small-button" onClick={() => void refreshSelectedEndpointStatus()} type="button">
                      刷新状态
                    </button>
                    <button className="secondary-button small-button" onClick={() => openEditModal(selectedEndpoint)} type="button">
                      编辑
                    </button>
                  </div>
                ) : null}
              </div>

              {!selectedEndpoint ? (
                <div className="mcp-empty-state">
                  <strong>还没有配置远程接入点</strong>
                  <p className="helper-text">添加一个小智侧远程 ws / wss 地址后，这里会显示连接详情和工具暴露情况。</p>
                </div>
              ) : (
                <>
                  <div className="mcp-selected-banner">
                    <div className="mcp-selected-banner-copy">
                      <strong>{selectedEndpoint.name}</strong>
                      <p className="helper-text">{selectedEndpoint.description || "这个接入点还没有填写说明。"}</p>
                    </div>
                    <div className="mcp-selected-banner-meta">
                      <span className={`pill ${statusPillClass(selectedEndpoint.connection_status, selectedEndpoint.enabled)}`}>
                        {statusLabel(selectedEndpoint.connection_status, selectedEndpoint.enabled)}
                      </span>
                      <span className="tag">工具 {selectedEndpointTools.length}</span>
                    </div>
                  </div>

                  <div className="table-wrap mcp-table-wrap">
                    <table className="data-table mcp-data-table mcp-detail-table">
                      <tbody>
                        <tr>
                          <th scope="row">接入点名称</th>
                          <td>{selectedEndpoint.name}</td>
                        </tr>
                        <tr>
                          <th scope="row">连接状态</th>
                          <td>
                            <span className={`pill ${statusPillClass(selectedEndpoint.connection_status, selectedEndpoint.enabled)}`}>
                              {statusLabel(selectedEndpoint.connection_status, selectedEndpoint.enabled)}
                            </span>
                          </td>
                        </tr>
                        <tr>
                          <th scope="row">远程地址</th>
                          <td className="mcp-cell-code mcp-inline-break">{selectedEndpoint.url}</td>
                        </tr>
                        <tr>
                          <th scope="row">令牌参数名</th>
                          <td>{selectedEndpoint.token_query_param || "token"}</td>
                        </tr>
                        <tr>
                          <th scope="row">学科参数名</th>
                          <td>{selectedEndpoint.subject_query_param || "subject"}</td>
                        </tr>
                        <tr>
                          <th scope="row">最近连接时间</th>
                          <td>{formatDateTime(selectedEndpoint.connected_at)}</td>
                        </tr>
                        <tr>
                          <th scope="row">当前工具数</th>
                          <td>{selectedEndpointTools.length}</td>
                        </tr>
                        <tr>
                          <th scope="row">备注说明</th>
                          <td>{selectedEndpoint.description || "-"}</td>
                        </tr>
                      </tbody>
                    </table>
                  </div>

                  {selectedEndpoint.last_error ? (
                    <div className="feedback-banner feedback-error">{selectedEndpoint.last_error}</div>
                  ) : (
                    <div className="feedback-banner">
                      连接建立后，小智侧无需在网页端选择工具，直接通过这一条远程连接就能调用当前用户可用的全部 Brights MCP 工具。
                    </div>
                  )}
                </>
              )}
            </section>

            <section className="mcp-console-panel">
              <div className="section-header">
                <div>
                  <p className="section-eyebrow">All Tools</p>
                  <h3>当前连接后会自动可用的全部工具</h3>
                  <p className="helper-text">展示的是当前接入点连接成功后，小智侧可以直接调用的 Brights MCP 工具。</p>
                </div>
              </div>

              {toolPreviewLoadingID === selectedEndpoint?.id ? (
                <div className="feedback-banner">正在读取接入点工具列表...</div>
              ) : toolPreviewError ? (
                <div className="feedback-banner feedback-error">{toolPreviewError}</div>
              ) : selectedEndpointTools.length === 0 ? (
                <div className="feedback-banner">当前还没有可展示的工具，请先刷新连接状态。</div>
              ) : (
                <div className="table-wrap mcp-table-wrap">
                  <table className="data-table mcp-data-table">
                    <thead>
                      <tr>
                        <th scope="col">工具名称</th>
                        <th scope="col">方法名</th>
                        <th scope="col">入参数量</th>
                        <th scope="col">权限与分类</th>
                        <th scope="col">说明</th>
                      </tr>
                    </thead>
                    <tbody>
                      {selectedEndpointTools.map((tool) => (
                        <tr key={tool.name}>
                          <td>
                            <div className="mcp-cell-stack">
                              <strong>{tool.title || tool.name}</strong>
                              <span className="mcp-table-muted">{tool.sourceType || "builtin"}</span>
                            </div>
                          </td>
                          <td className="mcp-cell-code">{tool.name}</td>
                          <td>{countSchemaFields(tool.inputSchema)}</td>
                          <td>
                            <div className="mcp-cell-stack">
                              <span>{tool.category || "-"}</span>
                              <span className="mcp-table-muted">
                                {tool.requiresMembership ? "需要会员" : "无需会员"}
                                {tool.canUse === false ? " / 当前不可用" : ""}
                              </span>
                            </div>
                          </td>
                          <td>{tool.description || "暂无描述"}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </section>
          </div>
        </div>
      </section>

      {endpointModalOpen ? (
        <div
          className="mcp-endpoint-modal-backdrop"
          onClick={(event) => {
            if (event.target === event.currentTarget) {
              closeEndpointModal();
            }
          }}
        >
          <section className="mcp-endpoint-modal">
            <div className="mcp-modal-hero">
              <div>
                <p className="section-eyebrow">{editingEndpointID ? "Edit Endpoint" : "Add Endpoint"}</p>
                <h3>{editingEndpointID ? "编辑远程 MCP 接入点" : "添加远程 MCP 接入点"}</h3>
                <p className="helper-text">只需要填写小智侧的远程 ws / wss 地址。保存后由 Brights 后端主动连接，不需要浏览器直接建立 WSS 连接。</p>
              </div>
              <button className="secondary-button small-button" disabled={savingEndpoint} onClick={closeEndpointModal} type="button">
                关闭
              </button>
            </div>

            <div className="setup-form">
              <div className="mcp-modal-preview">
                <strong>当前接入身份</strong>
                <code>
                  学员：{learnerLabel}
                  {"\n"}
                  学科：{subjectLabel}
                </code>
              </div>

              <div className="form-grid-two">
                <label className="form-field">
                  <span>接入点名称</span>
                  <input
                    value={endpointForm.name}
                    onChange={(event) => {
                      setEndpointForm((current) => ({ ...current, name: event.target.value }));
                    }}
                    placeholder="例如：小智 AI 正式环境"
                  />
                </label>

                <label className="form-field">
                  <span>是否启用</span>
                  <select
                    value={endpointForm.enabled ? "enabled" : "disabled"}
                    onChange={(event) => {
                      setEndpointForm((current) => ({ ...current, enabled: event.target.value === "enabled" }));
                    }}
                  >
                    <option value="enabled">启用并由 Brights 自动连接</option>
                    <option value="disabled">先保存为停用状态</option>
                  </select>
                </label>

                <label className="form-field form-grid-span-two">
                  <span>远程 ws / wss 地址</span>
                  <input
                    value={endpointForm.url}
                    onChange={(event) => {
                      setEndpointForm((current) => ({ ...current, url: event.target.value }));
                    }}
                    placeholder="wss://example.xiaozhi.ai/mcp"
                  />
                </label>

                <label className="form-field form-grid-span-two">
                  <span>备注说明</span>
                  <textarea
                    rows={4}
                    value={endpointForm.description}
                    onChange={(event) => {
                      setEndpointForm((current) => ({ ...current, description: event.target.value }));
                    }}
                    placeholder="写清楚这个远程服务的用途、环境或负责人，方便后续维护。"
                  />
                </label>

                <label className="form-field">
                  <span>令牌参数名</span>
                  <input
                    value={endpointForm.token_query_param}
                    onChange={(event) => {
                      setEndpointForm((current) => ({ ...current, token_query_param: event.target.value }));
                    }}
                    placeholder="token"
                  />
                </label>

                <label className="form-field">
                  <span>学科参数名</span>
                  <input
                    value={endpointForm.subject_query_param}
                    onChange={(event) => {
                      setEndpointForm((current) => ({ ...current, subject_query_param: event.target.value }));
                    }}
                    placeholder="subject"
                  />
                </label>
              </div>

              <div className="mcp-modal-preview">
                <strong>连接说明</strong>
                <p className="helper-text">
                  保存后，Brights 会按当前用户身份把令牌和学科参数拼到远程地址上，再由后端主动连接小智侧的 WSS 服务。
                </p>
              </div>

              {endpointBusyMessage ? <div className="feedback-banner feedback-error">{endpointBusyMessage}</div> : null}

              <div className="button-row mcp-modal-actions">
                <button className="primary-button" disabled={savingEndpoint} onClick={() => void saveEndpoint()} type="button">
                  {savingEndpoint ? "正在保存..." : editingEndpointID ? "保存修改" : "添加接入点"}
                </button>
                <button className="secondary-button" disabled={savingEndpoint} onClick={closeEndpointModal} type="button">
                  取消
                </button>
              </div>
            </div>
          </section>
        </div>
      ) : null}
    </>
  );
}

function parseErrorMessage(error: unknown) {
  if (error instanceof Error) {
    return error.message;
  }
  return String(error);
}

function resolveOverallStatus(endpoints: MCPEndpoint[], token: string, loading: boolean) {
  if (!token.trim()) {
    return "disconnected";
  }
  if (loading || endpoints.some((item) => item.connection_status === "connecting")) {
    return "connecting";
  }
  if (endpoints.some((item) => item.is_connected)) {
    return "connected";
  }
  if (endpoints.some((item) => item.connection_status === "error")) {
    return "error";
  }
  return "disconnected";
}

function overallStatusToClassName(status: string) {
  switch (status) {
    case "connected":
      return "pill-success";
    case "connecting":
      return "pill-warning";
    case "error":
      return "pill-danger";
    default:
      return "pill-muted";
  }
}

function overallStatusToLabel(status: string) {
  switch (status) {
    case "connected":
      return "远程服务在线";
    case "connecting":
      return "连接中";
    case "error":
      return "连接异常";
    default:
      return "未连接";
  }
}

function statusPillClass(status?: string, enabled?: boolean) {
  if (!enabled) {
    return "pill-muted";
  }
  switch (status) {
    case "connected":
      return "pill-success";
    case "connecting":
      return "pill-warning";
    case "error":
      return "pill-danger";
    default:
      return "pill-muted";
  }
}

function statusLabel(status?: string, enabled?: boolean) {
  if (!enabled) {
    return "已停用";
  }
  switch (status) {
    case "connected":
      return "在线";
    case "connecting":
      return "连接中";
    case "error":
      return "异常";
    default:
      return "未连接";
  }
}

function formatDateTime(value?: string) {
  if (!value) {
    return "-";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString("zh-CN");
}

function countSchemaFields(schema?: Record<string, unknown>) {
  const properties = schema?.properties;
  if (!properties || typeof properties !== "object" || Array.isArray(properties)) {
    return 0;
  }
  return Object.keys(properties).length;
}

function buildEndpointPreviewVersion(endpoint: MCPEndpoint) {
  return [endpoint.connection_status || "", endpoint.connected_at || "", endpoint.updated_at || ""].join("|");
}
