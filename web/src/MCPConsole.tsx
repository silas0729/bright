import { useEffect, useMemo, useRef, useState } from "react";

import { api, buildMCPWebSocketUrl, type MCPInfo, type MCPInfoTool } from "./api";

type MCPConsoleProps = {
  subjectKey: string;
  token: string;
  learnerName: string;
};

type MCPConnectionState = "disconnected" | "connecting" | "connected";

type MCPEnvelope = {
  jsonrpc: string;
  id?: number | string | null;
  method?: string;
  params?: unknown;
  result?: unknown;
  error?: {
    code?: number;
    message?: string;
    data?: unknown;
  };
};

type MCPLogEntry = {
  id: number;
  direction: "sent" | "received" | "status" | "error";
  text: string;
  timestamp: string;
};

const defaultArgsByTool: Record<string, Record<string, unknown>> = {
  search_words: {
    query: "travel",
    page: 1,
    page_size: 10,
  },
  list_classification_stats: {
    page: 1,
    page_size: 8,
  },
  list_categories: {
    kind: "topic",
  },
};

export default function MCPConsole(props: MCPConsoleProps) {
  const [mcpInfo, setMcpInfo] = useState<MCPInfo | null>(null);
  const [infoError, setInfoError] = useState("");
  const [infoLoading, setInfoLoading] = useState(false);

  const [connectionState, setConnectionState] = useState<MCPConnectionState>("disconnected");
  const [selectedTool, setSelectedTool] = useState("search_words");
  const [argumentsText, setArgumentsText] = useState("");
  const [logs, setLogs] = useState<MCPLogEntry[]>([]);
  const [callResultText, setCallResultText] = useState("");
  const [connectionError, setConnectionError] = useState("");

  const socketRef = useRef<WebSocket | null>(null);
  const logIdRef = useRef(1);
  const initializeWaiterRef = useRef<{
    resolve: () => void;
    reject: (error: Error) => void;
  } | null>(null);
  const listToolsWaiterRef = useRef<{
    resolve: (payload: MCPInfoTool[]) => void;
    reject: (error: Error) => void;
  } | null>(null);
  const callToolWaiterRef = useRef<{
    resolve: (payload: unknown) => void;
    reject: (error: Error) => void;
  } | null>(null);

  const tools = useMemo(() => mcpInfo?.tools ?? [], [mcpInfo]);
  const resolvedSocketUrl = useMemo(() => buildMCPWebSocketUrl(props.subjectKey), [props.subjectKey]);

  useEffect(() => {
    const initialArgs = {
      subject_key: props.subjectKey,
      ...(defaultArgsByTool[selectedTool] ?? {}),
    };
    setArgumentsText(JSON.stringify(initialArgs, null, 2));
  }, [props.subjectKey, selectedTool]);

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
    return () => {
      const socket = socketRef.current;
      socketRef.current = null;
      if (socket && socket.readyState === WebSocket.OPEN) {
        socket.close(1000, "component unmounted");
      }
    };
  }, []);

  function appendLog(direction: MCPLogEntry["direction"], text: string) {
    const entry: MCPLogEntry = {
      id: logIdRef.current++,
      direction,
      text,
      timestamp: new Date().toLocaleTimeString("zh-CN", { hour12: false }),
    };
    setLogs((current) => [entry, ...current].slice(0, 60));
  }

  function sendPayload(payload: MCPEnvelope) {
    const socket = socketRef.current;
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      throw new Error("MCP websocket is not connected");
    }
    const text = JSON.stringify(payload);
    socket.send(text);
    appendLog("sent", text);
  }

  function parseErrorMessage(error: unknown) {
    if (error instanceof Error) {
      return error.message;
    }
    return String(error);
  }

  async function connect() {
    if (!props.token.trim()) {
      setConnectionError("请先登录学员账号，再建立 MCP websocket 连接。");
      return;
    }
    if (!props.subjectKey.trim()) {
      setConnectionError("请先选择学科。");
      return;
    }
    if (!resolvedSocketUrl) {
      setConnectionError("当前环境无法推导 websocket 地址。");
      return;
    }

    if (socketRef.current && socketRef.current.readyState === WebSocket.OPEN) {
      appendLog("status", "websocket already connected");
      return;
    }

    setConnectionError("");
    setConnectionState("connecting");
    appendLog("status", `connecting ${resolvedSocketUrl}`);

    const socket = new WebSocket(resolvedSocketUrl);
    socketRef.current = socket;

    try {
      await new Promise<void>((resolve, reject) => {
        const cleanup = () => {
          socket.removeEventListener("open", handleOpen);
          socket.removeEventListener("error", handleError);
        };

        const handleOpen = () => {
          cleanup();
          resolve();
        };

        const handleError = () => {
          cleanup();
          reject(new Error("websocket connection failed"));
        };

        socket.addEventListener("open", handleOpen);
        socket.addEventListener("error", handleError);
      });
    } catch (error) {
      const message = parseErrorMessage(error);
      setConnectionState("disconnected");
      setConnectionError(message);
      appendLog("error", message);
      socketRef.current = null;
      socket.close();
      return;
    }

    socket.onmessage = (event) => {
      const text = typeof event.data === "string" ? event.data : "[binary message]";
      appendLog("received", text);

      if (typeof event.data !== "string") {
        return;
      }

      let payload: MCPEnvelope;
      try {
        payload = JSON.parse(event.data) as MCPEnvelope;
      } catch {
        return;
      }

      if (payload.id === 1 && initializeWaiterRef.current) {
        if (payload.error) {
          initializeWaiterRef.current.reject(new Error(payload.error.message || "initialize failed"));
        } else {
          initializeWaiterRef.current.resolve();
        }
        initializeWaiterRef.current = null;
        return;
      }

      if (payload.id === 2 && listToolsWaiterRef.current) {
        if (payload.error) {
          listToolsWaiterRef.current.reject(new Error(payload.error.message || "tools/list failed"));
        } else {
          const toolPayload = ((payload.result as { tools?: MCPInfoTool[] } | undefined)?.tools ?? []) as MCPInfoTool[];
          listToolsWaiterRef.current.resolve(toolPayload);
        }
        listToolsWaiterRef.current = null;
        return;
      }

      if (payload.id === 3 && callToolWaiterRef.current) {
        if (payload.error) {
          callToolWaiterRef.current.reject(new Error(payload.error.message || "tools/call failed"));
        } else {
          callToolWaiterRef.current.resolve(payload.result);
        }
        callToolWaiterRef.current = null;
      }
    };

    socket.onclose = (event) => {
      if (socketRef.current === socket) {
        socketRef.current = null;
      }
      setConnectionState("disconnected");
      appendLog("status", `socket closed code=${event.code} reason=${event.reason || "none"}`);
    };

    socket.onerror = () => {
      appendLog("error", "websocket error event");
    };

    try {
      await initializeAndLoadTools();
      setConnectionState("connected");
      appendLog("status", `connected as ${props.learnerName || "learner"}`);
    } catch (error) {
      const message = parseErrorMessage(error);
      setConnectionError(message);
      setConnectionState("disconnected");
      appendLog("error", message);
      socket.close();
    }
  }

  async function initializeAndLoadTools() {
    await new Promise<void>((resolve, reject) => {
      initializeWaiterRef.current = { resolve, reject };
      sendPayload({
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: {
          protocolVersion: "2024-11-05",
          capabilities: {},
          clientInfo: {
            name: "brights-web-console",
            version: "1.0.0",
          },
        },
      });
    });

    sendPayload({
      jsonrpc: "2.0",
      method: "notifications/initialized",
    });

    const listedTools = await new Promise<MCPInfoTool[]>((resolve, reject) => {
      listToolsWaiterRef.current = { resolve, reject };
      sendPayload({
        jsonrpc: "2.0",
        id: 2,
        method: "tools/list",
      });
    });

    setMcpInfo((current) =>
      current
        ? {
            ...current,
            tools: listedTools,
          }
        : {
            name: "brights-mcp",
            version: "0.1.0",
            protocolVersion: "2024-11-05",
            websocketPath: "/mcp",
            websocketURL: resolvedSocketUrl,
            availableMethods: ["initialize", "ping", "tools/list", "tools/call"],
            tools: listedTools,
          },
    );
  }

  function disconnect() {
    const socket = socketRef.current;
    socketRef.current = null;
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.close(1000, "manual disconnect");
    }
    setConnectionState("disconnected");
  }

  async function callSelectedTool() {
    if (!socketRef.current || socketRef.current.readyState !== WebSocket.OPEN) {
      setConnectionError("请先建立 websocket 连接。");
      return;
    }

    let parsedArguments: Record<string, unknown> = {};
    try {
      parsedArguments = argumentsText.trim()
        ? (JSON.parse(argumentsText) as Record<string, unknown>)
        : {};
    } catch {
      setConnectionError("工具参数不是合法 JSON。");
      return;
    }

    setConnectionError("");
    setCallResultText("");

    try {
      const result = await new Promise<unknown>((resolve, reject) => {
        callToolWaiterRef.current = { resolve, reject };
        sendPayload({
          jsonrpc: "2.0",
          id: 3,
          method: "tools/call",
          params: {
            name: selectedTool,
            arguments: {
              subject_key: props.subjectKey,
              ...parsedArguments,
            },
          },
        });
      });

      setCallResultText(JSON.stringify(result, null, 2));
    } catch (error) {
      const message = parseErrorMessage(error);
      setConnectionError(message);
      appendLog("error", message);
    }
  }

  return (
    <section className="content-card profile-card">
      <div className="section-header">
        <div>
          <p className="section-eyebrow">MCP / WSS</p>
          <h2>前端实时连接 MCP websocket</h2>
          <p className="helper-text">
            这里就是前端真正发起 <code>ws://</code> / <code>wss://</code> 连接的入口。当前会复用已登录学员的 token，并按学科建立连接。
          </p>
        </div>
        <span
          className={
            connectionState === "connected"
              ? "pill pill-success"
              : connectionState === "connecting"
                ? "pill pill-warning"
                : "pill pill-muted"
          }
        >
          {connectionState === "connected"
            ? "已连接"
            : connectionState === "connecting"
              ? "连接中"
              : "未连接"}
        </span>
      </div>

      <div className="mcp-console-grid">
        <div className="mcp-console-panel">
          <dl className="metric-list">
            <div>
              <dt>当前学员</dt>
              <dd>{props.learnerName || "未登录"}</dd>
            </div>
            <div>
              <dt>学科</dt>
              <dd>{props.subjectKey || "-"}</dd>
            </div>
            <div>
              <dt>连接地址</dt>
              <dd className="mcp-inline-break">{resolvedSocketUrl || "-"}</dd>
            </div>
          </dl>

          <div className="button-row">
            <button className="primary-button" disabled={connectionState === "connecting"} onClick={connect} type="button">
              {connectionState === "connected" ? "重新握手" : "连接 websocket"}
            </button>
            <button className="secondary-button" disabled={connectionState === "disconnected"} onClick={disconnect} type="button">
              断开连接
            </button>
          </div>

          {infoLoading ? <div className="feedback-banner">正在读取 MCP 信息...</div> : null}
          {infoError ? <div className="feedback-banner feedback-error">{infoError}</div> : null}
          {connectionError ? <div className="feedback-banner feedback-error">{connectionError}</div> : null}

          {mcpInfo ? (
            <div className="mcp-info-card">
              <p className="helper-text">
                服务端：<strong>{mcpInfo.name}</strong> {mcpInfo.version}，协议版本 {mcpInfo.protocolVersion}
              </p>
              <p className="helper-text">
                方法：{mcpInfo.availableMethods.join(" / ")}
              </p>
            </div>
          ) : null}
        </div>

        <div className="mcp-console-panel">
          <label className="form-field">
            <span>选择工具</span>
            <select
              value={selectedTool}
              onChange={(event) => {
                setSelectedTool(event.target.value);
              }}
            >
              {(tools.length > 0 ? tools : fallbackTools).map((tool) => (
                <option key={tool.name} value={tool.name}>
                  {tool.title || tool.name}
                </option>
              ))}
            </select>
          </label>

          <label className="form-field">
            <span>工具参数 JSON</span>
            <textarea
              className="mcp-json-editor"
              rows={10}
              value={argumentsText}
              onChange={(event) => {
                setArgumentsText(event.target.value);
              }}
            />
          </label>

          <div className="button-row">
            <button className="primary-button" disabled={connectionState !== "connected"} onClick={callSelectedTool} type="button">
              调用 tools/call
            </button>
          </div>

          {callResultText ? (
            <div className="mcp-output-card">
              <strong>调用结果</strong>
              <pre>{callResultText}</pre>
            </div>
          ) : null}
        </div>
      </div>

      <div className="mcp-log-card">
        <div className="section-header">
          <div>
            <p className="section-eyebrow">会话日志</p>
            <h3>握手、收发包和错误都在这里</h3>
          </div>
        </div>
        {logs.length === 0 ? (
          <div className="feedback-banner">建立连接后，这里会显示 initialize、tools/list、tools/call 的收发记录。</div>
        ) : (
          <div className="mcp-log-list">
            {logs.map((entry) => (
              <article className={`mcp-log-entry mcp-log-entry-${entry.direction}`} key={entry.id}>
                <div className="mcp-log-entry-meta">
                  <span className="pill pill-muted">{entry.direction}</span>
                  <span>{entry.timestamp}</span>
                </div>
                <pre>{entry.text}</pre>
              </article>
            ))}
          </div>
        )}
      </div>
    </section>
  );
}

const fallbackTools: MCPInfoTool[] = [
  {
    name: "search_words",
    title: "Search Words",
    description: "Search Brights words.",
    inputSchema: {},
  },
];
