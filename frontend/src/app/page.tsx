"use client";

import { useEffect, useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import {
  Activity,
  BrainCircuit,
  ChevronDown,
  Database,
  FileText,
  Menu,
  Plus,
  Sparkles,
  Trash2,
  UploadCloud
} from "lucide-react";
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
  fileName?: string;
  fileType?: string;
  result?: any;
  timestamp: Date;
}

const steps = [
  { title: "Cache", icon: <Database size={16} />, key: "cache" },
  { title: "Analysis", icon: <BrainCircuit size={16} />, key: "gemini" },
  { title: "Storage", icon: <Activity size={16} />, key: "bigquery" }
];

const samplePrompts = [
  {
    title: "Budget Variance",
    accent: "from-sky-500/20 via-cyan-500/10 to-transparent",
    border: "hover:border-sky-300",
    copy:
      "Quarterly budget variance report showing 15% overspend in marketing budget. Key drivers: paid advertising costs increased due to seasonal campaigns, staffing costs higher than budgeted. Recommend reallocating 8% from contingency fund to cover overages."
  },
  {
    title: "Security Incident",
    accent: "from-rose-500/20 via-orange-500/10 to-transparent",
    border: "hover:border-rose-300",
    copy:
      "Security incident: Unauthorized access detected on customer database. Affected records: 2,500 customer profiles. Time window: 2024-03-20 14:30 to 15:45 UTC. Root cause: Expired API key not rotated. Impact: customer email addresses, phone numbers exposed. Mitigation: Revoked compromised key, enabled API rotation policy."
  },
  {
    title: "Customer Feedback",
    accent: "from-fuchsia-500/20 via-violet-500/10 to-transparent",
    border: "hover:border-fuchsia-300",
    copy:
      "Product feedback summary from Q1 user interviews (50 respondents): Top issues: (1) Slow export functionality - 24 mentions, (2) Unclear pricing tiers - 18 mentions, (3) Missing API documentation - 12 mentions. Feature requests: Dark mode (21), Bulk operations (16), SSO integration (14). CSAT score: 7.2/10. Churn risk: 15% of respondents considering alternatives."
  },
  {
    title: "Compliance Review",
    accent: "from-amber-500/20 via-yellow-500/10 to-transparent",
    border: "hover:border-amber-300",
    copy:
      "Compliance audit findings: SOC2 Type II certification status - 3 open items: (1) Backup recovery procedure missing documentation, (2) Access control review overdue (Oct 2024), (3) Encryption keys stored without HSM backup. Audit deadline: April 2024. Risk level: Medium. Estimated remediation time: 2 weeks."
  }
];

function formatSessionTitle(text: string) {
  return text.length > 42 ? `${text.slice(0, 42)}...` : text;
}

function formatTimestamp(value: Date) {
  return new Date(value).toLocaleString([], {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

function buildDisplayResponse(result: any) {
  if (!result) {
    return null;
  }

  return {
    urgency: result.urgency ?? "UNKNOWN",
    confidence:
      typeof result.confidence === "number"
        ? Number(result.confidence.toFixed(2))
        : null,
    summary: result.summary ?? "",
    possible_condition: result.possible_condition ?? "",
    actions: Array.isArray(result.actions) ? result.actions : [],
    do_not: Array.isArray(result.do_not) ? result.do_not : [],
    reasoning: result.reasoning ?? "",
  };
}

export default function Home() {
  const [textInput, setTextInput] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<any>(null);
  const [events, setEvents] = useState<ProgressEvent[]>([]);
  const [expandedRaw, setExpandedRaw] = useState(false);
  const [history, setHistory] = useState<ChatSession[]>([]);
  const [activeSession, setActiveSession] = useState<string | null>(null);
  const [sidebarOpen, setSidebarOpen] = useState(false);

  useEffect(() => {
    const saved = localStorage.getItem("axiom_history");
    if (saved) {
      setHistory(JSON.parse(saved));
    }
  }, []);

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
    setExpandedRaw(false);

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

      const newSession: ChatSession = {
        id: sessionId,
        input: inputText,
        fileName: file?.name,
        fileType: file?.type,
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
    setExpandedRaw(false);

    if (session.fileName) {
      setFile(new File([], session.fileName, { type: session.fileType }));
    } else {
      setFile(null);
    }
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
    setExpandedRaw(false);
  };

  const latestResult = result?.result;

  const urgencyTone =
    latestResult?.urgency === "CRITICAL" || latestResult?.urgency === "HIGH"
      ? "bg-rose-500/15 text-rose-700 ring-rose-200"
      : latestResult?.urgency === "MEDIUM"
      ? "bg-amber-500/15 text-amber-700 ring-amber-200"
      : "bg-emerald-500/15 text-emerald-700 ring-emerald-200";

  const displayResponse = buildDisplayResponse(latestResult);
  const primaryResponseText = displayResponse
    ? JSON.stringify(displayResponse, null, 2)
    : result?.error || "Run analysis to see the response.";

  return (
    <div className="relative min-h-screen overflow-hidden bg-[radial-gradient(circle_at_top_left,_rgba(14,165,233,0.18),_transparent_30%),radial-gradient(circle_at_top_right,_rgba(244,114,182,0.14),_transparent_26%),linear-gradient(180deg,_#f8fbff_0%,_#eef4ff_48%,_#f7f9fc_100%)]">
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(rgba(15,23,42,0.03)_1px,transparent_1px),linear-gradient(90deg,rgba(15,23,42,0.03)_1px,transparent_1px)] bg-[size:32px_32px]" />

      {sidebarOpen && (
        <div
          className="fixed inset-0 z-30 bg-slate-950/35 backdrop-blur-sm md:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      <div className="relative z-10 flex min-h-screen">
        <aside
          className={`fixed inset-y-0 left-0 z-40 flex w-72 flex-col border-r border-white/50 bg-white/80 shadow-[0_20px_60px_rgba(15,23,42,0.12)] backdrop-blur-xl transition-transform duration-300 md:relative md:translate-x-0 ${
            sidebarOpen ? "translate-x-0" : "-translate-x-full"
          }`}
        >
          <div className="border-b border-slate-200/80 p-5">
            <div className="rounded-3xl border border-slate-200/80 bg-[linear-gradient(135deg,_rgba(14,165,233,0.14),_rgba(255,255,255,0.9),_rgba(244,114,182,0.12))] p-4 shadow-sm">
              <div className="mb-4 flex items-center gap-3">
                <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-slate-950 text-white shadow-lg shadow-sky-200/50">
                  <Activity size={22} />
                </div>
                <div>
                  <p className="text-sm font-semibold uppercase tracking-[0.24em] text-sky-700">Axiom Bridge</p>
                  <p className="text-sm text-slate-600">Decision intelligence cockpit</p>
                </div>
              </div>

              <button
                onClick={newChat}
                className="flex w-full items-center justify-center gap-2 rounded-2xl bg-slate-950 px-4 py-3 text-sm font-semibold text-white transition hover:-translate-y-0.5 hover:bg-slate-800 hover:shadow-xl"
              >
                <Plus size={18} />
                New Analysis
              </button>
            </div>
          </div>

          <div className="flex items-center justify-between px-5 pt-5">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-500">Recent Sessions</p>
              <p className="mt-1 text-sm text-slate-600">{history.length} saved runs</p>
            </div>
          </div>

          <div className="flex-1 space-y-3 overflow-y-auto px-4 py-4">
            {history.length === 0 ? (
              <div className="rounded-3xl border border-dashed border-slate-300 bg-white/70 p-5 text-center">
                <p className="text-sm font-semibold text-slate-700">No conversations yet</p>
                <p className="mt-2 text-xs text-slate-500">Run your first analysis and it will appear here.</p>
              </div>
            ) : (
              history.map(session => (
                <button
                  key={session.id}
                  onClick={() => {
                    loadSession(session);
                    setSidebarOpen(false);
                  }}
                  className={`group w-full rounded-3xl border p-4 text-left transition ${
                    activeSession === session.id
                      ? "border-sky-300 bg-sky-50/80 shadow-lg shadow-sky-100"
                      : "border-white/60 bg-white/70 hover:-translate-y-0.5 hover:border-slate-200 hover:bg-white hover:shadow-lg"
                  }`}
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-semibold text-slate-800">{formatSessionTitle(session.input)}</p>
                      <div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-slate-500">
                        <span className="rounded-full bg-slate-900/5 px-2.5 py-1">{formatTimestamp(session.timestamp)}</span>
                        {session.fileName && (
                          <span className="rounded-full bg-sky-100 px-2.5 py-1 font-medium text-sky-700">
                            {formatSessionTitle(session.fileName)}
                          </span>
                        )}
                      </div>
                    </div>

                    <span
                      onClick={(e) => deleteSession(session.id, e)}
                      className="rounded-full p-2 text-slate-400 opacity-0 transition hover:bg-red-50 hover:text-red-500 group-hover:opacity-100"
                    >
                      <Trash2 size={14} />
                    </span>
                  </div>
                </button>
              ))
            )}
          </div>

          {history.length > 0 && (
            <div className="border-t border-slate-200/80 p-4">
              <button
                onClick={clearHistory}
                className="w-full rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm font-semibold text-slate-600 transition hover:border-red-200 hover:bg-red-50 hover:text-red-600"
              >
                Clear history
              </button>
            </div>
          )}
        </aside>

        <main className="flex min-w-0 flex-1 flex-col">
          <header className="border-b border-white/50 bg-white/55 px-4 py-5 backdrop-blur-xl sm:px-8">
            <div className="mx-auto flex max-w-7xl items-start justify-between gap-4">
              <div className="flex min-w-0 items-start gap-3">
                <button
                  onClick={() => setSidebarOpen(!sidebarOpen)}
                  className="mt-1 rounded-2xl border border-slate-200 bg-white/80 p-2.5 text-slate-700 transition hover:bg-white md:hidden"
                >
                  <Menu size={20} />
                </button>

                <div>
                  <div className="mb-2 flex flex-wrap items-center gap-2">
                    <span className="inline-flex items-center gap-2 rounded-full bg-sky-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.18em] text-sky-700">
                      <Sparkles size={14} />
                      Live Intelligence
                    </span>
                    <span className="rounded-full bg-slate-900/5 px-3 py-1 text-xs font-medium text-slate-600">
                      Streamed pipeline visibility
                    </span>
                  </div>
                  <h1 className="text-3xl font-black tracking-[-0.04em] text-slate-950 sm:text-5xl">Enterprise signals, distilled fast.</h1>
                  <p className="mt-3 max-w-3xl text-sm text-slate-600 sm:text-base">
                    Drop in incidents, audits, reports, or feedback and Axiom Bridge turns the raw payload into urgency, actions, and structured intelligence.
                  </p>
                </div>
              </div>

              <div className="hidden rounded-3xl border border-white/60 bg-white/80 p-4 shadow-lg shadow-slate-200/60 lg:block">
                <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">This run</p>
                <div className="mt-3 grid grid-cols-3 gap-3">
                  <div className="rounded-2xl bg-slate-950 px-4 py-3 text-white">
                    <p className="text-xs uppercase tracking-[0.18em] text-white/60">Sessions</p>
                    <p className="mt-2 text-2xl font-bold">{history.length}</p>
                  </div>
                  <div className="rounded-2xl bg-sky-50 px-4 py-3">
                    <p className="text-xs uppercase tracking-[0.18em] text-sky-700">Pipeline</p>
                    <p className="mt-2 text-2xl font-bold text-slate-900">{steps.length}</p>
                  </div>
                  <div className="rounded-2xl bg-emerald-50 px-4 py-3">
                    <p className="text-xs uppercase tracking-[0.18em] text-emerald-700">Mode</p>
                    <p className="mt-2 text-lg font-bold text-slate-900">Streaming</p>
                  </div>
                </div>
              </div>
            </div>
          </header>

          <div className="flex-1 overflow-y-auto px-4 py-6 sm:px-8 sm:py-8">
            <div className="mx-auto grid max-w-7xl grid-cols-1 gap-6 xl:grid-cols-[1.05fr_0.95fr]">
              <section className="card card-elevated">
                <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
                  <div>
                    <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">Input Workspace</p>
                    <h2 className="mt-2 text-2xl font-bold text-slate-950">Analyze a report or upload a file</h2>
                    <p className="mt-2 text-sm text-slate-600">Paste source material, attach a document, then send it through the decision pipeline.</p>
                  </div>
                  <div className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-right">
                    <p className="text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">Ready state</p>
                    <p className="mt-1 text-sm font-semibold text-slate-800">{file ? "Text + file staged" : textInput ? "Text staged" : "Awaiting input"}</p>
                  </div>
                </div>

                <div className="space-y-6">
                  <div className="rounded-[28px] border border-slate-200 bg-white/90 p-4 shadow-[0_12px_30px_rgba(15,23,42,0.04)]">
                    <div className="mb-3 flex items-center gap-2">
                      <div className="rounded-2xl bg-sky-100 p-2 text-sky-700">
                        <FileText size={16} />
                      </div>
                      <div>
                        <p className="text-sm font-semibold text-slate-900">Text input</p>
                        <p className="text-xs text-slate-500">Incidents, audits, reports, escalations, feedback</p>
                      </div>
                    </div>

                    <textarea
                      value={textInput}
                      onChange={(e) => setTextInput(e.target.value)}
                      placeholder="Paste incident reports, budget variance, compliance findings, customer complaints, or structured JSON..."
                      className="min-h-[220px] w-full rounded-[24px] border border-slate-200 bg-slate-50/70 p-5 text-sm font-medium text-slate-800 placeholder:text-slate-400 focus:border-sky-400 focus:bg-white focus:ring-4 focus:ring-sky-100"
                    />
                  </div>

                  <div className="grid gap-4 lg:grid-cols-[1fr_auto]">
                    <label
                      htmlFor="file-input"
                      className="group relative block cursor-pointer overflow-hidden rounded-[28px] border border-dashed border-slate-300 bg-[linear-gradient(135deg,_rgba(14,165,233,0.08),_rgba(255,255,255,0.95),_rgba(16,185,129,0.08))] p-5 transition hover:border-sky-400 hover:shadow-lg"
                    >
                      <input
                        id="file-input"
                        type="file"
                        accept="image/*,audio/*,text/*,application/json,application/xml"
                        onChange={(e) => setFile(e.target.files?.[0] || null)}
                        className="hidden"
                      />
                      <div className="flex items-center gap-4">
                        <div className="rounded-2xl bg-slate-950 p-3 text-white transition group-hover:scale-105">
                          <UploadCloud size={20} />
                        </div>
                        <div>
                          <p className="text-sm font-semibold text-slate-900">Upload supporting file</p>
                          <p className="mt-1 text-xs text-slate-500">Drag and drop or click to select an image, audio, or text document</p>
                        </div>
                      </div>
                    </label>

                    <button
                      onClick={handleProcess}
                      disabled={loading || (!textInput && !file)}
                      className="flex min-h-[112px] items-center justify-center rounded-[28px] bg-slate-950 px-8 py-5 text-center text-base font-semibold text-white transition hover:-translate-y-0.5 hover:bg-slate-800 hover:shadow-[0_18px_40px_rgba(15,23,42,0.22)] disabled:translate-y-0 disabled:bg-slate-400"
                    >
                      {loading ? (
                        <span className="flex items-center gap-3">
                          <span className="inline-flex h-5 w-5 animate-spin rounded-full border-2 border-white/30 border-t-white" />
                          Analyzing
                        </span>
                      ) : (
                        "Run analysis"
                      )}
                    </button>
                  </div>

                  {file && (
                    <div className="inline-flex items-center gap-2 rounded-full border border-emerald-200 bg-emerald-50 px-4 py-2 text-sm font-medium text-emerald-700">
                      <span className="h-2.5 w-2.5 rounded-full bg-emerald-500" />
                      {file.name}
                    </div>
                  )}

                  <div>
                    <div className="mb-4 flex items-center justify-between">
                      <div>
                        <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">Quick Starts</p>
                        <p className="mt-1 text-sm text-slate-600">Drop in a sample prompt to preview the pipeline.</p>
                      </div>
                    </div>

                    <div className="grid gap-3 md:grid-cols-2">
                      {samplePrompts.map((prompt) => (
                        <button
                          key={prompt.title}
                          onClick={() => setTextInput(prompt.copy)}
                          className={`relative overflow-hidden rounded-[26px] border border-white/70 bg-white p-4 text-left shadow-[0_10px_24px_rgba(15,23,42,0.05)] transition hover:-translate-y-1 hover:shadow-xl ${prompt.border}`}
                        >
                          <div className={`absolute inset-0 bg-gradient-to-br ${prompt.accent}`} />
                          <div className="relative">
                            <p className="text-sm font-semibold text-slate-900">{prompt.title}</p>
                            <p className="mt-2 text-xs leading-6 text-slate-600">{prompt.copy.slice(0, 118)}...</p>
                          </div>
                        </button>
                      ))}
                    </div>
                  </div>
                </div>
              </section>

              <section className="card card-elevated flex min-h-[720px] flex-col">
                <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
                  <div>
                    <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">Output Panel</p>
                    <h2 className="mt-2 text-2xl font-bold text-slate-950">Structured response</h2>
                    <p className="mt-2 text-sm text-slate-600">Track the pipeline live, then review one compact response object.</p>
                  </div>
                </div>

                {loading ? (
                  <div className="space-y-4">
                    <div className="rounded-[28px] border border-slate-200 bg-slate-50 p-4">
                      <p className="text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">Pipeline status</p>
                      <p className="mt-2 text-sm text-slate-700">Axiom Bridge is streaming step-by-step progress as your input moves through the stack.</p>
                    </div>

                    <div className="rounded-[30px] border border-slate-200 bg-[linear-gradient(135deg,_rgba(14,165,233,0.12),_rgba(255,255,255,0.96),_rgba(244,114,182,0.08))] p-6">
                      <div className="flex items-center justify-between gap-3">
                        <p className="text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">Preparing response</p>
                        <span className="inline-flex items-center gap-2 rounded-full bg-sky-100 px-3 py-1 text-xs font-semibold text-sky-700">
                          <span className="h-2 w-2 animate-pulse rounded-full bg-sky-500" />
                          Streaming
                        </span>
                      </div>

                      <div className="mt-4 rounded-[24px] bg-slate-950 p-5 shadow-[inset_0_1px_0_rgba(255,255,255,0.04)]">
                        <div className="space-y-3">
                          <div className="h-4 w-32 animate-pulse rounded-full bg-emerald-200/20" />
                          <div className="h-4 w-5/6 animate-pulse rounded-full bg-emerald-200/15" />
                          <div className="h-4 w-4/6 animate-pulse rounded-full bg-emerald-200/15" />
                          <div className="h-4 w-3/4 animate-pulse rounded-full bg-emerald-200/15" />
                          <div className="h-4 w-2/3 animate-pulse rounded-full bg-emerald-200/15" />
                          <div className="pt-2 text-xs font-medium text-slate-400">
                            {events[events.length - 1]?.message || "Waiting for the backend to finish the structured response..."}
                          </div>
                        </div>
                      </div>
                    </div>

                    {steps.map((step, idx) => {
                      const status = getStepStatus(step.key);
                      const currentEvent = events.find(e => e.step === step.key);
                      const isActive = status === "processing" || status === "checking";
                      const isComplete = status === "complete";

                      return (
                        <div
                          key={step.key}
                          className={`rounded-[26px] border p-4 transition ${
                            isActive
                              ? "border-sky-300 bg-sky-50 shadow-lg shadow-sky-100"
                              : isComplete
                              ? "border-emerald-200 bg-emerald-50"
                              : "border-slate-200 bg-white"
                          }`}
                        >
                          <div className="flex items-start gap-4">
                            <div
                              className={`flex h-11 w-11 items-center justify-center rounded-2xl text-sm font-semibold ${
                                isActive
                                  ? "bg-sky-600 text-white"
                                  : isComplete
                                  ? "bg-emerald-600 text-white"
                                  : "bg-slate-100 text-slate-600"
                              }`}
                            >
                              {isActive ? "..." : isComplete ? "OK" : idx + 1}
                            </div>

                            <div className="min-w-0 flex-1">
                              <div className="flex items-center gap-2">
                                <span className="text-slate-700">{step.icon}</span>
                                <p className="text-sm font-semibold text-slate-900">{step.title}</p>
                              </div>
                              <p className="mt-2 text-sm text-slate-600">
                                {currentEvent?.message || "Waiting for this stage to begin."}
                              </p>
                            </div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                ) : !result ? (
                  <div className="flex flex-1 flex-col justify-between gap-6">
                    <div className="rounded-[30px] border border-dashed border-slate-300 bg-[linear-gradient(135deg,_rgba(14,165,233,0.08),_rgba(255,255,255,0.92),_rgba(244,114,182,0.06))] p-8">
                      <p className="text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">Waiting for analysis</p>
                      <h3 className="mt-3 text-2xl font-bold text-slate-950">Your response will land here.</h3>
                      <p className="mt-3 max-w-xl text-sm text-slate-600">
                        Once you run a prompt, you&apos;ll get a simple text response, pipeline progress, and optional raw JSON if you want to inspect the payload.
                      </p>
                    </div>

                    <div className="grid gap-3 md:grid-cols-2">
                      {steps.map((step) => (
                        <div key={step.key} className="rounded-[24px] border border-white/70 bg-white p-4 shadow-[0_10px_24px_rgba(15,23,42,0.05)]">
                          <div className="flex items-center gap-3">
                            <div className="rounded-2xl bg-slate-100 p-2 text-slate-700">{step.icon}</div>
                            <div>
                              <p className="text-sm font-semibold text-slate-900">{step.title}</p>
                              <p className="text-xs text-slate-500">Visible once the stream starts</p>
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : result.error ? (
                  <div className="rounded-[28px] border border-rose-200 bg-rose-50 p-5">
                    <p className="text-xs font-semibold uppercase tracking-[0.18em] text-rose-600">Connection Error</p>
                    <p className="mt-2 text-lg font-semibold text-rose-900">{result.error}</p>
                    {result.details && <p className="mt-2 text-sm text-rose-700">{result.details}</p>}
                  </div>
                ) : latestResult ? (
                  <div className="flex flex-1 flex-col gap-5">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="rounded-full bg-slate-900 px-3 py-1.5 text-xs font-semibold uppercase tracking-[0.18em] text-white">
                        {result.source || "Unknown source"}
                      </span>
                      <span className={`rounded-full px-3 py-1.5 text-xs font-semibold uppercase tracking-[0.16em] ring-1 ${urgencyTone}`}>
                        {latestResult.urgency || "Unknown"}
                      </span>
                    </div>

                    <div className="rounded-[30px] border border-slate-200 bg-[linear-gradient(135deg,_rgba(14,165,233,0.12),_rgba(255,255,255,0.96),_rgba(244,114,182,0.08))] p-6">
                      <p className="text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">Response</p>
                      <div className="mt-4 overflow-x-auto rounded-[24px] bg-slate-950 p-5 shadow-[inset_0_1px_0_rgba(255,255,255,0.04)]">
                        <pre className="whitespace-pre-wrap break-words font-mono text-sm leading-7 text-emerald-100">
                          {primaryResponseText}
                        </pre>
                      </div>
                    </div>

                    <div className="rounded-[30px] border border-slate-200 bg-slate-950 p-3 shadow-[0_18px_40px_rgba(2,6,23,0.22)]">
                      <button
                        onClick={() => setExpandedRaw(!expandedRaw)}
                        className="flex w-full items-center justify-between rounded-[22px] px-3 py-3 text-left text-sm font-semibold text-white transition hover:bg-white/5"
                      >
                        <div>
                          <p>Raw JSON</p>
                          <p className="mt-1 text-xs font-medium text-slate-400">Inspect the structured payload returned by the pipeline</p>
                        </div>
                        <ChevronDown size={18} className={`transition-transform duration-300 ${expandedRaw ? "rotate-180" : ""}`} />
                      </button>

                      <AnimatePresence initial={false}>
                        {expandedRaw && (
                          <motion.div
                            initial={{ opacity: 0, height: 0 }}
                            animate={{ opacity: 1, height: "auto" }}
                            exit={{ opacity: 0, height: 0 }}
                            transition={{ duration: 0.2 }}
                            className="overflow-hidden"
                          >
                            <div className="pt-3">
                              <JSONViewer data={latestResult} />
                            </div>
                          </motion.div>
                        )}
                      </AnimatePresence>
                    </div>
                  </div>
                ) : (
                  <p className="text-sm text-slate-500">Run analysis to see results.</p>
                )}
              </section>
            </div>
          </div>
        </main>
      </div>
    </div>
  );
}
