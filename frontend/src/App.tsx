"use client";

import { useState, useRef, useEffect } from "react";
import { Send, Sparkles } from "lucide-react";
import "./App.css";
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
      400
    );
    return () => clearInterval(interval);
  }, []);

  return <span>Thinking{dots}</span>;
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
        content: structured ? structured.reasoning || "" : response, // reasoning if JSON, raw text otherwise
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

          {/* Directory inspection */}
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
                {data.result.map((p: any) => `${p.pid}\t${p.name}`).join("\n")}
              </div>
            </div>
          )}

          {/* File read */}
          {data.action === "read_file" && data.result && (
            <pre className="airi-pre">
              {JSON.stringify(data.result, null, 2)}
            </pre>
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
                <p className="message-text">{m.content}</p>

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
