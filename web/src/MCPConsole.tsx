import { useEffect, useMemo, useState } from "react";

import {
  api,
  buildMCPWebSocketUrlWithToken,
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
  const [toolPreviewLoadingID, setToolPreviewLoadingID] = useState<number | null>(null);
  const [toolPreviewError, setToolPreviewError] = useState("");

  const [endpointModalOpen, setEndpointModalOpen] = useState(false);
  const [editingEndpointID, setEditingEndpointID] = useState<number | null>(null);
  const [endpointForm, setEndpointForm] = useState<EndpointFormState>(emptyEndpointForm);
  const [savingEndpoint, setSavingEndpoint] = useState(false);
  const [endpointBusyMessage, setEndpointBusyMessage] = useState("");
  const [copiedMessage, setCopiedMessage] = useState("");

  const localSocketURL = useMemo(
    () => buildMCPWebSocketUrlWithToken(props.subjectKey, props.token),
    [props.subjectKey, props.token],
  );

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
    if (toolPreviewMap[selectedEndpoint.id]) {
      return;
    }
    void loadEndpointToolPreview(selectedEndpoint);
  }, [props.token, selectedEndpoint, toolPreviewMap]);

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

  async function loadEndpointToolPreview(endpoint: MCPEndpoint) {
    if (!props.token.trim()) {
      return;
    }
    if (toolPreviewMap[endpoint.id]) {
      return;
    }

    setToolPreviewLoadingID(endpoint.id);
    setToolPreviewError("");
    try {
      const payload = await api.getLearnerMCPEndpointTools(props.token, endpoint.id);
      setToolPreviewMap((current) => ({
        ...current,
        [endpoint.id]: payload.tools,
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
      setEndpointBusyMessage("请先登录学员账号，再刷新 MCP 接入状态。");
      return;
    }

    setRefreshing(true);
    setEndpointBusyMessage("");
    setEndpointsError("");
    setCopiedMessage("");
    try {
      const payload = await api.refreshLearnerMCPConnections(props.token);
      setEndpoints(payload.endpoints);
      setEndpointBusyMessage("已刷新远程接入点状态。");
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
      setEndpointModalOpen(false);
      resetEndpointForm();
      setEndpointBusyMessage(editingEndpointID ? "接入点配置已更新。" : "新的 MCP 接入点已添加。");
      void loadEndpoints({ silent: true });
    } catch (error) {
      setEndpointBusyMessage(parseErrorMessage(error));
    } finally {
      setSavingEndpoint(false);
    }
  }

  async function deleteEndpoint(endpoint: MCPEndpoint) {
    if (!props.token.trim()) {
      setEndpointBusyMessage("请先登录学员账号，再删除接入点。");
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
      if (selectedEndpointID === endpoint.id) {
        setSelectedEndpointID(null);
      }
      setEndpointBusyMessage("接入点已删除。");
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

  function openMCPInfoPage() {
    const url = buildMCPInfoURL(props.subjectKey);
    if (!url) {
      return;
    }
    window.open(url, "_blank", "noopener,noreferrer");
  }

  const overallStatus = resolveOverallStatus(endpoints, props.token, endpointsLoading || refreshing);
  const overallStatusClass = overallStatusToClassName(overallStatus);
  const overallStatusLabel = overallStatusToLabel(overallStatus);

  return (
    <>
      <section className="content-card profile-card mcp-hub-card" id="mcp">
        <div className="section-header">
          <div>
            <p className="section-eyebrow">MCP Access</p>
            <h2>MCP接入点</h2>
            <p className="helper-text mcp-hero-note">
              你只需要在这里维护远程 ws / wss 地址。真正的连接由 Brights 后端完成，远程服务连上后就能以当前学员身份调用全部
              Brights MCP 工具，数据返回会自动按会员权限控制。
            </p>
          </div>
          <div className="mcp-header-actions">
            <span className={`pill ${overallStatusClass}`}>{overallStatusLabel}</span>
            <button className="primary-button small-button" onClick={openCreateModal} type="button">
              添加接入点
            </button>
          </div>
        </div>

        <div className="mcp-hub-layout">
          <div className="mcp-hub-main">
            <div className="mcp-connection-hero">
              <div className="mcp-connection-hero-copy">
                <p className="section-eyebrow">Unified Endpoint</p>
                <h3>一个连接点，对外暴露全部 Brights 工具</h3>
                <p className="helper-text">
                  当前学员 <strong>{props.learnerName || "未登录"}</strong>
                  {" · "}
                  当前学科 <strong>{props.subjectKey || "-"}</strong>
                </p>
              </div>

              <div className="mcp-mode-pills">
                <span className="mcp-mode-pill mcp-mode-pill-active">后端统一连接远程服务</span>
                <span className="mcp-mode-pill">工具列表自动全部暴露</span>
                <span className="mcp-mode-pill">返回结果按会员权限控制</span>
              </div>
            </div>

            <div className="mcp-summary-grid">
              <article className="mcp-summary-card">
                <span>统一入口</span>
                <strong>{mcpInfo?.websocketPath || "/mcp"}</strong>
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
                  <p className="section-eyebrow">Entry URL</p>
                  <h3>Brights 统一 MCP 入口</h3>
                </div>
              </div>

              <label className="form-field">
                <span>当前可复制的连接地址</span>
                <div className="mcp-address-preview">
                  <code>{localSocketURL || "请先登录并选择学科后再复制"}</code>
                </div>
              </label>

              <div className="button-row">
                <button
                  className="primary-button"
                  disabled={!localSocketURL}
                  onClick={() => {
                    void copyText(localSocketURL, "统一 MCP 连接地址已复制。");
                  }}
                  type="button"
                >
                  复制入口
                </button>
                <button className="secondary-button" onClick={() => void refreshAllConnections()} type="button">
                  {refreshing ? "正在刷新..." : "刷新连接"}
                </button>
                <button className="secondary-button" onClick={openMCPInfoPage} type="button">
                  查看接入说明
                </button>
              </div>

              {infoLoading ? <div className="feedback-banner">正在读取 MCP 接口信息...</div> : null}
              {infoError ? <div className="feedback-banner feedback-error">{infoError}</div> : null}
              {copiedMessage ? <div className="feedback-banner feedback-success">{copiedMessage}</div> : null}
              {endpointBusyMessage ? <div className="feedback-banner">{endpointBusyMessage}</div> : null}
              {endpointsError ? <div className="feedback-banner feedback-error">{endpointsError}</div> : null}

              {mcpInfo ? (
                <div className="mcp-service-strip">
                  <strong>{mcpInfo.name}</strong>
                  <span>{mcpInfo.version}</span>
                  <span>协议 {mcpInfo.protocolVersion}</span>
                  <span>支持 {mcpInfo.availableMethods.join(" / ")}</span>
                </div>
              ) : null}
            </section>

            <section className="mcp-console-panel">
              <div className="section-header">
                <div>
                  <p className="section-eyebrow">Selected Endpoint</p>
                  <h3>当前接入点状态</h3>
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
                  <p className="helper-text">添加一个 ws / wss 地址后，Brights 后端就会尝试主动连接这个远程服务。</p>
                </div>
              ) : (
                <>
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
                          <td className="mcp-inline-break mcp-cell-code">{selectedEndpoint.url}</td>
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
                      连接建立后，远程服务无需在网页端选择工具，直接通过这个接入点就能拿到全部 Brights MCP 工具。
                    </div>
                  )}
                </>
              )}
            </section>

            <section className="mcp-console-panel">
              <div className="section-header">
                <div>
                  <p className="section-eyebrow">All Tools</p>
                  <h3>连接成功后自动可用的全部工具</h3>
                  <p className="helper-text">这里展示的是当前接入点会对外暴露的 Brights MCP 工具，不需要在网页端逐个勾选。</p>
                </div>
              </div>

              {toolPreviewLoadingID === selectedEndpoint?.id ? (
                <div className="feedback-banner">正在读取接入点工具清单...</div>
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
                        <th scope="col">入参字段</th>
                        <th scope="col">说明</th>
                      </tr>
                    </thead>
                    <tbody>
                      {selectedEndpointTools.map((tool) => (
                        <tr key={tool.name}>
                          <td>
                            <strong>{tool.title || tool.name}</strong>
                          </td>
                          <td className="mcp-cell-code">{tool.name}</td>
                          <td>{countSchemaFields(tool.inputSchema)}</td>
                          <td>{tool.description || "暂无描述"}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </section>
          </div>

          <aside className="mcp-endpoint-sidebar">
            <div className="mcp-sidebar-head">
              <div>
                <p className="section-eyebrow">Remote ws / wss</p>
                <h3>远程接入点列表</h3>
              </div>
              <button className="secondary-button small-button" onClick={openCreateModal} type="button">
                新增
              </button>
            </div>

            <p className="helper-text">
              这里保存的是你要连接的远程服务地址。浏览器不会直接连接这些地址，真正的连接动作由 Brights 后端统一完成。
            </p>

            {endpointsLoading ? <div className="feedback-banner">正在加载接入点列表...</div> : null}

            <div className="mcp-endpoint-list">
              {endpoints.length === 0 ? (
                <div className="mcp-empty-state">
                  <strong>还没有远程接入点</strong>
                  <p className="helper-text">点击“新增”，手动填写一个远程 ws / wss 地址即可开始接入。</p>
                </div>
              ) : (
                <div className="table-wrap mcp-table-wrap">
                  <table className="data-table mcp-data-table">
                    <thead>
                      <tr>
                        <th scope="col">接入点</th>
                        <th scope="col">远程地址</th>
                        <th scope="col">状态</th>
                        <th scope="col">工具数</th>
                        <th scope="col">最近连接</th>
                        <th scope="col">操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      {endpoints.map((endpoint) => {
                        const endpointTools = toolPreviewMap[endpoint.id] ?? globalTools;
                        const isSelected = endpoint.id === selectedEndpoint?.id;

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
                                <strong>{endpoint.name}</strong>
                                <span className="mcp-table-muted">{endpoint.description || "未填写说明"}</span>
                              </div>
                            </td>
                            <td className="mcp-cell-code mcp-inline-break">{endpoint.url}</td>
                            <td>
                              <div className="mcp-cell-stack">
                                <span className={`pill ${statusPillClass(endpoint.connection_status, endpoint.enabled)}`}>
                                  {statusLabel(endpoint.connection_status, endpoint.enabled)}
                                </span>
                                {endpoint.last_error ? (
                                  <span className="mcp-table-error">{endpoint.last_error}</span>
                                ) : (
                                  <span className="mcp-table-muted">{endpoint.enabled ? "自动连接已开启" : "已停用"}</span>
                                )}
                              </div>
                            </td>
                            <td>{endpointTools.length}</td>
                            <td>{formatDateTime(endpoint.connected_at)}</td>
                            <td>
                              <div className="mcp-action-group">
                                <button
                                  className="secondary-button small-button"
                                  disabled={toolPreviewLoadingID === endpoint.id}
                                  onClick={(event) => {
                                    event.stopPropagation();
                                    selectEndpoint(endpoint);
                                  }}
                                  type="button"
                                >
                                  {toolPreviewLoadingID === endpoint.id ? "加载中..." : "查看"}
                                </button>
                                <button
                                  className="secondary-button small-button"
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
            </div>
          </aside>
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
            <div className="section-header">
              <div>
                <p className="section-eyebrow">{editingEndpointID ? "Edit Endpoint" : "Add Endpoint"}</p>
                <h3>{editingEndpointID ? "编辑 MCP 接入点" : "添加 MCP 接入点"}</h3>
                <p className="helper-text">
                  这里只需要填写远程 ws / wss 地址。保存后，Brights 后端会按当前学员身份主动连接这个远程服务。
                </p>
              </div>
              <button className="secondary-button small-button" disabled={savingEndpoint} onClick={closeEndpointModal} type="button">
                关闭
              </button>
            </div>

            <div className="setup-form">
              <div className="mcp-modal-preview">
                <strong>当前接入身份</strong>
                <code>
                  学员：{props.learnerName || "未登录"}
                  {"\n"}
                  学科：{props.subjectKey || "-"}
                </code>
              </div>

              <label className="form-field">
                <span>接入点名称</span>
                <input
                  value={endpointForm.name}
                  onChange={(event) => {
                    setEndpointForm((current) => ({ ...current, name: event.target.value }));
                  }}
                  placeholder="例如：Xiaozhi 远程服务"
                />
              </label>

              <label className="form-field">
                <span>远程 ws / wss 地址</span>
                <input
                  value={endpointForm.url}
                  onChange={(event) => {
                    setEndpointForm((current) => ({ ...current, url: event.target.value }));
                  }}
                  placeholder="wss://example.com/mcp"
                />
              </label>

              <label className="form-field">
                <span>备注说明</span>
                <textarea
                  rows={4}
                  value={endpointForm.description}
                  onChange={(event) => {
                    setEndpointForm((current) => ({ ...current, description: event.target.value }));
                  }}
                  placeholder="写清楚这个远程服务的用途，方便后续维护。"
                />
              </label>

              <label className="checkbox-field">
                <input
                  checked={endpointForm.enabled}
                  onChange={(event) => {
                    setEndpointForm((current) => ({ ...current, enabled: event.target.checked }));
                  }}
                  type="checkbox"
                />
                <span>保存后立即启用，并由 Brights 后端自动建立远程连接。</span>
              </label>

              {endpointBusyMessage ? <div className="feedback-banner feedback-error">{endpointBusyMessage}</div> : null}

              <div className="button-row">
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
      return "已有远程服务在线";
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

function buildMCPInfoURL(subjectKey: string) {
  if (typeof window === "undefined") {
    return "";
  }

  const url = new URL("/mcp/info", window.location.origin);
  if (subjectKey.trim()) {
    url.searchParams.set("subject", subjectKey.trim());
  }
  return url.toString();
}
