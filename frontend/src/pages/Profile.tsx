import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { User, Lock } from 'lucide-react';
import { toast } from 'sonner';
import api from '../lib/api';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Input } from '../components/ui/Input';
import { Badge } from '../components/ui/Badge';
import { useUser } from '../lib/store';

export function Profile() {
  const user = useUser();
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');

  const changePwMutation = useMutation({
    mutationFn: (data: { old_password: string; new_password: string }) =>
      api.post('/auth/change-password', data),
    onSuccess: () => {
      toast.success('Password changed successfully');
      setOldPassword('');
      setNewPassword('');
    },
    onError: (err: { response?: { data?: { error?: string } } }) => {
      toast.error(err.response?.data?.error ?? 'Failed to change password');
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (newPassword.length < 8) {
      toast.error('New password must be at least 8 characters');
      return;
    }
    changePwMutation.mutate({ old_password: oldPassword, new_password: newPassword });
  };

  return (
    <div className="animate-fade-in max-w-2xl">
      <div className="mb-8 flex items-center gap-4">
        <div className="w-11 h-11 rounded-xl bg-indigo-500/10 ring-1 ring-indigo-500/10 flex items-center justify-center">
          <User size={20} className="text-indigo-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">Profile</h1>
          <p className="text-zinc-500 text-sm">Manage your account settings</p>
        </div>
      </div>

      <Card className="mb-6">
        <div className="p-6 space-y-4">
          <h2 className="text-sm font-semibold text-zinc-200 mb-4">Account Information</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-xs text-zinc-500 mb-1">Name</p>
              <p className="text-sm text-zinc-200">{user?.name}</p>
            </div>
            <div>
              <p className="text-xs text-zinc-500 mb-1">Email</p>
              <p className="text-sm text-zinc-200">{user?.email}</p>
            </div>
            <div>
              <p className="text-xs text-zinc-500 mb-1">Role</p>
              <Badge status={user?.role ?? 'customer'} />
            </div>
            <div>
              <p className="text-xs text-zinc-500 mb-1">User ID</p>
              <p className="text-xs text-zinc-500 font-mono">{user?.id}</p>
            </div>
          </div>
        </div>
      </Card>

      <Card>
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          <div className="flex items-center gap-2 mb-2">
            <Lock size={16} className="text-zinc-400" />
            <h2 className="text-sm font-semibold text-zinc-200">Change Password</h2>
          </div>
          <Input
            label="Current Password"
            type="password"
            value={oldPassword}
            onChange={(e) => setOldPassword(e.target.value)}
            required
          />
          <Input
            label="New Password"
            type="password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            placeholder="At least 8 characters"
            required
          />
          <div className="flex justify-end pt-2">
            <Button type="submit" loading={changePwMutation.isPending} disabled={!oldPassword || !newPassword}>
              Update Password
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
