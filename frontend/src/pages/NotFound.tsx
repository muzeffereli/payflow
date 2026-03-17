import { Link } from 'react-router-dom';
import { Home } from 'lucide-react';
import { Button } from '../components/ui/Button';

export function NotFound() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-[var(--color-bg)]">
      <div className="text-center">
        <p className="text-7xl font-bold text-zinc-800 mb-4">404</p>
        <h1 className="text-2xl font-bold text-zinc-100 mb-2">Page not found</h1>
        <p className="text-zinc-500 mb-8 text-sm">
          The page you're looking for doesn't exist or has been moved.
        </p>
        <Link to="/dashboard">
          <Button>
            <Home size={14} /> Back to Dashboard
          </Button>
        </Link>
      </div>
    </div>
  );
}
