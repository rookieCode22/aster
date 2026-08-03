import { useState } from 'react';
import { Navigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

export default function Login() {
  const { user, login, register } = useAuth();
  const [isRegister, setIsRegister] = useState(false);
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setBusy(true);
    try {
      if (isRegister) {
        await register(username, email, password);
      } else {
        await login(username, password);
      }
    } catch (err: any) {
      setError(err.message || 'Authentication failed');
    } finally {
      setBusy(false);
    }
  };

  // 登录/注册成功后 user 变为有值，自动跳转到 Dashboard
  if (user) {
    return <Navigate to="/" replace />;
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-950 via-gray-900 to-gray-950">
      <div className="w-full max-w-md p-8 space-y-6">
        {/* Logo */}
        <div className="text-center space-y-2">
          <div className="text-5xl mb-4">🛡️</div>
          <h1 className="text-3xl font-bold text-white tracking-tight">Aster</h1>
          <p className="text-gray-400 text-sm">AI-Powered Security Agent Platform</p>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="space-y-4 bg-gray-900/50 border border-gray-800 rounded-xl p-6 backdrop-blur">
          <h2 className="text-lg font-semibold text-gray-200">
            {isRegister ? 'Create Account' : 'Sign In'}
          </h2>

          <input
            type="text"
            placeholder="Username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            className="w-full px-4 py-2.5 bg-gray-800 border border-gray-700 rounded-lg text-gray-100 placeholder-gray-500 focus:outline-none focus:border-emerald-500 transition"
            required
          />

          {isRegister && (
            <input
              type="email"
              placeholder="Email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full px-4 py-2.5 bg-gray-800 border border-gray-700 rounded-lg text-gray-100 placeholder-gray-500 focus:outline-none focus:border-emerald-500 transition"
              required
            />
          )}

          <input
            type="password"
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full px-4 py-2.5 bg-gray-800 border border-gray-700 rounded-lg text-gray-100 placeholder-gray-500 focus:outline-none focus:border-emerald-500 transition"
            required
            minLength={6}
          />

          {error && (
            <div className="text-red-400 text-sm bg-red-900/20 border border-red-800 rounded-lg px-3 py-2">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={busy}
            className="w-full py-2.5 bg-emerald-600 hover:bg-emerald-500 disabled:bg-emerald-800 disabled:cursor-not-allowed text-white font-medium rounded-lg transition"
          >
            {busy ? 'Processing...' : isRegister ? 'Create Account' : 'Sign In'}
          </button>

          <button
            type="button"
            onClick={() => { setIsRegister(!isRegister); setError(''); }}
            className="w-full text-sm text-gray-400 hover:text-emerald-400 transition"
          >
            {isRegister ? 'Already have an account? Sign In' : "Don't have an account? Register"}
          </button>
        </form>
      </div>
    </div>
  );
}
