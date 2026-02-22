

export default function LoginRequiredBanner({ open, info, onClose, onLogin }) {
  if (!open) return null;

  const action = info?.action ?? "this action";

  return (
    <div className="fixed top-3 left-1/2 -translate-x-1/2 bg-yellow-100 border border-yellow-400 text-yellow-900 px-4 py-2 rounded-lg shadow">
      <div className="flex items-center gap-3">
        <div className="text-sm">
          Login is required to perform: <b>{action}</b>
        </div>
      </div>
    </div>
  );
}