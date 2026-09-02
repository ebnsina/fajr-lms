import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import type { Course } from '$lib/api';

type Item = {
	id: string;
	source: 'quiz' | 'assignment' | 'manual';
	title: string;
	category: string;
	points_possible: number;
	weight: number;
};

type Score = {
	item_id: string;
	points: number | null;
	points_possible: number;
	percent: number | null;
	overridden: boolean;
	note?: string;
};

type Learner = {
	enrollment_id: string;
	full_name: string;
	scores: Score[];
	percent: number;
	items_graded: number;
	items_total: number;
};

export const load: PageServerLoad = async ({ params, locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	const course = await api<Course>(`/v1/courses/${params.slug}`, scoped).then(
		(outline: unknown) => (outline as { course: Course }).course
	);

	try {
		const book = await api<{ items: Item[]; learners: Learner[] }>(
			`/v1/courses/${course.id}/gradebook`,
			scoped
		);
		return { course, ...book, slug: params.slug };
	} catch (failure) {
		if (failure instanceof ApiFailure && failure.status === 403) {
			error(403, 'Only staff can see the whole class.');
		}
		throw failure;
	}
};

const scoped = (locals: App.Locals, cookies: { get: (n: string) => string | undefined }, fetch: typeof globalThis.fetch) => {
	const tenant = cookies.get('fajr_tenant');
	if (!locals.token || !tenant) redirect(303, '/login');
	return { token: locals.token, tenant, fetch };
};

export const actions: Actions = {
	// An empty box clears the teacher's score and lets the computed one apply again.
	setGrade: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const itemID = String(form.get('item_id') ?? '');
		const enrollmentID = String(form.get('enrollment_id') ?? '');
		const raw = String(form.get('points') ?? '').trim();
		const path = `/v1/grade-items/${itemID}/learners/${enrollmentID}`;

		try {
			if (raw === '') {
				await api(path, { method: 'DELETE', ...scoped(locals, cookies, fetch) }).catch(
					(failure) => {
						// Nothing to clear is the same outcome as clearing it.
						if (failure instanceof ApiFailure && failure.status === 404) return;
						throw failure;
					}
				);
				return { cleared: true };
			}

			const points = Number(raw);
			if (!Number.isFinite(points) || points < 0) {
				return fail(422, { message: 'A score is a number, zero or more.' });
			}
			await api(path, {
				method: 'PUT',
				body: { points: Math.trunc(points), note: String(form.get('note') ?? '') },
				...scoped(locals, cookies, fetch)
			});
		} catch (failure) {
			if (failure instanceof ApiFailure) return fail(failure.status, { message: failure.error.message });
			throw failure;
		}
		return { saved: true };
	},

	addItem: async ({ params, request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const title = String(form.get('title') ?? '').trim();
		const points = Number(form.get('points_possible'));
		const weight = Number(form.get('weight'));

		if (!title) return fail(422, { message: 'Give the item a name.' });
		if (!Number.isFinite(points) || points < 1) {
			return fail(422, { message: 'Say what it is out of.' });
		}

		try {
			await api(`/v1/courses/${form.get('course_id')}/grade-items`, {
				method: 'POST',
				body: {
					title,
					points_possible: Math.trunc(points),
					weight: Number.isFinite(weight) ? Math.trunc(weight) : 100,
					category: 'manual'
				},
				...scoped(locals, cookies, fetch)
			});
		} catch (failure) {
			if (failure instanceof ApiFailure) return fail(failure.status, { message: failure.error.message });
			throw failure;
		}
		return { added: true, slug: params.slug };
	}
};
