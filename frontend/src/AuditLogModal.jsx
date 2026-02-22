import { useEffect, useState } from "react";
import { GetAuditLogs } from "../wailsjs/go/main/App";


function fmtTs(ts) {
  if (!ts) return "-";
  const d = new Date(ts);
  if (isNaN(d.getTime())) return ts; // if backend returns raw string
  return d.toLocaleString();
}

export default function AuditLogModal({ open, onClose }) {
  const [rows, setRows] = useState([]);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [q, setQ] = useState("");

  useEffect(() => {
    if (!open) return;
    (async () => {
      setBusy(true);
      setErr("");
      try {
        const data = await GetAuditLogs(500);
        setRows(Array.isArray(data) ? data : []);
      } catch (e) {
        setErr(String(e?.message ?? e));
      } finally {
        setBusy(false);
      }
    })();
  }, [open]);

  if (!open) return null;

  const filtered = rows.filter((r) => {
    if (!q.trim()) return true;
    const s = (q || "").toLowerCase();
    return (
      String(r.action ?? "").toLowerCase().includes(s) ||
      String(r.username ?? "").toLowerCase().includes(s) ||
      String(r.target ?? "").toLowerCase().includes(s) ||
      String(r.detailsJson ?? r.detailsJSON ?? "").toLowerCase().includes(s)
    );
  });

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center">
      <div className="bg-white w-[1100px] max-w-[95vw] h-[650px] max-h-[90vh] rounded-xl p-5 flex flex-col">
        <div className="flex items-center justify-between">
          <div className="text-lg font-semibold">System Event Log</div>
          <button className="border px-3 py-1 rounded" onClick={onClose}>
            Close
          </button>
        </div>

        <div className="mt-3 flex items-center gap-2">
          <input
            className="border p-2 rounded w-[420px]"
            placeholder="Search (user, action, target, details...)"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
          <button
            className="border px-3 py-2 rounded"
            onClick={async () => {
              setBusy(true);
              setErr("");
              try {
                const data = await GetAuditLogs(500);
                setRows(Array.isArray(data) ? data : []);
              } catch (e) {
                setErr(String(e?.message ?? e));
              } finally {
                setBusy(false);
              }
            }}
            disabled={busy}
          >
            {busy ? "Loading..." : "Refresh"}
          </button>

          <div className="text-sm text-gray-600 ml-auto">
            Showing {filtered.length} / {rows.length}
          </div>
        </div>

        {err && <div className="mt-2 text-sm text-red-600">{err}</div>}

        <div className="mt-3 border rounded overflow-auto flex-1">
          <table className="w-full text-sm">
            <thead className="sticky top-0 bg-gray-100">
              <tr>
                <th className="text-left p-2 border-b">Time</th>
                <th className="text-left p-2 border-b">User</th>
                <th className="text-left p-2 border-b">Action</th>
                <th className="text-left p-2 border-b">Target</th>
                <th className="text-left p-2 border-b">Details</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((r) => (
                <tr key={r.id} className="odd:bg-white even:bg-gray-50">
                  <td className="p-2 border-b whitespace-nowrap">{fmtTs(r.ts)}</td>
                  <td className="p-2 border-b whitespace-nowrap">{r.username || "-"}</td>
                  <td className="p-2 border-b whitespace-nowrap">{r.action || "-"}</td>
                  <td className="p-2 border-b whitespace-nowrap">{r.target || "-"}</td>
                  <td className="p-2 border-b break-all">
                    {r.detailsJson ?? r.detailsJSON ?? "-"}
                  </td>
                </tr>
              ))}
              {filtered.length === 0 && !busy && (
                <tr>
                  <td className="p-3 text-gray-500" colSpan={5}>
                    No events found.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        <div className="mt-3 text-xs text-gray-500">
          Tip: You can filter by username, action (MONITOR_START, DEVICE_IMPORT, QUIT_DENIED…), IP, etc.
        </div>
      </div>
    </div>
  );
}