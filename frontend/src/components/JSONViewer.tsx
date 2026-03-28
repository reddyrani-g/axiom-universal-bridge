"use client";

import React, { useState } from "react";
import { ChevronRight } from "lucide-react";

interface JSONViewerProps {
  data: any;
  indent?: number;
  label?: string;
}

const INDENT_PX = 18;

function getValueTone(value: unknown) {
  if (value === null) return "text-slate-500";
  if (typeof value === "boolean") return "text-sky-300";
  if (typeof value === "number") return "text-emerald-300";
  if (typeof value === "string") return "text-amber-200";
  return "text-slate-300";
}

function getPreview(value: unknown) {
  if (Array.isArray(value)) return `${value.length} item${value.length === 1 ? "" : "s"}`;
  if (value && typeof value === "object") return `${Object.keys(value).length} field${Object.keys(value).length === 1 ? "" : "s"}`;
  if (typeof value === "string") return `"${value.length > 40 ? `${value.slice(0, 40)}...` : value}"`;
  return String(value);
}

function JSONNode({ data, indent = 0, label }: JSONViewerProps) {
  const [collapsed, setCollapsed] = useState(indent > 1);

  const isArray = Array.isArray(data);
  const isObject = Boolean(data) && typeof data === "object" && !isArray;
  const isCollection = isArray || isObject;

  if (!isCollection) {
    return (
      <div className="flex items-start gap-2 leading-6">
        {label && <span className="json-key">"{label}"</span>}
        {label && <span className="text-slate-500">:</span>}
        <span className={getValueTone(data)}>{typeof data === "string" ? `"${data}"` : String(data)}</span>
      </div>
    );
  }

  const entries = isArray ? data.map((value, index) => [String(index), value] as const) : Object.entries(data);
  const openingBracket = isArray ? "[" : "{";
  const closingBracket = isArray ? "]" : "}";

  if (entries.length === 0) {
    return (
      <div className="flex items-start gap-2 leading-6">
        {label && <span className="json-key">"{label}"</span>}
        {label && <span className="text-slate-500">:</span>}
        <span className="text-slate-500">
          {openingBracket}
          {closingBracket}
        </span>
      </div>
    );
  }

  return (
    <div>
      <button
        type="button"
        onClick={() => setCollapsed(prev => !prev)}
        className="group flex w-full items-start gap-2 rounded-lg px-2 py-1 text-left transition hover:bg-white/5"
      >
        <ChevronRight
          size={14}
          className={`mt-1 flex-shrink-0 text-slate-500 transition-transform ${collapsed ? "" : "rotate-90"}`}
        />
        <div className="min-w-0 flex-1 leading-6">
          <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
            {label && <span className="json-key">"{label}"</span>}
            {label && <span className="text-slate-500">:</span>}
            <span className="text-slate-400">
              {openingBracket}
              {collapsed ? closingBracket : ""}
            </span>
            <span className="rounded-full border border-white/10 bg-white/5 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-[0.18em] text-slate-400">
              {getPreview(data)}
            </span>
          </div>
          {collapsed && <p className="mt-1 truncate text-xs text-slate-500">{entries.slice(0, 3).map(([, value]) => getPreview(value)).join(" • ")}</p>}
        </div>
      </button>

      {!collapsed && (
        <div
          className="mt-1 space-y-1 border-l border-white/10"
          style={{ marginLeft: `${indent === 0 ? 0 : 8}px`, paddingLeft: `${INDENT_PX}px` }}
        >
          {entries.map(([entryLabel, value]) => (
            <JSONNode
              key={`${label ?? "root"}-${entryLabel}`}
              data={value}
              indent={indent + 1}
              label={isArray ? undefined : entryLabel}
            />
          ))}
          <div className="pl-2 text-slate-500">{closingBracket}</div>
        </div>
      )}
    </div>
  );
}

export default function JSONViewer({ data }: JSONViewerProps) {
  return (
    <div className="rounded-2xl border border-white/10 bg-[#07111f] p-3 text-xs font-medium text-slate-200 shadow-[inset_0_1px_0_rgba(255,255,255,0.03)]">
      <JSONNode data={data} />
    </div>
  );
}
