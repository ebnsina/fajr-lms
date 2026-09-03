<script lang="ts">
	import { dismissible } from '$lib/actions/dismiss';
	import Check from '@lucide/svelte/icons/check';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import Search from '@lucide/svelte/icons/search';

	type Option = { value: string; label: string; hint?: string };

	let {
		options,
		value = $bindable(),
		label,
		id,
		placeholder = 'Choose one',
		searchFrom = 8,
		onchange
	}: {
		options: Option[];
		value: string | null;
		label: string;
		id: string;
		placeholder?: string;
		/** Above this many options it becomes a combobox with a filter. */
		searchFrom?: number;
		onchange?: (value: string) => void;
	} = $props();

	let open = $state(false);
	let query = $state('');
	let active = $state(0);
	let trigger = $state<HTMLButtonElement | null>(null);
	let list = $state<HTMLUListElement | null>(null);
	let field = $state<HTMLInputElement | null>(null);

	let searchable = $derived(options.length >= searchFrom);
	let shown = $derived(
		query.trim() === ''
			? options
			: options.filter((o) => o.label.toLowerCase().includes(query.trim().toLowerCase()))
	);
	let chosen = $derived(options.find((o) => o.value === value) ?? null);

	function close(returnFocus = true) {
		open = false;
		query = '';
		if (returnFocus) trigger?.focus({ preventScroll: true });
	}

	function pick(option: Option) {
		value = option.value;
		onchange?.(option.value);
		close();
	}

	function onKey(event: KeyboardEvent) {
		if (event.key === 'ArrowDown') {
			event.preventDefault();
			active = Math.min(active + 1, shown.length - 1);
		} else if (event.key === 'ArrowUp') {
			event.preventDefault();
			active = Math.max(active - 1, 0);
		} else if (event.key === 'Home') {
			event.preventDefault();
			active = 0;
		} else if (event.key === 'End') {
			event.preventDefault();
			active = shown.length - 1;
		} else if (event.key === 'Enter' && shown[active]) {
			event.preventDefault();
			pick(shown[active]);
		}
	}

	// preventScroll, because focusing inside an absolutely positioned panel
	// otherwise scrolls every ancestor and shifts the page under the field.
	$effect(() => {
		if (!open) return;
		active = Math.max(0, shown.findIndex((o) => o.value === value));
		if (searchable) field?.focus({ preventScroll: true });
		else list?.focus({ preventScroll: true });
	});

	// scrollIntoView would scroll ancestors too, so the list scrolls itself.
	$effect(() => {
		if (!open || !list) return;
		const item = list.querySelector<HTMLElement>('[data-active="true"]');
		if (!item) return;
		const top = item.getBoundingClientRect().top - list.getBoundingClientRect().top + list.scrollTop;
		if (top < list.scrollTop) list.scrollTop = top;
		else if (top + item.offsetHeight > list.scrollTop + list.clientHeight)
			list.scrollTop = top + item.offsetHeight - list.clientHeight;
	});
</script>

<div class="relative">
	<button
		bind:this={trigger}
		{id}
		class="field flex items-center gap-2 text-start"
		type="button"
		role="combobox"
		aria-expanded={open}
		aria-controls="{id}-listbox"
		aria-haspopup="listbox"
		aria-label={label}
		onclick={() => (open = !open)}
	>
		<span class="min-w-0 flex-1 truncate" class:text-ink-faint={!chosen} dir="auto">
			{chosen?.label ?? placeholder}
		</span>
		<ChevronDown
			class="shrink-0 text-ink-faint transition-transform duration-150 motion-reduce:transition-none"
			style={open ? 'transform: rotate(180deg)' : ''}
			size={15}
			aria-hidden="true"
		/>
	</button>

	{#if open}
		<div
			class="absolute inset-x-0 top-full z-50 mt-1.5 overflow-hidden rounded-xl border border-line-strong bg-surface"
			use:dismissible={close}
		>
			{#if searchable}
				<!-- A filter appears only once the list is long enough to need one. -->
				<div class="flex items-center gap-2 border-b border-line px-3">
					<Search class="shrink-0 text-ink-faint" size={15} aria-hidden="true" />
					<input
						bind:this={field}
						bind:value={query}
						class="w-full bg-transparent py-2.5 text-sm outline-none"
						type="text"
						placeholder="filter"
						aria-label="Filter {label}"
						aria-controls="{id}-listbox"
						onkeydown={onKey}
					/>
				</div>
			{/if}

			<ul
				bind:this={list}
				id="{id}-listbox"
				class="max-h-64 list-none overflow-y-auto p-1 outline-none"
				role="listbox"
				aria-label={label}
				tabindex="-1"
				onkeydown={onKey}
			>
				{#each shown as option, index (option.value)}
					<li>
						<button
							class="flex w-full items-center gap-2 rounded-xl px-3 py-2 text-start text-sm transition-colors"
							class:bg-sunken={index === active}
							type="button"
							role="option"
							aria-selected={option.value === value}
							data-active={index === active}
							onmouseenter={() => (active = index)}
							onclick={() => pick(option)}
						>
							<span class="min-w-0 flex-1">
								<span class="block truncate" dir="auto">{option.label}</span>
								{#if option.hint}
									<span class="block truncate text-xs text-ink-soft" dir="auto">{option.hint}</span>
								{/if}
							</span>
							{#if option.value === value}
								<Check class="shrink-0 text-brand-text" size={15} aria-hidden="true" />
							{/if}
						</button>
					</li>
				{:else}
					<li class="px-3 py-2 text-sm text-ink-faint" dir="auto">Nothing matches that.</li>
				{/each}
			</ul>
		</div>
	{/if}
</div>
