import type { Action } from 'svelte/action';

/** Closes an open menu on an outside pointerdown or Escape. Shared by every
    dropdown so the open/close logic itself stays in the component. */
export const dismissible: Action<HTMLElement, () => void> = (node, onClose) => {
	function handlePointer(event: PointerEvent) {
		if (!node.contains(event.target as Node)) onClose();
	}
	function handleKey(event: KeyboardEvent) {
		if (event.key === 'Escape') onClose();
	}
	document.addEventListener('pointerdown', handlePointer, true);
	document.addEventListener('keydown', handleKey);
	return {
		update(next) {
			onClose = next;
		},
		destroy() {
			document.removeEventListener('pointerdown', handlePointer, true);
			document.removeEventListener('keydown', handleKey);
		}
	};
};
