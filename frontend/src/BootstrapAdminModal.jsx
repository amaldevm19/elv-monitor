import {useState, useEffect} from 'react';

import {
    CreateFirstAdminUser, 
} from "../wailsjs/go/main/App"

export default function BootstrapAdminModal({ open, onClose, onCreated }) {
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("");
  const [password2, setPassword2] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  useEffect(() => {
    if (!open) {
      setErr("");
      setBusy(false);
      setPassword("");
      setPassword2("");
    }
  }, [open]);

  if (!open) return null;

  const submit = async () => {
    setErr("");
    if (!username.trim()) return setErr("Username is required");
    if (!password) return setErr("Password is required");
    if (password.length < 6) return setErr("Password must be at least 6 characters");
    if (password !== password2) return setErr("Passwords do not match");

    setBusy(true);
    try {
      await CreateFirstAdminUser(username.trim(), password);
      onCreated(username.trim(), password); // optionally auto-login after create
    } catch (e) {
      setErr(String(e?.message ?? e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center">
      <div className="bg-white w-[420px] rounded-xl p-5">
        <div className="text-lg font-semibold">Create Admin (First Run)</div>
        <div className="text-sm text-gray-600 mt-1">
          No users exist yet. Create the first admin account.
        </div>

        <div className="mt-4 space-y-3">
          <input className="w-full border p-2 rounded"
                 placeholder="Username"
                 value={username}
                 onChange={(e)=>setUsername(e.target.value)} />
          <input className="w-full border p-2 rounded"
                 placeholder="Password"
                 type="password"
                 value={password}
                 onChange={(e)=>setPassword(e.target.value)} />
          <input className="w-full border p-2 rounded"
                 placeholder="Confirm Password"
                 type="password"
                 value={password2}
                 onChange={(e)=>setPassword2(e.target.value)} />
          {err && <div className="text-red-600 text-sm">{err}</div>}
        </div>

        <div className="mt-5 flex justify-end gap-2">
          {/* Do NOT allow close if bootstrap required */}
          <button className="px-4 py-2 rounded bg-blue-600 text-white disabled:opacity-50"
                  onClick={submit}
                  disabled={busy}>
            {busy ? "Creating..." : "Create Admin"}
          </button>
        </div>
      </div>
    </div>
  );
}