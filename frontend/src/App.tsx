"use client";

import {
  useState,
  useRef,
  useEffect,
  Component,
  ErrorInfo,
  ReactNode,
} from "react";
import { Send, Sparkles } from "lucide-react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";
import "./App.css";
import "highlight.js/styles/github-dark.css";
import { SendUserMessage } from "../wailsjs/go/main/App";

interface Message {
  id: string;
  role: "user" | "assistant" | "thinking";
  timestamp: Date;
  content: string;
  structured?: any; // parsed JSON response from backend
}

function ThinkingDots() {
  const [dots, setDots] = useState("");

  useEffect(() => {
    const interval = setInterval(
      () => setDots((prev) => (prev.length < 3 ? prev + "." : "")),
      400,
    );
    return () => clearInterval(interval);
  }, []);

  return <span>Thinking{dots}</span>;
}

// Error Boundary for markdown rendering
class MarkdownErrorBoundary extends Component<
  { children: ReactNode },
  { hasError: boolean; content: string }
> {
  constructor(props: { children: ReactNode }) {
    super(props);
    this.state = { hasError: false, content: "" };
  }

  static getDerivedStateFromError() {
    return { hasError: true };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error("Markdown rendering error:", error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div
          style={{
            color: "#c00",
            padding: "8px",
            background: "#fee",
            borderRadius: "4px",
          }}
        >
          Error rendering markdown. Showing plain text below:
          <pre
            style={{ whiteSpace: "pre-wrap", marginTop: "8px", color: "#000" }}
          >
            {this.props.children}
          </pre>
        </div>
      );
    }

    return this.props.children;
  }
}

// Safe markdown wrapper component
function SafeMarkdown({ content }: { content: string }) {
  return (
    <MarkdownErrorBoundary>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeHighlight]}
      >
        {content}
      </ReactMarkdown>
    </MarkdownErrorBoundary>
  );
}

export default function AiriChat() {
  const [messages, setMessages] = useState<Message[]>([
    {
      id: "1",
      content: "Hi! I'm Airi. How can I assist you today?",
      role: "assistant",
      timestamp: new Date(),
    },
  ]);

  const [input, setInput] = useState("");
  const [isThinking, setIsThinking] = useState(false);

  const messagesEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () =>
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });

  useEffect(() => {
    scrollToBottom();
  }, [messages, isThinking]);

  // ----- Detect output pattern -----
  const parseStructuredOutput = (text: string) => {
    try {
      const parsed = JSON.parse(text);
      // Only return if it has action, args, reasoning
      if (parsed.action && parsed.args && parsed.reasoning) return parsed;
    } catch (e) {
      // Not JSON, treat as conversational text
      return null;
    }
    return null;
  };

  const handleSend = async () => {
    if (!input.trim()) return;

    const userMessage: Message = {
      id: Date.now().toString(),
      content: input,
      role: "user",
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setInput("");
    setIsThinking(true);

    try {
      const response = await SendUserMessage(userMessage.content);

      let structured = parseStructuredOutput(response);

      const aiMessage: Message = {
        id: (Date.now() + 1).toString(),
        content: structured
          ? structured.reasoning || "Action completed"
          : response, // reasoning if JSON, raw text otherwise
        structured, // undefined if conversational
        role: "assistant",
        timestamp: new Date(),
      };

      setMessages((prev) => [...prev, aiMessage]);
    } catch (err: any) {
      setMessages((prev) => [
        ...prev,
        {
          id: (Date.now() + 2).toString(),
          role: "assistant",
          timestamp: new Date(),
          content: "Error: " + err.message,
        },
      ]);
    }

    setIsThinking(false);
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  // ----- Render structured data nicely -----
  const renderStructuredCard = (data: any) => {
    if (!data) return null;

    // Safety check to prevent crashes
    try {
      return (
        <div className="airi-card">
          <div className="airi-card-header">
            <Sparkles size={16} />
            <span>Airi System Action</span>
          </div>

          <div className="airi-card-body">
            <p>
              <b>Action:</b> {data.action}
            </p>

            {/* Directory inspection (simple) */}
            {data.action === "inspect_directory" && data.result && (
              <>
                <p>
                  <b>Directory Files:</b>
                </p>
                <ul>
                  {data.result.map((file: string, idx: number) => (
                    <li key={idx}>{file}</li>
                  ))}
                </ul>
              </>
            )}

            {/* Directory inspection (recursive) */}
            {data.action === "inspect_directory_recursive" && data.result && (
              <>
                <p>
                  <b>Files Found:</b>{" "}
                  {Array.isArray(data.result) ? data.result.length : 0}
                </p>
                <div
                  style={{
                    maxHeight: "300px",
                    overflowY: "auto",
                    background: "#eee",
                    padding: "8px",
                    borderRadius: "8px",
                    fontFamily: "monospace",
                    fontSize: "12px",
                    marginTop: "6px",
                  }}
                >
                  {Array.isArray(data.result) &&
                    data.result.map((file: string, idx: number) => (
                      <div key={idx}>{file}</div>
                    ))}
                </div>
              </>
            )}

            {/* Search files */}
            {data.action === "search_files" && data.result && (
              <>
                <p>
                  <b>Files Found:</b>{" "}
                  {Array.isArray(data.result) ? data.result.length : 0}
                </p>
                <div
                  style={{
                    maxHeight: "300px",
                    overflowY: "auto",
                    background: "#eee",
                    padding: "8px",
                    borderRadius: "8px",
                    fontFamily: "monospace",
                    fontSize: "12px",
                    marginTop: "6px",
                  }}
                >
                  {Array.isArray(data.result) &&
                    data.result.map((file: string, idx: number) => (
                      <div key={idx}>{file}</div>
                    ))}
                </div>
              </>
            )}

            {/* Find by extension */}
            {data.action === "find_by_extension" && data.result && (
              <>
                <p>
                  <b>Files Found:</b>{" "}
                  {Array.isArray(data.result) ? data.result.length : 0}
                </p>
                <div
                  style={{
                    maxHeight: "300px",
                    overflowY: "auto",
                    background: "#eee",
                    padding: "8px",
                    borderRadius: "8px",
                    fontFamily: "monospace",
                    fontSize: "12px",
                    marginTop: "6px",
                  }}
                >
                  {Array.isArray(data.result) &&
                    data.result.map((file: string, idx: number) => (
                      <div key={idx}>{file}</div>
                    ))}
                </div>
              </>
            )}

            {/* Directory tree */}
            {data.action === "directory_tree" && data.result && (
              <pre className="airi-pre">{data.result}</pre>
            )}

            {/* Delete file */}
            {data.action === "delete_file" && data.result && (
              <p style={{ color: "#16a34a", fontWeight: 600 }}>
                ✓ {data.result}
              </p>
            )}

            {/* Delete files batch */}
            {data.action === "delete_files_batch" && data.result && (
              <>
                <p style={{ color: "#16a34a", fontWeight: 600 }}>
                  ✓ Deleted {data.result.deleted_count} file(s)
                </p>
                {data.result.failed_count > 0 && (
                  <p style={{ color: "#dc2626", fontWeight: 600 }}>
                    ✗ Failed to delete {data.result.failed_count} file(s)
                  </p>
                )}
                {data.result.deleted && data.result.deleted.length > 0 && (
                  <>
                    <p>
                      <b>Deleted files:</b>
                    </p>
                    <div
                      style={{
                        maxHeight: "200px",
                        overflowY: "auto",
                        background: "#eee",
                        padding: "8px",
                        borderRadius: "8px",
                        fontFamily: "monospace",
                        fontSize: "12px",
                        marginTop: "6px",
                      }}
                    >
                      {data.result.deleted.map((file: string, idx: number) => (
                        <div key={idx}>✓ {file}</div>
                      ))}
                    </div>
                  </>
                )}
                {data.result.failed && data.result.failed.length > 0 && (
                  <>
                    <p>
                      <b>Failed:</b>
                    </p>
                    <div
                      style={{
                        maxHeight: "200px",
                        overflowY: "auto",
                        background: "#fee",
                        padding: "8px",
                        borderRadius: "8px",
                        fontFamily: "monospace",
                        fontSize: "12px",
                        marginTop: "6px",
                      }}
                    >
                      {data.result.failed.map((error: string, idx: number) => (
                        <div key={idx}>✗ {error}</div>
                      ))}
                    </div>
                  </>
                )}
              </>
            )}

            {/* Process list */}
            {data.action === "list_processes" && data.result && (
              <div
                style={{
                  maxHeight: "200px",
                  overflowY: "auto",
                  background: "#eee",
                  padding: "8px",
                  borderRadius: "8px",
                  fontFamily: "monospace",
                  whiteSpace: "pre-wrap",
                  marginTop: "6px",
                }}
              >
                <div>
                  <b>PID Name</b>
                </div>
                <div>
                  {data.result
                    .map((p: any) => `${p.pid}\t${p.name}`)
                    .join("\n")}
                </div>
              </div>
            )}

            {/* File read */}
            {data.action === "read_file" && data.result && (
              <pre className="airi-pre">
                {typeof data.result === "string"
                  ? data.result
                  : JSON.stringify(data.result, null, 2)}
              </pre>
            )}

            {/* File write */}
            {data.action === "write_file" && data.result && (
              <p style={{ color: "#16a34a", fontWeight: 600 }}>
                ✓ {data.result}
              </p>
            )}

            {/* Kill output */}
            {data.action === "kill_process" && data.result && (
              <pre className="airi-pre">
                {JSON.stringify(data.result, null, 2)}
              </pre>
            )}
          </div>
        </div>
      );
    } catch (error) {
      console.error("Error rendering structured card:", error);
      return (
        <div
          className="airi-card"
          style={{ background: "#fee", borderColor: "#fcc" }}
        >
          <p style={{ color: "#c00" }}>Error displaying action result</p>
        </div>
      );
    }
  };

  return (
    <div className="chat-container">
      {/* Header */}
      <header className="chat-header">
        <div className="header-content">
          <div className="logo-container">
            <Sparkles className="logo-icon" />
          </div>
          <div>
            <h1 className="app-title">airi</h1>
            <p className="app-subtitle">AI OS Assistant</p>
          </div>
        </div>
      </header>

      {/* Messages */}
      <div className="messages-container">
        <div className="messages-wrapper">
          {messages.map((m) => (
            <div key={m.id} className={`message ${m.role}`}>
              <div className="message-bubble">
                {m.role === "user" ? (
                  <p className="message-text">{m.content}</p>
                ) : (
                  <div className="message-text markdown-content">
                    <SafeMarkdown content={m.content} />
                  </div>
                )}

                {/* If result is structured JSON, show pretty card */}
                {m.role === "assistant" && renderStructuredCard(m.structured)}
              </div>
            </div>
          ))}

          {isThinking && (
            <div className="message assistant">
              <div className="message-bubble">
                <p className="message-text">
                  <ThinkingDots />
                </p>
              </div>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>
      </div>

      {/* Input */}
      <div className="input-area">
        <div className="input-wrapper">
          <div className="input-group">
            <textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyPress}
              placeholder="Type your message..."
              rows={1}
              className="message-input"
            />
            <button
              onClick={handleSend}
              disabled={!input.trim()}
              className="send-button"
            >
              <Send className="send-icon" />
            </button>
          </div>
        </div>
      </div>

      {/* Inline styling for cards */}
      <style>{`
        .airi-card {
          margin-top: 10px;
          padding: 12px;
          border-radius: 12px;
          background: #f7f7fa;
          border: 1px solid #e5e7eb;
        }
        .airi-card-header {
          display: flex;
          align-items: center;
          gap: 6px;
          font-weight: 600;
          margin-bottom: 6px;
        }
        .airi-card-body {
          font-size: 14px;
        }
        .airi-pre {
          background: #eee;
          padding: 8px;
          border-radius: 8px;
          white-space: pre-wrap;
          font-family: monospace;
          margin-top: 6px;
        }
      `}</style>
    </div>
  );
}
