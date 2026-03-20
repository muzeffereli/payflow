import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Mail, Lock, CreditCard, ArrowRight, Shield, Zap, Globe } from 'lucide-react';
import { authApi, getApiErrorMessage } from '../lib/api';
import { useAuthStore } from '../lib/store';
import { Input } from '../components/ui/Input';
import { Button } from '../components/ui/Button';

export function Login() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { setAuth } = useAuthStore();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await authApi.login({ email, password }); // sets httpOnly cookies
      const meRes = await authApi.me();
      setAuth(meRes.data);
      navigate('/dashboard');
    } catch (error: unknown) {
      setError(getApiErrorMessage(error, 'Login failed'));
    } finally {
      setLoading(false);
    }
  };

  const features = [
    { icon: <Shield size={16} />, label: 'Enterprise-grade security' },
    { icon: <Zap size={16} />,    label: 'Real-time event processing' },
    { icon: <Globe size={16} />,  label: 'Multi-currency support' },
  ];

  return (
    <div className="min-h-screen flex">
      <div className="hidden lg:flex lg:w-[52%] relative overflow-hidden" style={{ background: 'var(--color-surface)' }}>
        <div className="absolute inset-0 dot-grid" />
        <div className="absolute top-1/4 left-1/2 -translate-x-1/2 w-[420px] h-[420px] bg-indigo-600/[0.07] rounded-full blur-[100px]" />

        <div className="relative flex flex-col justify-between p-10 w-full">
          <div className="flex items-center gap-2.5">
            <div className="w-7 h-7 rounded-lg bg-indigo-600 flex items-center justify-center">
              <CreditCard size={13} className="text-white" />
            </div>
            <span className="text-sm font-semibold text-zinc-100">PayFlow</span>
          </div>

          <div className="max-w-md">
            <h1 className="text-3xl font-semibold text-zinc-100 leading-snug tracking-tight mb-3">
              Manage payments{' '}
              <span className="text-indigo-400">with confidence</span>
            </h1>
            <p className="text-zinc-500 text-[15px] leading-relaxed mb-8">
              Process orders, track payments, and grow your business with a modern platform built for scale.
            </p>

            <div className="space-y-3">
              {features.map((f) => (
                <div key={f.label} className="flex items-center gap-3 text-sm text-zinc-400">
                  <div className="w-8 h-8 rounded-lg bg-white/5 ring-1 ring-white/[0.06] flex items-center justify-center text-indigo-400 shrink-0">
                    {f.icon}
                  </div>
                  {f.label}
                </div>
              ))}
            </div>
          </div>

          <p className="text-xs text-zinc-700">&copy; 2026 PayFlow</p>
        </div>
      </div>

      <div className="flex-1 flex items-center justify-center p-6" style={{ background: 'var(--color-bg)' }}>
        <div className="w-full max-w-[340px] animate-fade-in">
          <div className="lg:hidden flex items-center gap-2.5 mb-8">
            <div className="w-7 h-7 rounded-lg bg-indigo-600 flex items-center justify-center">
              <CreditCard size={13} className="text-white" />
            </div>
            <span className="text-sm font-semibold text-zinc-100">PayFlow</span>
          </div>

          <h2 className="text-xl font-semibold text-zinc-100 tracking-tight">Sign in</h2>
          <p className="text-zinc-500 mt-1 text-sm mb-6">Enter your credentials to continue</p>

          <form onSubmit={handleSubmit} className="space-y-4">
            <Input label="Email" type="email" placeholder="you@example.com" value={email} onChange={(e) => setEmail(e.target.value)} icon={<Mail size={14} />} required />
            <Input label="Password" type="password" placeholder="Your password" value={password} onChange={(e) => setPassword(e.target.value)} icon={<Lock size={14} />} required />

            {error && (
              <div className="rounded-lg bg-red-500/10 px-3 py-2 text-sm text-red-400">
                {error}
              </div>
            )}

            <Button type="submit" loading={loading} className="w-full" size="lg">
              Continue <ArrowRight size={15} />
            </Button>
          </form>

          <p className="text-center text-sm text-zinc-600 mt-6">
            No account?{' '}
            <Link to="/register" className="text-indigo-400 hover:text-indigo-300 transition-colors duration-150">
              Sign up
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
