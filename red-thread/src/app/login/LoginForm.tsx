'use client';

import { useActionState } from 'react';
import { login, type LoginState } from './actions';
import { Icon } from '@/components/Icon';

export function LoginForm({ next, hint }: { next: string; hint: string | null }) {
  const [state, action, pending] = useActionState<LoginState, FormData>(login, {});
  return (
    <form action={action} className="login-form">
      <input type="hidden" name="next" value={next} />
      <label className="field">
        <span>邮箱</span>
        <input name="email" type="email" required autoComplete="username" defaultValue={state.email} key={state.email} />
      </label>
      <label className="field">
        <span>暗号</span>
        <input name="password" type="password" required autoComplete="current-password" />
      </label>
      {state.error ? <p className="notice notice-red">{state.error}</p> : null}
      {hint ? <p className="notice">{hint}</p> : null}
      <button type="submit" className="btn btn-red" disabled={pending}>
        <Icon name="heart" size={17} filled />
        {pending ? '正在开门…' : '回家'}
      </button>
    </form>
  );
}
