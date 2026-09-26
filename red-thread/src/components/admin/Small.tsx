'use client';

import { useFormStatus } from 'react-dom';
import { Icon, type IconName } from '@/components/Icon';

/** Submit button that asks first. */
export function ConfirmSubmit({
  message,
  children,
  className = 'btn btn-danger',
  icon = 'trash',
}: {
  message: string;
  children: React.ReactNode;
  className?: string;
  icon?: IconName;
}) {
  const { pending } = useFormStatus();
  return (
    <button
      type="submit"
      className={className}
      disabled={pending}
      onClick={(event) => {
        if (!window.confirm(message)) event.preventDefault();
      }}
    >
      <Icon name={icon} size={16} />
      {children}
    </button>
  );
}

export function SaveButton({ children = '保存', className = 'btn btn-red' }: { children?: React.ReactNode; className?: string }) {
  const { pending } = useFormStatus();
  return (
    <button type="submit" className={className} disabled={pending}>
      <Icon name="check" size={17} />
      {pending ? '保存中…' : children}
    </button>
  );
}
