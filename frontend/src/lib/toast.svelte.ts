export type ToastType = 'success' | 'error' | 'warning';

export type ToastItem = {
  id: number;
  message: string;
  type: ToastType;
};

let nextId = 1;
export const toasts = $state<ToastItem[]>([]);

export function toast(message: string, type: ToastType = 'success', durationMs = 4000) {
  const id = nextId++;
  toasts.push({ id, message, type });
  setTimeout(() => dismiss(id), durationMs);
}

export function dismiss(id: number) {
  const index = toasts.findIndex((t) => t.id === id);
  if (index !== -1) toasts.splice(index, 1);
}
