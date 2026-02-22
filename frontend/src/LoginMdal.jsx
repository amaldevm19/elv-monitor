import {useState, useEffect} from 'react';


import {
    Login, 
} from "../wailsjs/go/main/App"

export default function LoginModal({ open, onClose, onLoggedIn }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  useEffect(() => {
    if (!open) {
      setErr("");
      setBusy(false);
      setPassword("");
    }
  }, [open]);

  if (!open) return null;

  const submit = async () => {
    setErr("");
    if (!username.trim() || !password) return setErr("Enter username and password");
    setBusy(true);
    try {
      const dto = await Login(username.trim(), password);
      onLoggedIn(dto);
      onClose();
    } catch (e) {
      setErr("Invalid credentials");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center">
      <div className="bg-white w-[380px] rounded-xl p-5">
        <div className="text-lg font-semibold">Login</div>

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
          {err && <div className="text-red-600 text-sm">{err}</div>}
        </div>

        <div className="mt-5 flex justify-end gap-2">
          <button className="px-4 py-2 rounded border" onClick={onClose} disabled={busy}>
            Cancel
          </button>
          <button className="px-4 py-2 rounded bg-blue-600 text-white disabled:opacity-50"
                  onClick={submit}
                  disabled={busy}>
            {busy ? "Signing in..." : "Login"}
          </button>
        </div>
      </div>
    </div>
  );
}