import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';

export type Year = {
	id: string;
	name: string;
	starts_on: string;
	ends_on: string;
	is_current: boolean;
};

export type Term = Year & { year_id: string };

export type Klass = { id: string; name: string; rank: number };

export type Section = {
	section: { id: string; class_id: string; name: string; capacity: number | null };
	class_name: string;
	teacher_name: string | null;
	students: number;
};

export type Subject = {
	subject: { id: string; name: string; code: string; class_id: string | null };
	class_name: string | null;
};

export const load: PageServerLoad = async ({ locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	const [years, arrangement] = await Promise.all([
		api<{ years: Year[]; terms: Term[] }>('/v1/academics/years', scoped),
		api<{ classes: Klass[]; sections: Section[]; subjects: Subject[] }>(
			'/v1/academics/classes',
			scoped
		)
	]);

	return {
		years: years.years ?? [],
		terms: years.terms ?? [],
		classes: arrangement.classes ?? [],
		sections: arrangement.sections ?? [],
		subjects: arrangement.subjects ?? []
	};
};

const scoped = (
	locals: App.Locals,
	cookies: { get: (name: string) => string | undefined },
	fetch: typeof globalThis.fetch
) => {
	const tenant = cookies.get('fajr_tenant');
	if (!locals.token || !tenant) redirect(303, '/login');
	return { token: locals.token, tenant, fetch };
};

const failed = (cause: unknown) => {
	if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
	throw cause;
};

export const actions: Actions = {
	addYear: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api('/v1/academics/years', {
				method: 'POST',
				body: {
					name: String(form.get('name') ?? '').trim(),
					starts_on: String(form.get('starts_on') ?? ''),
					ends_on: String(form.get('ends_on') ?? '')
				},
				...scoped(locals, cookies, fetch)
			});
			return { saved: true };
		} catch (cause) {
			return failed(cause);
		}
	},

	currentYear: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/academics/years/${form.get('id')}/current`, {
				method: 'PUT',
				...scoped(locals, cookies, fetch)
			});
			return { saved: true };
		} catch (cause) {
			return failed(cause);
		}
	},

	addTerm: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/academics/years/${form.get('year_id')}/terms`, {
				method: 'POST',
				body: {
					name: String(form.get('name') ?? '').trim(),
					starts_on: String(form.get('starts_on') ?? ''),
					ends_on: String(form.get('ends_on') ?? '')
				},
				...scoped(locals, cookies, fetch)
			});
			return { saved: true };
		} catch (cause) {
			return failed(cause);
		}
	},

	currentTerm: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/academics/terms/${form.get('id')}/current`, {
				method: 'PUT',
				...scoped(locals, cookies, fetch)
			});
			return { saved: true };
		} catch (cause) {
			return failed(cause);
		}
	},

	addClass: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api('/v1/academics/classes', {
				method: 'POST',
				body: {
					name: String(form.get('name') ?? '').trim(),
					rank: Number(form.get('rank') ?? 0)
				},
				...scoped(locals, cookies, fetch)
			});
			return { saved: true };
		} catch (cause) {
			return failed(cause);
		}
	},

	removeClass: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/academics/classes/${form.get('id')}`, {
				method: 'DELETE',
				...scoped(locals, cookies, fetch)
			});
			return { removed: true };
		} catch (cause) {
			return failed(cause);
		}
	},

	addSection: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const capacity = Number(form.get('capacity') ?? 0);
		try {
			await api(`/v1/academics/classes/${form.get('class_id')}/sections`, {
				method: 'POST',
				body: {
					name: String(form.get('name') ?? '').trim(),
					capacity: capacity > 0 ? capacity : null
				},
				...scoped(locals, cookies, fetch)
			});
			return { saved: true };
		} catch (cause) {
			return failed(cause);
		}
	},

	removeSection: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/academics/sections/${form.get('id')}`, {
				method: 'DELETE',
				...scoped(locals, cookies, fetch)
			});
			return { removed: true };
		} catch (cause) {
			return failed(cause);
		}
	},

	addSubject: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const classID = String(form.get('class_id') ?? '');
		try {
			await api('/v1/academics/subjects', {
				method: 'POST',
				body: {
					name: String(form.get('name') ?? '').trim(),
					code: String(form.get('code') ?? '').trim(),
					class_id: classID || null
				},
				...scoped(locals, cookies, fetch)
			});
			return { saved: true };
		} catch (cause) {
			return failed(cause);
		}
	},

	removeSubject: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/academics/subjects/${form.get('id')}`, {
				method: 'DELETE',
				...scoped(locals, cookies, fetch)
			});
			return { removed: true };
		} catch (cause) {
			return failed(cause);
		}
	}
};
