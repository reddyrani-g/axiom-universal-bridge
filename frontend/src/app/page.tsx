"use client";

import { useState, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { ShieldCheck, Database, BrainCircuit, Activity, UploadCloud, ChevronDown, Trash2, Plus } from "lucide-react";
import JSONViewer from "@/components/JSONViewer";

interface ProgressEvent {
  step: string;
  status: string;
  message: string;
  timestamp: string;
  data?: any;
}

interface ChatSession {
  id: string;
  input: string;
  result?: any;
  timestamp: Date;
}

const steps = [
  { title: "Cache", icon: <Database size={16} />, key: "cache" },
  { title: "Redaction", icon: <ShieldCheck size={16} />, key: "dlp" },
  { title: "Analysis", icon: <BrainCircuit size={16} />, key: "gemini" },
  { title: "Storage", icon: <Activity size={16} />, key: "bigquery" }
];

export default function Home() {
  const [textInput, setTextInput] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<any>(null);
  const [events, setEvents] = useState<ProgressEvent[]>([]);
  const [expandedRaw, setExpandedRaw] = useState(false);
  const [history, setHistory] = useState<ChatSession[]>([]);
  const [activeSession, setActiveSession] = useState<string | null>(null);

  // Load history from localStorage
  useEffect(() => {
    const saved = localStorage.getItem("axiom_history");
    if (saved) {
      setHistory(JSON.parse(saved));
    }
  }, []);

  // Save history to localStorage
  useEffect(() => {
    localStorage.setItem("axiom_history", JSON.stringify(history));
  }, [history]);

  const getStepStatus = (stepKey: string) => {
    const event = events.find(e => e.step === stepKey);
    return event?.status || "pending";
  };

  const handleProcess = async () => {
    if (!textInput && !file) return;
    setLoading(true);
    setResult(null);
    setEvents([]);

    const inputText = textInput || `File: ${file?.name}`;
    const sessionId = Date.now().toString();

    try {
      const formData = new FormData();
      if (textInput) formData.append("text", textInput);
      if (file) formData.append("file", file);

      const response = await fetch("/api/process?stream=true", {
        method: "POST",
        body: formData,
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const reader = response.body?.getReader();
      if (!reader) throw new Error("No response body");

      const decoder = new TextDecoder();
      let buffer = "";
      let finalResult: any = null;

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";

        for (const line of lines) {
          if (line.startsWith("data: ")) {
            try {
              const eventData = JSON.parse(line.slice(6));
              setEvents(prev => [...prev, eventData]);

              if (eventData.step === "complete" && eventData.status === "success") {
                finalResult = eventData.data;
                setResult(eventData.data);
              }
            } catch (e) {
              console.error("Failed to parse event:", e);
            }
          }
        }
      }

      // Add to history
      const newSession: ChatSession = {
        id: sessionId,
        input: inputText,
        result: finalResult,
        timestamp: new Date()
      };
      setHistory(prev => [newSession, ...prev.slice(0, 49)]);
      setActiveSession(sessionId);
    } catch (e) {
      console.error(e);
      setResult({
        error: "Failed to connect to Bridge API",
        details: String(e)
      });
    }

    setLoading(false);
    setTextInput("");
    setFile(null);
  };

  const loadSession = (session: ChatSession) => {
    setActiveSession(session.id);
    setTextInput(session.input);
    setResult(session.result);
  };

  const deleteSession = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setHistory(prev => prev.filter(s => s.id !== id));
    if (activeSession === id) {
      setActiveSession(null);
      setResult(null);
    }
  };

  const clearHistory = () => {
    if (confirm("Clear all chat history?")) {
      setHistory([]);
      setActiveSession(null);
      setResult(null);
    }
  };

  const newChat = () => {
    setActiveSession(null);
    setResult(null);
    setTextInput("");
    setFile(null);
    setEvents([]);
  };

  return (
    <div className="flex h-screen bg-white">
      {/* Sidebar */}
      <div className="w-64 border-r border-gray-200 bg-gray-50 flex flex-col">
        <div className="p-4 border-b border-gray-200">
          <button
            onClick={newChat}
            className="w-full flex items-center gap-2 px-4 py-2 bg-white border border-gray-300 text-gray-700 rounded-8 hover:bg-gray-50 transition text-sm font-medium"
          >
            <Plus size={18} /> New Chat
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-3 space-y-2">
          {history.length === 0 ? (
            <p className="text-xs text-gray-500 text-center pt-4">No history yet</p>
          ) : (
            history.map(session => (
              <div
                key={session.id}
                onClick={() => loadSession(session)}
                className={`p-3 rounded-8 cursor-pointer transition text-sm ${
                  activeSession === session.id
                    ? "bg-blue-100 border border-blue-300"
                    : "bg-white border border-gray-200 hover:border-gray-300"
                }`}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="flex-1 min-w-0">
                    <p className="text-gray-800 truncate font-medium text-xs">
                      {session.input.substring(0, 30)}...
                    </p>
                    <p className="text-gray-500 text-xs mt-1">
                      {new Date(session.timestamp).toLocaleDateString()}
                    </p>
                  </div>
                  <button
                    onClick={(e) => deleteSession(session.id, e)}
                    className="text-gray-400 hover:text-red-500 transition"
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              </div>
            ))
          )}
        </div>

        {history.length > 0 && (
          <div className="p-3 border-t border-gray-200">
            <button
              onClick={clearHistory}
              className="w-full text-xs text-gray-600 hover:text-red-600 font-medium transition"
            >
              Clear history
            </button>
          </div>
        )}
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Header */}
        <div className="border-b border-gray-200 bg-white px-8 py-6">
          <h1 className="text-3xl font-bold text-gray-900">Axiom Bridge</h1>
          <p className="text-gray-600 mt-1">AI-powered decision engine</p>
        </div>

        {/* Content Area */}
        <div className="flex-1 overflow-y-auto">
          <div className="max-w-6xl mx-auto px-8 py-6">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* Input Panel */}
              <div className="card">
                <h2 className="text-lg font-bold text-gray-900 mb-4">Input</h2>

                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Text Input
                    </label>
                    <textarea
                      value={textInput}
                      onChange={(e) => setTextInput(e.target.value)}
                      placeholder="Paste context, analysis, or reports..."
                      className="w-full h-32 p-3 border border-gray-300 rounded-8 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Or Upload File
                    </label>
                    <input
                      type="file"
                      onChange={(e) => setFile(e.target.files?.[0] || null)}
                      className="block w-full text-sm text-gray-500"
                    />
                    {file && <p className="text-xs text-green-600 mt-2">✓ {file.name}</p>}
                  </div>

                  <button
                    onClick={handleProcess}
                    disabled={loading || (!textInput && !file)}
                    className="btn-primary w-full"
                  >
                    {loading ? "Processing..." : "Analyze"}
                  </button>
                </div>
              </div>

              {/* Results Panel */}
              <div className="card flex flex-col">
                <h2 className="text-lg font-bold text-gray-900 mb-4">Results</h2>

                {loading && (
                  <div className="space-y-3">
                    {steps.map((step, idx) => {
                      const status = getStepStatus(step.key);
                      const isActive = status === "processing" || status === "checking";

                      return (
                        <div key={idx} className={`step-item ${isActive ? "active" : ""}`}>
                          <div className={`step-icon ${isActive ? "bg-blue-500" : "bg-gray-200"}`}>
                            <span>{isActive ? "⟳" : step.icon}</span>
                          </div>
                          <div>
                            <p className="text-sm font-medium text-gray-900">{step.title}</p>
                            {events.find(e => e.step === step.key) && (
                              <p className="text-xs text-gray-600 mt-0.5">
                                {events.find(e => e.step === step.key)?.message}
                              </p>
                            )}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}

                {!loading && result && (
                  <div className="space-y-4 flex-1">
                    {result.error ? (
                      <div className="error-message">
                        {result.error}
                      </div>
                    ) : result.result ? (
                      <>
                        <div className="flex gap-2">
                          <span className="badge badge-info">{result.source}</span>
                          <span className={`badge ${
                            result.result?.urgency === "CRITICAL" || result.result?.urgency === "HIGH"
                              ? "badge-error"
                              : "badge-success"
                          }`}>
                            {result.result?.urgency}
                          </span>
                        </div>

                        <div className="result-section">
                          <h4>Summary</h4>
                          <p className="text-sm text-gray-700 line-clamp-3">{result.result?.summary}</p>
                        </div>

                        {result.result?.action_items?.length > 0 && (
                          <div className="result-section">
                            <h4>Actions</h4>
                            <ul className="text-sm text-gray-700 space-y-1">
                              {result.result.action_items.slice(0, 3).map((item: string, i: number) => (
                                <li key={i}>• {item}</li>
                              ))}
                              {result.result.action_items.length > 3 && (
                                <li className="text-gray-500">+ {result.result.action_items.length - 3} more</li>
                              )}
                            </ul>
                          </div>
                        )}

                        {result.result?.entities?.length > 0 && (
                          <div className="result-section">
                            <h4>Entities</h4>
                            <div className="flex flex-wrap gap-2">
                              {result.result.entities.slice(0, 4).map((entity: string, i: number) => (
                                <span key={i} className="badge badge-info">
                                  {entity}
                                </span>
                              ))}
                            </div>
                          </div>
                        )}

                        <button
                          onClick={() => setExpandedRaw(!expandedRaw)}
                          className="text-xs text-gray-600 hover:text-gray-900 font-medium flex items-center gap-1"
                        >
                          <ChevronDown size={14} className={`transition ${expandedRaw ? "rotate-180" : ""}`} />
                          View Raw
                        </button>

                        <AnimatePresence>
                          {expandedRaw && (
                            <motion.div
                              initial={{ opacity: 0, height: 0 }}
                              animate={{ opacity: 1, height: "auto" }}
                              exit={{ opacity: 0, height: 0 }}
                              className="bg-gray-100 p-3 rounded-8 overflow-auto max-h-40 text-xs"
                            >
                              <JSONViewer data={result.result} />
                            </motion.div>
                          )}
                        </AnimatePresence>
                      </>
                    ) : (
                      <p className="text-gray-500 text-sm">Run analysis to see results</p>
                    )}
                  </div>
                )}

                {!loading && !result && (
                  <p className="text-gray-500 text-sm">Results will appear here</p>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

