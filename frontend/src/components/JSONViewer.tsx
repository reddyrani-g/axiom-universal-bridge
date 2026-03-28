import React from "react";

interface JSONViewerProps {
  data: any;
  indent?: number;
}

const JSONViewer: React.FC<JSONViewerProps> = ({ data, indent = 0 }) => {
  if (data === null) return <span className="text-slate-400">null</span>;
  if (typeof data === "boolean") return <span className="text-indigo-400">{String(data)}</span>;
  if (typeof data === "number") return <span className="text-emerald-400">{data}</span>;
  if (typeof data === "string") return <span className="text-teal-400">"{data}"</span>;

  if (Array.isArray(data)) {
    if (data.length === 0) return <span className="text-slate-400">[]</span>;
    return (
      <div>
        <span className="text-slate-400">[</span>
        {data.map((item, idx) => (
          <div key={idx} style={{ marginLeft: `${indent + 1}em` }} className="text-slate-300">
            <JSONViewer data={item} indent={indent + 1} />
            {idx < data.length - 1 && <span className="text-slate-400">,</span>}
          </div>
        ))}
        <div style={{ marginLeft: `${indent}em` }} className="text-slate-400">
          ]
        </div>
      </div>
    );
  }

  if (typeof data === "object") {
    const keys = Object.keys(data);
    if (keys.length === 0) return <span className="text-slate-400">{"{}"}</span>;
    return (
      <div>
        <span className="text-slate-400">{"{"}</span>
        {keys.map((key, idx) => (
          <div key={key} style={{ marginLeft: `${indent + 1}em` }} className="text-slate-300">
            <span className="text-violet-400">"{key}"</span>
            <span className="text-slate-400">: </span>
            <JSONViewer data={data[key]} indent={indent + 1} />
            {idx < keys.length - 1 && <span className="text-slate-400">,</span>}
          </div>
        ))}
        <div style={{ marginLeft: `${indent}em` }} className="text-slate-400">
          {"}"}
        </div>
      </div>
    );
  }

  return <span className="text-slate-400">{String(data)}</span>;
};

export default JSONViewer;
